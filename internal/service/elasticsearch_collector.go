package service

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"inspection-tool/internal/client/vm"
	"inspection-tool/internal/config"
	"inspection-tool/internal/model"
	"inspection-tool/internal/util"
)

type ElasticsearchCollector struct {
	vmClient       *vm.Client
	config         *config.ElasticsearchInspectionConfig
	metrics        []*model.ElasticsearchMetricDefinition
	instanceFilter *ElasticsearchInstanceFilter
	logger         zerolog.Logger
}

type ElasticsearchInstanceFilter struct {
	AddressPatterns []string
	NamePatterns    []string
	BusinessGroups  []string
	Tags            map[string]string
}

func NewElasticsearchCollector(
	cfg *config.ElasticsearchInspectionConfig,
	vmClient *vm.Client,
	metrics []*model.ElasticsearchMetricDefinition,
	logger zerolog.Logger,
) *ElasticsearchCollector {
	c := &ElasticsearchCollector{
		vmClient: vmClient,
		config:   cfg,
		metrics:  metrics,
		logger:   logger.With().Str("component", "elasticsearch-collector").Logger(),
	}
	c.instanceFilter = c.buildInstanceFilter()
	return c
}

func (c *ElasticsearchCollector) buildInstanceFilter() *ElasticsearchInstanceFilter {
	if c.config == nil {
		return nil
	}

	filter := c.config.InstanceFilter
	if len(filter.AddressPatterns) == 0 &&
		len(filter.NamePatterns) == 0 &&
		len(filter.BusinessGroups) == 0 &&
		len(filter.Tags) == 0 {
		return nil
	}

	return &ElasticsearchInstanceFilter{
		AddressPatterns: filter.AddressPatterns,
		NamePatterns:    filter.NamePatterns,
		BusinessGroups:  filter.BusinessGroups,
		Tags:            filter.Tags,
	}
}

func (c *ElasticsearchCollector) GetConfig() *config.ElasticsearchInspectionConfig {
	return c.config
}

func (c *ElasticsearchCollector) GetMetrics() []*model.ElasticsearchMetricDefinition {
	return c.metrics
}

func (c *ElasticsearchCollector) GetInstanceFilter() *ElasticsearchInstanceFilter {
	return c.instanceFilter
}

func (f *ElasticsearchInstanceFilter) IsEmpty() bool {
	if f == nil {
		return true
	}
	return len(f.AddressPatterns) == 0 &&
		len(f.NamePatterns) == 0 &&
		len(f.BusinessGroups) == 0 &&
		len(f.Tags) == 0
}

func (f *ElasticsearchInstanceFilter) ToVMHostFilter() *vm.HostFilter {
	if f == nil || f.IsEmpty() {
		return nil
	}

	if len(f.BusinessGroups) == 0 && len(f.Tags) == 0 {
		return nil
	}

	return &vm.HostFilter{
		BusinessGroups: f.BusinessGroups,
		Tags:           f.Tags,
	}
}

func (c *ElasticsearchCollector) DiscoverInstances(ctx context.Context) ([]*model.ElasticsearchInstance, error) {
	c.logger.Info().Msg("starting Elasticsearch instance discovery")

	vmFilter := c.instanceFilter.ToVMHostFilter()

	query := "elasticsearch_jvm_mem_heap_used_percent"
	results, err := queryResultsWithHostFilterFallback(ctx, c.vmClient, c.logger, query, vmFilter)
	if err != nil {
		c.logger.Error().Err(err).Msg("failed to query elasticsearch_jvm_mem_heap_used_percent metric")
		return nil, fmt.Errorf("failed to query elasticsearch metrics: %w", err)
	}

	c.logger.Debug().Int("raw_results", len(results)).Msg("received elasticsearch query results")

	var instances []*model.ElasticsearchInstance
	seenNodes := make(map[string]bool)

	for _, result := range results {
		nodeName := result.Labels["name"]
		nodeHost := result.Labels["host"]
		clusterName := result.Labels["cluster"]

		if nodeName == "" && nodeHost == "" {
			c.logger.Warn().Interface("labels", result.Labels).Msg("missing name and host labels")
			continue
		}

		nodeID := nodeName
		if nodeID == "" {
			nodeID = nodeHost
		}

		if seenNodes[nodeID] {
			c.logger.Debug().Str("node", nodeID).Msg("skipping duplicate node")
			continue
		}

		if !c.matchesNamePatterns(nodeName) {
			c.logger.Debug().Str("name", nodeName).Msg("node name filtered out")
			continue
		}

		if !c.matchesAddressPatterns(nodeHost) {
			c.logger.Debug().Str("host", nodeHost).Msg("node host filtered out")
			continue
		}

		instance := &model.ElasticsearchInstance{
			Address:     fmt.Sprintf("%s:9200", nodeHost),
			IP:          nodeHost,
			Port:        9200,
			ClusterName: clusterName,
		}

		instances = append(instances, instance)
		seenNodes[nodeID] = true
	}

	if err := c.enrichInstancesWithRoles(ctx, instances, vmFilter); err != nil {
		c.logger.Warn().Err(err).Msg("failed to enrich instances with roles, continuing without roles")
	}

	c.logger.Info().
		Int("discovered", len(instances)).
		Int("filtered_out", len(results)-len(instances)).
		Msg("Elasticsearch instance discovery completed")

	return instances, nil
}

