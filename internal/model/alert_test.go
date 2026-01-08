package model

import "testing"

// ============================================================================
// NewAlert Tests
// ============================================================================

func TestNewAlert(t *testing.T) {
	tests := []struct {
		name         string
		hostname     string
		metricName   string
		currentValue float64
		level        AlertLevel
	}{
		{
			name:         "warning alert",
			hostname:     "server-01",
			metricName:   "cpu_usage",
			currentValue: 75.5,
			level:        AlertLevelWarning,
		},
		{
			name:         "critical alert",
			hostname:     "server-02",
			metricName:   "memory_usage",
			currentValue: 95.0,
			level:        AlertLevelCritical,
		},
		{
			name:         "normal level",
			hostname:     "server-03",
			metricName:   "disk_usage",
			currentValue: 50.0,
			level:        AlertLevelNormal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alert := NewAlert(tt.hostname, tt.metricName, tt.currentValue, tt.level)

			if alert.Hostname != tt.hostname {
				t.Errorf("Hostname = %q, want %q", alert.Hostname, tt.hostname)
			}
			if alert.MetricName != tt.metricName {
				t.Errorf("MetricName = %q, want %q", alert.MetricName, tt.metricName)
			}
			if alert.CurrentValue != tt.currentValue {
				t.Errorf("CurrentValue = %v, want %v", alert.CurrentValue, tt.currentValue)
			}
			if alert.Level != tt.level {
				t.Errorf("Level = %v, want %v", alert.Level, tt.level)
			}
		})
	}
}

// ============================================================================
// Alert.IsWarning Tests
// ============================================================================

func TestAlert_IsWarning(t *testing.T) {
	tests := []struct {
		name     string
		level    AlertLevel
		expected bool
	}{
		{
			name:     "warning level returns true",
			level:    AlertLevelWarning,
			expected: true,
		},
		{
			name:     "critical level returns false",
			level:    AlertLevelCritical,
			expected: false,
		},
		{
			name:     "normal level returns false",
			level:    AlertLevelNormal,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alert := &Alert{Level: tt.level}
			if got := alert.IsWarning(); got != tt.expected {
				t.Errorf("IsWarning() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// ============================================================================
// Alert.IsCritical Tests
// ============================================================================

func TestAlert_IsCritical(t *testing.T) {
	tests := []struct {
		name     string
		level    AlertLevel
		expected bool
	}{
		{
			name:     "critical level returns true",
			level:    AlertLevelCritical,
			expected: true,
		},
		{
			name:     "warning level returns false",
			level:    AlertLevelWarning,
			expected: false,
		},
		{
			name:     "normal level returns false",
			level:    AlertLevelNormal,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alert := &Alert{Level: tt.level}
			if got := alert.IsCritical(); got != tt.expected {
				t.Errorf("IsCritical() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// ============================================================================
// NewAlertSummary Tests
// ============================================================================

// ============================================================================
// UnifiedAlert Conversion Tests
// ============================================================================

func TestNewUnifiedAlertFromHostAlert(t *testing.T) {
	tests := []struct {
		name   string
		alert  *Alert
		expect *UnifiedAlert
	}{
		{
			name:   "nil alert returns nil",
			alert:  nil,
			expect: nil,
		},
		{
			name: "valid host alert converts correctly",
			alert: &Alert{
				Hostname:          "server-01",
				MetricName:        "cpu_usage",
				MetricDisplayName: "CPU利用率",
				CurrentValue:      85.5,
				FormattedValue:    "85.5%",
				WarningThreshold:  70,
				CriticalThreshold: 90,
				Level:             AlertLevelWarning,
				Message:           "CPU usage high",
			},
			expect: &UnifiedAlert{
				SourceType:        AlertSourceHost,
				Identifier:        "server-01",
				Level:             AlertLevelWarning,
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
				Level:             AlertLevelWarning,
				Message:           "Connection usage high",
			},
			expect: &UnifiedAlert{
				SourceType:        AlertSourceMySQL,
				Identifier:        "192.168.1.1:3306",
				Level:             AlertLevelWarning,
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
				Level:             AlertLevelCritical,
				Message:           "Replication lag critical",
			},
			expect: &UnifiedAlert{
				SourceType:        AlertSourceRedis,
				Identifier:        "192.168.1.1:6379",
				Level:             AlertLevelCritical,
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
		level    AlertLevel
		expected bool
	}{
		{"warning returns true", AlertLevelWarning, true},
		{"critical returns false", AlertLevelCritical, false},
		{"normal returns false", AlertLevelNormal, false},
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
		level    AlertLevel
		expected bool
	}{
		{"critical returns true", AlertLevelCritical, true},
		{"warning returns false", AlertLevelWarning, false},
		{"normal returns false", AlertLevelNormal, false},
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

// ============================================================================
// NewAlertSummary Tests
// ============================================================================

func TestNewAlertSummary(t *testing.T) {
	tests := []struct {
		name             string
		alerts           []*Alert
		expectedTotal    int
		expectedWarning  int
		expectedCritical int
	}{
		{
			name:             "empty list",
			alerts:           []*Alert{},
			expectedTotal:    0,
			expectedWarning:  0,
			expectedCritical: 0,
		},
		{
			name:             "nil list",
			alerts:           nil,
			expectedTotal:    0,
			expectedWarning:  0,
			expectedCritical: 0,
		},
		{
			name: "mixed alerts",
			alerts: []*Alert{
				{Level: AlertLevelWarning},
				{Level: AlertLevelCritical},
				{Level: AlertLevelWarning},
				{Level: AlertLevelCritical},
				{Level: AlertLevelCritical},
			},
			expectedTotal:    5,
			expectedWarning:  2,
			expectedCritical: 3,
		},
		{
			name: "only warnings",
			alerts: []*Alert{
				{Level: AlertLevelWarning},
				{Level: AlertLevelWarning},
			},
			expectedTotal:    2,
			expectedWarning:  2,
			expectedCritical: 0,
		},
		{
			name: "only criticals",
			alerts: []*Alert{
				{Level: AlertLevelCritical},
				{Level: AlertLevelCritical},
			},
			expectedTotal:    2,
			expectedWarning:  0,
			expectedCritical: 2,
		},
		{
			name: "with nil elements",
			alerts: []*Alert{
				{Level: AlertLevelWarning},
				nil,
				{Level: AlertLevelCritical},
				nil,
			},
			expectedTotal:    2,
			expectedWarning:  1,
			expectedCritical: 1,
		},
		{
			name: "with normal level (not counted as warning or critical)",
			alerts: []*Alert{
				{Level: AlertLevelNormal},
				{Level: AlertLevelWarning},
				{Level: AlertLevelCritical},
			},
			expectedTotal:    3,
			expectedWarning:  1,
			expectedCritical: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := NewAlertSummary(tt.alerts)

			if summary.TotalAlerts != tt.expectedTotal {
				t.Errorf("TotalAlerts = %d, want %d", summary.TotalAlerts, tt.expectedTotal)
			}
			if summary.WarningCount != tt.expectedWarning {
				t.Errorf("WarningCount = %d, want %d", summary.WarningCount, tt.expectedWarning)
			}
			if summary.CriticalCount != tt.expectedCritical {
				t.Errorf("CriticalCount = %d, want %d", summary.CriticalCount, tt.expectedCritical)
			}
		})
	}
}
