// Package service provides business logic services for the inspection tool.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"inspection-tool/internal/client/n9e"
	"inspection-tool/internal/client/vm"
	"inspection-tool/internal/config"
	"inspection-tool/internal/model"
)

// FailedHost represents a host that failed during data collection.
type FailedHost struct {
	Hostname string // 主机名
	Error    string // 错误信息
}

// CollectionResult contains the results of a complete data collection operation.
type CollectionResult struct {
	Hosts       []*model.HostMeta             // 主机元信息列表
	HostMetrics map[string]*model.HostMetrics // 按主机名分组的指标数据
	FailedHosts []FailedHost                  // 采集失败的主机
	CollectedAt time.Time                     // 采集时间
}

// Collector is the data collection service that integrates N9E and VM clients.
type Collector struct {
	n9eClient  *n9e.Client
	vmClient   *vm.Client
	config     *config.Config
	metrics    []*model.MetricDefinition
	hostFilter *vm.HostFilter
	logger     zerolog.Logger
}

// NewCollector creates a new Collector instance.
func NewCollector(
	cfg *config.Config,
	n9eClient *n9e.Client,
	vmClient *vm.Client,
	metrics []*model.MetricDefinition,
	logger zerolog.Logger,
) *Collector {
	c := &Collector{
		n9eClient: n9eClient,
		vmClient:  vmClient,
		config:    cfg,
		metrics:   metrics,
		logger:    logger.With().Str("component", "collector").Logger(),
	}

	// Build VM host filter from config
	c.hostFilter = c.buildVMHostFilter()

	return c
}

// buildVMHostFilter converts config.HostFilter to vm.HostFilter.
func (c *Collector) buildVMHostFilter() *vm.HostFilter {
	if c.config == nil {
		return nil
	}

	cfgFilter := c.config.Inspection.HostFilter
	if len(cfgFilter.BusinessGroups) == 0 && len(cfgFilter.Tags) == 0 {
		return nil
	}

	return &vm.HostFilter{
		BusinessGroups: cfgFilter.BusinessGroups,
		Tags:           cfgFilter.Tags,
	}
}

// CollectAll executes the complete data collection workflow.
// It collects host metadata from N9E and metrics from VictoriaMetrics.
func (c *Collector) CollectAll(ctx context.Context) (*CollectionResult, error) {
	collectedAt := time.Now()
	c.logger.Info().Msg("starting data collection")

	// Step 1: Collect host metadata from N9E
	hosts, err := c.CollectHostMetas(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect host metas: %w", err)
	}

	if len(hosts) == 0 {
		c.logger.Warn().Msg("no hosts found from N9E")
		return &CollectionResult{
			Hosts:       hosts,
			HostMetrics: make(map[string]*model.HostMetrics),
			FailedHosts: nil,
			CollectedAt: collectedAt,
		}, nil
	}

	// Step 2: Collect metrics from VictoriaMetrics
	hostMetrics, err := c.CollectMetrics(ctx, hosts, c.metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to collect metrics: %w", err)
	}

	// Step 3: Identify hosts that have no metrics (potential failures)
	var failedHosts []FailedHost
	for _, host := range hosts {
		if hm, exists := hostMetrics[host.Hostname]; !exists || len(hm.Metrics) == 0 {
			failedHosts = append(failedHosts, FailedHost{
				Hostname: host.Hostname,
				Error:    "no metrics collected",
			})
		}
	}

	c.logger.Info().
		Int("total_hosts", len(hosts)).
		Int("hosts_with_metrics", len(hostMetrics)).
		Int("failed_hosts", len(failedHosts)).
		Msg("data collection completed")

	return &CollectionResult{
		Hosts:       hosts,
		HostMetrics: hostMetrics,
		FailedHosts: failedHosts,
		CollectedAt: collectedAt,
	}, nil
}

// CollectHostMetas retrieves host metadata from the N9E API.
func (c *Collector) CollectHostMetas(ctx context.Context) ([]*model.HostMeta, error) {
	c.logger.Debug().Msg("collecting host metas from N9E")

	hosts, err := c.n9eClient.GetHostMetas(ctx)
	if err != nil {
		c.logger.Error().Err(err).Msg("failed to get host metas from N9E")
		return nil, fmt.Errorf("N9E API error: %w", err)
	}

	c.logger.Info().Int("count", len(hosts)).Msg("collected host metas successfully")
	return hosts, nil
}

// CollectMetrics retrieves metric data from VictoriaMetrics for all hosts.
func (c *Collector) CollectMetrics(
	ctx context.Context,
	hosts []*model.HostMeta,
	metrics []*model.MetricDefinition,
) (map[string]*model.HostMetrics, error) {
	c.logger.Debug().
		Int("host_count", len(hosts)).
		Int("metric_count", len(metrics)).
		Msg("collecting metrics from VictoriaMetrics")

	// Initialize HostMetrics for each host
	hostMetricsMap := make(map[string]*model.HostMetrics, len(hosts))
	for _, host := range hosts {
		hostMetricsMap[host.Hostname] = model.NewHostMetrics(host.Hostname)
	}

	// Separate pending and active metrics
	var pendingMetrics []*model.MetricDefinition
	var activeMetrics []*model.MetricDefinition

	for _, metric := range metrics {
		if metric.IsPending() {
			pendingMetrics = append(pendingMetrics, metric)
		} else {
			activeMetrics = append(activeMetrics, metric)
		}
	}

	// Set N/A for pending metrics
	c.setPendingMetrics(hostMetricsMap, pendingMetrics)

	// Collect active metrics
	for _, metric := range activeMetrics {
		if metric.HasExpandLabel() {
			// Handle metrics that need to be expanded by label (e.g., disk by path)
			if err := c.collectExpandedMetric(ctx, metric, hostMetricsMap); err != nil {
				c.logger.Warn().
					Err(err).
					Str("metric", metric.Name).
					Msg("failed to collect expanded metric, continuing with other metrics")
			}
		} else {
			// Handle regular metrics
			if err := c.collectSimpleMetric(ctx, metric, hostMetricsMap); err != nil {
				c.logger.Warn().
					Err(err).
					Str("metric", metric.Name).
					Msg("failed to collect metric, continuing with other metrics")
			}
		}
	}

	return hostMetricsMap, nil
}

