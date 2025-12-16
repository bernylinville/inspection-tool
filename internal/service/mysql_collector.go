// Package service provides business logic services for the inspection tool.
package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"inspection-tool/internal/client/vm"
	"inspection-tool/internal/config"
	"inspection-tool/internal/model"
)

// MySQLCollector is the data collection service for MySQL instances.
// It integrates with VictoriaMetrics to collect MySQL monitoring metrics.
type MySQLCollector struct {
	vmClient       *vm.Client
	config         *config.MySQLInspectionConfig
	metrics        []*model.MySQLMetricDefinition
	instanceFilter *MySQLInstanceFilter
	logger         zerolog.Logger
}

// MySQLInstanceFilter defines filtering criteria for MySQL instances.
// Mirrors vm.HostFilter but with MySQL-specific address pattern matching.
type MySQLInstanceFilter struct {
	AddressPatterns []string          // Address patterns (e.g., "172.18.182.*")
	BusinessGroups  []string          // Business groups (OR relation)
	Tags            map[string]string // Tags (AND relation)
}

// NewMySQLCollector creates a new MySQLCollector instance.
func NewMySQLCollector(
	cfg *config.MySQLInspectionConfig,
	vmClient *vm.Client,
	metrics []*model.MySQLMetricDefinition,
	logger zerolog.Logger,
) *MySQLCollector {
	c := &MySQLCollector{
		vmClient: vmClient,
		config:   cfg,
		metrics:  metrics,
		logger:   logger.With().Str("component", "mysql-collector").Logger(),
	}

	// Build instance filter from config
	c.instanceFilter = c.buildInstanceFilter()

	return c
}

// buildInstanceFilter converts config.MySQLFilter to MySQLInstanceFilter.
func (c *MySQLCollector) buildInstanceFilter() *MySQLInstanceFilter {
	if c.config == nil {
		return nil
	}

	filter := c.config.InstanceFilter
	if len(filter.AddressPatterns) == 0 &&
		len(filter.BusinessGroups) == 0 &&
		len(filter.Tags) == 0 {
		return nil
	}

	return &MySQLInstanceFilter{
		AddressPatterns: filter.AddressPatterns,
		BusinessGroups:  filter.BusinessGroups,
		Tags:            filter.Tags,
	}
}

// GetConfig returns the MySQL inspection configuration.
func (c *MySQLCollector) GetConfig() *config.MySQLInspectionConfig {
	return c.config
}

// GetMetrics returns the list of metric definitions.
func (c *MySQLCollector) GetMetrics() []*model.MySQLMetricDefinition {
	return c.metrics
}

// GetInstanceFilter returns the instance filter.
func (c *MySQLCollector) GetInstanceFilter() *MySQLInstanceFilter {
	return c.instanceFilter
}

// IsEmpty returns true if the instance filter has no filtering criteria.
func (f *MySQLInstanceFilter) IsEmpty() bool {
	if f == nil {
		return true
	}
	return len(f.AddressPatterns) == 0 &&
		len(f.BusinessGroups) == 0 &&
		len(f.Tags) == 0
}

// ToVMHostFilter converts MySQLInstanceFilter to vm.HostFilter.
// Note: AddressPatterns are not supported in vm.HostFilter and will be
// handled separately in the DiscoverInstances method.
func (f *MySQLInstanceFilter) ToVMHostFilter() *vm.HostFilter {
	if f == nil || f.IsEmpty() {
		return nil
	}

	// Only include business groups and tags, address patterns are handled separately
	if len(f.BusinessGroups) == 0 && len(f.Tags) == 0 {
		return nil
	}

	return &vm.HostFilter{
		BusinessGroups: f.BusinessGroups,
		Tags:           f.Tags,
	}
}

