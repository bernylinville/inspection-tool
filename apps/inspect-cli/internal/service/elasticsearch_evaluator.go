package service

import (
	"fmt"

	"github.com/rs/zerolog"

	"inspection-tool/apps/inspect-cli/internal/config"
	"inspection-tool/apps/inspect-cli/internal/model"
)

type ElasticsearchEvaluationResult struct {
	Address string                            `json:"address"`
	Status  model.ElasticsearchInstanceStatus `json:"status"`
	Alerts  []*model.ElasticsearchAlert       `json:"alerts"`
}

type ElasticsearchEvaluator struct {
	thresholds *config.ElasticsearchThresholds
	metricDefs map[string]*model.ElasticsearchMetricDefinition
	logger     zerolog.Logger
}

func NewElasticsearchEvaluator(
	thresholds *config.ElasticsearchThresholds,
	metrics []*model.ElasticsearchMetricDefinition,
	logger zerolog.Logger,
) *ElasticsearchEvaluator {
	metricDefs := make(map[string]*model.ElasticsearchMetricDefinition)
	for _, m := range metrics {
		metricDefs[m.Name] = m
	}

	return &ElasticsearchEvaluator{
		thresholds: thresholds,
		metricDefs: metricDefs,
		logger:     logger.With().Str("component", "elasticsearch_evaluator").Logger(),
	}
}

func (e *ElasticsearchEvaluator) EvaluateAll(
	results map[string]*model.ElasticsearchInspectionResult,
) []*ElasticsearchEvaluationResult {
	evalResults := make([]*ElasticsearchEvaluationResult, 0, len(results))

	for _, result := range results {
		evalResult := e.Evaluate(result)
		evalResults = append(evalResults, evalResult)
	}

	e.logger.Info().
		Int("total_instances", len(evalResults)).
		Msg("Elasticsearch evaluation completed")

	return evalResults
}

func (e *ElasticsearchEvaluator) Evaluate(
	result *model.ElasticsearchInspectionResult,
) *ElasticsearchEvaluationResult {
	evalResult := &ElasticsearchEvaluationResult{
		Address: result.GetAddress(),
		Status:  model.ElasticsearchStatusNormal,
		Alerts:  make([]*model.ElasticsearchAlert, 0),
	}

	if result.Error != "" {
		evalResult.Status = model.ElasticsearchStatusFailed
		e.logger.Debug().
			Str("address", result.GetAddress()).
			Str("error", result.Error).
			Msg("skipping evaluation for failed instance")
		return evalResult
	}

	if alert := e.evaluateHeapMemoryUsage(result); alert != nil {
		evalResult.Alerts = append(evalResult.Alerts, alert)
	}

	if alert := e.evaluateCPUUsage(result); alert != nil {
		evalResult.Alerts = append(evalResult.Alerts, alert)
	}

	if alert := e.evaluateDiskUsage(result); alert != nil {
		evalResult.Alerts = append(evalResult.Alerts, alert)
	}

	if alert := e.evaluateFileHandleUsage(result); alert != nil {
		evalResult.Alerts = append(evalResult.Alerts, alert)
	}

	if alert := e.evaluateCircuitBreaker(result); alert != nil {
		evalResult.Alerts = append(evalResult.Alerts, alert)
	}

	if alert := e.evaluateThreadPoolRejected(result); alert != nil {
		evalResult.Alerts = append(evalResult.Alerts, alert)
	}

	evalResult.Status = e.determineInstanceStatus(evalResult.Alerts)

	result.Status = evalResult.Status
	result.Alerts = evalResult.Alerts

	e.logger.Debug().
		Str("address", result.GetAddress()).
		Str("status", string(evalResult.Status)).
		Int("alert_count", len(evalResult.Alerts)).
		Msg("instance evaluation completed")

	return evalResult
}

func (e *ElasticsearchEvaluator) evaluateHeapMemoryUsage(
	result *model.ElasticsearchInspectionResult,
) *model.ElasticsearchAlert {
	usage := result.HeapMemoryPercent

	if usage >= e.thresholds.HeapMemoryUsageCritical {
		return e.createAlert(
			result.GetAddress(),
			"heap_memory_percent",
			usage,
			model.AlertLevelCritical,
		)
	}

	if usage >= e.thresholds.HeapMemoryUsageWarning {
		return e.createAlert(
			result.GetAddress(),
			"heap_memory_percent",
			usage,
			model.AlertLevelWarning,
		)
	}

	return nil
}

func (e *ElasticsearchEvaluator) evaluateCPUUsage(
	result *model.ElasticsearchInspectionResult,
) *model.ElasticsearchAlert {
	usage := result.CPUPercent

	if usage >= e.thresholds.CPUUsageCritical {
		return e.createAlert(
			result.GetAddress(),
			"cpu_percent",
			usage,
			model.AlertLevelCritical,
		)
	}

	if usage >= e.thresholds.CPUUsageWarning {
		return e.createAlert(
			result.GetAddress(),
			"cpu_percent",
			usage,
			model.AlertLevelWarning,
		)
	}

	return nil
}