// collectSimpleMetric collects a single metric without label expansion.
func (c *Collector) collectSimpleMetric(
	ctx context.Context,
	metric *model.MetricDefinition,
	hostMetricsMap map[string]*model.HostMetrics,
) error {
	c.logger.Debug().
		Str("metric", metric.Name).
		Str("query", metric.Query).
		Msg("collecting simple metric")

	// Execute query with optional host filter
	results, err := c.vmClient.QueryByIdentWithFilter(ctx, metric.Query, c.hostFilter)
	if err != nil {
		return fmt.Errorf("query failed for %s: %w", metric.Name, err)
	}

	// Map results to hosts
	matchedCount := 0
	for ident, result := range results {
		// Try to match by hostname (clean ident)
		hostname := model.CleanIdent(ident)
		if hostMetrics, exists := hostMetricsMap[hostname]; exists {
			mv := model.NewMetricValue(metric.Name, result.Value)
			mv.Timestamp = time.Now().Unix()
			hostMetrics.SetMetric(mv)
			matchedCount++
		}
	}

	c.logger.Debug().
		Str("metric", metric.Name).
		Int("results", len(results)).
		Int("matched", matchedCount).
		Msg("simple metric collected")

	return nil
}

// collectExpandedMetric collects a metric that should be expanded by a label.
// For example, disk metrics expanded by "path" label.
func (c *Collector) collectExpandedMetric(
	ctx context.Context,
	metric *model.MetricDefinition,
	hostMetricsMap map[string]*model.HostMetrics,
) error {
	c.logger.Debug().
		Str("metric", metric.Name).
		Str("expand_by", metric.ExpandByLabel).
		Str("query", metric.Query).
		Msg("collecting expanded metric")

	// Execute query - need raw results to access labels
	results, err := c.vmClient.QueryResultsWithFilter(ctx, metric.Query, c.hostFilter)
	if err != nil {
		return fmt.Errorf("query failed for %s: %w", metric.Name, err)
	}

	// Group results by host and track max value for aggregation
	hostMaxValues := make(map[string]float64)
	hostExpandedMetrics := make(map[string][]*model.MetricValue)

	for _, result := range results {
		hostname := model.CleanIdent(result.Ident)
		if hostname == "" {
			continue
		}

		// Check if host exists in our map
		if _, exists := hostMetricsMap[hostname]; !exists {
			continue
		}

		// Get the expansion label value (e.g., path)
		labelValue := result.Labels[metric.ExpandByLabel]
		if labelValue == "" {
			labelValue = "unknown"
		}

		// Create expanded metric name (e.g., "disk_usage:/home")
		expandedName := fmt.Sprintf("%s:%s", metric.Name, labelValue)

		// Create metric value with labels
		mv := model.NewMetricValue(expandedName, result.Value)
		mv.Timestamp = time.Now().Unix()
		mv.Labels = map[string]string{
			metric.ExpandByLabel: labelValue,
		}

		// Track for this host
		hostExpandedMetrics[hostname] = append(hostExpandedMetrics[hostname], mv)

		// Track max value for aggregation
		if current, exists := hostMaxValues[hostname]; !exists || result.Value > current {
			hostMaxValues[hostname] = result.Value
		}
	}

	// Apply expanded metrics to hosts
	for hostname, metrics := range hostExpandedMetrics {
		hostMetrics := hostMetricsMap[hostname]
		for _, mv := range metrics {
			hostMetrics.SetMetric(mv)
		}
	}

	// Apply aggregated max value for alert evaluation
	if metric.Aggregate == model.AggregateMax {
		for hostname, maxValue := range hostMaxValues {
			if hostMetrics, exists := hostMetricsMap[hostname]; exists {
				aggregatedName := fmt.Sprintf("%s_max", metric.Name)
				mv := model.NewMetricValue(aggregatedName, maxValue)
				mv.Timestamp = time.Now().Unix()
				hostMetrics.SetMetric(mv)
			}
		}
	}

	c.logger.Debug().
		Str("metric", metric.Name).
		Int("results", len(results)).
		Int("hosts_with_data", len(hostExpandedMetrics)).
		Msg("expanded metric collected")

	return nil
}

// setPendingMetrics sets N/A values for all pending metrics on all hosts.
func (c *Collector) setPendingMetrics(
	hostMetricsMap map[string]*model.HostMetrics,
	pendingMetrics []*model.MetricDefinition,
) {
	if len(pendingMetrics) == 0 {
		return
	}

	c.logger.Debug().
		Int("pending_count", len(pendingMetrics)).
		Msg("setting N/A for pending metrics")

	for _, metric := range pendingMetrics {
		for _, hostMetrics := range hostMetricsMap {
			mv := model.NewNAMetricValue(metric.Name)
			hostMetrics.SetMetric(mv)
		}
	}
}
