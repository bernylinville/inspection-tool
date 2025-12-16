// Package service provides business logic services for the inspection tool.
package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

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
