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

// =============================================================================
// Tomcat Collector
// =============================================================================

// TomcatCollector is the data collection service for Tomcat instances.
// It integrates with VictoriaMetrics to collect Tomcat monitoring metrics
// and N9E to obtain host IP addresses.
type TomcatCollector struct {
	vmClient       *vm.Client
	n9eClient      *n9e.Client // 用于获取 IP 地址
	config         *config.TomcatInspectionConfig
	metrics        []*model.TomcatMetricDefinition
	metricDefs     map[string]*model.TomcatMetricDefinition
	instanceFilter *TomcatInstanceFilter
	logger         zerolog.Logger
}

// TomcatInstanceFilter defines filtering criteria for Tomcat instances.
// UNIQUE: Supports both HostnamePatterns AND ContainerPatterns (dual deployment).
type TomcatInstanceFilter struct {
	HostnamePatterns  []string          // Hostname patterns (glob, e.g., "GX-MFUI-*")
	ContainerPatterns []string          // Container name patterns (glob, e.g., "tomcat-18001")
	BusinessGroups    []string          // Business groups (OR relation)
	Tags              map[string]string // Tags (AND relation)
}

// NewTomcatCollector creates a new TomcatCollector instance.
func NewTomcatCollector(
	cfg *config.TomcatInspectionConfig,
	vmClient *vm.Client,
	n9eClient *n9e.Client,
	metrics []*model.TomcatMetricDefinition,
	logger zerolog.Logger,
) *TomcatCollector {
	c := &TomcatCollector{
		vmClient:  vmClient,
		n9eClient: n9eClient,
		config:    cfg,
		metrics:   metrics,
		logger:    logger.With().Str("component", "tomcat-collector").Logger(),
	}

	// Build metric definitions map for fast lookup
	c.metricDefs = make(map[string]*model.TomcatMetricDefinition, len(metrics))
	for _, m := range metrics {
		c.metricDefs[m.Name] = m
	}

	// Build instance filter from config
	c.instanceFilter = c.buildInstanceFilter()

	return c
}

// buildInstanceFilter converts config.TomcatFilter to TomcatInstanceFilter.
func (c *TomcatCollector) buildInstanceFilter() *TomcatInstanceFilter {
	if c.config == nil {
		return nil
	}

	filter := c.config.InstanceFilter
	if len(filter.HostnamePatterns) == 0 &&
		len(filter.ContainerPatterns) == 0 &&
		len(filter.BusinessGroups) == 0 &&
		len(filter.Tags) == 0 {
		return nil
	}

	return &TomcatInstanceFilter{
		HostnamePatterns:  filter.HostnamePatterns,
		ContainerPatterns: filter.ContainerPatterns,
		BusinessGroups:    filter.BusinessGroups,
		Tags:              filter.Tags,
	}
}

// GetConfig returns the Tomcat inspection configuration.
func (c *TomcatCollector) GetConfig() *config.TomcatInspectionConfig {
	return c.config
}

// GetMetrics returns the list of metric definitions.
func (c *TomcatCollector) GetMetrics() []*model.TomcatMetricDefinition {
	return c.metrics
}

// GetInstanceFilter returns the instance filter.
func (c *TomcatCollector) GetInstanceFilter() *TomcatInstanceFilter {
	return c.instanceFilter
}

// IsEmpty returns true if the instance filter has no filtering criteria.
func (f *TomcatInstanceFilter) IsEmpty() bool {
	if f == nil {
		return true
	}
	return len(f.HostnamePatterns) == 0 &&
		len(f.ContainerPatterns) == 0 &&
		len(f.BusinessGroups) == 0 &&
		len(f.Tags) == 0
}

// ToVMHostFilter converts TomcatInstanceFilter to vm.HostFilter.
// Note: HostnamePatterns and ContainerPatterns are not supported in vm.HostFilter
// and will be handled separately in the DiscoverInstances method.
func (f *TomcatInstanceFilter) ToVMHostFilter() *vm.HostFilter {
	if f == nil || f.IsEmpty() {
		return nil
	}

	// Only include business groups and tags, patterns are handled separately
	if len(f.BusinessGroups) == 0 && len(f.Tags) == 0 {
		return nil
	}

	return &vm.HostFilter{
		BusinessGroups: f.BusinessGroups,
		Tags:           f.Tags,
	}
}

