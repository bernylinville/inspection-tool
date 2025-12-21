// Package service provides business logic services for the inspection tool.
package service

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"inspection-tool/internal/client/n9e"
	"inspection-tool/internal/client/vm"
	"inspection-tool/internal/config"
	"inspection-tool/internal/model"
)

// NginxCollector is the data collection service for Nginx/OpenResty instances.
// It integrates with VictoriaMetrics to collect Nginx monitoring metrics
// and N9E to obtain host IP addresses.
type NginxCollector struct {
	vmClient       *vm.Client
	n9eClient      *n9e.Client // 用于获取 IP 地址
	config         *config.NginxInspectionConfig
	metrics        []*model.NginxMetricDefinition
	metricDefs     map[string]*model.NginxMetricDefinition
	instanceFilter *NginxInstanceFilter
	logger         zerolog.Logger
}

// NginxInstanceFilter defines filtering criteria for Nginx instances.
// Uses HostnamePatterns instead of AddressPatterns (different from MySQL).
type NginxInstanceFilter struct {
	HostnamePatterns []string          // Hostname patterns (glob, e.g., "GX-NM-*")
	BusinessGroups   []string          // Business groups (OR relation)
	Tags             map[string]string // Tags (AND relation)
}

// NewNginxCollector creates a new NginxCollector instance.
func NewNginxCollector(
	cfg *config.NginxInspectionConfig,
	vmClient *vm.Client,
	n9eClient *n9e.Client,
	metrics []*model.NginxMetricDefinition,
	logger zerolog.Logger,
) *NginxCollector {
	c := &NginxCollector{
		vmClient:  vmClient,
		n9eClient: n9eClient,
		config:    cfg,
		metrics:   metrics,
		logger:    logger.With().Str("component", "nginx-collector").Logger(),
	}

	// Build metric definitions map for fast lookup
	c.metricDefs = make(map[string]*model.NginxMetricDefinition, len(metrics))
	for _, m := range metrics {
		c.metricDefs[m.Name] = m
	}

	// Build instance filter from config
	c.instanceFilter = c.buildInstanceFilter()

	return c
}

// buildInstanceFilter converts config.NginxFilter to NginxInstanceFilter.
func (c *NginxCollector) buildInstanceFilter() *NginxInstanceFilter {
	if c.config == nil {
		return nil
	}

	filter := c.config.InstanceFilter
	if len(filter.HostnamePatterns) == 0 &&
		len(filter.BusinessGroups) == 0 &&
		len(filter.Tags) == 0 {
		return nil
	}

	return &NginxInstanceFilter{
		HostnamePatterns: filter.HostnamePatterns,
		BusinessGroups:   filter.BusinessGroups,
		Tags:             filter.Tags,
	}
}

// GetConfig returns the Nginx inspection configuration.
func (c *NginxCollector) GetConfig() *config.NginxInspectionConfig {
	return c.config
}

// GetMetrics returns the list of metric definitions.
func (c *NginxCollector) GetMetrics() []*model.NginxMetricDefinition {
	return c.metrics
}

// GetInstanceFilter returns the instance filter.
func (c *NginxCollector) GetInstanceFilter() *NginxInstanceFilter {
	return c.instanceFilter
}

// IsEmpty returns true if the instance filter has no filtering criteria.
func (f *NginxInstanceFilter) IsEmpty() bool {
	if f == nil {
		return true
	}
	return len(f.HostnamePatterns) == 0 &&
		len(f.BusinessGroups) == 0 &&
		len(f.Tags) == 0
}

// ToVMHostFilter converts NginxInstanceFilter to vm.HostFilter.
// Note: HostnamePatterns are not supported in vm.HostFilter and will be
// handled separately in the DiscoverInstances method.
func (f *NginxInstanceFilter) ToVMHostFilter() *vm.HostFilter {
	if f == nil || f.IsEmpty() {
		return nil
	}

	// Only include business groups and tags, hostname patterns are handled separately
	if len(f.BusinessGroups) == 0 && len(f.Tags) == 0 {
		return nil
	}

	return &vm.HostFilter{
		BusinessGroups: f.BusinessGroups,
		Tags:           f.Tags,
	}
}

// =============================================================================
// Nginx 实例发现
// =============================================================================

