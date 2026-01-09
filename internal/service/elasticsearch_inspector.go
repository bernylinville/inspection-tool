package service

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"inspection-tool/internal/config"
	"inspection-tool/internal/model"
)

type ElasticsearchInspector struct {
	collector ElasticsearchInstanceCollector
	evaluator ElasticsearchInstanceEvaluator
	config    *config.Config
	timezone  *time.Location
	version   string
	logger    zerolog.Logger
}

type ElasticsearchInspectorOption func(*ElasticsearchInspector)

func NewElasticsearchInspector(
	cfg *config.Config,
	collector ElasticsearchInstanceCollector,
	evaluator ElasticsearchInstanceEvaluator,
	logger zerolog.Logger,
	opts ...ElasticsearchInspectorOption,
) (*ElasticsearchInspector, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if collector == nil {
		return nil, fmt.Errorf("collector cannot be nil")
	}
	if evaluator == nil {
		return nil, fmt.Errorf("evaluator cannot be nil")
	}

	tzName := defaultTimezone
	if cfg.Report.Timezone != "" {
		tzName = cfg.Report.Timezone
	}

	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return nil, fmt.Errorf("failed to load timezone %s: %w", tzName, err)
	}

	i := &ElasticsearchInspector{
		collector: collector,
		evaluator: evaluator,
		config:    cfg,
		timezone:  loc,
		version:   "dev",
		logger:    logger.With().Str("component", "elasticsearch_inspector").Logger(),
	}

	for _, opt := range opts {
		opt(i)
	}

	return i, nil
}

func WithElasticsearchVersion(version string) ElasticsearchInspectorOption {
	return func(i *ElasticsearchInspector) {
		i.version = version
	}
}

func (i *ElasticsearchInspector) GetTimezone() *time.Location {
	return i.timezone
}

func (i *ElasticsearchInspector) GetVersion() string {
	return i.version
}

func (i *ElasticsearchInspector) Inspect(ctx context.Context) (*model.ElasticsearchInspectionResults, error) {
	startTime := time.Now().In(i.timezone)
	i.logger.Info().
		Time("start_time", startTime).
		Str("timezone", i.timezone.String()).
		Msg("starting Elasticsearch inspection")

	result := model.NewElasticsearchInspectionResults(startTime)
	result.Version = i.version

	i.logger.Debug().Msg("step 1: discovering Elasticsearch instances")
	instances, err := i.collector.DiscoverInstances(ctx)
	if err != nil {
		i.logger.Error().Err(err).Msg("instance discovery failed")
		return nil, fmt.Errorf("instance discovery failed: %w", err)
	}

	if len(instances) == 0 {
		i.logger.Warn().Msg("no Elasticsearch instances found, completing inspection with empty result")
		endTime := time.Now().In(i.timezone)
		result.Finalize(endTime)
		return result, nil
	}

	i.logger.Info().Int("instance_count", len(instances)).Msg("discovered Elasticsearch instances")

	i.logger.Debug().Msg("step 2: loading Elasticsearch metric definitions")
	metrics := i.collector.GetMetrics()
	if len(metrics) == 0 {
		i.logger.Error().Msg("no Elasticsearch metrics defined")
		return nil, fmt.Errorf("no Elasticsearch metrics defined")
	}

	i.logger.Debug().
		Int("instance_count", len(instances)).
		Int("metric_count", len(metrics)).
		Msg("step 3: collecting Elasticsearch metrics")

	resultsMap, err := i.collector.CollectMetrics(ctx, instances, metrics)
	if err != nil {
		i.logger.Error().Err(err).Msg("metrics collection failed")
		return nil, fmt.Errorf("metrics collection failed: %w", err)
	}

	i.logger.Debug().Msg("step 4: evaluating thresholds")
	evalResults := i.evaluator.EvaluateAll(resultsMap)

	i.logger.Debug().Msg("step 5: aggregating results")
	for _, evalResult := range evalResults {
		if inspResult, ok := resultsMap[evalResult.Address]; ok {
			result.Results = append(result.Results, inspResult)
			result.Alerts = append(result.Alerts, inspResult.Alerts...)
		}
	}

	endTime := time.Now().In(i.timezone)
	result.Finalize(endTime)

	i.logger.Info().
		Int("total_instances", result.Summary.TotalInstances).
		Int("normal_instances", result.Summary.NormalInstances).
		Int("warning_instances", result.Summary.WarningInstances).
		Int("critical_instances", result.Summary.CriticalInstances).
		Int("failed_instances", result.Summary.FailedInstances).
		Int("total_alerts", result.AlertSummary.TotalAlerts).
		Dur("duration", result.Duration).
		Msg("Elasticsearch inspection completed")

	return result, nil
}
