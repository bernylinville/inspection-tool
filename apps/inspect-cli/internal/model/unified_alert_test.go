package model

import (
	"testing"

	pkgmodel "inspection-tool/pkg/model"
)

// ============================================================================
// UnifiedAlert Conversion Tests
// ============================================================================

func TestNewUnifiedAlertFromHostAlert(t *testing.T) {
	tests := []struct {
		name   string
		alert  *pkgmodel.Alert
		expect *UnifiedAlert
	}{
		{
			name:   "nil alert returns nil",
			alert:  nil,
			expect: nil,
		},
		{
			name: "valid host alert converts correctly",
			alert: &pkgmodel.Alert{
				Hostname:          "server-01",
				MetricName:        "cpu_usage",
				MetricDisplayName: "CPU利用率",
				CurrentValue:      85.5,
				FormattedValue:    "85.5%",
				WarningThreshold:  70,
				CriticalThreshold: 90,
				Level:             pkgmodel.AlertLevelWarning,
				Message:           "CPU usage high",
			},
			expect: &UnifiedAlert{
				SourceType:        AlertSourceHost,
				Identifier:        "server-01",
				Level:             pkgmodel.AlertLevelWarning,
				MetricName:        "cpu_usage",
				MetricDisplayName: "CPU利用率",
				CurrentValue:      85.5,
				FormattedValue:    "85.5%",
				WarningThreshold:  70,
				CriticalThreshold: 90,
				Message:           "CPU usage high",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewUnifiedAlertFromHostAlert(tt.alert)
			if tt.expect == nil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got.SourceType != tt.expect.SourceType {
				t.Errorf("SourceType = %v, want %v", got.SourceType, tt.expect.SourceType)
			}
			if got.Identifier != tt.expect.Identifier {
				t.Errorf("Identifier = %v, want %v", got.Identifier, tt.expect.Identifier)
			}
			if got.Level != tt.expect.Level {
				t.Errorf("Level = %v, want %v", got.Level, tt.expect.Level)
			}
			if got.MetricName != tt.expect.MetricName {
				t.Errorf("MetricName = %v, want %v", got.MetricName, tt.expect.MetricName)
			}
			if got.CurrentValue != tt.expect.CurrentValue {
				t.Errorf("CurrentValue = %v, want %v", got.CurrentValue, tt.expect.CurrentValue)
			}
		})
	}
}

func TestNewUnifiedAlertFromMySQLAlert(t *testing.T) {
	tests := []struct {
		name   string
		alert  *MySQLAlert
		expect *UnifiedAlert
	}{
		{
			name:   "nil alert returns nil",
			alert:  nil,
			expect: nil,
		},
		{
			name: "valid mysql alert converts correctly",
			alert: &MySQLAlert{
				Address:           "192.168.1.1:3306",
				MetricName:        "connection_usage",
				MetricDisplayName: "连接使用率",
				CurrentValue:      75,
				FormattedValue:    "75%",
				WarningThreshold:  70,
				CriticalThreshold: 90,
				Level:             pkgmodel.AlertLevelWarning,
				Message:           "Connection usage high",
			},
			expect: &UnifiedAlert{
				SourceType:        AlertSourceMySQL,
				Identifier:        "192.168.1.1:3306",
				Level:             pkgmodel.AlertLevelWarning,
				MetricName:        "connection_usage",
				MetricDisplayName: "连接使用率",
				CurrentValue:      75,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewUnifiedAlertFromMySQLAlert(tt.alert)
			if tt.expect == nil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got.SourceType != tt.expect.SourceType {
				t.Errorf("SourceType = %v, want %v", got.SourceType, tt.expect.SourceType)
			}
			if got.Identifier != tt.expect.Identifier {
				t.Errorf("Identifier = %v, want %v", got.Identifier, tt.expect.Identifier)
			}
			if got.Level != tt.expect.Level {
				t.Errorf("Level = %v, want %v", got.Level, tt.expect.Level)
			}
		})
	}
}

func TestNewUnifiedAlertFromRedisAlert(t *testing.T) {
	tests := []struct {
		name   string
		alert  *RedisAlert
		expect *UnifiedAlert
	}{
		{
			name:   "nil alert returns nil",
			alert:  nil,
			expect: nil,
		},
		{
			name: "valid redis alert converts correctly",
			alert: &RedisAlert{
				Address:           "192.168.1.1:6379",
				MetricName:        "replication_lag",
				MetricDisplayName: "复制延迟",
				CurrentValue:      1048576,
				FormattedValue:    "1 MB",
				WarningThreshold:  1048576,
				CriticalThreshold: 10485760,
				Level:             pkgmodel.AlertLevelCritical,
				Message:           "Replication lag critical",
			},
			expect: &UnifiedAlert{
				SourceType:        AlertSourceRedis,
				Identifier:        "192.168.1.1:6379",
				Level:             pkgmodel.AlertLevelCritical,
				MetricName:        "replication_lag",
				MetricDisplayName: "复制延迟",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewUnifiedAlertFromRedisAlert(tt.alert)
			if tt.expect == nil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got.SourceType != tt.expect.SourceType {
				t.Errorf("SourceType = %v, want %v", got.SourceType, tt.expect.SourceType)
			}
			if got.Identifier != tt.expect.Identifier {
				t.Errorf("Identifier = %v, want %v", got.Identifier, tt.expect.Identifier)
			}
			if got.Level != tt.expect.Level {
				t.Errorf("Level = %v, want %v", got.Level, tt.expect.Level)
			}
		})
	}
}

func TestUnifiedAlert_IsWarning(t *testing.T) {
	tests := []struct {
		name     string
		level    pkgmodel.AlertLevel
		expected bool
	}{
		{"warning returns true", pkgmodel.AlertLevelWarning, true},
		{"critical returns false", pkgmodel.AlertLevelCritical, false},
		{"normal returns false", pkgmodel.AlertLevelNormal, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &UnifiedAlert{Level: tt.level}
			if got := u.IsWarning(); got != tt.expected {
				t.Errorf("IsWarning() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUnifiedAlert_IsCritical(t *testing.T) {
	tests := []struct {
		name     string
		level    pkgmodel.AlertLevel
		expected bool
	}{
		{"critical returns true", pkgmodel.AlertLevelCritical, true},
		{"warning returns false", pkgmodel.AlertLevelWarning, false},
		{"normal returns false", pkgmodel.AlertLevelNormal, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &UnifiedAlert{Level: tt.level}
			if got := u.IsCritical(); got != tt.expected {
				t.Errorf("IsCritical() = %v, want %v", got, tt.expected)
			}
		})
	}
}