// DiscoverInstances discovers all MySQL instances by querying mysql_up metric.
// It filters instances based on the configured InstanceFilter and returns
// a list of MySQLInstance objects with ClusterMode set from config.
//
// Only instances with mysql_up = 1 (connection OK) are included.
func (c *MySQLCollector) DiscoverInstances(ctx context.Context) ([]*model.MySQLInstance, error) {
	c.logger.Info().Msg("starting MySQL instance discovery")

	// Step 1: 构建 VM HostFilter（不包含 AddressPatterns）
	vmFilter := c.instanceFilter.ToVMHostFilter()

	// Step 2: 查询 mysql_up == 1（仅在线实例）
	query := "mysql_up == 1"
	results, err := c.vmClient.QueryResultsWithFilter(ctx, query, vmFilter)
	if err != nil {
		c.logger.Error().Err(err).Msg("failed to query mysql_up metric")
		return nil, fmt.Errorf("failed to query mysql_up: %w", err)
	}

	c.logger.Debug().Int("raw_results", len(results)).Msg("received mysql_up query results")

	// Step 3: 提取地址并构建实例
	var instances []*model.MySQLInstance
	seenAddresses := make(map[string]bool)

	for _, result := range results {
		// 3.1 提取地址
		address := c.extractAddress(result.Labels)
		if address == "" {
			c.logger.Warn().Interface("labels", result.Labels).Msg("missing address label")
			continue
		}

		// 3.2 去重
		if seenAddresses[address] {
			c.logger.Debug().Str("address", address).Msg("skipping duplicate address")
			continue
		}

		// 3.3 地址模式过滤（后置过滤）
		if !c.matchesAddressPatterns(address) {
			c.logger.Debug().Str("address", address).Msg("address filtered out")
			continue
		}

		// 3.4 创建实例
		instance := model.NewMySQLInstanceWithClusterMode(
			address,
			model.MySQLClusterMode(c.config.ClusterMode),
		)

		if instance == nil {
			c.logger.Warn().Str("address", address).Msg("failed to parse address")
			continue
		}

		instances = append(instances, instance)
		seenAddresses[address] = true
	}

	c.logger.Info().
		Int("discovered", len(instances)).
		Int("filtered_out", len(results)-len(instances)).
		Msg("MySQL instance discovery completed")

	return instances, nil
}

// extractAddress extracts MySQL instance address from metric labels.
// Tries the following labels in order: "address", "instance", "server".
// Returns empty string if no address label is found.
func (c *MySQLCollector) extractAddress(labels map[string]string) string {
	addressLabels := []string{"address", "instance", "server"}

	for _, label := range addressLabels {
		if addr, ok := labels[label]; ok && addr != "" {
			return addr
		}
	}

	return ""
}

// matchesAddressPatterns checks if an address matches any configured patterns.
// Returns true if no patterns configured or address matches at least one pattern.
func (c *MySQLCollector) matchesAddressPatterns(address string) bool {
	if c.instanceFilter == nil || len(c.instanceFilter.AddressPatterns) == 0 {
		return true
	}

	for _, pattern := range c.instanceFilter.AddressPatterns {
		if matchAddressPattern(address, pattern) {
			return true
		}
	}

	return false
}

// matchAddressPattern checks if an address matches a pattern with wildcard support.
// Supports wildcard '*' which matches any sequence of characters.
// Examples:
//   - "172.18.182.*" matches "172.18.182.91:3306"
//   - "192.168.1.100:*" matches "192.168.1.100:3306"
//   - "*" matches all addresses
func matchAddressPattern(address, pattern string) bool {
	// 精确匹配优化
	if address == pattern {
		return true
	}

	// 无通配符
	if !strings.Contains(pattern, "*") {
		return false
	}

	// 转换为正则表达式
	regexPattern := regexp.QuoteMeta(pattern)
	regexPattern = strings.ReplaceAll(regexPattern, "\\*", ".*")
	regexPattern = "^" + regexPattern + "$"

	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return false
	}

	return re.MatchString(address)
}

// =============================================================================
// MySQL 指标采集相关方法
// =============================================================================

