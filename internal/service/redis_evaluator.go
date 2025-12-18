// Package service provides business logic services for the inspection tool.
package service

import (
	"fmt"

	"github.com/rs/zerolog"

	"inspection-tool/internal/config"
	"inspection-tool/internal/model"
)

// RedisEvaluationResult represents the evaluation result for a single Redis instance.
type RedisEvaluationResult struct {
	Address string                    `json:"address"` // 实例地址
	Status  model.RedisInstanceStatus `json:"status"`  // 实例整体状态
	Alerts  []*model.RedisAlert       `json:"alerts"`  // 告警列表
}

// RedisEvaluator evaluates Redis instance metrics against thresholds.
type RedisEvaluator struct {
	thresholds *config.RedisThresholds                 // 阈值配置
	metricDefs map[string]*model.RedisMetricDefinition // 指标定义映射（用于获取显示名称）
	logger     zerolog.Logger                          // 日志器
}

// NewRedisEvaluator creates a new RedisEvaluator with the given threshold configuration.
func NewRedisEvaluator(
	thresholds *config.RedisThresholds,
	metrics []*model.RedisMetricDefinition,
	logger zerolog.Logger,
) *RedisEvaluator {
	metricDefs := make(map[string]*model.RedisMetricDefinition)
	for _, m := range metrics {
		metricDefs[m.Name] = m
	}

	return &RedisEvaluator{
		thresholds: thresholds,
		metricDefs: metricDefs,
		logger:     logger.With().Str("component", "redis_evaluator").Logger(),
	}
}

// EvaluateAll evaluates all Redis instances and returns the complete evaluation results.
func (e *RedisEvaluator) EvaluateAll(
	results map[string]*model.RedisInspectionResult,
) []*RedisEvaluationResult {
	evalResults := make([]*RedisEvaluationResult, 0, len(results))

	for _, result := range results {
		evalResult := e.Evaluate(result)
		evalResults = append(evalResults, evalResult)
	}

	e.logger.Info().
		Int("total_instances", len(evalResults)).
		Msg("Redis evaluation completed")

	return evalResults
}

// Evaluate evaluates a single Redis instance against configured thresholds.
func (e *RedisEvaluator) Evaluate(
	result *model.RedisInspectionResult,
) *RedisEvaluationResult {
	evalResult := &RedisEvaluationResult{
		Address: result.GetAddress(),
		Status:  model.RedisStatusNormal,
		Alerts:  make([]*model.RedisAlert, 0),
	}

	// 跳过采集失败的实例
	if result.Error != "" {
		evalResult.Status = model.RedisStatusFailed
		e.logger.Debug().
			Str("address", result.GetAddress()).
			Str("error", result.Error).
			Msg("skipping evaluation for failed instance")
		return evalResult
	}

	// 评估连接使用率（所有节点）
	if alert := e.evaluateConnectionUsage(result); alert != nil {
		evalResult.Alerts = append(evalResult.Alerts, alert)
	}

	// 仅 slave 节点的评估
	if result.Instance != nil && result.Instance.Role.IsSlave() {
		// 评估主从链接状态
		if alert := e.evaluateMasterLinkStatus(result); alert != nil {
			evalResult.Alerts = append(evalResult.Alerts, alert)
		}

		// 评估复制延迟
		if alert := e.evaluateReplicationLag(result); alert != nil {
			evalResult.Alerts = append(evalResult.Alerts, alert)
		}
	}

	// 聚合状态：取最严重级别
	evalResult.Status = e.determineInstanceStatus(evalResult.Alerts)

	// 更新原始结果对象
	result.Status = evalResult.Status
	result.Alerts = evalResult.Alerts

	e.logger.Debug().
		Str("address", result.GetAddress()).
		Str("status", string(evalResult.Status)).
		Int("alert_count", len(evalResult.Alerts)).
		Msg("instance evaluation completed")

	return evalResult
}

// evaluateConnectionUsage evaluates connection usage percentage.
func (e *RedisEvaluator) evaluateConnectionUsage(
	result *model.RedisInspectionResult,
) *model.RedisAlert {
	usage := result.GetConnectionUsagePercent()

	// 比较阈值
	if usage >= e.thresholds.ConnectionUsageCritical {
		return e.createAlert(
			result.GetAddress(),
			"connection_usage",
			usage,
			model.AlertLevelCritical,
		)
	}

	if usage >= e.thresholds.ConnectionUsageWarning {
		return e.createAlert(
			result.GetAddress(),
			"connection_usage",
			usage,
			model.AlertLevelWarning,
		)
	}

	return nil // 正常，无告警
}

// evaluateMasterLinkStatus evaluates master-slave link status (slave nodes only).
func (e *RedisEvaluator) evaluateMasterLinkStatus(
	result *model.RedisInspectionResult,
) *model.RedisAlert {
	// 仅对 slave 节点检查（调用者已确认角色）
	if !result.MasterLinkStatus {
		return e.createAlert(
			result.GetAddress(),
			"master_link_status",
			0, // CurrentValue = 0 表示断开
			model.AlertLevelCritical,
		)
	}

	return nil
}