// =============================================================================
// Tomcat 实例发现
// =============================================================================

// DiscoverInstances discovers all Tomcat instances by querying tomcat_up metric.
// It extracts labels (agent_hostname, container, port, app_type, install_path, version, log_path, jvm_config)
// and retrieves IP addresses from N9E API.
//
// Returns a list of TomcatInstance objects.
func (c *TomcatCollector) DiscoverInstances(ctx context.Context) ([]*model.TomcatInstance, error) {
	c.logger.Info().Msg("starting Tomcat instance discovery")

	// Step 1: Query tomcat_up to get all running instances
	vmFilter := c.instanceFilter.ToVMHostFilter()
	results, err := c.vmClient.QueryResultsWithFilter(ctx, "tomcat_up == 1", vmFilter)
	if err != nil {
		c.logger.Error().Err(err).Msg("failed to query tomcat_up metric")
		return nil, fmt.Errorf("failed to query tomcat_up: %w", err)
	}

	c.logger.Debug().Int("raw_results", len(results)).Msg("received tomcat_up query results")

	// Step 2: Query tomcat_info to get instance metadata
	infoResults, err := c.vmClient.QueryResultsWithFilter(ctx, "tomcat_info", vmFilter)
	if err != nil {
		c.logger.Warn().Err(err).Msg("failed to query tomcat_info, continuing with limited metadata")
		infoResults = []vm.QueryResult{}
	}

	// Step 3: Build container map from tomcat_up
	containerMap := c.buildContainerMap(ctx, vmFilter)

	// Step 4: Build info map for fast lookup
	infoMap := c.buildInfoMap(infoResults)

	// Step 5: Extract instances and apply filters
	var instances []*model.TomcatInstance
	seenIdentifiers := make(map[string]bool)

	for _, result := range results {
		// 5.1 Extract hostname
		hostname := c.extractHostname(result.Labels)
		if hostname == "" {
			c.logger.Warn().Interface("labels", result.Labels).Msg("missing hostname labels")
			continue
		}

		// 5.2 Apply hostname pattern filter
		if !c.matchesHostnamePatterns(hostname) {
			c.logger.Debug().Str("hostname", hostname).Msg("hostname filtered out")
			continue
		}

		// 5.3 Get container
		container := containerMap[hostname]

		// 5.4 Apply container pattern filter (if container exists)
		if container != "" && !c.matchesContainerPatterns(container) {
			c.logger.Debug().
				Str("hostname", hostname).
				Str("container", container).
				Msg("container filtered out")
			continue
		}

		// 5.5 Extract port from info
		port := c.extractPort(infoMap, hostname)

		// 5.6 Generate identifier (container-first)
		identifier := model.GenerateTomcatIdentifier(hostname, port, container)
		if seenIdentifiers[identifier] {
			c.logger.Debug().Str("identifier", identifier).Msg("skipping duplicate instance")
			continue
		}

		// 5.7 Create instance
		var instance *model.TomcatInstance
		if container != "" {
			instance = model.NewTomcatInstanceWithContainer(hostname, container)
		} else if port > 0 {
			instance = model.NewTomcatInstance(hostname, port)
		} else {
			c.logger.Warn().
				Str("hostname", hostname).
				Msg("skipping instance: no port or container info")
			continue
		}

		if instance == nil {
			continue
		}

		// 5.8 Set additional fields from info labels
		c.populateInstanceFromInfo(instance, infoMap, hostname)

		// 5.9 Get IP from N9E
		ip := c.getIPFromN9E(ctx, hostname)
		instance.SetIP(ip)

		instances = append(instances, instance)
		seenIdentifiers[identifier] = true
	}

	c.logger.Info().
		Int("discovered", len(instances)).
		Int("filtered_out", len(results)-len(instances)).
		Msg("Tomcat instance discovery completed")

	return instances, nil
}

