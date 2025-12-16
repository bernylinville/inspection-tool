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

// MySQLInspector orchestrates the complete MySQL inspection workflow, coordinating
// instance discovery, data collection, threshold evaluation, and result aggregation.
type MySQLInspector struct {
	collector *MySQLCollector
	evaluator *MySQLEvaluator
	config    *config.Config
	timezone  *time.Location
	version   string
	logger    zerolog.Logger
}

// MySQLInspectorOption is a functional option for configuring a MySQLInspector.
type MySQLInspectorOption func(*MySQLInspector)

// NewMySQLInspector creates a new MySQLInspector with the given dependencies.
//
// Parameters:
//   - cfg: Complete configuration including MySQL inspection config
//   - collector: MySQL data collector
//   - evaluator: Threshold evaluator
//   - logger: Structured logger
//   - opts: Optional configuration via functional options
//
// Returns:
//   - *MySQLInspector: Configured inspector instance
//   - error: Timezone loading error or validation failure
func NewMySQLInspector(
	cfg *config.Config,
	collector *MySQLCollector,
	evaluator *MySQLEvaluator,
	logger zerolog.Logger,
	opts ...MySQLInspectorOption,
) (*MySQLInspector, error) {
	// 验证必填参数
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if collector == nil {
		return nil, fmt.Errorf("collector cannot be nil")
	}
	if evaluator == nil {
		return nil, fmt.Errorf("evaluator cannot be nil")
	}

	// 确定时区（从配置或使用默认值）
	tzName := defaultTimezone
	if cfg.Report.Timezone != "" {
		tzName = cfg.Report.Timezone
	}

	// 加载时区
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return nil, fmt.Errorf("failed to load timezone %s: %w", tzName, err)
	}

	i := &MySQLInspector{
		collector: collector,
		evaluator: evaluator,
		config:    cfg,
		timezone:  loc,
		version:   "dev",
		logger:    logger.With().Str("component", "mysql_inspector").Logger(),
	}

	// 应用函数选项
	for _, opt := range opts {
		opt(i)
	}

	return i, nil
}

// WithMySQLVersion sets the tool version to include in the inspection result.
func WithMySQLVersion(version string) MySQLInspectorOption {
	return func(i *MySQLInspector) {
		i.version = version
	}
}

// GetTimezone returns the configured timezone.
func (i *MySQLInspector) GetTimezone() *time.Location {
	return i.timezone
}

// GetVersion returns the configured version.
func (i *MySQLInspector) GetVersion() string {
	return i.version
}

// Inspect executes the complete MySQL inspection workflow:
// 1. Discovers MySQL instances
// 2. Collects metrics for all instances
// 3. Evaluates thresholds and generates alerts
// 4. Aggregates results into MySQLInspectionResults
//
// Returns:
//   - *model.MySQLInspectionResults: Complete inspection result with summary
//   - error: Fatal errors that prevent inspection (discovery/config loading failures)
func (i *MySQLInspector) Inspect(ctx context.Context) (*model.MySQLInspectionResults, error) {
	// Step 1: 记录开始时间（Asia/Shanghai）
	startTime := time.Now().In(i.timezone)
	i.logger.Info().
		Time("start_time", startTime).
		Str("timezone", i.timezone.String()).
		Msg("starting MySQL inspection")

	// Step 2: 创建结果容器
	result := model.NewMySQLInspectionResults(startTime)
	result.Version = i.version

	// Step 3: 发现实例
	i.logger.Debug().Msg("step 1: discovering MySQL instances")
	instances, err := i.collector.DiscoverInstances(ctx)
	if err != nil {
		i.logger.Error().Err(err).Msg("instance discovery failed")
		return nil, fmt.Errorf("instance discovery failed: %w", err)
	}

	// Step 4: 空实例列表处理（优雅降级）
	if len(instances) == 0 {
		i.logger.Warn().Msg("no MySQL instances found, completing inspection with empty result")
		endTime := time.Now().In(i.timezone)
		result.Finalize(endTime)
		return result, nil
	}

	i.logger.Info().Int("instance_count", len(instances)).Msg("discovered MySQL instances")

	// Step 5: 采集指标（使用 collector 内部已加载的 metrics）
	i.logger.Debug().Msg("step 2: loading MySQL metric definitions")
	metrics := i.collector.GetMetrics()
	if len(metrics) == 0 {
		i.logger.Error().Msg("no MySQL metrics defined")
		return nil, fmt.Errorf("no MySQL metrics defined")
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

	// 从 Metrics map 填充字段（为评估器准备数据）
	for _, inspResult := range resultsMap {
		if inspResult == nil {
			continue
		}
		if maxConnMetric := inspResult.GetMetric("max_connections"); maxConnMetric != nil {
			inspResult.MaxConnections = int(maxConnMetric.RawValue)
		}
		if currConnMetric := inspResult.GetMetric("current_connections"); currConnMetric != nil {
			inspResult.CurrentConnections = int(currConnMetric.RawValue)
		}
		if mgrCountMetric := inspResult.GetMetric("mgr_member_count"); mgrCountMetric != nil {
			inspResult.MGRMemberCount = int(mgrCountMetric.RawValue)
		}
		if mgrStateMetric := inspResult.GetMetric("mgr_state_online"); mgrStateMetric != nil {
			inspResult.MGRStateOnline = mgrStateMetric.RawValue > 0
		}
	}

	// Step 6: 评估阈值
	i.logger.Debug().
		Int("results_count", len(resultsMap)).
		Msg("step 4: evaluating thresholds")

	_ = i.evaluator.EvaluateAll(resultsMap)

	// Step 7: 构建结果
	i.logger.Debug().Msg("step 5: building inspection results")
	i.buildInspectionResults(result, resultsMap)

	// Step 8: 最终化（计算 Duration、Summary、AlertSummary）
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
		Msg("MySQL inspection completed")

	return result, nil
}

// buildInspectionResults merges collection results into MySQLInspectionResults.
func (i *MySQLInspector) buildInspectionResults(
	result *model.MySQLInspectionResults,
	resultsMap map[string]*model.MySQLInspectionResult,
) {
	// 遍历所有实例结果
	for _, inspResult := range resultsMap {
		if inspResult == nil {
			continue
		}

		// 转换时间戳到配置的时区
		inspResult.CollectedAt = inspResult.CollectedAt.In(i.timezone)

		// 添加到结果容器（自动聚合告警）
		result.AddResult(inspResult)
	}

	i.logger.Debug().
		Int("total_results", len(result.Results)).
		Int("total_alerts", len(result.Alerts)).
		Msg("inspection results merged")
}