// evaluateReplicationLag evaluates replication lag (slave nodes only).
func (e *RedisEvaluator) evaluateReplicationLag(
	result *model.RedisInspectionResult,
) *model.RedisAlert {
	// 仅对 slave 节点检查（调用者已确认角色）
	lag := result.ReplicationLag

	// 比较阈值
	if lag >= e.thresholds.ReplicationLagCritical {
		return e.createAlert(
			result.GetAddress(),
			"replication_lag",
			float64(lag),
			model.AlertLevelCritical,
		)
	}

	if lag >= e.thresholds.ReplicationLagWarning {
		return e.createAlert(
			result.GetAddress(),
			"replication_lag",
			float64(lag),
			model.AlertLevelWarning,
		)
	}

	return nil
}

// determineInstanceStatus determines the overall instance status based on alerts.
// 状态聚合优先级：Critical > Warning > Normal
func (e *RedisEvaluator) determineInstanceStatus(
	alerts []*model.RedisAlert,
) model.RedisInstanceStatus {
	hasCritical := false
	hasWarning := false

	for _, alert := range alerts {
		if alert.Level == model.AlertLevelCritical {
			hasCritical = true
		} else if alert.Level == model.AlertLevelWarning {
			hasWarning = true
		}
	}

	if hasCritical {
		return model.RedisStatusCritical
	}
	if hasWarning {
		return model.RedisStatusWarning
	}

	return model.RedisStatusNormal
}

// createAlert creates a RedisAlert with formatted message.
func (e *RedisEvaluator) createAlert(
	address string,
	metricName string,
	currentValue float64,
	level model.AlertLevel,
) *model.RedisAlert {
	// 获取指标显示名称
	displayName := metricName
	if def, exists := e.metricDefs[metricName]; exists {
		displayName = def.GetDisplayName()
	}

	// 格式化当前值
	formattedValue := e.formatValue(currentValue, metricName)

	// 生成告警消息
	message := e.generateAlertMessage(metricName, currentValue, level)

	// 获取阈值
	warningThreshold, criticalThreshold := e.getThresholds(metricName)

	return &model.RedisAlert{
		Address:           address,
		MetricName:        metricName,
		MetricDisplayName: displayName,
		CurrentValue:      currentValue,
		FormattedValue:    formattedValue,
		WarningThreshold:  warningThreshold,
		CriticalThreshold: criticalThreshold,
		Level:             level,
		Message:           message,
	}
}

// formatValue formats the metric value based on metric type.
func (e *RedisEvaluator) formatValue(value float64, metricName string) string {
	switch metricName {
	case "connection_usage":
		return fmt.Sprintf("%.1f%%", value)
	case "replication_lag":
		// Convert bytes to human-readable format
		if value >= 1073741824 { // 1GB
			return fmt.Sprintf("%.2f GB", value/1073741824)
		}
		if value >= 1048576 { // 1MB
			return fmt.Sprintf("%.2f MB", value/1048576)
		}
		if value >= 1024 { // 1KB
			return fmt.Sprintf("%.2f KB", value/1024)
		}
		return fmt.Sprintf("%.0f B", value)
	case "master_link_status":
		if value == 0 {
			return "断开"
		}
		return "正常"
	default:
		return fmt.Sprintf("%.2f", value)
	}
}

// generateAlertMessage generates a human-readable alert message.
func (e *RedisEvaluator) generateAlertMessage(
	metricName string,
	currentValue float64,
	level model.AlertLevel,
) string {
	warningThreshold, criticalThreshold := e.getThresholds(metricName)

	switch metricName {
	case "connection_usage":
		if level == model.AlertLevelCritical {
			return fmt.Sprintf("连接使用率为 %.1f%%，已超过严重阈值 %.1f%%",
				currentValue, criticalThreshold)
		}
		return fmt.Sprintf("连接使用率为 %.1f%%，已超过警告阈值 %.1f%%",
			currentValue, warningThreshold)

	case "replication_lag":
		formattedLag := e.formatValue(currentValue, metricName)
		if level == model.AlertLevelCritical {
			return fmt.Sprintf("复制延迟为 %s，已超过严重阈值 %.0f MB",
				formattedLag, float64(e.thresholds.ReplicationLagCritical)/1048576)
		}
		return fmt.Sprintf("复制延迟为 %s，已超过警告阈值 %.0f MB",
			formattedLag, float64(e.thresholds.ReplicationLagWarning)/1048576)

	case "master_link_status":
		return "主从链接断开（master_link_status = 0）"

	default:
		return fmt.Sprintf("%s 指标异常，当前值: %.2f", metricName, currentValue)
	}
}

// getThresholds returns the warning and critical thresholds for a metric.
func (e *RedisEvaluator) getThresholds(metricName string) (warning float64, critical float64) {
	switch metricName {
	case "connection_usage":
		return e.thresholds.ConnectionUsageWarning, e.thresholds.ConnectionUsageCritical
	case "replication_lag":
		return float64(e.thresholds.ReplicationLagWarning), float64(e.thresholds.ReplicationLagCritical)
	case "master_link_status":
		return 1, 1 // 断开即严重，无警告阈值
	default:
		return 0, 0
	}
}
