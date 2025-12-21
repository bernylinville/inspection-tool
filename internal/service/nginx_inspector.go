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

// NginxInspector orchestrates the complete Nginx inspection workflow, coordinating
// instance discovery, data collection, threshold evaluation, and result aggregation.
type NginxInspector struct {
	collector *NginxCollector
	evaluator *NginxEvaluator
	config    *config.Config
	timezone  *time.Location
	version   string
	logger    zerolog.Logger
}

// NginxInspectorOption is a functional option for configuring a NginxInspector.
type NginxInspectorOption func(*NginxInspector)

// NewNginxInspector creates a new NginxInspector with the given dependencies.
//
// Parameters:
//   - cfg: Complete configuration including Nginx inspection config
//   - collector: Nginx data collector
//   - evaluator: Threshold evaluator
//   - logger: Structured logger
//   - opts: Optional configuration via functional options
//
// Returns:
//   - *NginxInspector: Configured inspector instance
//   - error: Timezone loading error or validation failure
func NewNginxInspector(
	cfg *config.Config,
	collector *NginxCollector,
	evaluator *NginxEvaluator,
	logger zerolog.Logger,
	opts ...NginxInspectorOption,
) (*NginxInspector, error) {
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

	i := &NginxInspector{
		collector: collector,
		evaluator: evaluator,
		config:    cfg,
		timezone:  loc,
		version:   "dev",
		logger:    logger.With().Str("component", "nginx_inspector").Logger(),
	}

	// 应用函数选项
	for _, opt := range opts {
		opt(i)
	}

	return i, nil
}

// WithNginxVersion sets the tool version to include in the inspection result.
func WithNginxVersion(version string) NginxInspectorOption {
	return func(i *NginxInspector) {
		i.version = version
	}
}

// GetTimezone returns the configured timezone.
func (i *NginxInspector) GetTimezone() *time.Location {
	return i.timezone
}

// GetVersion returns the configured version.
func (i *NginxInspector) GetVersion() string {
	return i.version
}

// Inspect executes the complete Nginx inspection workflow:
// 1. Discovers Nginx instances
// 2. Collects metrics for all instances
// 3. Collects upstream status if configured
// 4. Evaluates thresholds and generates alerts
// 5. Aggregates results into NginxInspectionResults
//
// Returns:
//   - *model.NginxInspectionResults: Complete inspection result with summary
//   - error: Fatal errors that prevent inspection (discovery/config loading failures)
func (i *NginxInspector) Inspect(ctx context.Context) (*model.NginxInspectionResults, error) {
	// Step 1: 记录开始时间（Asia/Shanghai）
	startTime := time.Now().In(i.timezone)
	i.logger.Info().
		Time("start_time", startTime).
		Str("timezone", i.timezone.String()).
		Msg("starting Nginx inspection")

	// Step 2: 创建结果容器
	result := model.NewNginxInspectionResults(startTime)
	result.Version = i.version

	// Step 3: 发现实例
	i.logger.Debug().Msg("step 1: discovering Nginx instances")
	instances, err := i.collector.DiscoverInstances(ctx)
	if err != nil {
		i.logger.Error().Err(err).Msg("instance discovery failed")
		return nil, fmt.Errorf("instance discovery failed: %w", err)
	}

	// Step 4: 空实例列表处理（优雅降级）
	if len(instances) == 0 {
		i.logger.Warn().Msg("no Nginx instances found, completing inspection with empty result")
		endTime := time.Now().In(i.timezone)
		result.Finalize(endTime)
		return result, nil
	}

	i.logger.Info().Int("instance_count", len(instances)).Msg("discovered Nginx instances")

	// Step 5: 采集指标（使用 collector 内部已加载的 metrics）
	i.logger.Debug().Msg("step 2: loading Nginx metric definitions")
	metrics := i.collector.GetMetrics()
	if len(metrics) == 0 {
		i.logger.Error().Msg("no Nginx metrics defined")
		return nil, fmt.Errorf("no Nginx metrics defined")
	}

	i.logger.Debug().
		Int("total_metrics", len(metrics)).
		Msg("Nginx metrics loaded")

	// Step 6: 采集指标数据
	i.logger.Debug().Msg("step 3: collecting Nginx metrics")
	metricsResults, err := i.collector.CollectMetrics(ctx, instances, metrics)
	if err != nil {
		i.logger.Error().Err(err).Msg("metrics collection failed")
		return nil, fmt.Errorf("metrics collection failed: %w", err)
	}

	// Step 7: 采集 Upstream 状态（可选）
	i.logger.Debug().Msg("step 4: collecting Nginx upstream status")
	if err := i.collector.CollectUpstreamStatus(ctx, metricsResults); err != nil {
		i.logger.Warn().Err(err).Msg("upstream status collection failed, continuing with other metrics")
		// Upstream 采集失败不是致命错误，继续处理
	}

	// Step 8: 评估阈值并生成告警
	i.logger.Debug().Msg("step 5: evaluating Nginx metrics against thresholds")
	evalResults := i.evaluator.EvaluateAll(metricsResults)

	// Step 9: 整理结果
	i.logger.Debug().Msg("step 6: organizing Nginx inspection results")
	for _, evalResult := range evalResults {
		if inspResult, exists := metricsResults[evalResult.Identifier]; exists {
			// 添加到结果列表
			result.AddResult(inspResult)
		}
	}

	// Step 10: 完成统计
	endTime := time.Now().In(i.timezone)
	result.Finalize(endTime)

	// Step 11: 记录完成信息
	i.logger.Info().
		Time("end_time", endTime).
		Dur("duration", result.Duration).
		Int("total_instances", result.Summary.TotalInstances).
		Int("normal_instances", result.Summary.NormalInstances).
		Int("warning_instances", result.Summary.WarningInstances).
		Int("critical_instances", result.Summary.CriticalInstances).
		Int("failed_instances", result.Summary.FailedInstances).
		Int("total_alerts", result.AlertSummary.TotalAlerts).
		Msg("Nginx inspection completed")

	// Step 12: 如果有严重告警，额外记录
	if result.HasCritical() {
		i.logger.Warn().
			Int("critical_count", result.Summary.CriticalInstances).
			Int("critical_alerts", result.AlertSummary.CriticalCount).
			Msg("Nginx inspection found critical issues")
	}

	return result, nil
}

// IsEnabled returns true if Nginx inspection is enabled in the configuration.
func (i *NginxInspector) IsEnabled() bool {
	return i.config != nil && i.config.Nginx.Enabled
}

// GetConfig returns the Nginx inspection configuration.
func (i *NginxInspector) GetConfig() *config.NginxInspectionConfig {
	if i.config == nil {
		return nil
	}
	return &i.config.Nginx
}