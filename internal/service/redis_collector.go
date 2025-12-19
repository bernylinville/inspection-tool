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
)

// RedisCollector is the data collection service for Redis instances.
// It integrates with VictoriaMetrics to collect Redis monitoring metrics.
type RedisCollector struct {
	vmClient       *vm.Client
	config         *config.RedisInspectionConfig
	metrics        []*model.RedisMetricDefinition
	instanceFilter *RedisInstanceFilter
	logger         zerolog.Logger
}

// RedisInstanceFilter defines filtering criteria for Redis instances.
type RedisInstanceFilter struct {
	AddressPatterns []string          // Address patterns (glob, e.g., "192.18.102.*")
	BusinessGroups  []string          // Business groups (OR relation)
	Tags            map[string]string // Tags (AND relation)
}

// NewRedisCollector creates a new Redis collector.
func NewRedisCollector(
	cfg *config.RedisInspectionConfig,
	vmClient *vm.Client,
	metrics []*model.RedisMetricDefinition,
	logger zerolog.Logger,
) *RedisCollector {
	c := &RedisCollector{
		vmClient: vmClient,
		config:   cfg,
		metrics:  metrics,
		logger:   logger.With().Str("component", "redis-collector").Logger(),
	}

	// Build instance filter from config
	c.instanceFilter = c.buildInstanceFilter()

	return c
}

// buildInstanceFilter creates an instance filter from the config.
func (c *RedisCollector) buildInstanceFilter() *RedisInstanceFilter {
	if c.config == nil {
		return nil
	}

	filter := c.config.InstanceFilter
	if len(filter.AddressPatterns) == 0 &&
		len(filter.BusinessGroups) == 0 &&
		len(filter.Tags) == 0 {
		return nil
	}

	return &RedisInstanceFilter{
		AddressPatterns: filter.AddressPatterns,
		BusinessGroups:  filter.BusinessGroups,
		Tags:            filter.Tags,
	}
}

// GetConfig returns the Redis inspection configuration.
func (c *RedisCollector) GetConfig() *config.RedisInspectionConfig {
	return c.config
}

// GetMetrics returns the list of metric definitions.
func (c *RedisCollector) GetMetrics() []*model.RedisMetricDefinition {
	return c.metrics
}

// GetInstanceFilter returns the instance filter.
func (c *RedisCollector) GetInstanceFilter() *RedisInstanceFilter {
	return c.instanceFilter
}

// IsEmpty returns true if the instance filter has no filtering criteria.
func (f *RedisInstanceFilter) IsEmpty() bool {
	if f == nil {
		return true
	}
	return len(f.AddressPatterns) == 0 &&
		len(f.BusinessGroups) == 0 &&
		len(f.Tags) == 0
}

