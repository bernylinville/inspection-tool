// Package service provides business logic services for the inspection tool.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"inspection-tool/internal/config"
	"inspection-tool/internal/model"
)

// RedisInspector orchestrates the complete Redis inspection workflow, coordinating
// instance discovery, data collection, threshold evaluation, and result aggregation.
type RedisInspector struct {
	collector *RedisCollector
	evaluator *RedisEvaluator
	config    *config.Config
	timezone  *time.Location
	version   string
	logger    zerolog.Logger
}

// RedisInspectorOption is a functional option for configuring a RedisInspector.
type RedisInspectorOption func(*RedisInspector)

// NewRedisInspector creates a new RedisInspector with the given dependencies.
//
// Parameters:
//   - cfg: Complete configuration including Redis inspection config
//   - collector: Redis data collector
//   - evaluator: Threshold evaluator
//   - logger: Structured logger
//   - opts: Optional configuration via functional options
//
// Returns:
//   - *RedisInspector: Configured inspector instance
//   - error: Timezone loading error or validation failure
func NewRedisInspector(
	cfg *config.Config,
	collector *RedisCollector,
	evaluator *RedisEvaluator,
	logger zerolog.Logger,
	opts ...RedisInspectorOption,
) (*RedisInspector, error) {
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

	i := &RedisInspector{
		collector: collector,
		evaluator: evaluator,
		config:    cfg,
		timezone:  loc,
		version:   "dev",
		logger:    logger.With().Str("component", "redis_inspector").Logger(),
	}

	for _, opt := range opts {
		opt(i)
	}

	return i, nil
}

// WithRedisVersion sets the tool version to include in the inspection result.
func WithRedisVersion(version string) RedisInspectorOption {
	return func(i *RedisInspector) {
		i.version = version
	}
}

// GetTimezone returns the configured timezone.
func (i *RedisInspector) GetTimezone() *time.Location {
	return i.timezone
}

// GetVersion returns the configured version.
func (i *RedisInspector) GetVersion() string {
	return i.version
}

// Inspect executes the complete Redis inspection workflow:
// 1. Discovers Redis instances
// 2. Collects metrics for all instances
// 3. Evaluates thresholds and generates alerts
// 4. Aggregates results into RedisInspectionResults
//
// Returns:
//   - *model.RedisInspectionResults: Complete inspection result with summary
//   - error: Fatal errors that prevent inspection (discovery/config loading failures)
func (i *RedisInspector) Inspect(ctx context.Context) (*model.RedisInspectionResults, error) {
	// Step 1: Record start time in configured timezone
	startTime := time.Now().In(i.timezone)
	i.logger.Info().
		Time("start_time", startTime).
		Str("timezone", i.timezone.String()).
		Msg("starting Redis inspection")

	// Step 2: Create result container
	result := model.NewRedisInspectionResults(startTime)
	result.Version = i.version

	// Step 3: Discover instances
	i.logger.Debug().Msg("step 1: discovering Redis instances")
	instances, err := i.collector.DiscoverInstances(ctx)
	if err != nil {
		i.logger.Error().Err(err).Msg("instance discovery failed")
		return nil, fmt.Errorf("instance discovery failed: %w", err)
	}

	// Step 4: Handle empty instance list (graceful degradation)
	if len(instances) == 0 {
		i.logger.Warn().Msg("no Redis instances found, completing inspection with empty result")
		endTime := time.Now().In(i.timezone)
		result.Finalize(endTime)
		return result, nil
	}

	i.logger.Info().Int("instance_count", len(instances)).Msg("discovered Redis instances")

	// Step 5: Load metric definitions
	i.logger.Debug().Msg("step 2: loading Redis metric definitions")
	metrics := i.collector.GetMetrics()
	if len(metrics) == 0 {
		i.logger.Error().Msg("no Redis metrics defined")
		return nil, fmt.Errorf("no Redis metrics defined")
	}

	// Step 6: Collect metrics
	i.logger.Debug().
		Int("instance_count", len(instances)).
		Int("metric_count", len(metrics)).
		Msg("step 3: collecting metrics")

	resultsMap, err := i.collector.CollectMetrics(ctx, instances, metrics)
	if err != nil {
		i.logger.Error().Err(err).Msg("metrics collection failed")
		return nil, fmt.Errorf("metrics collection failed: %w", err)
	}

	// Step 7: Evaluate thresholds
	i.logger.Debug().
		Int("results_count", len(resultsMap)).
		Msg("step 4: evaluating thresholds")

	_ = i.evaluator.EvaluateAll(resultsMap)

	// Step 8: Build inspection results
	i.logger.Debug().Msg("step 5: building inspection results")
	i.buildInspectionResults(result, resultsMap)

	// Step 9: Finalize (calculate Duration, Summary, AlertSummary)
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
		Msg("Redis inspection completed")

	return result, nil
}

// buildInspectionResults merges collection results into RedisInspectionResults.
func (i *RedisInspector) buildInspectionResults(
	result *model.RedisInspectionResults,
	resultsMap map[string]*model.RedisInspectionResult,
) {
	for _, inspResult := range resultsMap {
		if inspResult == nil {
			continue
		}

		// Convert timestamp to configured timezone
		inspResult.CollectedAt = inspResult.CollectedAt.In(i.timezone)

		// Add to result container (automatically aggregates alerts)
		result.AddResult(inspResult)
	}

	i.logger.Debug().
		Int("total_results", len(result.Results)).
		Int("total_alerts", len(result.Alerts)).
		Msg("inspection results merged")
}
