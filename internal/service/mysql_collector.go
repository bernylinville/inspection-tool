// Package service provides business logic services for the inspection tool.
package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"inspection-tool/internal/client/vm"
	"inspection-tool/internal/config"
	"inspection-tool/internal/model"
	"inspection-tool/internal/util"
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
		address := util.ExtractAddress(result.Labels)
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

// matchesAddressPatterns checks if an address matches any configured patterns.
// Returns true if no patterns configured or address matches at least one pattern.
func (c *MySQLCollector) matchesAddressPatterns(address string) bool {
	if c.instanceFilter == nil || len(c.instanceFilter.AddressPatterns) == 0 {
		return true
	}

	return util.MatchAnyPattern(address, c.instanceFilter.AddressPatterns)
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
		address := util.ExtractAddress(result.Labels)
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
		address := util.ExtractAddress(result.Labels)
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

	// Step 6: Post-process - populate struct fields from metrics
	for _, result := range resultsMap {
		c.populateResultFields(result)
	}

	c.logger.Info().
		Int("instances", len(instances)).
		Int("active_metrics", len(filteredMetrics)).
		Int("pending_metrics", len(pendingMetrics)).
		Msg("MySQL metrics collection completed")

	return resultsMap, nil
}

// populateResultFields maps collected metrics to MySQLInspectionResult struct fields.
// This method translates metric values to their corresponding struct fields.
func (c *MySQLCollector) populateResultFields(result *model.MySQLInspectionResult) {
	if result == nil {
		return
	}

	// mysql_non_root_user → NonRootUser
	if mv := result.GetMetric("non_root_user"); mv != nil && !mv.IsNA {
		if mv.RawValue == 1 {
			result.NonRootUser = "是"
		} else {
			result.NonRootUser = "否"
		}
	}

	// mysql_up → ConnectionStatus
	if mv := result.GetMetric("mysql_up"); mv != nil && !mv.IsNA {
		result.ConnectionStatus = mv.RawValue == 1
	}

	// max_connections → MaxConnections
	if mv := result.GetMetric("max_connections"); mv != nil && !mv.IsNA {
		result.MaxConnections = int(mv.RawValue)
	}

	// current_connections → CurrentConnections
	if mv := result.GetMetric("current_connections"); mv != nil && !mv.IsNA {
		result.CurrentConnections = int(mv.RawValue)
	}

	// binlog_file_count → BinlogEnabled
	if mv := result.GetMetric("binlog_file_count"); mv != nil && !mv.IsNA {
		result.BinlogEnabled = mv.RawValue > 0
	}

	// binlog_expire_seconds → BinlogExpireSeconds
	if mv := result.GetMetric("binlog_expire_seconds"); mv != nil && !mv.IsNA {
		result.BinlogExpireSeconds = int(mv.RawValue)
	}

	// slow_query_log → SlowQueryLogEnabled
	if mv := result.GetMetric("slow_query_log"); mv != nil && !mv.IsNA {
		result.SlowQueryLogEnabled = mv.RawValue == 1
	}

	// slow_query_log_file → SlowQueryLogPath
	if mv := result.GetMetric("slow_query_log_file"); mv != nil && !mv.IsNA && mv.StringValue != "" {
		result.SlowQueryLogPath = mv.StringValue
	}

	// uptime → Uptime
	if mv := result.GetMetric("uptime"); mv != nil && !mv.IsNA {
		result.Uptime = int64(mv.RawValue)
	}

	// MGR related fields
	if mv := result.GetMetric("mgr_member_count"); mv != nil && !mv.IsNA {
		result.MGRMemberCount = int(mv.RawValue)
	}

	if mv := result.GetMetric("mgr_role_primary"); mv != nil && !mv.IsNA {
		if mv.RawValue == 1 {
			result.MGRRole = model.MGRRolePrimary
		} else {
			result.MGRRole = model.MGRRoleSecondary
		}
		// Also extract server_id from member_id label
		if result.Instance != nil && mv.StringValue != "" {
			result.Instance.SetServerID(mv.StringValue)
		}
	}

	if mv := result.GetMetric("mgr_state_online"); mv != nil && !mv.IsNA {
		result.MGRStateOnline = mv.RawValue == 1
	}

	// mysql_version → Instance.Version
	if mv := result.GetMetric("mysql_version"); mv != nil && !mv.IsNA && mv.StringValue != "" {
		if result.Instance != nil {
			result.Instance.SetVersion(mv.StringValue, "")
		}
	}

	// server_id → Instance.ServerID (fallback if not set by mgr_role_primary)
	if mv := result.GetMetric("server_id"); mv != nil && !mv.IsNA {
		if result.Instance != nil && result.Instance.ServerID == "" {
			result.Instance.SetServerID(fmt.Sprintf("%.0f", mv.RawValue))
		}
	}

	// remote_users_count → RemoteUsersCount
	if mv := result.GetMetric("remote_users_count"); mv != nil && !mv.IsNA {
		result.RemoteUsersCount = int(mv.RawValue)
	}

	// remote_user_info → RemoteUsers (collect user names from StringValue)
	// Note: Each remote_user_info metric represents one user with host='%'
	// The user name is extracted from the 'user' label via label_extract
	if mv := result.GetMetric("remote_user_info"); mv != nil && !mv.IsNA && mv.StringValue != "" {
		// Add the user to the list if not already present
		userExists := false
		for _, u := range result.RemoteUsers {
			if u == mv.StringValue {
				userExists = true
				break
			}
		}
		if !userExists {
			result.RemoteUsers = append(result.RemoteUsers, mv.StringValue)
		}
	}

	// Set collected timestamp
	result.CollectedAt = time.Now()
}
