// Package service provides business logic services for the inspection tool.
package service

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog"

	"inspection-tool/internal/config"
	"inspection-tool/internal/model"
)

// metricThresholdMap maps metric names to their corresponding threshold config field names.
var metricThresholdMap = map[string]string{
	"cpu_usage":         "cpu_usage",
	"memory_usage":      "memory_usage",
	"disk_usage_max":    "disk_usage", // 磁盘使用聚合最大值用于告警判断
	"processes_zombies": "zombie_processes",
	"load_per_core":     "load_per_core",
}

// HostEvaluationResult contains the evaluation result for a single host.
type HostEvaluationResult struct {
	Hostname string                        `json:"hostname"`
	Status   model.HostStatus              `json:"status"`
	Metrics  map[string]*model.MetricValue `json:"metrics"`
	Alerts   []*model.Alert                `json:"alerts"`
}

// EvaluationResult contains the complete evaluation results for all hosts.
type EvaluationResult struct {
	HostResults []*HostEvaluationResult `json:"host_results"`
	Alerts      []*model.Alert          `json:"alerts"`
	Summary     *model.AlertSummary     `json:"summary"`
}

// Evaluator performs threshold evaluation on collected metrics.
type Evaluator struct {
	thresholds *config.ThresholdsConfig
	metricDefs map[string]*model.MetricDefinition // 指标定义映射，用于获取显示名称
	logger     zerolog.Logger
}

// NewEvaluator creates a new Evaluator with the given threshold configuration.
func NewEvaluator(thresholds *config.ThresholdsConfig, metrics []*model.MetricDefinition, logger zerolog.Logger) *Evaluator {
	metricDefs := make(map[string]*model.MetricDefinition)
	for _, m := range metrics {
		metricDefs[m.Name] = m
	}

	return &Evaluator{
		thresholds: thresholds,
		metricDefs: metricDefs,
		logger:     logger.With().Str("component", "evaluator").Logger(),
	}
}

// EvaluateAll evaluates all hosts and returns the complete evaluation result.
func (e *Evaluator) EvaluateAll(hostMetrics map[string]*model.HostMetrics) *EvaluationResult {
	result := &EvaluationResult{
		HostResults: make([]*HostEvaluationResult, 0, len(hostMetrics)),
		Alerts:      make([]*model.Alert, 0),
	}

	for hostname, metrics := range hostMetrics {
		hostResult := e.EvaluateHost(hostname, metrics)
		result.HostResults = append(result.HostResults, hostResult)
		result.Alerts = append(result.Alerts, hostResult.Alerts...)
	}

	result.Summary = model.NewAlertSummary(result.Alerts)

	e.logger.Info().
		Int("total_hosts", len(result.HostResults)).
		Int("total_alerts", result.Summary.TotalAlerts).
		Int("warning_count", result.Summary.WarningCount).
		Int("critical_count", result.Summary.CriticalCount).
		Msg("evaluation completed")

	return result
}

// EvaluateHost evaluates all metrics for a single host.
func (e *Evaluator) EvaluateHost(hostname string, hostMetrics *model.HostMetrics) *HostEvaluationResult {
	result := &HostEvaluationResult{
		Hostname: hostname,
		Status:   model.HostStatusNormal,
		Metrics:  make(map[string]*model.MetricValue),
		Alerts:   make([]*model.Alert, 0),
	}

	if hostMetrics == nil || hostMetrics.Metrics == nil {
		e.logger.Warn().Str("hostname", hostname).Msg("no metrics to evaluate")
		return result
	}

	result.Metrics = hostMetrics.Metrics

	for metricName, metricValue := range hostMetrics.Metrics {
		if metricValue == nil {
			continue
		}

		alert := e.evaluateMetric(hostname, metricName, metricValue)
		if alert != nil {
			result.Alerts = append(result.Alerts, alert)
		}
	}

	result.Status = e.determineHostStatus(result.Alerts)

	e.logger.Debug().
		Str("hostname", hostname).
		Str("status", string(result.Status)).
		Int("alerts", len(result.Alerts)).
		Msg("host evaluation completed")

	return result
}

