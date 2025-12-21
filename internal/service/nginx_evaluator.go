// Package service provides business logic services for the inspection tool.
package service

import (
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"inspection-tool/internal/config"
	"inspection-tool/internal/model"
)

// NginxEvaluationResult represents the evaluation result for a single Nginx instance.
type NginxEvaluationResult struct {
	Identifier string                     `json:"identifier"` // 实例标识符
	Status     model.NginxInstanceStatus  `json:"status"`     // 实例整体状态
	Alerts     []*model.NginxAlert        `json:"alerts"`     // 告警列表
}

// NginxEvaluator evaluates Nginx instance metrics against thresholds.
type NginxEvaluator struct {
	thresholds *config.NginxThresholds                    // 阈值配置
	metricDefs map[string]*model.NginxMetricDefinition    // 指标定义映射（用于获取显示名称）
	timezone   *time.Location                             // 时区（用于格式化错误时间）
	logger     zerolog.Logger                             // 日志器
}

// NewNginxEvaluator creates a new NginxEvaluator with the given threshold configuration.
func NewNginxEvaluator(
	thresholds *config.NginxThresholds,
	metrics []*model.NginxMetricDefinition,
	timezone *time.Location,
	logger zerolog.Logger,
) *NginxEvaluator {
	metricDefs := make(map[string]*model.NginxMetricDefinition)
	for _, m := range metrics {
		metricDefs[m.Name] = m
	}

	if timezone == nil {
		timezone, _ = time.LoadLocation("Asia/Shanghai")
	}

	return &NginxEvaluator{
		thresholds: thresholds,
		metricDefs: metricDefs,
		timezone:   timezone,
		logger:     logger.With().Str("component", "nginx_evaluator").Logger(),
	}
}

// EvaluateAll evaluates all Nginx instances and returns the complete evaluation results.
func (e *NginxEvaluator) EvaluateAll(
	results map[string]*model.NginxInspectionResult,
) []*NginxEvaluationResult {
	evalResults := make([]*NginxEvaluationResult, 0, len(results))

	for _, result := range results {
		evalResult := e.Evaluate(result)
		evalResults = append(evalResults, evalResult)
	}

	e.logger.Info().
		Int("total_instances", len(evalResults)).
		Msg("Nginx evaluation completed")

	return evalResults
}

// Evaluate evaluates a single Nginx instance against configured thresholds.
//
// Evaluation rules:
//  1. Connection status (nginx_up): nginx_up=0 → Critical
//  2. Connection usage: >90% Critical, >70% Warning
//  3. Last error time: <10min Critical, <60min Warning (inverted logic)
//  4. Error page config (4xx/5xx): =0 → Critical
//  5. Non-root user: =0 → Critical
//  6. Upstream status: status_code=0 → Critical
func (e *NginxEvaluator) Evaluate(
	result *model.NginxInspectionResult,
) *NginxEvaluationResult {
	evalResult := &NginxEvaluationResult{
		Identifier: result.GetIdentifier(),
		Status:     model.NginxStatusNormal,
		Alerts:     make([]*model.NginxAlert, 0),
	}

	// 跳过采集失败的实例
	if result.Error != "" {
		evalResult.Status = model.NginxStatusFailed
		e.logger.Debug().
			Str("identifier", result.GetIdentifier()).
			Str("error", result.Error).
			Msg("skipping evaluation for failed instance")
		return evalResult
	}

	// 1. 评估连接状态（nginx_up）
	if alert := e.evaluateConnectionStatus(result); alert != nil {
		evalResult.Alerts = append(evalResult.Alerts, alert)
	}

	// 2. 评估连接使用率
	if alert := e.evaluateConnectionUsage(result); alert != nil {
		evalResult.Alerts = append(evalResult.Alerts, alert)
	}

	// 3. 评估最近错误日志时间
	if alert := e.evaluateLastErrorTime(result); alert != nil {
		evalResult.Alerts = append(evalResult.Alerts, alert)
	}

	// 4. 评估错误页配置（4xx）
	if alert := e.evaluateErrorPage4xx(result); alert != nil {
		evalResult.Alerts = append(evalResult.Alerts, alert)
	}

	// 5. 评估错误页配置（5xx）
	if alert := e.evaluateErrorPage5xx(result); alert != nil {
		evalResult.Alerts = append(evalResult.Alerts, alert)
	}

	// 6. 评估非 root 用户启动
	if alert := e.evaluateNonRootUser(result); alert != nil {
		evalResult.Alerts = append(evalResult.Alerts, alert)
	}

	// 7. 评估 Upstream 后端状态
	upstreamAlerts := e.evaluateUpstreamStatus(result)
	evalResult.Alerts = append(evalResult.Alerts, upstreamAlerts...)

	// 聚合状态：取最严重级别
	evalResult.Status = e.determineInstanceStatus(evalResult.Alerts)

	// 更新原始结果对象
	result.Status = evalResult.Status
	result.Alerts = evalResult.Alerts

	// 格式化最近错误时间
	result.LastErrorTimeFormatted = result.FormatLastErrorTime(e.timezone)

	e.logger.Debug().
		Str("identifier", result.GetIdentifier()).
		Str("status", string(evalResult.Status)).
		Int("alert_count", len(evalResult.Alerts)).
		Msg("instance evaluation completed")

	return evalResult
}

