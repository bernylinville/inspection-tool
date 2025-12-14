// Package model provides data models for the inspection tool.
package model

// AlertLevel represents the severity level of an alert.
type AlertLevel string

const (
	AlertLevelNormal   AlertLevel = "normal"   // 正常
	AlertLevelWarning  AlertLevel = "warning"  // 警告
	AlertLevelCritical AlertLevel = "critical" // 严重
)

// Alert represents a threshold violation alert for a host metric.
type Alert struct {
	Hostname          string            `json:"hostname"`            // 主机名
	MetricName        string            `json:"metric_name"`         // 指标名称
	MetricDisplayName string            `json:"metric_display_name"` // 指标中文显示名称
	CurrentValue      float64           `json:"current_value"`       // 当前值
	FormattedValue    string            `json:"formatted_value"`     // 格式化后的当前值
	WarningThreshold  float64           `json:"warning_threshold"`   // 警告阈值
	CriticalThreshold float64           `json:"critical_threshold"`  // 严重阈值
	Level             AlertLevel        `json:"level"`               // 告警级别
	Message           string            `json:"message"`             // 告警消息
	Labels            map[string]string `json:"labels,omitempty"`    // 额外标签（如磁盘路径）
}

// NewAlert creates a new Alert with the given parameters.
func NewAlert(hostname, metricName string, currentValue float64, level AlertLevel) *Alert {
	return &Alert{
		Hostname:     hostname,
		MetricName:   metricName,
		CurrentValue: currentValue,
		Level:        level,
	}
}

// IsWarning returns true if this alert is at warning level.
func (a *Alert) IsWarning() bool {
	return a.Level == AlertLevelWarning
}

// IsCritical returns true if this alert is at critical level.
func (a *Alert) IsCritical() bool {
	return a.Level == AlertLevelCritical
}

// AlertSummary provides aggregated alert statistics.
type AlertSummary struct {
	TotalAlerts   int `json:"total_alerts"`   // 告警总数
	WarningCount  int `json:"warning_count"`  // 警告级别数量
	CriticalCount int `json:"critical_count"` // 严重级别数量
}

// NewAlertSummary creates a new AlertSummary from a list of alerts.
func NewAlertSummary(alerts []*Alert) *AlertSummary {
	summary := &AlertSummary{}
	for _, alert := range alerts {
		if alert == nil {
			continue
		}
		summary.TotalAlerts++
		switch alert.Level {
		case AlertLevelWarning:
			summary.WarningCount++
		case AlertLevelCritical:
			summary.CriticalCount++
		}
	}
	return summary
}