// filterMetricsByClusterMode filters metrics based on cluster mode.
// Returns only metrics that are applicable to the configured cluster mode.
//
// Rules:
// - Metrics with no cluster_mode restriction are always included
// - Metrics with cluster_mode restriction are only included if they match
func (c *MySQLCollector) filterMetricsByClusterMode(
	metrics []*model.MySQLMetricDefinition,
	clusterMode model.MySQLClusterMode,
) []*model.MySQLMetricDefinition {
	var filtered []*model.MySQLMetricDefinition

	for _, metric := range metrics {
		if metric.IsForClusterMode(clusterMode) {
			filtered = append(filtered, metric)
		} else {
			c.logger.Debug().
				Str("metric", metric.Name).
				Str("required_mode", metric.ClusterMode).
				Str("current_mode", string(clusterMode)).
				Msg("skipping metric not applicable to current cluster mode")
		}
	}

	c.logger.Info().
		Int("total", len(metrics)).
		Int("filtered", len(filtered)).
		Str("cluster_mode", string(clusterMode)).
		Msg("filtered metrics by cluster mode")

	return filtered
}

// setPendingMetrics sets N/A values for all pending metrics on all instances.
// Pending metrics are those that are not yet implemented (status="pending" or no query).
func (c *MySQLCollector) setPendingMetrics(
	resultsMap map[string]*model.MySQLInspectionResult,
	pendingMetrics []*model.MySQLMetricDefinition,
) {
	if len(pendingMetrics) == 0 {
		return
	}

	c.logger.Debug().
		Int("pending_count", len(pendingMetrics)).
		Msg("setting N/A for pending MySQL metrics")

	for _, metric := range pendingMetrics {
		for _, result := range resultsMap {
			mv := &model.MySQLMetricValue{
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
// This method is called concurrently by multiple goroutines, protected by mutex.
//
// If the metric has label_extract, it delegates to collectLabelExtractMetric.
// Otherwise, it directly queries and stores the metric value.
func (c *MySQLCollector) collectMetricConcurrent(
	ctx context.Context,
	metric *model.MySQLMetricDefinition,
	instances []*model.MySQLInstance,
	resultsMap map[string]*model.MySQLInspectionResult,
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
		Msg("collecting MySQL metric (concurrent)")

	// Query VictoriaMetrics
	vmFilter := c.instanceFilter.ToVMHostFilter()
	results, err := c.vmClient.QueryResultsWithFilter(ctx, metric.Query, vmFilter)
	if err != nil {
		return fmt.Errorf("query failed for %s: %w", metric.Name, err)
	}

	// Create address map for fast lookup
	addressMap := make(map[string]*model.MySQLInstance, len(instances))
	for _, instance := range instances {
		addressMap[instance.Address] = instance
	}

	// Match results to instances by address
	mu.Lock()
	defer mu.Unlock()

	matchedCount := 0
	for _, result := range results {
		address := c.extractAddress(result.Labels)
		if address == "" {
			continue
		}

		// Apply address pattern filtering (post-filter)
		if !c.matchesAddressPatterns(address) {
			continue
		}

		// Check if this address belongs to our instances
		if _, exists := addressMap[address]; !exists {
			continue
		}

		// Add metric value to result
		if inspResult, ok := resultsMap[address]; ok {
			mv := &model.MySQLMetricValue{
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
		Msg("MySQL metric collected (concurrent)")

	return nil
}

// collectLabelExtractMetric collects metrics that extract values from labels.
// This handles special metrics like mysql_version (extracts "version" label)
// and mgr_role_primary (extracts "member_id" label for Server ID).
func (c *MySQLCollector) collectLabelExtractMetric(
	ctx context.Context,
	metric *model.MySQLMetricDefinition,
	instances []*model.MySQLInstance,
	resultsMap map[string]*model.MySQLInspectionResult,
	mu *sync.Mutex,
) error {
	c.logger.Debug().
		Str("metric", metric.Name).
		Str("label_extract", metric.LabelExtract).
		Msg("collecting label extract metric")

	vmFilter := c.instanceFilter.ToVMHostFilter()
	results, err := c.vmClient.QueryResultsWithFilter(ctx, metric.Query, vmFilter)
	if err != nil {
		return fmt.Errorf("query failed for %s: %w", metric.Name, err)
	}

	// Create address map for fast lookup
	addressMap := make(map[string]*model.MySQLInstance, len(instances))
	for _, instance := range instances {
		addressMap[instance.Address] = instance
	}

	mu.Lock()
	defer mu.Unlock()

	matchedCount := 0
	for _, result := range results {
		address := c.extractAddress(result.Labels)
		if address == "" {
			continue
		}

		// Apply address pattern filtering
		if !c.matchesAddressPatterns(address) {
			continue
		}

		// Check if this address belongs to our instances
		if _, exists := addressMap[address]; !exists {
			continue
		}

		// Extract value from label
		extractedValue := result.Labels[metric.LabelExtract]
		if extractedValue == "" {
			c.logger.Warn().
				Str("metric", metric.Name).
				Str("address", address).
				Str("label", metric.LabelExtract).
				Msg("label value not found")
			continue
		}

		// Add metric value to result
		if inspResult, ok := resultsMap[address]; ok {
			mv := &model.MySQLMetricValue{
				Name:        metric.Name,
				RawValue:    result.Value,
				StringValue: extractedValue, // Extracted label value
				Timestamp:   time.Now().Unix(),
				Labels:      result.Labels,
			}
			inspResult.SetMetric(mv)
			matchedCount++
		}
	}

	c.logger.Debug().
		Str("metric", metric.Name).
		Int("matched", matchedCount).
		Msg("label extract metric collected")

	return nil
}

// CollectMetrics retrieves metric data from VictoriaMetrics for all MySQL instances.
//
// Flow:
//  1. Initialize result objects for each instance
//  2. Separate pending and active metrics
//  3. Set N/A for pending metrics
//  4. Filter active metrics by cluster_mode
//  5. Concurrently collect active metrics (errgroup + concurrency limit)
//  6. Return results map (key = address)
//
// Single metric failure does not abort the entire collection.
func (c *MySQLCollector) CollectMetrics(
	ctx context.Context,
	instances []*model.MySQLInstance,
	metrics []*model.MySQLMetricDefinition,
) (map[string]*model.MySQLInspectionResult, error) {
	c.logger.Debug().
		Int("instance_count", len(instances)).
		Int("metric_count", len(metrics)).
		Msg("collecting MySQL metrics from VictoriaMetrics")

	// Step 1: Initialize results map (indexed by address)
	resultsMap := make(map[string]*model.MySQLInspectionResult, len(instances))
	for _, instance := range instances {
		resultsMap[instance.Address] = model.NewMySQLInspectionResult(instance)
	}

	// Step 2: Separate pending and active metrics
	var pendingMetrics []*model.MySQLMetricDefinition
	var activeMetrics []*model.MySQLMetricDefinition

	for _, metric := range metrics {
		if metric.IsPending() {
			pendingMetrics = append(pendingMetrics, metric)
		} else {
			activeMetrics = append(activeMetrics, metric)
		}
	}

	// Step 3: Set N/A for pending metrics
	c.setPendingMetrics(resultsMap, pendingMetrics)

	// Step 4: Filter active metrics by cluster_mode
	clusterMode := model.MySQLClusterMode(c.config.ClusterMode)
	filteredMetrics := c.filterMetricsByClusterMode(activeMetrics, clusterMode)

	if len(filteredMetrics) == 0 {
		c.logger.Warn().Msg("no active metrics to collect after filtering")
		return resultsMap, nil
	}

	// Step 5: Concurrently collect active metrics
	g, ctx := errgroup.WithContext(ctx)
	concurrency := 20 // Default concurrency
	g.SetLimit(concurrency)

	var mu sync.Mutex // Protects resultsMap from concurrent writes

	for _, metric := range filteredMetrics {
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

	c.logger.Info().
		Int("instances", len(instances)).
		Int("active_metrics", len(filteredMetrics)).
		Int("pending_metrics", len(pendingMetrics)).
		Msg("MySQL metrics collection completed")

	return resultsMap, nil
}