// evaluateConnectionStatus evaluates nginx_up status.
// nginx_up=0 → Critical
func (e *NginxEvaluator) evaluateConnectionStatus(
	result *model.NginxInspectionResult,
) *model.NginxAlert {
	if !result.Up {
		return e.createAlert(
			result.GetIdentifier(),
			"nginx_up",
			0,
			model.AlertLevelCritical,
		)
	}
	return nil
}

// evaluateConnectionUsage evaluates connection usage percentage.
// >90% → Critical, >70% → Warning
func (e *NginxEvaluator) evaluateConnectionUsage(
	result *model.NginxInspectionResult,
) *model.NginxAlert {
	usage := result.ConnectionUsagePercent

	// 无法计算时不告警（usage < 0）
	if usage < 0 {
		return nil
	}

	if usage >= e.thresholds.ConnectionUsageCritical {
		return e.createAlert(
			result.GetIdentifier(),
			"connection_usage",
			usage,
			model.AlertLevelCritical,
		)
	}

	if usage >= e.thresholds.ConnectionUsageWarning {
		return e.createAlert(
			result.GetIdentifier(),
			"connection_usage",
			usage,
			model.AlertLevelWarning,
		)
	}

	return nil
}

// evaluateLastErrorTime evaluates the last error log timestamp.
// Logic is INVERTED: more recent error = more severe
//   - <10 minutes → Critical
//   - <60 minutes → Warning
//   - timestamp=0 (no error) → Normal
func (e *NginxEvaluator) evaluateLastErrorTime(
	result *model.NginxInspectionResult,
) *model.NginxAlert {
	timestamp := result.LastErrorTimestamp

	// 0 表示从未有错误日志，正常
	if timestamp == 0 {
		return nil
	}

	now := time.Now().Unix()
	minutesSinceError := (now - timestamp) / 60

	// 严重：10 分钟内有错误
	criticalMinutes := int64(e.thresholds.LastErrorCriticalMinutes)
	if minutesSinceError <= criticalMinutes {
		return e.createAlert(
			result.GetIdentifier(),
			"last_error_time",
			float64(minutesSinceError),
			model.AlertLevelCritical,
		)
	}

	// 警告：60 分钟内有错误
	warningMinutes := int64(e.thresholds.LastErrorWarningMinutes)
	if minutesSinceError <= warningMinutes {
		return e.createAlert(
			result.GetIdentifier(),
			"last_error_time",
			float64(minutesSinceError),
			model.AlertLevelWarning,
		)
	}

	return nil
}