// buildContainerMap builds a map of hostname -> container from tomcat_up.
func (c *TomcatCollector) buildContainerMap(ctx context.Context, vmFilter *vm.HostFilter) map[string]string {
	containerMap := make(map[string]string)

	results, err := c.vmClient.QueryResultsWithFilter(ctx, "tomcat_up", vmFilter)
	if err != nil {
		c.logger.Warn().Err(err).Msg("failed to query tomcat_up for container info")
		return containerMap
	}

	for _, result := range results {
		hostname := c.extractHostname(result.Labels)
		container := result.Labels["container"]
		if hostname != "" && container != "" {
			containerMap[hostname] = container
		}
	}

	return containerMap
}

// buildInfoMap builds a map of hostname -> labels from tomcat_info.
func (c *TomcatCollector) buildInfoMap(results []vm.QueryResult) map[string]map[string]string {
	infoMap := make(map[string]map[string]string)

	for _, result := range results {
		hostname := c.extractHostname(result.Labels)
		if hostname != "" {
			infoMap[hostname] = result.Labels
		}
	}

	return infoMap
}

// extractHostname extracts hostname from metric labels.
// Tries: agent_hostname > ident > host
func (c *TomcatCollector) extractHostname(labels map[string]string) string {
	for _, key := range []string{"agent_hostname", "ident", "host"} {
		if val := labels[key]; val != "" {
			return val
		}
	}
	return ""
}

// extractPort extracts port from tomcat_info labels.
func (c *TomcatCollector) extractPort(infoMap map[string]map[string]string, hostname string) int {
	if info, ok := infoMap[hostname]; ok {
		if portStr := info["port"]; portStr != "" {
			if port, err := strconv.Atoi(portStr); err == nil {
				return port
			}
		}
	}
	return 0
}

// populateInstanceFromInfo populates instance fields from tomcat_info labels.
func (c *TomcatCollector) populateInstanceFromInfo(
	instance *model.TomcatInstance,
	infoMap map[string]map[string]string,
	hostname string,
) {
	info, ok := infoMap[hostname]
	if !ok {
		return
	}

	if appType := info["app_type"]; appType != "" {
		instance.SetApplicationType(appType)
	}
	if version := info["version"]; version != "" {
		instance.SetVersion(version)
	}
	if installPath := info["install_path"]; installPath != "" {
		instance.SetInstallPath(installPath)
	}
	if logPath := info["log_path"]; logPath != "" {
		instance.SetLogPath(logPath)
	}
	if jvmConfig := info["jvm_config"]; jvmConfig != "" {
		instance.SetJVMConfig(jvmConfig)
	}
}