func (c *ElasticsearchCollector) enrichInstancesWithRoles(
	ctx context.Context,
	instances []*model.ElasticsearchInstance,
	vmFilter *vm.HostFilter,
) error {
	query := `elasticsearch_nodes_roles{role="master"}`
	results, err := queryResultsWithHostFilterFallback(ctx, c.vmClient, c.logger, query, vmFilter)
	if err != nil {
		return fmt.Errorf("failed to query node roles: %w", err)
	}

	hostToRoles := make(map[string][]string)
	for _, result := range results {
		host := result.Labels["host"]
		role := result.Labels["role"]
		if host != "" && role != "" && result.Value == 1 {
			hostToRoles[host] = append(hostToRoles[host], role)
		}
	}

	for _, instance := range instances {
		if roles, ok := hostToRoles[instance.IP]; ok {
			for _, role := range roles {
				if role == "master" {
					instance.Role = model.ElasticsearchRoleMaster
					break
				}
			}
			if instance.Role == "" {
				if len(roles) > 0 {
					instance.Role = model.ElasticsearchRole(roles[0])
				}
			}
		}
	}

	return nil
}

func (c *ElasticsearchCollector) matchesNamePatterns(name string) bool {
	if c.instanceFilter == nil || len(c.instanceFilter.NamePatterns) == 0 {
		return true
	}
	return util.MatchAnyPattern(name, c.instanceFilter.NamePatterns)
}

func (c *ElasticsearchCollector) matchesAddressPatterns(host string) bool {
	if c.instanceFilter == nil || len(c.instanceFilter.AddressPatterns) == 0 {
		return true
	}
	return util.MatchAnyPattern(host, c.instanceFilter.AddressPatterns)
}

func (c *ElasticsearchCollector) setPendingMetrics(
	resultsMap map[string]*model.ElasticsearchInspectionResult,
	pendingMetrics []*model.ElasticsearchMetricDefinition,
) {
	if len(pendingMetrics) == 0 {
		return
	}

	c.logger.Debug().
		Int("pending_count", len(pendingMetrics)).
		Msg("setting N/A for pending Elasticsearch metrics")

	for _, metric := range pendingMetrics {
		for _, result := range resultsMap {
			mv := &model.ElasticsearchMetricValue{
				Name:           metric.Name,
				RawValue:       0,
				FormattedValue: "N/A",
				IsNA:           true,
			}
			result.SetMetric(mv)
		}
	}
}

