// Package service provides business logic services for the inspection tool.
package service

import (
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"inspection-tool/internal/config"
	"inspection-tool/internal/model"
)

// =============================================================================
// Tomcat Evaluator
// =============================================================================

// TomcatEvaluationResult represents the evaluation result for a single Tomcat instance.
type TomcatEvaluationResult struct {
	Identifier string                       `json:"identifier"` // 实例标识符
	Status     model.TomcatInstanceStatus   `json:"status"`     // 实例整体状态
	Alerts     []*model.TomcatAlert         `json:"alerts"`     // 告警列表
}

// TomcatEvaluator evaluates Tomcat instance metrics against thresholds.
type TomcatEvaluator struct {
	thresholds *config.TomcatThresholds                   // 阈值配置
	metricDefs map[string]*model.TomcatMetricDefinition // 指标定义映射（用于获取显示名称）
	timezone   *time.Location                             // 时区（用于格式化时间）
	logger     zerolog.Logger                            // 日志器
}

// NewTomcatEvaluator creates a new TomcatEvaluator with the given threshold configuration.
func NewTomcatEvaluator(
	thresholds *config.TomcatThresholds,
	metrics []*model.TomcatMetricDefinition,
	timezone *time.Location,
	logger zerolog.Logger,
) *TomcatEvaluator {
	metricDefs := make(map[string]*model.TomcatMetricDefinition)
	for _, m := range metrics {
		metricDefs[m.Name] = m
	}

	return &TomcatEvaluator{
		thresholds: thresholds,
		metricDefs: metricDefs,
		timezone:   timezone,
		logger:     logger.With().Str("component", "tomcat_evaluator").Logger(),
	}
}

// EvaluateAll evaluates all Tomcat instances and returns the complete evaluation results.
func (e *TomcatEvaluator) EvaluateAll(
	results map[string]*model.TomcatInspectionResult,
) []*TomcatEvaluationResult {
	evalResults := make([]*TomcatEvaluationResult, 0, len(results))

	for _, result := range results {
		evalResult := e.Evaluate(result)
		evalResults = append(evalResults, evalResult)
	}

	e.logger.Info().
		Int("total_instances", len(evalResults)).
		Msg("Tomcat evaluation completed")

	return evalResults
}

// Evaluate evaluates a single Tomcat instance against configured thresholds.
func (e *TomcatEvaluator) Evaluate(
	result *model.TomcatInspectionResult,
) *TomcatEvaluationResult {
	evalResult := &TomcatEvaluationResult{
		Identifier: result.GetIdentifier(),
		Status:     model.TomcatStatusNormal,
		Alerts:     make([]*model.TomcatAlert, 0),
	}

	// Skip failed instances
	if result.Error != "" {
		evalResult.Status = model.TomcatStatusFailed
		e.logger.Debug().
			Str("identifier", result.GetIdentifier()).
			Str("error", result.Error).
			Msg("skipping evaluation for failed instance")
		return evalResult
	}

	// 1. Evaluate up status (tomcat_up)
	if alert := e.evaluateUpStatus(result); alert != nil {
		evalResult.Alerts = append(evalResult.Alerts, alert)
	}

	// 2. Evaluate non-root user (tomcat_non_root_user)
	if alert := e.evaluateNonRootUser(result); alert != nil {
		evalResult.Alerts = append(evalResult.Alerts, alert)
	}

	// 3. Evaluate last error time (tomcat_last_error_timestamp) - INVERTED LOGIC
	if alert := e.evaluateLastErrorTime(result); alert != nil {
		evalResult.Alerts = append(evalResult.Alerts, alert)
	}

	// Aggregate status
	evalResult.Status = e.determineInstanceStatus(evalResult.Alerts)

	// Update original result
	result.Status = evalResult.Status
	result.Alerts = evalResult.Alerts

	// Format time fields
	result.UptimeFormatted = result.FormatUptime(e.timezone)
	result.LastErrorTimeFormatted = result.FormatLastErrorTime(e.timezone)

	e.logger.Debug().
		Str("identifier", result.GetIdentifier()).
		Str("status", string(evalResult.Status)).
		Int("alert_count", len(evalResult.Alerts)).
		Msg("instance evaluation completed")

	return evalResult
}

// evaluateUpStatus evaluates tomcat_up status.
// tomcat_up = 0 -> Critical
func (e *TomcatEvaluator) evaluateUpStatus(
	result *model.TomcatInspectionResult,
) *model.TomcatAlert {
	mv := result.GetMetric("tomcat_up")
	if mv == nil || mv.IsNA {
		return nil // Metric not collected, skip
	}

	if !result.Up {
		return e.createAlert(
			result.GetIdentifier(),
			"tomcat_up",
			0,
			model.AlertLevelCritical,
		)
	}

	return nil
}

