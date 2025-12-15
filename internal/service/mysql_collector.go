// Package service provides business logic services for the inspection tool.
package service

import (
	"github.com/rs/zerolog"

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