// getIPFromN9E retrieves the IP address for a hostname from N9E API.
// Returns "N/A" if the hostname is not found or an error occurs.
func (c *TomcatCollector) getIPFromN9E(ctx context.Context, hostname string) string {
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
func (c *TomcatCollector) matchesHostnamePatterns(hostname string) bool {
	if c.instanceFilter == nil || len(c.instanceFilter.HostnamePatterns) == 0 {
		return true
	}

	for _, pattern := range c.instanceFilter.HostnamePatterns {
		if matchPattern(hostname, pattern) {
			return true
		}
	}

	return false
}

// matchesContainerPatterns checks if a container name matches any configured patterns.
// Returns true if no patterns configured or container matches at least one pattern.
func (c *TomcatCollector) matchesContainerPatterns(container string) bool {
	if c.instanceFilter == nil || len(c.instanceFilter.ContainerPatterns) == 0 {
		return true
	}

	for _, pattern := range c.instanceFilter.ContainerPatterns {
		if matchPattern(container, pattern) {
			return true
		}
	}

	return false
}

// matchPattern checks if a value matches a pattern with wildcard support.
// Supports wildcard '*' which matches any sequence of characters.
func matchPattern(value, pattern string) bool {
	// Exact match optimization
	if value == pattern {
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

	return re.MatchString(value)
}

// =============================================================================
// Tomcat 指标采集
// =============================================================================

// CollectMetrics retrieves metric data from VictoriaMetrics for all Tomcat instances.
//
// Flow:
//  1. Initialize result objects for each instance
//  2. Separate pending and active metrics
//  3. Set N/A for pending metrics
//  4. Concurrently collect active metrics (errgroup + concurrency limit)
//  5. Extract field values from metrics
//  6. Return results map (key = identifier)
//
// Single metric failure does not abort the entire collection.
func (c *TomcatCollector) CollectMetrics(
	ctx context.Context,
	instances []*model.TomcatInstance,
	metrics []*model.TomcatMetricDefinition,
) (map[string]*model.TomcatInspectionResult, error) {
	c.logger.Debug().
		Int("instance_count", len(instances)).
		Int("metric_count", len(metrics)).
		Msg("collecting Tomcat metrics from VictoriaMetrics")

	// Step 1: Initialize results map (indexed by identifier)
	resultsMap := make(map[string]*model.TomcatInspectionResult, len(instances))
	for _, instance := range instances {
		resultsMap[instance.Identifier] = model.NewTomcatInspectionResult(instance)
	}

	// Step 2: Separate pending and active metrics
	var pendingMetrics []*model.TomcatMetricDefinition
	var activeMetrics []*model.TomcatMetricDefinition

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
		Msg("Tomcat metrics collection completed")

	return resultsMap, nil
}

// setPendingMetrics sets N/A values for all pending metrics on all instances.
func (c *TomcatCollector) setPendingMetrics(
	resultsMap map[string]*model.TomcatInspectionResult,
	pendingMetrics []*model.TomcatMetricDefinition,
) {
	if len(pendingMetrics) == 0 {
		return
	}

	c.logger.Debug().
		Int("pending_count", len(pendingMetrics)).
		Msg("setting N/A for pending Tomcat metrics")

	for _, metric := range pendingMetrics {
		for _, result := range resultsMap {
			mv := &model.TomcatMetricValue{
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
func (c *TomcatCollector) collectMetricConcurrent(
	ctx context.Context,
	metric *model.TomcatMetricDefinition,
	instances []*model.TomcatInstance,
	resultsMap map[string]*model.TomcatInspectionResult,
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
		Msg("collecting Tomcat metric (concurrent)")

	// Query VictoriaMetrics
	vmFilter := c.instanceFilter.ToVMHostFilter()
	results, err := c.vmClient.QueryResultsWithFilter(ctx, metric.Query, vmFilter)
	if err != nil {
		return fmt.Errorf("query failed for %s: %w", metric.Name, err)
	}

	// Build identifier map for fast lookup
	identifierMap := make(map[string]*model.TomcatInstance, len(instances))
	for _, instance := range instances {
		identifierMap[instance.Identifier] = instance
	}

	// Match results to instances
	mu.Lock()
	defer mu.Unlock()

	matchedCount := 0
	for _, result := range results {
		// Extract identifier
		identifier := c.extractIdentifier(result.Labels, identifierMap)
		if identifier == "" {
			continue
		}

		// Find matching instance result
		inspResult, ok := resultsMap[identifier]
		if !ok {
			continue
		}

		// Create metric value (result.Value is already float64)
		mv := &model.TomcatMetricValue{
			Name:      metric.Name,
			RawValue:  result.Value,
			Timestamp: time.Now().Unix(),
			Labels:    result.Labels,
		}
		inspResult.SetMetric(mv)
		matchedCount++
	}

	c.logger.Debug().
		Str("metric", metric.Name).
		Int("matched", matchedCount).
		Msg("metric collection completed")

	return nil
}

// collectLabelExtractMetric handles metrics with label_extract (tomcat_info).
func (c *TomcatCollector) collectLabelExtractMetric(
	ctx context.Context,
	metric *model.TomcatMetricDefinition,
	instances []*model.TomcatInstance,
	resultsMap map[string]*model.TomcatInspectionResult,
	mu *sync.Mutex,
) error {
	c.logger.Debug().
		Str("metric", metric.Name).
		Str("query", metric.Query).
		Msg("collecting Tomcat metric with label extraction")

	// Query VictoriaMetrics
	vmFilter := c.instanceFilter.ToVMHostFilter()
	results, err := c.vmClient.QueryResultsWithFilter(ctx, metric.Query, vmFilter)
	if err != nil {
		return fmt.Errorf("query failed for %s: %w", metric.Name, err)
	}

	// Build identifier map for fast lookup
	identifierMap := make(map[string]*model.TomcatInstance, len(instances))
	for _, instance := range instances {
		identifierMap[instance.Identifier] = instance
	}

	mu.Lock()
	defer mu.Unlock()

	matchedCount := 0
	for _, result := range results {
		// Extract identifier
		identifier := c.extractIdentifier(result.Labels, identifierMap)
		if identifier == "" {
			continue
		}

		// Find matching instance result
		inspResult, ok := resultsMap[identifier]
		if !ok {
			continue
		}

		// Build extracted value from labels
		var values []string
		for _, label := range metric.LabelExtract {
			if val := result.Labels[label]; val != "" {
				values = append(values, val)
			}
		}

		// Create metric value with extracted string value
		mv := &model.TomcatMetricValue{
			Name:        metric.Name,
			RawValue:    result.Value,
			StringValue: strings.Join(values, ", "),
			Timestamp:   time.Now().Unix(),
			Labels:      result.Labels,
		}
		inspResult.SetMetric(mv)
		matchedCount++
	}

	c.logger.Debug().
		Str("metric", metric.Name).
		Int("matched", matchedCount).
		Msg("label extraction metric collection completed")

	return nil
}

// extractIdentifier extracts the instance identifier from metric labels.
// Tries container-first: hostname:container, then hostname:port.
func (c *TomcatCollector) extractIdentifier(
	labels map[string]string,
	identifierMap map[string]*model.TomcatInstance,
) string {
	hostname := labels["agent_hostname"]
	if hostname == "" {
		hostname = labels["ident"]
	}
	if hostname == "" {
		return ""
	}

	// Try container first
	container := labels["container"]
	if container != "" {
		identifier := model.GenerateTomcatIdentifier(hostname, 0, container)
		if _, exists := identifierMap[identifier]; exists {
			return identifier
		}
	}

	// Try port
	portStr := labels["port"]
	if portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			identifier := model.GenerateTomcatIdentifier(hostname, port, "")
			if _, exists := identifierMap[identifier]; exists {
				return identifier
			}
		}
	}

	return ""
}

// extractFieldsFromMetrics extracts metric values to result struct fields.
func (c *TomcatCollector) extractFieldsFromMetrics(resultsMap map[string]*model.TomcatInspectionResult) {
	for _, result := range resultsMap {
		// tomcat_up -> Up
		if mv := result.GetMetric("tomcat_up"); mv != nil && !mv.IsNA {
			result.Up = mv.RawValue == 1
		}

		// tomcat_connections -> Connections
		if mv := result.GetMetric("tomcat_connections"); mv != nil && !mv.IsNA {
			result.Connections = int(mv.RawValue)
		}

		// tomcat_uptime_seconds -> UptimeSeconds
		if mv := result.GetMetric("tomcat_uptime_seconds"); mv != nil && !mv.IsNA {
			result.UptimeSeconds = int64(mv.RawValue)
		}

		// tomcat_last_error_timestamp -> LastErrorTimestamp
		if mv := result.GetMetric("tomcat_last_error_timestamp"); mv != nil && !mv.IsNA {
			result.LastErrorTimestamp = int64(mv.RawValue)
		}

		// tomcat_non_root_user -> NonRootUser
		if mv := result.GetMetric("tomcat_non_root_user"); mv != nil && !mv.IsNA {
			result.NonRootUser = mv.RawValue == 1
		}

		// Set collected time
		result.CollectedAt = time.Now()
	}
}