// ToVMHostFilter converts RedisInstanceFilter to vm.HostFilter.
// Note: AddressPatterns are not supported in vm.HostFilter and will be
// handled separately in the DiscoverInstances method.
func (f *RedisInstanceFilter) ToVMHostFilter() *vm.HostFilter {
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

// DiscoverInstances discovers all Redis instances by querying redis_up metric.
// It filters instances based on the configured InstanceFilter and returns
// a list of RedisInstance objects with Role set from replica_role label.
func (c *RedisCollector) DiscoverInstances(ctx context.Context) ([]*model.RedisInstance, error) {
	c.logger.Info().Msg("starting Redis instance discovery")

	// Step 1: Build VM HostFilter (excludes AddressPatterns)
	vmFilter := c.instanceFilter.ToVMHostFilter()

	// Step 2: Query redis_up metric
	query := "redis_up"
	results, err := c.vmClient.QueryResultsWithFilter(ctx, query, vmFilter)
	if err != nil {
		c.logger.Error().Err(err).Msg("failed to query redis_up metric")
		return nil, fmt.Errorf("failed to query redis_up: %w", err)
	}

	c.logger.Debug().Int("raw_results", len(results)).Msg("received redis_up query results")

	// Step 3: Extract addresses and build instances
	var instances []*model.RedisInstance
	seenAddresses := make(map[string]bool)

	for _, result := range results {
		// 3.1 Extract address
		address := c.extractAddress(result.Labels)
		if address == "" {
			c.logger.Warn().Interface("labels", result.Labels).Msg("missing address label")
			continue
		}

		// 3.2 Deduplicate
		if seenAddresses[address] {
			c.logger.Debug().Str("address", address).Msg("skipping duplicate address")
			continue
		}

		// 3.3 Address pattern filtering (post-filter)
		if !c.matchesAddressPatterns(address) {
			c.logger.Debug().Str("address", address).Msg("address filtered out")
			continue
		}

		// 3.4 Extract role from replica_role label
		role := c.extractRole(result.Labels)

		// 3.5 Create instance with role
		instance := model.NewRedisInstanceWithRole(address, role)
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
		Msg("Redis instance discovery completed")

	return instances, nil
}

// extractAddress extracts Redis instance address from metric labels.
// Tries the following labels in order: "address", "instance", "server".
// Returns empty string if no address label is found.
func (c *RedisCollector) extractAddress(labels map[string]string) string {
	addressLabels := []string{"address", "instance", "server"}

	for _, label := range addressLabels {
		if addr, ok := labels[label]; ok && addr != "" {
			return addr
		}
	}

	return ""
}

// extractRole extracts Redis role from metric labels.
// Uses "replica_role" label to determine master/slave role.
// Returns RedisRoleUnknown if the label is missing or has unexpected value.
func (c *RedisCollector) extractRole(labels map[string]string) model.RedisRole {
	roleLabel, ok := labels["replica_role"]
	if !ok || roleLabel == "" {
		return model.RedisRoleUnknown
	}

	switch roleLabel {
	case "master":
		return model.RedisRoleMaster
	case "slave":
		return model.RedisRoleSlave
	default:
		c.logger.Debug().
			Str("replica_role", roleLabel).
			Msg("unknown replica_role value, defaulting to unknown")
		return model.RedisRoleUnknown
	}
}

// matchesAddressPatterns checks if an address matches any configured patterns.
// Returns true if no patterns configured or address matches at least one pattern.
func (c *RedisCollector) matchesAddressPatterns(address string) bool {
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

// =============================================================================
// Redis 指标采集相关方法
// =============================================================================

// setPendingMetrics sets N/A values for all pending metrics on all instances.
// Pending metrics are those that are not yet implemented (status="pending" or no query).
func (c *RedisCollector) setPendingMetrics(
	resultsMap map[string]*model.RedisInspectionResult,
	pendingMetrics []*model.RedisMetricDefinition,
) {
	if len(pendingMetrics) == 0 {
		return
	}

	c.logger.Debug().
		Int("pending_count", len(pendingMetrics)).
		Msg("setting N/A for pending Redis metrics")

	for _, metric := range pendingMetrics {
		for _, result := range resultsMap {
			mv := &model.RedisMetricValue{
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
func (c *RedisCollector) collectMetricConcurrent(
	ctx context.Context,
	metric *model.RedisMetricDefinition,
	instances []*model.RedisInstance,
	resultsMap map[string]*model.RedisInspectionResult,
	mu *sync.Mutex,
) error {
	c.logger.Debug().
		Str("metric", metric.Name).
		Str("query", metric.Query).
		Msg("collecting Redis metric (concurrent)")

	// Query VictoriaMetrics
	vmFilter := c.instanceFilter.ToVMHostFilter()
	results, err := c.vmClient.QueryResultsWithFilter(ctx, metric.Query, vmFilter)
	if err != nil {
		return fmt.Errorf("query failed for %s: %w", metric.Name, err)
	}

	// Create address map for fast lookup
	addressMap := make(map[string]*model.RedisInstance, len(instances))
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
			mv := &model.RedisMetricValue{
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
		Msg("Redis metric collected (concurrent)")

	return nil
}

// verifyRoles performs dual-source role verification for all instances.
// Primary source: replica_role label (already set during DiscoverInstances)
// Secondary source: redis_connected_slaves and redis_master_link_status metrics
//
// Logic:
//   - If role is already master/slave from replica_role, keep it
//   - If role is unknown and connected_slaves > 0, set to master
//   - If role is unknown and connected_slaves == 0 and has master_link_status, set to slave
func (c *RedisCollector) verifyRoles(resultsMap map[string]*model.RedisInspectionResult) {
	for address, result := range resultsMap {
		if result.Instance == nil {
			continue
		}

		// Primary source already set from replica_role in DiscoverInstances
		currentRole := result.Instance.Role

		// If role is already determined, skip secondary verification
		if currentRole == model.RedisRoleMaster || currentRole == model.RedisRoleSlave {
			c.logger.Debug().
				Str("address", address).
				Str("role", string(currentRole)).
				Msg("role already determined from replica_role label")
			continue
		}

		// Secondary verification: check redis_connected_slaves metric
		connectedSlavesMetric := result.GetMetric("redis_connected_slaves")
		masterLinkStatusMetric := result.GetMetric("redis_master_link_status")

		if connectedSlavesMetric != nil && !connectedSlavesMetric.IsNA {
			if connectedSlavesMetric.RawValue > 0 {
				result.Instance.Role = model.RedisRoleMaster
				c.logger.Info().
					Str("address", address).
					Float64("connected_slaves", connectedSlavesMetric.RawValue).
					Msg("role determined as master from connected_slaves > 0")
			} else if masterLinkStatusMetric != nil && !masterLinkStatusMetric.IsNA {
				// connected_slaves == 0 and has master_link_status metric -> slave
				result.Instance.Role = model.RedisRoleSlave
				c.logger.Info().
					Str("address", address).
					Msg("role determined as slave from master_link_status presence")
			}
		}
	}
}

// calculateReplicationLag calculates replication lag for slave nodes.
// Formula: master_repl_offset - slave_repl_offset
// Only calculated for instances with role == slave
func (c *RedisCollector) calculateReplicationLag(resultsMap map[string]*model.RedisInspectionResult) {
	for address, result := range resultsMap {
		if result.Instance == nil {
			continue
		}

		// Only calculate for slave nodes
		if result.Instance.Role != model.RedisRoleSlave {
			continue
		}

		// Get master_repl_offset (the offset known to slave about master)
		masterOffset := result.GetMetric("redis_master_repl_offset")
		slaveOffset := result.GetMetric("redis_slave_repl_offset")

		// Both metrics must be present and not N/A
		if masterOffset == nil || masterOffset.IsNA {
			c.logger.Debug().
				Str("address", address).
				Msg("master_repl_offset not available, skipping lag calculation")
			continue
		}

		if slaveOffset == nil || slaveOffset.IsNA {
			c.logger.Debug().
				Str("address", address).
				Msg("slave_repl_offset not available, skipping lag calculation")
			continue
		}

		// Calculate lag: master_offset - slave_offset
		lag := int64(masterOffset.RawValue) - int64(slaveOffset.RawValue)
		if lag < 0 {
			lag = 0 // Shouldn't happen, but protect against negative values
		}

		result.MasterReplOffset = int64(masterOffset.RawValue)
		result.SlaveReplOffset = int64(slaveOffset.RawValue)
		result.ReplicationLag = lag

		c.logger.Debug().
			Str("address", address).
			Int64("master_offset", result.MasterReplOffset).
			Int64("slave_offset", result.SlaveReplOffset).
			Int64("lag_bytes", lag).
			Msg("calculated replication lag for slave")
	}
}

// populateResultFields maps collected metrics to RedisInspectionResult struct fields.
// This method translates metric values to their corresponding struct fields.
func (c *RedisCollector) populateResultFields(result *model.RedisInspectionResult) {
	if result == nil {
		return
	}

	// redis_up -> ConnectionStatus
	if m := result.GetMetric("redis_up"); m != nil && !m.IsNA {
		result.ConnectionStatus = m.RawValue == 1
	}

	// redis_cluster_enabled -> ClusterEnabled
	if m := result.GetMetric("redis_cluster_enabled"); m != nil && !m.IsNA {
		result.ClusterEnabled = m.RawValue == 1
		if result.Instance != nil {
			result.Instance.SetClusterEnabled(result.ClusterEnabled)
		}
	}

	// redis_master_link_status -> MasterLinkStatus (slave only)
	if m := result.GetMetric("redis_master_link_status"); m != nil && !m.IsNA {
		result.MasterLinkStatus = m.RawValue == 1
	}

	// redis_maxclients -> MaxClients
	if m := result.GetMetric("redis_maxclients"); m != nil && !m.IsNA {
		result.MaxClients = int(m.RawValue)
	}

	// redis_connected_clients -> ConnectedClients
	if m := result.GetMetric("redis_connected_clients"); m != nil && !m.IsNA {
		result.ConnectedClients = int(m.RawValue)
	}

	// redis_connected_slaves -> ConnectedSlaves (master only)
	if m := result.GetMetric("redis_connected_slaves"); m != nil && !m.IsNA {
		result.ConnectedSlaves = int(m.RawValue)
	}

	// redis_master_port -> MasterPort (slave only)
	if m := result.GetMetric("redis_master_port"); m != nil && !m.IsNA {
		result.MasterPort = int(m.RawValue)
	}

	// redis_uptime_in_seconds -> Uptime
	if m := result.GetMetric("redis_uptime_in_seconds"); m != nil && !m.IsNA {
		result.Uptime = int64(m.RawValue)
	}

	// Set collected timestamp
	result.CollectedAt = time.Now()
}

// CollectMetrics retrieves metric data from VictoriaMetrics for all Redis instances.
//
// Flow:
//  1. Initialize result objects for each instance
//  2. Separate pending and active metrics
//  3. Set N/A for pending metrics
//  4. Concurrently collect active metrics (errgroup + concurrency limit)
//  5. Post-process: Role verification and replication lag calculation
//  6. Return results map (key = address)
//
// Single metric failure does not abort the entire collection.
func (c *RedisCollector) CollectMetrics(
	ctx context.Context,
	instances []*model.RedisInstance,
	metrics []*model.RedisMetricDefinition,
) (map[string]*model.RedisInspectionResult, error) {
	c.logger.Debug().
		Int("instance_count", len(instances)).
		Int("metric_count", len(metrics)).
		Msg("collecting Redis metrics from VictoriaMetrics")

	// Step 1: Initialize results map (indexed by address)
	resultsMap := make(map[string]*model.RedisInspectionResult, len(instances))
	for _, instance := range instances {
		resultsMap[instance.Address] = model.NewRedisInspectionResult(instance)
	}

	// Step 2: Separate pending and active metrics
	var pendingMetrics []*model.RedisMetricDefinition
	var activeMetrics []*model.RedisMetricDefinition

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

	// Step 5: Post-process
	c.verifyRoles(resultsMap)
	c.calculateReplicationLag(resultsMap)
	for _, result := range resultsMap {
		c.populateResultFields(result)
	}

	c.logger.Info().
		Int("instances", len(instances)).
		Int("active_metrics", len(activeMetrics)).
		Int("pending_metrics", len(pendingMetrics)).
		Msg("Redis metrics collection completed")

	return resultsMap, nil
}