// DiscoverInstances discovers all Nginx instances by querying nginx_info metric.
// It extracts labels (agent_hostname, port, app_type, install_path, version)
// and retrieves IP addresses from N9E API.
//
// Returns a list of NginxInstance objects.
func (c *NginxCollector) DiscoverInstances(ctx context.Context) ([]*model.NginxInstance, error) {
	c.logger.Info().Msg("starting Nginx instance discovery")

	// Step 1: Query nginx_info to get all instances
	vmFilter := c.instanceFilter.ToVMHostFilter()
	results, err := c.vmClient.QueryResultsWithFilter(ctx, "nginx_info", vmFilter)
	if err != nil {
		c.logger.Error().Err(err).Msg("failed to query nginx_info metric")
		return nil, fmt.Errorf("failed to query nginx_info: %w", err)
	}

	c.logger.Debug().Int("raw_results", len(results)).Msg("received nginx_info query results")

	// Step 2: Query nginx_up to get container info (for container deployments)
	containerMap := c.getContainerMap(ctx, vmFilter)

	// Step 3: Extract instances and apply filters
	var instances []*model.NginxInstance
	seenIdentifiers := make(map[string]bool)

	for _, result := range results {
		// 3.1 Extract hostname
		hostname := result.Labels["agent_hostname"]
		if hostname == "" {
			// Fallback to ident label if agent_hostname is missing
			hostname = result.Labels["ident"]
			if hostname == "" {
				c.logger.Warn().Interface("labels", result.Labels).Msg("missing both agent_hostname and ident labels")
				continue
			}
		}

		// 3.2 Apply hostname pattern filter
		if !c.matchesHostnamePatterns(hostname) {
			c.logger.Debug().Str("hostname", hostname).Msg("hostname filtered out")
			continue
		}

		// 3.3 Extract port
		portStr := result.Labels["port"]
		port := 0
		if portStr != "" {
			if p, err := strconv.Atoi(portStr); err == nil {
				port = p
			}
		}

		// 3.4 Check for container deployment
		container := containerMap[hostname]

		// 3.5 Generate identifier and check for duplicates
		identifier := model.GenerateNginxIdentifier(hostname, port, container)
		if seenIdentifiers[identifier] {
			c.logger.Debug().Str("identifier", identifier).Msg("skipping duplicate instance")
			continue
		}

		// 3.6 Create instance
		var instance *model.NginxInstance
		if container != "" {
			instance = model.NewNginxInstanceWithContainer(hostname, container)
			if port > 0 {
				instance.Port = port
			}
		} else if port > 0 {
			instance = model.NewNginxInstance(hostname, port)
		} else {
			c.logger.Warn().
				Str("hostname", hostname).
				Msg("skipping instance: no port or container info")
			continue
		}

		if instance == nil {
			continue
		}

		// 3.7 Set additional fields from labels
		if appType := result.Labels["app_type"]; appType != "" {
			instance.SetApplicationType(appType)
		}
		if version := result.Labels["version"]; version != "" {
			instance.SetVersion(version)
		}
		if installPath := result.Labels["install_path"]; installPath != "" {
			instance.SetInstallPath(installPath)
		}

		// 3.8 Get IP address from N9E API
		ip := c.getIPFromN9E(ctx, hostname)
		instance.SetIP(ip)

		instances = append(instances, instance)
		seenIdentifiers[identifier] = true
	}

	c.logger.Info().
		Int("discovered", len(instances)).
		Int("filtered_out", len(results)-len(instances)).
		Msg("Nginx instance discovery completed")

	return instances, nil
}

// getContainerMap queries nginx_up to get container info for each hostname.
// Returns a map[hostname]container.
func (c *NginxCollector) getContainerMap(ctx context.Context, vmFilter *vm.HostFilter) map[string]string {
	containerMap := make(map[string]string)

	results, err := c.vmClient.QueryResultsWithFilter(ctx, "nginx_up", vmFilter)
	if err != nil {
		c.logger.Warn().Err(err).Msg("failed to query nginx_up for container info")
		return containerMap
	}

	for _, result := range results {
		// First try agent_hostname, fallback to ident
		hostname := result.Labels["agent_hostname"]
		if hostname == "" {
			hostname = result.Labels["ident"]
		}
		container := result.Labels["container"]
		if hostname != "" && container != "" {
			containerMap[hostname] = container
		}
	}

	return containerMap
}