// evaluateErrorPage4xx evaluates 4xx error page configuration.
// =0 (not configured) → Critical
func (e *NginxEvaluator) evaluateErrorPage4xx(
	result *model.NginxInspectionResult,
) *model.NginxAlert {
	// 只有在指标被采集且值为 0 时才告警
	mv := result.GetMetric("nginx_error_page_4xx")
	if mv == nil || mv.IsNA {
		return nil // 指标未采集，不告警
	}

	if !result.ErrorPage4xxConfigured {
		return e.createAlert(
			result.GetIdentifier(),
			"error_page_4xx",
			0,
			model.AlertLevelCritical,
		)
	}

	return nil
}

// evaluateErrorPage5xx evaluates 5xx error page configuration.
// =0 (not configured) → Critical
func (e *NginxEvaluator) evaluateErrorPage5xx(
	result *model.NginxInspectionResult,
) *model.NginxAlert {
	// 只有在指标被采集且值为 0 时才告警
	mv := result.GetMetric("nginx_error_page_5xx")
	if mv == nil || mv.IsNA {
		return nil // 指标未采集，不告警
	}

	if !result.ErrorPage5xxConfigured {
		return e.createAlert(
			result.GetIdentifier(),
			"error_page_5xx",
			0,
			model.AlertLevelCritical,
		)
	}

	return nil
}

// evaluateNonRootUser evaluates non-root user startup.
// =0 (root user) → Critical (security risk)
func (e *NginxEvaluator) evaluateNonRootUser(
	result *model.NginxInspectionResult,
) *model.NginxAlert {
	// 只有在指标被采集且值为 0 时才告警
	mv := result.GetMetric("nginx_non_root_user")
	if mv == nil || mv.IsNA {
		return nil // 指标未采集，不告警
	}

	if !result.NonRootUser {
		return e.createAlert(
			result.GetIdentifier(),
			"non_root_user",
			0,
			model.AlertLevelCritical,
		)
	}

	return nil
}