// evaluateNonRootUser evaluates tomcat_non_root_user.
// = 0 (root user) -> Critical (security risk)
func (e *TomcatEvaluator) evaluateNonRootUser(
	result *model.TomcatInspectionResult,
) *model.TomcatAlert {
	mv := result.GetMetric("tomcat_non_root_user")
	if mv == nil || mv.IsNA {
		return nil // Metric not collected, skip
	}

	if !result.NonRootUser {
		return e.createAlert(
			result.GetIdentifier(),
			"tomcat_non_root_user",
			0,
			model.AlertLevelCritical,
		)
	}

	return nil
}

// evaluateLastErrorTime evaluates tomcat_last_error_timestamp.
// LOGIC IS INVERTED: more recent error = more severe
//   - <10 minutes -> Critical
//   - <60 minutes -> Warning
//   - timestamp = 0 (no error) -> Normal
func (e *TomcatEvaluator) evaluateLastErrorTime(
	result *model.TomcatInspectionResult,
) *model.TomcatAlert {
	mv := result.GetMetric("tomcat_last_error_timestamp")
	if mv == nil || mv.IsNA {
		return nil // Metric not collected, skip
	}

	timestamp := result.LastErrorTimestamp

	// 0 means no error ever, normal
	if timestamp == 0 {
		return nil
	}

	now := time.Now().Unix()
	minutesSinceError := (now - timestamp) / 60

	// Critical: error within 10 minutes
	criticalMinutes := int64(e.thresholds.LastErrorCriticalMinutes)
	if minutesSinceError <= criticalMinutes {
		return e.createAlert(
			result.GetIdentifier(),
			"tomcat_last_error_timestamp",
			float64(minutesSinceError),
			model.AlertLevelCritical,
		)
	}

	// Warning: error within 60 minutes
	warningMinutes := int64(e.thresholds.LastErrorWarningMinutes)
	if minutesSinceError <= warningMinutes {
		return e.createAlert(
			result.GetIdentifier(),
			"tomcat_last_error_timestamp",
			float64(minutesSinceError),
			model.AlertLevelWarning,
		)
	}

	return nil
}

// determineInstanceStatus determines overall status based on alerts.
// Priority: Critical > Warning > Normal
func (e *TomcatEvaluator) determineInstanceStatus(
	alerts []*model.TomcatAlert,
) model.TomcatInstanceStatus {
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
		return model.TomcatStatusCritical
	}
	if hasWarning {
		return model.TomcatStatusWarning
	}
	return model.TomcatStatusNormal
}

// createAlert creates a TomcatAlert with formatted message.
func (e *TomcatEvaluator) createAlert(
	identifier string,
	metricName string,
	currentValue float64,
	level model.AlertLevel,
) *model.TomcatAlert {
	// Get display name
	displayName := metricName
	if def, exists := e.metricDefs[metricName]; exists {
		displayName = def.GetDisplayName()
	}

	// Format value
	formattedValue := e.formatValue(currentValue, metricName)

	// Generate message
	message := e.generateAlertMessage(metricName, currentValue, level)

	// Get thresholds
	warningThreshold, criticalThreshold := e.getThresholds(metricName)

	return &model.TomcatAlert{
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

// formatValue formats metric value for display.
func (e *TomcatEvaluator) formatValue(value float64, metricName string) string {
	switch metricName {
	case "tomcat_up":
		if value == 0 {
			return "停止"
		}
		return "运行"
	case "tomcat_non_root_user":
		if value == 0 {
			return "root 用户"
		}
		return "普通用户"
	case "tomcat_last_error_timestamp":
		return fmt.Sprintf("%.0f 分钟前", value)
	default:
		return fmt.Sprintf("%.2f", value)
	}
}

// generateAlertMessage generates human-readable alert message.
func (e *TomcatEvaluator) generateAlertMessage(
	metricName string,
	currentValue float64,
	level model.AlertLevel,
) string {
	switch metricName {
	case "tomcat_up":
		return "Tomcat 实例连接失败 (tomcat_up=0)"
	case "tomcat_non_root_user":
		return "Tomcat 以 root 用户启动，存在安全风险 (tomcat_non_root_user=0)"
	case "tomcat_last_error_timestamp":
		minutes := int(currentValue)
		if level == model.AlertLevelCritical {
			return fmt.Sprintf("最近 %d 分钟内有错误日志（严重阈值: %d 分钟）",
				minutes, e.thresholds.LastErrorCriticalMinutes)
		}
		return fmt.Sprintf("最近 %d 分钟内有错误日志（警告阈值: %d 分钟）",
			minutes, e.thresholds.LastErrorWarningMinutes)
	default:
		return fmt.Sprintf("%s 指标异常，当前值: %.2f", metricName, currentValue)
	}
}

// getThresholds returns warning and critical thresholds for a metric.
func (e *TomcatEvaluator) getThresholds(metricName string) (warning float64, critical float64) {
	switch metricName {
	case "tomcat_last_error_timestamp":
		// INVERTED: warning > critical (60 > 10)
		return float64(e.thresholds.LastErrorWarningMinutes),
			float64(e.thresholds.LastErrorCriticalMinutes)
	case "tomcat_up":
		return 1, 1
	case "tomcat_non_root_user":
		return 1, 1
	default:
		return 0, 0
	}
}