// getIPFromN9E retrieves the IP address for a hostname from N9E API.
// Returns "N/A" if the hostname is not found or an error occurs.
func (c *NginxCollector) getIPFromN9E(ctx context.Context, hostname string) string {
	if c.n9eClient == nil {
		return "N/A"
	}

	hostMeta, err := c.n9eClient.GetHostMetaByIdent(ctx, hostname)
	if err != nil {
		c.logger.Debug().
			Err(err).
			Str("hostname", hostname).
			Msg("failed to get host meta from N9E")
		return "N/A"
	}

	if hostMeta == nil || hostMeta.IP == "" {
		return "N/A"
	}

	return hostMeta.IP
}

// matchesHostnamePatterns checks if a hostname matches any configured patterns.
// Returns true if no patterns configured or hostname matches at least one pattern.
func (c *NginxCollector) matchesHostnamePatterns(hostname string) bool {
	if c.instanceFilter == nil || len(c.instanceFilter.HostnamePatterns) == 0 {
		return true
	}

	for _, pattern := range c.instanceFilter.HostnamePatterns {
		if matchHostnamePattern(hostname, pattern) {
			return true
		}
	}

	return false
}

// matchHostnamePattern checks if a hostname matches a pattern with wildcard support.
// Supports wildcard '*' which matches any sequence of characters.
// Examples:
//   - "GX-NM-*" matches "GX-NM-MNS-NGX-01"
//   - "*-NGX-*" matches "GX-NM-MNS-NGX-01"
//   - "*" matches all hostnames
func matchHostnamePattern(hostname, pattern string) bool {
	// Exact match optimization
	if hostname == pattern {
		return true
	}

	// No wildcard
	if !strings.Contains(pattern, "*") {
		return false
	}

	// Convert to regex
	regexPattern := regexp.QuoteMeta(pattern)
	regexPattern = strings.ReplaceAll(regexPattern, "\\*", ".*")
	regexPattern = "^" + regexPattern + "$"

	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return false
	}

	return re.MatchString(hostname)
}

// =============================================================================
// Nginx 指标采集
// =============================================================================

