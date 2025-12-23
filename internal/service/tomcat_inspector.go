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

// TomcatInspector orchestrates the complete Tomcat inspection workflow, coordinating
// instance discovery, data collection, threshold evaluation, and result aggregation.
type TomcatInspector struct {
	collector *TomcatCollector
	evaluator *TomcatEvaluator
	config    *config.Config
	timezone  *time.Location
	version   string
	logger    zerolog.Logger
}

// TomcatInspectorOption is a functional option for configuring a TomcatInspector.
type TomcatInspectorOption func(*TomcatInspector)

// NewTomcatInspector creates a new TomcatInspector with the given dependencies.
//
// Parameters:
//   - cfg: Complete configuration including Tomcat inspection config
//   - collector: Tomcat data collector
//   - evaluator: Threshold evaluator
//   - logger: Structured logger
//   - opts: Optional configuration via functional options
//
// Returns:
//   - *TomcatInspector: Configured inspector instance
//   - error: Timezone loading error or validation failure
func NewTomcatInspector(
	cfg *config.Config,
	collector *TomcatCollector,
	evaluator *TomcatEvaluator,
	logger zerolog.Logger,
	opts ...TomcatInspectorOption,
) (*TomcatInspector, error) {
	// Validate required parameters
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if collector == nil {
		return nil, fmt.Errorf("collector cannot be nil")
	}
	if evaluator == nil {
		return nil, fmt.Errorf("evaluator cannot be nil")
	}

	// Determine timezone (from config or use default)
	tzName := defaultTimezone
	if cfg.Report.Timezone != "" {
		tzName = cfg.Report.Timezone
	}

	// Load timezone
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return nil, fmt.Errorf("failed to load timezone %s: %w", tzName, err)
	}

	i := &TomcatInspector{
		collector: collector,
		evaluator: evaluator,
		config:    cfg,
		timezone:  loc,
		version:   "dev",
		logger:    logger.With().Str("component", "tomcat_inspector").Logger(),
	}

	// Apply functional options
	for _, opt := range opts {
		opt(i)
	}

	return i, nil
}

// WithTomcatVersion sets the tool version to include in the inspection result.
func WithTomcatVersion(version string) TomcatInspectorOption {
	return func(i *TomcatInspector) {
		i.version = version
	}
}

// GetTimezone returns the configured timezone.
func (i *TomcatInspector) GetTimezone() *time.Location {
	return i.timezone
}

// GetVersion returns the configured version.
func (i *TomcatInspector) GetVersion() string {
	return i.version
}

// Inspect executes the complete Tomcat inspection workflow:
// 1. Discovers Tomcat instances
// 2. Collects metrics for all instances
// 3. Evaluates thresholds and generates alerts
// 4. Aggregates results into TomcatInspectionResults
//
// Returns:
//   - *model.TomcatInspectionResults: Complete inspection result with summary
//   - error: Fatal errors that prevent inspection (discovery/config loading failures)
func (i *TomcatInspector) Inspect(ctx context.Context) (*model.TomcatInspectionResults, error) {
	// Step 1: Record start time (Asia/Shanghai)
	startTime := time.Now().In(i.timezone)
	i.logger.Info().
		Time("start_time", startTime).
		Str("timezone", i.timezone.String()).
		Msg("starting Tomcat inspection")

	// Step 2: Create result container
	result := model.NewTomcatInspectionResults(startTime)
	result.Version = i.version

	// Step 3: Discover instances
	i.logger.Debug().Msg("step 1: discovering Tomcat instances")
	instances, err := i.collector.DiscoverInstances(ctx)
	if err != nil {
		i.logger.Error().Err(err).Msg("instance discovery failed")
		return nil, fmt.Errorf("instance discovery failed: %w", err)
	}

	// Step 4: Handle empty instance list (graceful degradation)
	if len(instances) == 0 {
		i.logger.Warn().Msg("no Tomcat instances found, completing inspection with empty result")
		endTime := time.Now().In(i.timezone)
		result.Finalize(endTime)
		return result, nil
	}

	i.logger.Info().Int("instance_count", len(instances)).Msg("discovered Tomcat instances")

	// Step 5: Load metric definitions (use collector's internal metrics)
	i.logger.Debug().Msg("step 2: loading Tomcat metric definitions")
	metrics := i.collector.GetMetrics()
	if len(metrics) == 0 {
		i.logger.Error().Msg("no Tomcat metrics defined")
		return nil, fmt.Errorf("no Tomcat metrics defined")
	}

	i.logger.Debug().
		Int("instance_count", len(instances)).
		Int("metric_count", len(metrics)).
		Msg("step 3: collecting metrics")

	resultsMap, err := i.collector.CollectMetrics(ctx, instances, metrics)
	if err != nil {
		i.logger.Error().Err(err).Msg("metrics collection failed")
		return nil, fmt.Errorf("metrics collection failed: %w", err)
	}

	// Step 6: Evaluate thresholds
	i.logger.Debug().
		Int("results_count", len(resultsMap)).
		Msg("step 4: evaluating thresholds")

	_ = i.evaluator.EvaluateAll(resultsMap)

	// Step 7: Build results
	i.logger.Debug().Msg("step 5: building inspection results")
	i.buildInspectionResults(result, resultsMap)

	// Step 8: Finalize (calculate Duration, Summary, AlertSummary)
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
		Msg("Tomcat inspection completed")

	// Step 9: Log critical alerts if any
	if result.HasCritical() {
		i.logger.Warn().
			Int("critical_count", result.Summary.CriticalInstances).
			Int("critical_alerts", result.AlertSummary.CriticalCount).
			Msg("Tomcat inspection found critical issues")
	}

	return result, nil
}

// buildInspectionResults merges collection results into TomcatInspectionResults.
func (i *TomcatInspector) buildInspectionResults(
	result *model.TomcatInspectionResults,
	resultsMap map[string]*model.TomcatInspectionResult,
) {
	// Iterate through all instance results
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

// IsEnabled returns true if Tomcat inspection is enabled in the configuration.
func (i *TomcatInspector) IsEnabled() bool {
	return i.config != nil && i.config.Tomcat.Enabled
}

// GetConfig returns the Tomcat inspection configuration.
func (i *TomcatInspector) GetConfig() *config.TomcatInspectionConfig {
	if i.config == nil {
		return nil
	}
	return &i.config.Tomcat
}