// evaluateMetric evaluates a single metric and returns an Alert if threshold is exceeded.
// Returns nil if the metric is within normal range or is N/A.
func (e *Evaluator) evaluateMetric(hostname, metricName string, value *model.MetricValue) *model.Alert {
	// Skip N/A metrics (pending items or failed collection)
	if value.IsNA {
		value.Status = model.MetricStatusPending
		return nil
	}

	// Format the metric value for display
	value.FormattedValue = e.formatMetricValue(metricName, value.RawValue)

	// Skip expanded metrics (e.g., disk_usage:/home) - only evaluate aggregated metrics
	if strings.Contains(metricName, ":") {
		// Expanded metrics are for display only, don't trigger alerts
		// Evaluate status but don't generate alerts
		baseName := strings.Split(metricName, ":")[0]
		threshold := e.getThreshold(baseName + "_max")
		if threshold == nil {
			threshold = e.getThreshold(baseName)
		}
		if threshold != nil {
			e.setMetricStatus(value, threshold)
		}
		return nil
	}

	threshold := e.getThreshold(metricName)
	if threshold == nil {
		// No threshold configured for this metric, skip evaluation
		value.Status = model.MetricStatusNormal
		return nil
	}

	// Evaluate and set status
	level := e.evaluateThreshold(value.RawValue, threshold)
	e.setMetricStatus(value, threshold)

	// Only generate alert for warning or critical
	if level == model.AlertLevelNormal {
		return nil
	}

	// Build alert
	alert := &model.Alert{
		Hostname:          hostname,
		MetricName:        metricName,
		MetricDisplayName: e.getMetricDisplayName(metricName),
		CurrentValue:      value.RawValue,
		FormattedValue:    value.FormattedValue,
		WarningThreshold:  threshold.Warning,
		CriticalThreshold: threshold.Critical,
		Level:             level,
		Message:           e.buildAlertMessage(metricName, value.RawValue, level, threshold),
		Labels:            value.Labels,
	}

	return alert
}

// setMetricStatus sets the Status field of a MetricValue based on threshold evaluation.
func (e *Evaluator) setMetricStatus(value *model.MetricValue, threshold *config.ThresholdPair) {
	if value.RawValue >= threshold.Critical {
		value.Status = model.MetricStatusCritical
	} else if value.RawValue >= threshold.Warning {
		value.Status = model.MetricStatusWarning
	} else {
		value.Status = model.MetricStatusNormal
	}
}

// evaluateThreshold compares a value against warning and critical thresholds.
func (e *Evaluator) evaluateThreshold(value float64, threshold *config.ThresholdPair) model.AlertLevel {
	if value >= threshold.Critical {
		return model.AlertLevelCritical
	}
	if value >= threshold.Warning {
		return model.AlertLevelWarning
	}
	return model.AlertLevelNormal
}

// getThreshold retrieves the threshold configuration for a metric name.
func (e *Evaluator) getThreshold(metricName string) *config.ThresholdPair {
	if e.thresholds == nil {
		return nil
	}

	// Map metric name to threshold config field
	thresholdKey, ok := metricThresholdMap[metricName]
	if !ok {
		return nil
	}

	switch thresholdKey {
	case "cpu_usage":
		return &e.thresholds.CPUUsage
	case "memory_usage":
		return &e.thresholds.MemoryUsage
	case "disk_usage":
		return &e.thresholds.DiskUsage
	case "zombie_processes":
		return &e.thresholds.ZombieProcesses
	case "load_per_core":
		return &e.thresholds.LoadPerCore
	default:
		return nil
	}
}

// getMetricDisplayName retrieves the display name for a metric from definitions.
func (e *Evaluator) getMetricDisplayName(metricName string) string {
	// Handle expanded metrics (e.g., disk_usage:/home → disk_usage)
	baseName := metricName
	if idx := strings.Index(metricName, ":"); idx > 0 {
		baseName = metricName[:idx]
	}
	// Handle aggregated metrics (e.g., disk_usage_max → disk_usage)
	if strings.HasSuffix(baseName, "_max") {
		baseName = strings.TrimSuffix(baseName, "_max")
	}

	if def, ok := e.metricDefs[baseName]; ok {
		return def.DisplayName
	}

	// Fallback to metric name if no definition found
	return metricName
}

