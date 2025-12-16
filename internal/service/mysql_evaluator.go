// Package service provides business logic services for the inspection tool.
package service

import (
	"fmt"

	"github.com/rs/zerolog"

	"inspection-tool/internal/config"
	"inspection-tool/internal/model"
)

// MySQLEvaluationResult represents the evaluation result for a single MySQL instance.
type MySQLEvaluationResult struct {
	Address string                      `json:"address"` // 实例地址
	Status  model.MySQLInstanceStatus   `json:"status"`  // 实例整体状态
	Alerts  []*model.MySQLAlert         `json:"alerts"`  // 告警列表
}

// MySQLEvaluator evaluates MySQL instance metrics against thresholds.
type MySQLEvaluator struct {
	thresholds *config.MySQLThresholds                    // 阈值配置
	metricDefs map[string]*model.MySQLMetricDefinition    // 指标定义映射（用于获取显示名称）
	logger     zerolog.Logger                             // 日志器
}

// NewMySQLEvaluator creates a new MySQLEvaluator with the given threshold configuration.
func NewMySQLEvaluator(
	thresholds *config.MySQLThresholds,
	metrics []*model.MySQLMetricDefinition,
	logger zerolog.Logger,
) *MySQLEvaluator {
	metricDefs := make(map[string]*model.MySQLMetricDefinition)
	for _, m := range metrics {
		metricDefs[m.Name] = m
	}

	return &MySQLEvaluator{
		thresholds: thresholds,
		metricDefs: metricDefs,
		logger:     logger.With().Str("component", "mysql_evaluator").Logger(),
	}
}

// EvaluateAll evaluates all MySQL instances and returns the complete evaluation results.
func (e *MySQLEvaluator) EvaluateAll(
	results map[string]*model.MySQLInspectionResult,
) []*MySQLEvaluationResult {
	evalResults := make([]*MySQLEvaluationResult, 0, len(results))

	for _, result := range results {
		evalResult := e.Evaluate(result)
		evalResults = append(evalResults, evalResult)
	}

	e.logger.Info().
		Int("total_instances", len(evalResults)).
		Msg("MySQL evaluation completed")

	return evalResults
}

// Evaluate evaluates a single MySQL instance against configured thresholds.
func (e *MySQLEvaluator) Evaluate(
	result *model.MySQLInspectionResult,
) *MySQLEvaluationResult {
	evalResult := &MySQLEvaluationResult{
		Address: result.GetAddress(),
		Status:  model.MySQLStatusNormal,
		Alerts:  make([]*model.MySQLAlert, 0),
	}

	// 跳过采集失败的实例
	if result.Error != "" {
		evalResult.Status = model.MySQLStatusFailed
		e.logger.Debug().
			Str("address", result.GetAddress()).
			Str("error", result.Error).
			Msg("skipping evaluation for failed instance")
		return evalResult
	}

	// 评估连接使用率（必评）
	if alert := e.evaluateConnectionUsage(result); alert != nil {
		evalResult.Alerts = append(evalResult.Alerts, alert)
	}

	// 如果是 MGR 模式，评估 MGR 指标
	if result.Instance.ClusterMode.IsMGR() {
		// 评估 MGR 成员数
		if alert := e.evaluateMGRMemberCount(result); alert != nil {
			evalResult.Alerts = append(evalResult.Alerts, alert)
		}

		// 评估 MGR 在线状态
		if alert := e.evaluateMGRStateOnline(result); alert != nil {
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
func (e *MySQLEvaluator) evaluateConnectionUsage(
	result *model.MySQLInspectionResult,
) *model.MySQLAlert {
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

// evaluateMGRMemberCount evaluates MGR cluster member count.
func (e *MySQLEvaluator) evaluateMGRMemberCount(
	result *model.MySQLInspectionResult,
) *model.MySQLAlert {
	count := result.MGRMemberCount
	expected := e.thresholds.MGRMemberCountExpected

	// 严重：掉 2 个及以上节点
	if count < expected-1 {
		return e.createAlert(
			result.GetAddress(),
			"mgr_member_count",
			float64(count),
			model.AlertLevelCritical,
		)
	}

	// 警告：掉 1 个节点
	if count < expected {
		return e.createAlert(
			result.GetAddress(),
			"mgr_member_count",
			float64(count),
			model.AlertLevelWarning,
		)
	}

	// 正常：成员数达到或超过期望值
	return nil
}

// evaluateMGRStateOnline evaluates MGR node online status.
func (e *MySQLEvaluator) evaluateMGRStateOnline(
	result *model.MySQLInspectionResult,
) *model.MySQLAlert {
	if !result.MGRStateOnline {
		return e.createAlert(
			result.GetAddress(),
			"mgr_state_online",
			0, // CurrentValue = 0 表示离线
			model.AlertLevelCritical,
		)
	}

	return nil
}

// determineInstanceStatus determines the overall instance status based on alerts.
// 状态聚合优先级：Critical > Warning > Normal
func (e *MySQLEvaluator) determineInstanceStatus(
	alerts []*model.MySQLAlert,
) model.MySQLInstanceStatus {
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
		return model.MySQLStatusCritical
	}
	if hasWarning {
		return model.MySQLStatusWarning
	}

	return model.MySQLStatusNormal
}

// createAlert creates a MySQLAlert with formatted message.
func (e *MySQLEvaluator) createAlert(
	address string,
	metricName string,
	currentValue float64,
	level model.AlertLevel,
) *model.MySQLAlert {
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

	return &model.MySQLAlert{
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
func (e *MySQLEvaluator) formatValue(value float64, metricName string) string {
	switch metricName {
	case "connection_usage":
		return fmt.Sprintf("%.1f%%", value)
	case "mgr_member_count":
		return fmt.Sprintf("%d", int(value))
	case "mgr_state_online":
		if value == 0 {
			return "离线"
		}
		return "在线"
	default:
		return fmt.Sprintf("%.2f", value)
	}
}

// generateAlertMessage generates a human-readable alert message.
func (e *MySQLEvaluator) generateAlertMessage(
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

	case "mgr_member_count":
		expected := e.thresholds.MGRMemberCountExpected
		count := int(currentValue)
		if level == model.AlertLevelCritical {
			dropped := expected - count
			return fmt.Sprintf("MGR 成员数为 %d，低于期望值 %d（掉 %d 个节点）",
				count, expected, dropped)
		}
		return fmt.Sprintf("MGR 成员数为 %d，低于期望值 %d（掉 1 个节点）",
			count, expected)

	case "mgr_state_online":
		return "MGR 节点离线（mgr_state_online = 0）"

	default:
		return fmt.Sprintf("%s 指标异常，当前值: %.2f", metricName, currentValue)
	}
}

// getThresholds returns the warning and critical thresholds for a metric.
func (e *MySQLEvaluator) getThresholds(metricName string) (warning float64, critical float64) {
	switch metricName {
	case "connection_usage":
		return e.thresholds.ConnectionUsageWarning, e.thresholds.ConnectionUsageCritical
	case "mgr_member_count":
		expected := float64(e.thresholds.MGRMemberCountExpected)
		return expected - 1, expected - 2 // 警告: expected-1, 严重: expected-2
	case "mgr_state_online":
		return 1, 1 // 离线即严重，无警告阈值
	default:
		return 0, 0
	}
}