// CollectMetrics retrieves metric data from VictoriaMetrics for all Nginx instances.
//
// Flow:
//  1. Initialize result objects for each instance
//  2. Separate pending and active metrics
//  3. Set N/A for pending metrics
//  4. Concurrently collect active metrics (errgroup + concurrency limit)
//  5. Return results map (key = identifier)
//
// Single metric failure does not abort the entire collection.
func (c *NginxCollector) CollectMetrics(
	ctx context.Context,
	instances []*model.NginxInstance,
	metrics []*model.NginxMetricDefinition,
) (map[string]*model.NginxInspectionResult, error) {
	c.logger.Debug().
		Int("instance_count", len(instances)).
		Int("metric_count", len(metrics)).
		Msg("collecting Nginx metrics from VictoriaMetrics")

	// Step 1: Initialize results map (indexed by identifier)
	resultsMap := make(map[string]*model.NginxInspectionResult, len(instances))
	for _, instance := range instances {
		resultsMap[instance.Identifier] = model.NewNginxInspectionResult(instance)
	}

	// Step 2: Separate pending and active metrics
	var pendingMetrics []*model.NginxMetricDefinition
	var activeMetrics []*model.NginxMetricDefinition

	for _, metric := range metrics {
		if metric.IsPending() {
			pendingMetrics = append(pendingMetrics, metric)
		} else {
			activeMetrics = append(activeMetrics, metric)
		}
	}

	// Step 3: Set N/A for pending metrics
	c.setPendingMetrics(resultsMap, pendingMetrics)

	if len(activeMetrics) == 0 {
		c.logger.Warn().Msg("no active metrics to collect")
		return resultsMap, nil
	}

	// Step 4: Concurrently collect active metrics
	g, ctx := errgroup.WithContext(ctx)
	concurrency := 20 // Default concurrency
	g.SetLimit(concurrency)

	var mu sync.Mutex // Protects resultsMap from concurrent writes

	for _, metric := range activeMetrics {
		metric := metric // Capture loop variable
		g.Go(func() error {
			err := c.collectMetricConcurrent(ctx, metric, instances, resultsMap, &mu)
			if err != nil {
				c.logger.Warn().
					Err(err).
					Str("metric", metric.Name).
					Msg("failed to collect metric, continuing with others")
			}
			return nil // Single metric failure does not abort
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("concurrent metric collection failed: %w", err)
	}

	// Step 5: Extract field values from metrics
	c.extractFieldsFromMetrics(resultsMap)

	c.logger.Info().
		Int("instances", len(instances)).
		Int("active_metrics", len(activeMetrics)).
		Int("pending_metrics", len(pendingMetrics)).
		Msg("Nginx metrics collection completed")

	return resultsMap, nil
}

// setPendingMetrics sets N/A values for all pending metrics on all instances.
func (c *NginxCollector) setPendingMetrics(
	resultsMap map[string]*model.NginxInspectionResult,
	pendingMetrics []*model.NginxMetricDefinition,
) {
	if len(pendingMetrics) == 0 {
		return
	}

	c.logger.Debug().
		Int("pending_count", len(pendingMetrics)).
		Msg("setting N/A for pending Nginx metrics")

	for _, metric := range pendingMetrics {
		for _, result := range resultsMap {
			mv := &model.NginxMetricValue{
				Name:           metric.Name,
				RawValue:       0,
				FormattedValue: "N/A",
				IsNA:           true,
			}
			result.SetMetric(mv)
		}
	}
}

// collectMetricConcurrent collects a single metric for all instances (concurrent-safe).
func (c *NginxCollector) collectMetricConcurrent(
	ctx context.Context,
	metric *model.NginxMetricDefinition,
	instances []*model.NginxInstance,
	resultsMap map[string]*model.NginxInspectionResult,
	mu *sync.Mutex,
) error {
	// If metric needs label extraction, use special handler
	if metric.HasLabelExtract() {
		return c.collectLabelExtractMetric(ctx, metric, instances, resultsMap, mu)
	}

	// Otherwise, collect numeric metric
	c.logger.Debug().
		Str("metric", metric.Name).
		Str("query", metric.Query).
		Msg("collecting Nginx metric (concurrent)")

	// Query VictoriaMetrics
	vmFilter := c.instanceFilter.ToVMHostFilter()
	results, err := c.vmClient.QueryResultsWithFilter(ctx, metric.Query, vmFilter)
	if err != nil {
		return fmt.Errorf("query failed for %s: %w", metric.Name, err)
	}

	// Create hostname map for fast lookup
	hostnameMap := make(map[string]*model.NginxInstance, len(instances))
	for _, instance := range instances {
		hostnameMap[instance.Hostname] = instance
	}

	// Match results to instances by hostname
	mu.Lock()
	defer mu.Unlock()

	matchedCount := 0
	for _, result := range results {
		// First try agent_hostname, fallback to ident
		hostname := result.Labels["agent_hostname"]
		if hostname == "" {
			hostname = result.Labels["ident"]
		}
		if hostname == "" {
			continue
		}

		// Apply hostname pattern filtering (post-filter)
		if !c.matchesHostnamePatterns(hostname) {
			continue
		}

		// Check if this hostname belongs to our instances
		instance, exists := hostnameMap[hostname]
		if !exists {
			continue
		}

		// Add metric value to result
		if inspResult, ok := resultsMap[instance.Identifier]; ok {
			mv := &model.NginxMetricValue{
				Name:      metric.Name,
				RawValue:  result.Value,
				Timestamp: time.Now().Unix(),
				Labels:    result.Labels,
			}
			inspResult.SetMetric(mv)
			matchedCount++
		}
	}

	c.logger.Debug().
		Str("metric", metric.Name).
		Int("results", len(results)).
		Int("matched", matchedCount).
		Msg("Nginx metric collected (concurrent)")

	return nil
}

// collectLabelExtractMetric collects metrics that extract values from labels.
// Handles metrics like nginx_info (extracts port, app_type, install_path, version)
// and nginx_last_error_timestamp (extracts error_log_path).
func (c *NginxCollector) collectLabelExtractMetric(
	ctx context.Context,
	metric *model.NginxMetricDefinition,
	instances []*model.NginxInstance,
	resultsMap map[string]*model.NginxInspectionResult,
	mu *sync.Mutex,
) error {
	c.logger.Debug().
		Str("metric", metric.Name).
		Interface("label_extract", metric.LabelExtract).
		Msg("collecting label extract metric")

	vmFilter := c.instanceFilter.ToVMHostFilter()
	results, err := c.vmClient.QueryResultsWithFilter(ctx, metric.Query, vmFilter)
	if err != nil {
		return fmt.Errorf("query failed for %s: %w", metric.Name, err)
	}

	// Create hostname map for fast lookup
	hostnameMap := make(map[string]*model.NginxInstance, len(instances))
	for _, instance := range instances {
		hostnameMap[instance.Hostname] = instance
	}

	mu.Lock()
	defer mu.Unlock()

	matchedCount := 0
	for _, result := range results {
		// First try agent_hostname, fallback to ident
		hostname := result.Labels["agent_hostname"]
		if hostname == "" {
			hostname = result.Labels["ident"]
		}
		if hostname == "" {
			continue
		}

		// Apply hostname pattern filtering
		if !c.matchesHostnamePatterns(hostname) {
			continue
		}

		// Check if this hostname belongs to our instances
		instance, exists := hostnameMap[hostname]
		if !exists {
			continue
		}

		// Build string value from extracted labels
		var extractedValues []string
		for _, label := range metric.LabelExtract {
			if val := result.Labels[label]; val != "" {
				extractedValues = append(extractedValues, val)
			}
		}
		extractedValue := strings.Join(extractedValues, ", ")

		// Add metric value to result
		if inspResult, ok := resultsMap[instance.Identifier]; ok {
			mv := &model.NginxMetricValue{
				Name:        metric.Name,
				RawValue:    result.Value,
				StringValue: extractedValue,
				Timestamp:   time.Now().Unix(),
				Labels:      result.Labels,
			}
			inspResult.SetMetric(mv)

			// Also update instance fields for nginx_last_error_timestamp
			if metric.Name == "nginx_last_error_timestamp" {
				if errorLogPath := result.Labels["error_log_path"]; errorLogPath != "" {
					inspResult.Instance.SetErrorLogPath(errorLogPath)
				}
			}

			matchedCount++
		}
	}

	c.logger.Debug().
		Str("metric", metric.Name).
		Int("matched", matchedCount).
		Msg("label extract metric collected")

	return nil
}

// extractFieldsFromMetrics extracts common fields from metrics to result struct.
// This populates fields like Up, ActiveConnections, WorkerProcesses, etc.
func (c *NginxCollector) extractFieldsFromMetrics(resultsMap map[string]*model.NginxInspectionResult) {
	for _, result := range resultsMap {
		// nginx_up → Up
		if mv := result.GetMetric("nginx_up"); mv != nil && !mv.IsNA {
			result.Up = mv.RawValue == 1
		}

		// nginx_active → ActiveConnections
		if mv := result.GetMetric("nginx_active"); mv != nil && !mv.IsNA {
			result.ActiveConnections = int(mv.RawValue)
		}

		// nginx_worker_processes → WorkerProcesses
		if mv := result.GetMetric("nginx_worker_processes"); mv != nil && !mv.IsNA {
			result.WorkerProcesses = int(mv.RawValue)
		}

		// nginx_worker_connections → WorkerConnections
		if mv := result.GetMetric("nginx_worker_connections"); mv != nil && !mv.IsNA {
			result.WorkerConnections = int(mv.RawValue)
		}

		// nginx_error_page_4xx → ErrorPage4xxConfigured
		if mv := result.GetMetric("nginx_error_page_4xx"); mv != nil && !mv.IsNA {
			result.ErrorPage4xxConfigured = mv.RawValue == 1
		}

		// nginx_error_page_5xx → ErrorPage5xxConfigured
		if mv := result.GetMetric("nginx_error_page_5xx"); mv != nil && !mv.IsNA {
			result.ErrorPage5xxConfigured = mv.RawValue == 1
		}

		// nginx_non_root_user → NonRootUser
		if mv := result.GetMetric("nginx_non_root_user"); mv != nil && !mv.IsNA {
			result.NonRootUser = mv.RawValue == 1
		}

		// nginx_last_error_timestamp → LastErrorTimestamp
		if mv := result.GetMetric("nginx_last_error_timestamp"); mv != nil && !mv.IsNA {
			result.LastErrorTimestamp = int64(mv.RawValue)
		}

		// Calculate connection usage percent
		result.CalculateConnectionUsagePercent()

		// Set collected time
		result.CollectedAt = time.Now()
	}
}

// =============================================================================
// Nginx Upstream 状态采集
// =============================================================================

// CollectUpstreamStatus collects Upstream backend status from VictoriaMetrics.
// Queries nginx_upstream_check_status_code, nginx_upstream_check_rise, and
// nginx_upstream_check_fall metrics.
//
// Results are associated with instances by hostname matching.
func (c *NginxCollector) CollectUpstreamStatus(
	ctx context.Context,
	resultsMap map[string]*model.NginxInspectionResult,
) error {
	c.logger.Debug().Msg("collecting Nginx upstream status")

	vmFilter := c.instanceFilter.ToVMHostFilter()

	// Step 1: Query status_code
	statusResults, err := c.vmClient.QueryResultsWithFilter(ctx, "nginx_upstream_check_status_code", vmFilter)
	if err != nil {
		c.logger.Warn().Err(err).Msg("failed to query nginx_upstream_check_status_code")
		return nil // Non-fatal: some instances may not have upstream
	}

	if len(statusResults) == 0 {
		c.logger.Debug().Msg("no upstream status data found")
		return nil
	}

	// Step 2: Query rise and fall counts
	riseResults, _ := c.vmClient.QueryResultsWithFilter(ctx, "nginx_upstream_check_rise", vmFilter)
	fallResults, _ := c.vmClient.QueryResultsWithFilter(ctx, "nginx_upstream_check_fall", vmFilter)

	// Build maps for rise/fall by key (hostname:upstream:backend)
	riseMap := c.buildUpstreamValueMap(riseResults)
	fallMap := c.buildUpstreamValueMap(fallResults)

	// Step 3: Build hostname to identifier map
	hostnameToIdentifier := make(map[string]string)
	for identifier, result := range resultsMap {
		if result.Instance != nil {
			hostnameToIdentifier[result.Instance.Hostname] = identifier
		}
	}

	// Step 4: Process status results
	for _, result := range statusResults {
		// First try agent_hostname, fallback to ident
		hostname := result.Labels["agent_hostname"]
		if hostname == "" {
			hostname = result.Labels["ident"]
		}
		if hostname == "" {
			continue
		}

		// Apply hostname filter
		if !c.matchesHostnamePatterns(hostname) {
			continue
		}

		// Find the corresponding inspection result
		identifier, exists := hostnameToIdentifier[hostname]
		if !exists {
			continue
		}

		inspResult := resultsMap[identifier]
		if inspResult == nil {
			continue
		}

		// Extract upstream info
		upstreamName := result.Labels["upstream"]
		backendAddr := result.Labels["name"]
		statusCode := int(result.Value)

		// Build upstream status key
		key := fmt.Sprintf("%s:%s:%s", hostname, upstreamName, backendAddr)

		upstreamStatus := model.NginxUpstreamStatus{
			UpstreamName:   upstreamName,
			BackendAddress: backendAddr,
			Status:         statusCode == 1,
			RiseCount:      int(riseMap[key]),
			FallCount:      int(fallMap[key]),
		}

		inspResult.AddUpstreamStatus(upstreamStatus)
	}

	c.logger.Info().
		Int("status_results", len(statusResults)).
		Msg("Nginx upstream status collection completed")

	return nil
}

// buildUpstreamValueMap builds a map from upstream results.
// Key format: "hostname:upstream:backend"
func (c *NginxCollector) buildUpstreamValueMap(results []vm.QueryResult) map[string]float64 {
	valueMap := make(map[string]float64)

	for _, result := range results {
		// First try agent_hostname, fallback to ident
		hostname := result.Labels["agent_hostname"]
		if hostname == "" {
			hostname = result.Labels["ident"]
		}
		upstreamName := result.Labels["upstream"]
		backendAddr := result.Labels["name"]

		if hostname == "" || upstreamName == "" || backendAddr == "" {
			continue
		}

		key := fmt.Sprintf("%s:%s:%s", hostname, upstreamName, backendAddr)
		valueMap[key] = result.Value
	}

	return valueMap
}