func (e *ElasticsearchEvaluator) evaluateDiskUsage(
	result *model.ElasticsearchInspectionResult,
) *model.ElasticsearchAlert {
	usage := result.DiskUsagePercent

	if usage >= e.thresholds.DiskUsageCritical {
		return e.createAlert(
			result.GetAddress(),
			"disk_usage_percent",
			usage,
			model.AlertLevelCritical,
		)
	}

	if usage >= e.thresholds.DiskUsageWarning {
		return e.createAlert(
			result.GetAddress(),
			"disk_usage_percent",
			usage,
			model.AlertLevelWarning,
		)
	}

	return nil
}

func (e *ElasticsearchEvaluator) evaluateFileHandleUsage(
	result *model.ElasticsearchInspectionResult,
) *model.ElasticsearchAlert {
	usage := result.FileHandleUsagePercent

	if usage >= e.thresholds.FileHandleUsageCritical {
		return e.createAlert(
			result.GetAddress(),
			"file_handle_usage_percent",
			usage,
			model.AlertLevelCritical,
		)
	}

	if usage >= e.thresholds.FileHandleUsageWarning {
		return e.createAlert(
			result.GetAddress(),
			"file_handle_usage_percent",
			usage,
			model.AlertLevelWarning,
		)
	}

	return nil
}

func (e *ElasticsearchEvaluator) evaluateCircuitBreaker(
	result *model.ElasticsearchInspectionResult,
) *model.ElasticsearchAlert {
	if result.CircuitBreakerTripped {
		return e.createAlert(
			result.GetAddress(),
			"circuit_breaker_tripped",
			1,
			model.AlertLevelWarning,
		)
	}
	return nil
}

func (e *ElasticsearchEvaluator) evaluateThreadPoolRejected(
	result *model.ElasticsearchInspectionResult,
) *model.ElasticsearchAlert {
	if result.ThreadPoolRejected {
		return e.createAlert(
			result.GetAddress(),
			"thread_pool_rejected",
			1,
			model.AlertLevelWarning,
		)
	}
	return nil
}

func (e *ElasticsearchEvaluator) determineInstanceStatus(
	alerts []*model.ElasticsearchAlert,
) model.ElasticsearchInstanceStatus {
	if len(alerts) == 0 {
		return model.ElasticsearchStatusNormal
	}

	hasCritical := false
	for _, alert := range alerts {
		if alert.Level == model.AlertLevelCritical {
			hasCritical = true
			break
		}
	}

	if hasCritical {
		return model.ElasticsearchStatusCritical
	}
	return model.ElasticsearchStatusWarning
}

func (e *ElasticsearchEvaluator) createAlert(
	address string,
	metricName string,
	currentValue float64,
	level model.AlertLevel,
) *model.ElasticsearchAlert {
	displayName := metricName
	if def, ok := e.metricDefs[metricName]; ok {
		displayName = def.DisplayName
	}

	var warningThreshold, criticalThreshold float64
	var message string

	switch metricName {
	case "heap_memory_percent":
		warningThreshold = e.thresholds.HeapMemoryUsageWarning
		criticalThreshold = e.thresholds.HeapMemoryUsageCritical
		message = fmt.Sprintf("JVM堆内存使用率 %.1f%% 超过阈值", currentValue)
	case "cpu_percent":
		warningThreshold = e.thresholds.CPUUsageWarning
		criticalThreshold = e.thresholds.CPUUsageCritical
		message = fmt.Sprintf("CPU使用率 %.1f%% 超过阈值", currentValue)
	case "disk_usage_percent":
		warningThreshold = e.thresholds.DiskUsageWarning
		criticalThreshold = e.thresholds.DiskUsageCritical
		message = fmt.Sprintf("磁盘使用率 %.1f%% 超过阈值", currentValue)
	case "file_handle_usage_percent":
		warningThreshold = e.thresholds.FileHandleUsageWarning
		criticalThreshold = e.thresholds.FileHandleUsageCritical
		message = fmt.Sprintf("文件句柄使用率 %.1f%% 超过阈值", currentValue)
	case "circuit_breaker_tripped":
		message = "熔断器已触发"
	case "thread_pool_rejected":
		message = "线程池存在拒绝请求"
	default:
		message = fmt.Sprintf("%s 异常: %.2f", displayName, currentValue)
	}

	return &model.ElasticsearchAlert{
		Address:           address,
		MetricName:        metricName,
		MetricDisplayName: displayName,
		CurrentValue:      currentValue,
		FormattedValue:    fmt.Sprintf("%.2f", currentValue),
		WarningThreshold:  warningThreshold,
		CriticalThreshold: criticalThreshold,
		Level:             level,
		Message:           message,
	}
}