func (c *ElasticsearchCollector) collectMetricConcurrent(
	ctx context.Context,
	metric *model.ElasticsearchMetricDefinition,
	instances []*model.ElasticsearchInstance,
	resultsMap map[string]*model.ElasticsearchInspectionResult,
	mu *sync.Mutex,
) error {
	if metric.HasLabelExtract() {
		return c.collectLabelExtractMetric(ctx, metric, instances, resultsMap, mu)
	}

	c.logger.Debug().
		Str("metric", metric.Name).
		Str("query", metric.Query).
		Msg("collecting Elasticsearch metric (concurrent)")

	vmFilter := c.instanceFilter.ToVMHostFilter()
	results, err := queryResultsWithHostFilterFallback(ctx, c.vmClient, c.logger, metric.Query, vmFilter)
	if err != nil {
		return fmt.Errorf("query failed for %s: %w", metric.Name, err)
	}

	hostMap := make(map[string]*model.ElasticsearchInstance, len(instances))
	for _, instance := range instances {
		hostMap[instance.IP] = instance
	}

	mu.Lock()
	defer mu.Unlock()

	matchedCount := 0
	for _, result := range results {
		host := result.Labels["host"]
		if host == "" {
			host = c.extractHostFromAddress(result.Labels["address"])
		}
		if host == "" {
			continue
		}

		if !c.matchesAddressPatterns(host) {
			continue
		}

		if _, exists := hostMap[host]; !exists {
			continue
		}

		nodeKey := host + ":9200"
		if inspResult, ok := resultsMap[nodeKey]; ok {
			mv := &model.ElasticsearchMetricValue{
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
		Msg("Elasticsearch metric collected (concurrent)")

	return nil
}

func (c *ElasticsearchCollector) collectLabelExtractMetric(
	ctx context.Context,
	metric *model.ElasticsearchMetricDefinition,
	instances []*model.ElasticsearchInstance,
	resultsMap map[string]*model.ElasticsearchInspectionResult,
	mu *sync.Mutex,
) error {
	c.logger.Debug().
		Str("metric", metric.Name).
		Str("label_extract", metric.LabelExtract).
		Msg("collecting label extract metric")

	vmFilter := c.instanceFilter.ToVMHostFilter()
	results, err := queryResultsWithHostFilterFallback(ctx, c.vmClient, c.logger, metric.Query, vmFilter)
	if err != nil {
		return fmt.Errorf("query failed for %s: %w", metric.Name, err)
	}

	hostMap := make(map[string]*model.ElasticsearchInstance, len(instances))
	for _, instance := range instances {
		hostMap[instance.IP] = instance
	}

	mu.Lock()
	defer mu.Unlock()

	matchedCount := 0
	for _, result := range results {
		host := result.Labels["host"]
		if host == "" {
			host = c.extractHostFromAddress(result.Labels["address"])
		}
		if host == "" {
			continue
		}

		if !c.matchesAddressPatterns(host) {
			continue
		}

		if _, exists := hostMap[host]; !exists {
			continue
		}

		extractedValue := result.Labels[metric.LabelExtract]
		if extractedValue == "" {
			c.logger.Warn().
				Str("metric", metric.Name).
				Str("host", host).
				Str("label", metric.LabelExtract).
				Msg("label value not found")
			continue
		}

		nodeKey := host + ":9200"
		if inspResult, ok := resultsMap[nodeKey]; ok {
			mv := &model.ElasticsearchMetricValue{
				Name:        metric.Name,
				RawValue:    result.Value,
				StringValue: extractedValue,
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

func (c *ElasticsearchCollector) extractHostFromAddress(address string) string {
	if address == "" {
		return ""
	}
	parsed, err := url.Parse(address)
	if err != nil {
		return ""
	}
	return parsed.Hostname()
}

func (c *ElasticsearchCollector) populateResultFields(result *model.ElasticsearchInspectionResult) {
	if result == nil {
		return
	}

	if m := result.GetMetric("node_up"); m != nil && !m.IsNA {
		result.ConnectionStatus = m.RawValue == 1
	}

	if m := result.GetMetric("heap_memory_percent"); m != nil && !m.IsNA {
		result.HeapMemoryPercent = m.RawValue
	}

	if m := result.GetMetric("cpu_percent"); m != nil && !m.IsNA {
		result.CPUPercent = m.RawValue
	}

	if m := result.GetMetric("disk_usage_percent"); m != nil && !m.IsNA {
		result.DiskUsagePercent = m.RawValue
	}

	if m := result.GetMetric("file_handle_usage_percent"); m != nil && !m.IsNA {
		result.FileHandleUsagePercent = m.RawValue
	}

	if m := result.GetMetric("circuit_breaker_tripped"); m != nil && !m.IsNA {
		result.CircuitBreakerTripped = m.RawValue > 0
	}

	if m := result.GetMetric("thread_pool_rejected"); m != nil && !m.IsNA {
		result.ThreadPoolRejected = m.RawValue > 0
	}

	if m := result.GetMetric("gc_collection_seconds"); m != nil && !m.IsNA {
		result.GCDurationSeconds = m.RawValue
	}

	if m := result.GetMetric("uptime_seconds"); m != nil && !m.IsNA {
		result.Uptime = int64(m.RawValue)
	}

	if m := result.GetMetric("cluster_name"); m != nil && !m.IsNA && m.StringValue != "" {
		if result.Instance != nil {
			result.Instance.SetClusterName(m.StringValue)
		}
	}

	result.CollectedAt = time.Now()
}

func (c *ElasticsearchCollector) CollectMetrics(
	ctx context.Context,
	instances []*model.ElasticsearchInstance,
	metrics []*model.ElasticsearchMetricDefinition,
) (map[string]*model.ElasticsearchInspectionResult, error) {
	c.logger.Debug().
		Int("instance_count", len(instances)).
		Int("metric_count", len(metrics)).
		Msg("collecting Elasticsearch metrics from VictoriaMetrics")

	resultsMap := make(map[string]*model.ElasticsearchInspectionResult, len(instances))
	for _, instance := range instances {
		resultsMap[instance.Address] = model.NewElasticsearchInspectionResult(instance)
	}

	var pendingMetrics []*model.ElasticsearchMetricDefinition
	var activeMetrics []*model.ElasticsearchMetricDefinition

	for _, metric := range metrics {
		if metric.IsPending() {
			pendingMetrics = append(pendingMetrics, metric)
		} else {
			activeMetrics = append(activeMetrics, metric)
		}
	}

	c.setPendingMetrics(resultsMap, pendingMetrics)

	if len(activeMetrics) == 0 {
		c.logger.Warn().Msg("no active metrics to collect")
		return resultsMap, nil
	}

	g, ctx := errgroup.WithContext(ctx)
	concurrency := 20
	g.SetLimit(concurrency)

	var mu sync.Mutex

	for _, metric := range activeMetrics {
		metric := metric
		g.Go(func() error {
			err := c.collectMetricConcurrent(ctx, metric, instances, resultsMap, &mu)
			if err != nil {
				c.logger.Warn().
					Err(err).
					Str("metric", metric.Name).
					Msg("failed to collect metric, continuing with others")
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("concurrent metric collection failed: %w", err)
	}

	for _, result := range resultsMap {
		c.populateResultFields(result)
	}

	c.logger.Info().
		Int("instances", len(instances)).
		Int("active_metrics", len(activeMetrics)).
		Int("pending_metrics", len(pendingMetrics)).
		Msg("Elasticsearch metrics collection completed")

	return resultsMap, nil
}
