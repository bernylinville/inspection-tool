// Package model provides data models for the inspection tool.
package model

import pkgmodel "inspection-tool/pkg/model"

// AlertSourceType represents the source component of an alert.
type AlertSourceType string

const (
	AlertSourceHost   AlertSourceType = "Host"
	AlertSourceMySQL  AlertSourceType = "MySQL"
	AlertSourceRedis  AlertSourceType = "Redis"
	AlertSourceNginx  AlertSourceType = "Nginx"
	AlertSourceTomcat AlertSourceType = "Tomcat"
)

// UnifiedAlert represents a unified alert structure for combined reporting.
// It aggregates alerts from all component types (Host, MySQL, Redis, Nginx, Tomcat)
// into a single standardized format for the unified "异常汇总" sheet/section.
type UnifiedAlert struct {
	SourceType        AlertSourceType   `json:"source_type"`         // 来源类型 (Host, MySQL, Redis, etc.)
	Identifier        string            `json:"identifier"`          // 实例标识 (hostname or IP:Port)
	Level             pkgmodel.AlertLevel `json:"level"`             // 告警级别
	MetricName        string            `json:"metric_name"`         // 指标名称
	MetricDisplayName string            `json:"metric_display_name"` // 指标中文显示名称
	CurrentValue      float64           `json:"current_value"`       // 当前值
	FormattedValue    string            `json:"formatted_value"`     // 格式化后的当前值
	WarningThreshold  float64           `json:"warning_threshold"`   // 警告阈值
	CriticalThreshold float64           `json:"critical_threshold"`  // 严重阈值
	Message           string            `json:"message"`             // 告警消息
}

// NewUnifiedAlertFromHostAlert converts a Host Alert to UnifiedAlert.
func NewUnifiedAlertFromHostAlert(alert *pkgmodel.Alert) *UnifiedAlert {
	if alert == nil {
		return nil
	}
	return &UnifiedAlert{
		SourceType:        AlertSourceHost,
		Identifier:        alert.Hostname,
		Level:             alert.Level,
		MetricName:        alert.MetricName,
		MetricDisplayName: alert.MetricDisplayName,
		CurrentValue:      alert.CurrentValue,
		FormattedValue:    alert.FormattedValue,
		WarningThreshold:  alert.WarningThreshold,
		CriticalThreshold: alert.CriticalThreshold,
		Message:           alert.Message,
	}
}

// NewUnifiedAlertFromMySQLAlert converts a MySQL Alert to UnifiedAlert.
func NewUnifiedAlertFromMySQLAlert(alert *MySQLAlert) *UnifiedAlert {
	if alert == nil {
		return nil
	}
	return &UnifiedAlert{
		SourceType:        AlertSourceMySQL,
		Identifier:        alert.Address,
		Level:             alert.Level,
		MetricName:        alert.MetricName,
		MetricDisplayName: alert.MetricDisplayName,
		CurrentValue:      alert.CurrentValue,
		FormattedValue:    alert.FormattedValue,
		WarningThreshold:  alert.WarningThreshold,
		CriticalThreshold: alert.CriticalThreshold,
		Message:           alert.Message,
	}
}

// NewUnifiedAlertFromRedisAlert converts a Redis Alert to UnifiedAlert.
func NewUnifiedAlertFromRedisAlert(alert *RedisAlert) *UnifiedAlert {
	if alert == nil {
		return nil
	}
	return &UnifiedAlert{
		SourceType:        AlertSourceRedis,
		Identifier:        alert.Address,
		Level:             alert.Level,
		MetricName:        alert.MetricName,
		MetricDisplayName: alert.MetricDisplayName,
		CurrentValue:      alert.CurrentValue,
		FormattedValue:    alert.FormattedValue,
		WarningThreshold:  alert.WarningThreshold,
		CriticalThreshold: alert.CriticalThreshold,
		Message:           alert.Message,
	}
}

// IsWarning returns true if this unified alert is at warning level.
func (u *UnifiedAlert) IsWarning() bool {
	return u.Level == pkgmodel.AlertLevelWarning
}

// IsCritical returns true if this unified alert is at critical level.
func (u *UnifiedAlert) IsCritical() bool {
	return u.Level == pkgmodel.AlertLevelCritical
}
