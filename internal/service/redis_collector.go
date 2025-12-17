// Package service provides business logic services for the inspection tool.
package service

import (
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
