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

const defaultTimezone = "Asia/Shanghai"

// Inspector orchestrates the complete inspection workflow, coordinating
// data collection, threshold evaluation, and result aggregation.
type Inspector struct {
	collector *Collector
	evaluator *Evaluator
	config    *config.Config
	timezone  *time.Location
	version   string
	logger    zerolog.Logger
}

// InspectorOption is a functional option for configuring an Inspector.
type InspectorOption func(*Inspector)

// NewInspector creates a new Inspector with the given dependencies.
func NewInspector(
	cfg *config.Config,
	collector *Collector,
	evaluator *Evaluator,
	logger zerolog.Logger,
	opts ...InspectorOption,
) (*Inspector, error) {
	// Determine timezone from config or use default
	tzName := defaultTimezone
	if cfg != nil && cfg.Report.Timezone != "" {
		tzName = cfg.Report.Timezone
	}

	// Load timezone
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return nil, fmt.Errorf("failed to load timezone %s: %w", tzName, err)
	}

	i := &Inspector{
		collector: collector,
		evaluator: evaluator,
		config:    cfg,
		timezone:  loc,
		version:   "dev",
		logger:    logger.With().Str("component", "inspector").Logger(),
	}

	// Apply options
	for _, opt := range opts {
		opt(i)
	}

	return i, nil
}

// WithVersion sets the tool version to include in the inspection result.
func WithVersion(version string) InspectorOption {
	return func(i *Inspector) {
		i.version = version
	}
}

// Run executes the complete inspection workflow:
// 1. Collects host metadata and metrics
// 2. Evaluates thresholds to generate alerts
// 3. Aggregates results into InspectionResult
func (i *Inspector) Run(ctx context.Context) (*model.InspectionResult, error) {
	startTime := time.Now().In(i.timezone)
	i.logger.Info().
		Time("start_time", startTime).
		Str("timezone", i.timezone.String()).
		Msg("starting inspection")

	// Create result container
	result := model.NewInspectionResult(startTime)
	result.Version = i.version

	// Step 1: Collect data
	i.logger.Debug().Msg("step 1: collecting data")
	collectionResult, err := i.collector.CollectAll(ctx)
	if err != nil {
		i.logger.Error().Err(err).Msg("data collection failed")
		return nil, fmt.Errorf("data collection failed: %w", err)
	}

	if len(collectionResult.Hosts) == 0 {
		i.logger.Warn().Msg("no hosts found, completing inspection with empty result")
		result.Finalize(time.Now().In(i.timezone))
		return result, nil
	}

	// Step 2: Evaluate thresholds
	i.logger.Debug().
		Int("hosts_with_metrics", len(collectionResult.HostMetrics)).
		Msg("step 2: evaluating thresholds")
	evalResult := i.evaluator.EvaluateAll(collectionResult.HostMetrics)

	// Step 3: Build inspection result by merging collection and evaluation results
	i.logger.Debug().Msg("step 3: building inspection result")
	i.buildInspectionResult(result, collectionResult, evalResult)

	// Step 4: Finalize result (calculate summaries)
	endTime := time.Now().In(i.timezone)
	result.Finalize(endTime)

	i.logger.Info().
		Int("total_hosts", result.Summary.TotalHosts).
		Int("normal_hosts", result.Summary.NormalHosts).
		Int("warning_hosts", result.Summary.WarningHosts).
		Int("critical_hosts", result.Summary.CriticalHosts).
		Int("failed_hosts", result.Summary.FailedHosts).
		Int("total_alerts", result.AlertSummary.TotalAlerts).
		Dur("duration", result.Duration).
		Msg("inspection completed")

	return result, nil
}

// buildInspectionResult merges collection and evaluation results into InspectionResult.
func (i *Inspector) buildInspectionResult(
	result *model.InspectionResult,
	collectionResult *CollectionResult,
	evalResult *EvaluationResult,
) {
	// Build a map of evaluation results by hostname for quick lookup
	evalByHost := make(map[string]*HostEvaluationResult)
	for _, hostEval := range evalResult.HostResults {
		if hostEval != nil {
			evalByHost[hostEval.Hostname] = hostEval
		}
	}

	// Build a set of failed hosts
	failedHosts := make(map[string]string) // hostname -> error message
	for _, failed := range collectionResult.FailedHosts {
		failedHosts[failed.Hostname] = failed.Error
	}

	// Process each host from collection result
	for _, hostMeta := range collectionResult.Hosts {
		if hostMeta == nil {
			continue
		}

		// Create HostResult from HostMeta
		hostResult := model.NewHostResult(hostMeta)
		hostResult.CollectedAt = collectionResult.CollectedAt.In(i.timezone)

		// Check if this host failed collection
		if errMsg, failed := failedHosts[hostMeta.Hostname]; failed {
			hostResult.Status = model.HostStatusFailed
			hostResult.Error = errMsg
			result.AddHost(hostResult)
			continue
		}

		// Merge evaluation results
		if hostEval, exists := evalByHost[hostMeta.Hostname]; exists {
			// Copy metrics from evaluation result
			hostResult.Metrics = hostEval.Metrics
			// Copy alerts from evaluation result
			hostResult.Alerts = hostEval.Alerts
			// Set status from evaluation result
			hostResult.Status = hostEval.Status
		}

		result.AddHost(hostResult)
	}
}

// GetTimezone returns the configured timezone.
func (i *Inspector) GetTimezone() *time.Location {
	return i.timezone
}

// GetVersion returns the configured version.
func (i *Inspector) GetVersion() string {
	return i.version
}