// evaluateUpstreamStatus evaluates all upstream backend statuses.
// status_code=0 → Critical for each unhealthy backend
func (e *NginxEvaluator) evaluateUpstreamStatus(
	result *model.NginxInspectionResult,
) []*model.NginxAlert {
	var alerts []*model.NginxAlert

	for _, upstream := range result.UpstreamStatus {
		if !upstream.Status {
			// Upstream 后端异常
			alert := e.createUpstreamAlert(
				result.GetIdentifier(),
				upstream,
			)
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

// createUpstreamAlert creates an alert for an unhealthy upstream backend.
func (e *NginxEvaluator) createUpstreamAlert(
	identifier string,
	upstream model.NginxUpstreamStatus,
) *model.NginxAlert {
	metricName := "upstream_status"
	displayName := "Upstream 后端状态"
	if def, exists := e.metricDefs["nginx_upstream_check_status_code"]; exists {
		displayName = def.GetDisplayName()
	}

	message := fmt.Sprintf("Upstream 后端异常: %s 在 %s 组，连续失败 %d 次",
		upstream.BackendAddress, upstream.UpstreamName, upstream.FallCount)

	return &model.NginxAlert{
		Identifier:        identifier,
		MetricName:        metricName,
		MetricDisplayName: displayName,
		CurrentValue:      0,
		FormattedValue:    "异常",
		WarningThreshold:  1,
		CriticalThreshold: 1,
		Level:             model.AlertLevelCritical,
		Message:           message,
	}
}

// determineInstanceStatus determines the overall instance status based on alerts.
// Status priority: Critical > Warning > Normal
func (e *NginxEvaluator) determineInstanceStatus(
	alerts []*model.NginxAlert,
) model.NginxInstanceStatus {
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
		return model.NginxStatusCritical
	}
	if hasWarning {
		return model.NginxStatusWarning
	}

	return model.NginxStatusNormal
}

// createAlert creates a NginxAlert with formatted message.
func (e *NginxEvaluator) createAlert(
	identifier string,
	metricName string,
	currentValue float64,
	level model.AlertLevel,
) *model.NginxAlert {
	// 获取指标显示名称
	displayName := e.getDisplayName(metricName)

	// 格式化当前值
	formattedValue := e.formatValue(currentValue, metricName)

	// 生成告警消息
	message := e.generateAlertMessage(metricName, currentValue, level)

	// 获取阈值
	warningThreshold, criticalThreshold := e.getThresholds(metricName)

	return &model.NginxAlert{
		Identifier:        identifier,
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

// getDisplayName returns the display name for a metric.
func (e *NginxEvaluator) getDisplayName(metricName string) string {
	// 映射内部名称到指标定义名称
	metricDefName := metricName
	switch metricName {
	case "connection_usage":
		// 连接使用率是计算值，无对应指标定义
		return "连接使用率"
	case "last_error_time":
		metricDefName = "nginx_last_error_timestamp"
	case "error_page_4xx":
		metricDefName = "nginx_error_page_4xx"
	case "error_page_5xx":
		metricDefName = "nginx_error_page_5xx"
	case "non_root_user":
		metricDefName = "nginx_non_root_user"
	}

	if def, exists := e.metricDefs[metricDefName]; exists {
		return def.GetDisplayName()
	}
	return metricName
}

// formatValue formats the metric value based on metric type.
func (e *NginxEvaluator) formatValue(value float64, metricName string) string {
	switch metricName {
	case "nginx_up":
		if value == 0 {
			return "停止"
		}
		return "运行"
	case "connection_usage":
		return fmt.Sprintf("%.1f%%", value)
	case "last_error_time":
		return fmt.Sprintf("%.0f 分钟前", value)
	case "error_page_4xx", "error_page_5xx":
		if value == 0 {
			return "未配置"
		}
		return "已配置"
	case "non_root_user":
		if value == 0 {
			return "root 用户"
		}
		return "普通用户"
	default:
		return fmt.Sprintf("%.2f", value)
	}
}

// generateAlertMessage generates a human-readable alert message.
func (e *NginxEvaluator) generateAlertMessage(
	metricName string,
	currentValue float64,
	level model.AlertLevel,
) string {
	warningThreshold, criticalThreshold := e.getThresholds(metricName)

	switch metricName {
	case "nginx_up":
		return "Nginx 实例连接失败 (nginx_up=0)"

	case "connection_usage":
		if level == model.AlertLevelCritical {
			return fmt.Sprintf("连接使用率为 %.1f%%，已超过严重阈值 %.1f%%",
				currentValue, criticalThreshold)
		}
		return fmt.Sprintf("连接使用率为 %.1f%%，已超过警告阈值 %.1f%%",
			currentValue, warningThreshold)

	case "last_error_time":
		minutes := int(currentValue)
		if level == model.AlertLevelCritical {
			return fmt.Sprintf("最近 %d 分钟内有错误日志（严重阈值: %d 分钟）",
				minutes, e.thresholds.LastErrorCriticalMinutes)
		}
		return fmt.Sprintf("最近 %d 分钟内有错误日志（警告阈值: %d 分钟）",
			minutes, e.thresholds.LastErrorWarningMinutes)

	case "error_page_4xx":
		return "未配置 4xx 错误页重定向 (nginx_error_page_4xx=0)"

	case "error_page_5xx":
		return "未配置 5xx 错误页重定向 (nginx_error_page_5xx=0)"

	case "non_root_user":
		return "Nginx 以 root 用户启动，存在安全风险 (nginx_non_root_user=0)"

	default:
		return fmt.Sprintf("%s 指标异常，当前值: %.2f", metricName, currentValue)
	}
}

// getThresholds returns the warning and critical thresholds for a metric.
func (e *NginxEvaluator) getThresholds(metricName string) (warning float64, critical float64) {
	switch metricName {
	case "connection_usage":
		return e.thresholds.ConnectionUsageWarning, e.thresholds.ConnectionUsageCritical
	case "last_error_time":
		// 注意：逻辑反转，warning > critical（60 > 10）
		return float64(e.thresholds.LastErrorWarningMinutes), float64(e.thresholds.LastErrorCriticalMinutes)
	case "nginx_up":
		return 1, 1 // nginx_up=0 即严重
	case "error_page_4xx", "error_page_5xx":
		return 1, 1 // 未配置即严重
	case "non_root_user":
		return 1, 1 // root 用户即严重
	default:
		return 0, 0
	}
}