// buildAlertMessage creates a human-readable alert message.
func (e *Evaluator) buildAlertMessage(metricName string, value float64, level model.AlertLevel, threshold *config.ThresholdPair) string {
	displayName := e.getMetricDisplayName(metricName)
	levelStr := "警告"
	thresholdValue := threshold.Warning
	if level == model.AlertLevelCritical {
		levelStr = "严重"
		thresholdValue = threshold.Critical
	}

	// Format the message based on metric type
	unit := e.getMetricUnit(metricName)
	if unit == "%" {
		return fmt.Sprintf("%s %s: %.1f%% (阈值: %.1f%%)", displayName, levelStr, value, thresholdValue)
	}
	return fmt.Sprintf("%s %s: %.2f (阈值: %.2f)", displayName, levelStr, value, thresholdValue)
}

// getMetricUnit retrieves the unit for a metric from definitions.
func (e *Evaluator) getMetricUnit(metricName string) string {
	// Handle expanded metrics
	baseName := metricName
	if idx := strings.Index(metricName, ":"); idx > 0 {
		baseName = metricName[:idx]
	}
	if strings.HasSuffix(baseName, "_max") {
		baseName = strings.TrimSuffix(baseName, "_max")
	}

	if def, ok := e.metricDefs[baseName]; ok {
		return def.Unit
	}
	return ""
}

// determineHostStatus determines the overall host status based on alerts.
// Returns the most severe alert level.
func (e *Evaluator) determineHostStatus(alerts []*model.Alert) model.HostStatus {
	if len(alerts) == 0 {
		return model.HostStatusNormal
	}

	hasCritical := false
	hasWarning := false

	for _, alert := range alerts {
		if alert == nil {
			continue
		}
		switch alert.Level {
		case model.AlertLevelCritical:
			hasCritical = true
		case model.AlertLevelWarning:
			hasWarning = true
		}
	}

	if hasCritical {
		return model.HostStatusCritical
	}
	if hasWarning {
		return model.HostStatusWarning
	}
	return model.HostStatusNormal
}

// formatMetricValue formats a raw metric value based on the metric definition.
func (e *Evaluator) formatMetricValue(metricName string, value float64) string {
	// Get base metric name (strip expanded suffix like ":/" or "_max")
	baseName := metricName
	if idx := strings.Index(metricName, ":"); idx > 0 {
		baseName = metricName[:idx]
	}
	if strings.HasSuffix(baseName, "_max") {
		baseName = strings.TrimSuffix(baseName, "_max")
	}

	// Get metric definition for format info
	def, ok := e.metricDefs[baseName]
	if !ok {
		// Fallback to default number format
		return fmt.Sprintf("%.2f", value)
	}

	// Format based on metric format type
	switch def.Format {
	case model.MetricFormatPercent:
		return fmt.Sprintf("%.1f%%", value)
	case model.MetricFormatSize:
		return formatBytes(int64(value))
	case model.MetricFormatDuration:
		return formatUptime(value)
	case model.MetricFormatNumber:
		// For counts/integers, show as integer if no decimals
		if value == float64(int64(value)) {
			return fmt.Sprintf("%.0f", value)
		}
		return fmt.Sprintf("%.2f", value)
	default:
		// Default: use unit hint if available
		if def.Unit == "%" {
			return fmt.Sprintf("%.1f%%", value)
		}
		if def.Unit == "个" || def.Unit == "core" {
			return fmt.Sprintf("%.0f", value)
		}
		return fmt.Sprintf("%.2f", value)
	}
}

// formatBytes formats bytes to human-readable size.
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// formatUptime formats seconds to human-readable uptime.
func formatUptime(seconds float64) string {
	days := int(seconds / 86400)
	hours := int((seconds - float64(days*86400)) / 3600)
	minutes := int((seconds - float64(days*86400) - float64(hours*3600)) / 60)

	if days > 0 {
		return fmt.Sprintf("%d天%d时%d分", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%d时%d分", hours, minutes)
	}
	return fmt.Sprintf("%d分钟", minutes)
}
