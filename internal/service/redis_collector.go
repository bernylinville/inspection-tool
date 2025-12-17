// Package service provides business logic services for the inspection tool.
package service

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

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
