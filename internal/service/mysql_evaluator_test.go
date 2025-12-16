package service

import (
	"testing"
	"time"

	"github.com/rs/zerolog"

	"inspection-tool/internal/config"
	"inspection-tool/internal/model"
)

// =============================================================================
// 辅助函数
// =============================================================================

// createTestMySQLThresholds creates default MySQL threshold config for testing.
func createTestMySQLThresholds() *config.MySQLThresholds {
	return &config.MySQLThresholds{
		ConnectionUsageWarning:  70,
		ConnectionUsageCritical: 90,
		MGRMemberCountExpected:  3,
	}
}

// createTestMySQLMetricDefs creates test MySQL metric definitions.
func createTestMySQLMetricDefs() []*model.MySQLMetricDefinition {
	return []*model.MySQLMetricDefinition{
		{Name: "connection_usage", DisplayName: "连接使用率"},
		{Name: "mgr_member_count", DisplayName: "MGR 成员数"},
		{Name: "mgr_state_online", DisplayName: "MGR 在线状态"},
		{Name: "mysql_up", DisplayName: "连接状态"},
	}
}

// createTestMySQLInspectionResult creates a test MySQL inspection result.
func createTestMySQLInspectionResult(address string, clusterMode model.MySQLClusterMode) *model.MySQLInspectionResult {
	instance := model.NewMySQLInstanceWithClusterMode(address, clusterMode)
	result := model.NewMySQLInspectionResult(instance)
	result.CollectedAt = time.Now()
	return result
}

// createTestMySQLEvaluator creates a test MySQL evaluator.
func createTestMySQLEvaluator() *MySQLEvaluator {
	return NewMySQLEvaluator(
		createTestMySQLThresholds(),
		createTestMySQLMetricDefs(),
		zerolog.Nop(),
	)
}

// =============================================================================
// TestNewMySQLEvaluator - 构造函数测试
// =============================================================================

func TestNewMySQLEvaluator(t *testing.T) {
	thresholds := createTestMySQLThresholds()
	metrics := createTestMySQLMetricDefs()
	logger := zerolog.Nop()

	evaluator := NewMySQLEvaluator(thresholds, metrics, logger)

	if evaluator == nil {
		t.Fatal("expected non-nil evaluator")
	}
	if evaluator.thresholds != thresholds {
		t.Error("thresholds not set correctly")
	}
	if len(evaluator.metricDefs) != len(metrics) {
		t.Errorf("expected %d metric definitions, got %d", len(metrics), len(evaluator.metricDefs))
	}

	// Verify metric definitions are accessible by name
	for _, m := range metrics {
		if _, exists := evaluator.metricDefs[m.Name]; !exists {
			t.Errorf("metric definition %s not found in map", m.Name)
		}
	}
}

// =============================================================================
// TestEvaluateConnectionUsage - 连接使用率评估测试
// =============================================================================

func TestEvaluateConnectionUsage(t *testing.T) {
	evaluator := createTestMySQLEvaluator()

	tests := []struct {
		name             string
		maxConnections   int
		currentConnections int
		expectedLevel    model.AlertLevel
		expectAlert      bool
	}{
		{
			name:             "75% usage - Warning",
			maxConnections:   1000,
			currentConnections: 750, // 75%
			expectedLevel:    model.AlertLevelWarning,
			expectAlert:      true,
		},
		{
			name:             "95% usage - Critical",
			maxConnections:   1000,
			currentConnections: 950, // 95%
			expectedLevel:    model.AlertLevelCritical,
			expectAlert:      true,
		},
		{
			name:             "50% usage - Normal",
			maxConnections:   1000,
			currentConnections: 500, // 50%
			expectAlert:      false,
		},
		{
			name:             "Exactly 70% - Warning",
			maxConnections:   1000,
			currentConnections: 700, // 70%
			expectedLevel:    model.AlertLevelWarning,
			expectAlert:      true,
		},
		{
			name:             "Exactly 90% - Critical",
			maxConnections:   1000,
			currentConnections: 900, // 90%
			expectedLevel:    model.AlertLevelCritical,
			expectAlert:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createTestMySQLInspectionResult("172.18.182.91:3306", model.ClusterModeMGR)
			result.MaxConnections = tt.maxConnections
			result.CurrentConnections = tt.currentConnections

			alert := evaluator.evaluateConnectionUsage(result)

			if tt.expectAlert {
				if alert == nil {
					t.Fatal("expected alert but got nil")
				}
				if alert.Level != tt.expectedLevel {
					t.Errorf("expected level %s, got %s", tt.expectedLevel, alert.Level)
				}
				if alert.MetricName != "connection_usage" {
					t.Errorf("expected metric name 'connection_usage', got %s", alert.MetricName)
				}
			} else {
				if alert != nil {
					t.Errorf("expected no alert but got level %s", alert.Level)
				}
			}
		})
	}
}

// =============================================================================
// TestEvaluateMGRMemberCount - MGR 成员数评估测试
// =============================================================================

func TestEvaluateMGRMemberCount(t *testing.T) {
	evaluator := createTestMySQLEvaluator()

	tests := []struct {
		name          string
		memberCount   int
		expectedLevel model.AlertLevel
		expectAlert   bool
	}{
		{
			name:          "Member count = expected - 1 (2/3) - Warning",
			memberCount:   2,
			expectedLevel: model.AlertLevelWarning,
			expectAlert:   true,
		},
		{
			name:          "Member count < expected - 1 (1/3) - Critical",
			memberCount:   1,
			expectedLevel: model.AlertLevelCritical,
			expectAlert:   true,
		},
		{
			name:        "Member count = expected (3/3) - Normal",
			memberCount: 3,
			expectAlert: false,
		},
		{
			name:        "Member count > expected (4/3) - Normal",
			memberCount: 4,
			expectAlert: false,
		},
		{
			name:          "Member count = 0 - Critical",
			memberCount:   0,
			expectedLevel: model.AlertLevelCritical,
			expectAlert:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createTestMySQLInspectionResult("172.18.182.91:3306", model.ClusterModeMGR)
			result.MGRMemberCount = tt.memberCount

			alert := evaluator.evaluateMGRMemberCount(result)

			if tt.expectAlert {
				if alert == nil {
					t.Fatal("expected alert but got nil")
				}
				if alert.Level != tt.expectedLevel {
					t.Errorf("expected level %s, got %s", tt.expectedLevel, alert.Level)
				}
				if alert.MetricName != "mgr_member_count" {
					t.Errorf("expected metric name 'mgr_member_count', got %s", alert.MetricName)
				}
			} else {
				if alert != nil {
					t.Errorf("expected no alert but got level %s", alert.Level)
				}
			}
		})
	}
}

// =============================================================================
// TestEvaluateMGRStateOnline - MGR 在线状态评估测试
// =============================================================================

func TestEvaluateMGRStateOnline(t *testing.T) {
	evaluator := createTestMySQLEvaluator()

	tests := []struct {
		name        string
		online      bool
		expectAlert bool
	}{
		{
			name:        "MGR node offline - Critical",
			online:      false,
			expectAlert: true,
		},
		{
			name:        "MGR node online - Normal",
			online:      true,
			expectAlert: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createTestMySQLInspectionResult("172.18.182.91:3306", model.ClusterModeMGR)
			result.MGRStateOnline = tt.online

			alert := evaluator.evaluateMGRStateOnline(result)

			if tt.expectAlert {
				if alert == nil {
					t.Fatal("expected alert but got nil")
				}
				if alert.Level != model.AlertLevelCritical {
					t.Errorf("expected level Critical, got %s", alert.Level)
				}
				if alert.MetricName != "mgr_state_online" {
					t.Errorf("expected metric name 'mgr_state_online', got %s", alert.MetricName)
				}
			} else {
				if alert != nil {
					t.Errorf("expected no alert but got level %s", alert.Level)
				}
			}
		})
	}
}

// =============================================================================
// TestEvaluate - 单实例评估测试
// =============================================================================

func TestEvaluate(t *testing.T) {
	evaluator := createTestMySQLEvaluator()

	t.Run("Normal instance with no alerts", func(t *testing.T) {
		result := createTestMySQLInspectionResult("172.18.182.91:3306", model.ClusterModeMGR)
		result.MaxConnections = 1000
		result.CurrentConnections = 500 // 50% - Normal
		result.MGRMemberCount = 3       // Expected
		result.MGRStateOnline = true    // Online

		evalResult := evaluator.Evaluate(result)

		if evalResult.Status != model.MySQLStatusNormal {
			t.Errorf("expected status Normal, got %s", evalResult.Status)
		}
		if len(evalResult.Alerts) != 0 {
			t.Errorf("expected 0 alerts, got %d", len(evalResult.Alerts))
		}
		if result.Status != model.MySQLStatusNormal {
			t.Error("result.Status should be updated to Normal")
		}
	})

	t.Run("Instance with connection usage warning", func(t *testing.T) {
		result := createTestMySQLInspectionResult("172.18.182.92:3306", model.ClusterModeMGR)
		result.MaxConnections = 1000
		result.CurrentConnections = 800 // 80% - Warning
		result.MGRMemberCount = 3       // Expected
		result.MGRStateOnline = true    // Online

		evalResult := evaluator.Evaluate(result)

		if evalResult.Status != model.MySQLStatusWarning {
			t.Errorf("expected status Warning, got %s", evalResult.Status)
		}
		if len(evalResult.Alerts) != 1 {
			t.Fatalf("expected 1 alert, got %d", len(evalResult.Alerts))
		}
		if evalResult.Alerts[0].Level != model.AlertLevelWarning {
			t.Errorf("expected alert level Warning, got %s", evalResult.Alerts[0].Level)
		}
	})

	t.Run("Instance with multiple alerts (connection + MGR)", func(t *testing.T) {
		result := createTestMySQLInspectionResult("172.18.182.93:3306", model.ClusterModeMGR)
		result.MaxConnections = 1000
		result.CurrentConnections = 950 // 95% - Critical
		result.MGRMemberCount = 1       // Critical (expected=3)
		result.MGRStateOnline = true    // Online

		evalResult := evaluator.Evaluate(result)

		if evalResult.Status != model.MySQLStatusCritical {
			t.Errorf("expected status Critical, got %s", evalResult.Status)
		}
		if len(evalResult.Alerts) != 2 {
			t.Fatalf("expected 2 alerts, got %d", len(evalResult.Alerts))
		}
		// Both should be critical
		for _, alert := range evalResult.Alerts {
			if alert.Level != model.AlertLevelCritical {
				t.Errorf("expected all alerts to be Critical, got %s", alert.Level)
			}
		}
	})

	t.Run("Failed instance - skip evaluation", func(t *testing.T) {
		result := createTestMySQLInspectionResult("172.18.182.94:3306", model.ClusterModeMGR)
		result.Error = "connection failed"

		evalResult := evaluator.Evaluate(result)

		if evalResult.Status != model.MySQLStatusFailed {
			t.Errorf("expected status Failed, got %s", evalResult.Status)
		}
		if len(evalResult.Alerts) != 0 {
			t.Errorf("expected 0 alerts for failed instance, got %d", len(evalResult.Alerts))
		}
	})

	t.Run("Non-MGR instance - skip MGR evaluation", func(t *testing.T) {
		result := createTestMySQLInspectionResult("172.18.182.95:3306", model.ClusterModeMasterSlave)
		result.MaxConnections = 1000
		result.CurrentConnections = 500 // 50% - Normal
		// MGR fields should not be evaluated
		result.MGRMemberCount = 0
		result.MGRStateOnline = false

		evalResult := evaluator.Evaluate(result)

		if evalResult.Status != model.MySQLStatusNormal {
			t.Errorf("expected status Normal, got %s", evalResult.Status)
		}
		if len(evalResult.Alerts) != 0 {
			t.Errorf("expected 0 alerts, got %d", len(evalResult.Alerts))
		}
	})
}

// =============================================================================
// TestEvaluateAll - 批量评估测试
// =============================================================================

func TestEvaluateAll(t *testing.T) {
	evaluator := createTestMySQLEvaluator()

	results := map[string]*model.MySQLInspectionResult{
		"172.18.182.91:3306": createTestMySQLInspectionResult("172.18.182.91:3306", model.ClusterModeMGR),
		"172.18.182.92:3306": createTestMySQLInspectionResult("172.18.182.92:3306", model.ClusterModeMGR),
		"172.18.182.93:3306": createTestMySQLInspectionResult("172.18.182.93:3306", model.ClusterModeMGR),
	}

	// Instance 1: Normal
	results["172.18.182.91:3306"].MaxConnections = 1000
	results["172.18.182.91:3306"].CurrentConnections = 500
	results["172.18.182.91:3306"].MGRMemberCount = 3
	results["172.18.182.91:3306"].MGRStateOnline = true

	// Instance 2: Warning
	results["172.18.182.92:3306"].MaxConnections = 1000
	results["172.18.182.92:3306"].CurrentConnections = 800
	results["172.18.182.92:3306"].MGRMemberCount = 3
	results["172.18.182.92:3306"].MGRStateOnline = true

	// Instance 3: Critical
	results["172.18.182.93:3306"].MaxConnections = 1000
	results["172.18.182.93:3306"].CurrentConnections = 950
	results["172.18.182.93:3306"].MGRMemberCount = 1
	results["172.18.182.93:3306"].MGRStateOnline = true

	evalResults := evaluator.EvaluateAll(results)

	if len(evalResults) != 3 {
		t.Fatalf("expected 3 evaluation results, got %d", len(evalResults))
	}

	// Count statuses
	var normalCount, warningCount, criticalCount int
	for _, er := range evalResults {
		switch er.Status {
		case model.MySQLStatusNormal:
			normalCount++
		case model.MySQLStatusWarning:
			warningCount++
		case model.MySQLStatusCritical:
			criticalCount++
		}
	}

	if normalCount != 1 {
		t.Errorf("expected 1 normal instance, got %d", normalCount)
	}
	if warningCount != 1 {
		t.Errorf("expected 1 warning instance, got %d", warningCount)
	}
	if criticalCount != 1 {
		t.Errorf("expected 1 critical instance, got %d", criticalCount)
	}
}

// =============================================================================
// TestDetermineInstanceStatus - 状态聚合测试
// =============================================================================

func TestDetermineInstanceStatus(t *testing.T) {
	evaluator := createTestMySQLEvaluator()

	t.Run("Only warning alerts", func(t *testing.T) {
		alerts := []*model.MySQLAlert{
			{Level: model.AlertLevelWarning},
			{Level: model.AlertLevelWarning},
		}
		status := evaluator.determineInstanceStatus(alerts)
		if status != model.MySQLStatusWarning {
			t.Errorf("expected Warning status, got %s", status)
		}
	})

	t.Run("Only critical alerts", func(t *testing.T) {
		alerts := []*model.MySQLAlert{
			{Level: model.AlertLevelCritical},
		}
		status := evaluator.determineInstanceStatus(alerts)
		if status != model.MySQLStatusCritical {
			t.Errorf("expected Critical status, got %s", status)
		}
	})

	t.Run("Mixed warning and critical alerts - should be Critical", func(t *testing.T) {
		alerts := []*model.MySQLAlert{
			{Level: model.AlertLevelWarning},
			{Level: model.AlertLevelCritical},
			{Level: model.AlertLevelWarning},
		}
		status := evaluator.determineInstanceStatus(alerts)
		if status != model.MySQLStatusCritical {
			t.Errorf("expected Critical status (highest priority), got %s", status)
		}
	})

	t.Run("No alerts - should be Normal", func(t *testing.T) {
		alerts := []*model.MySQLAlert{}
		status := evaluator.determineInstanceStatus(alerts)
		if status != model.MySQLStatusNormal {
			t.Errorf("expected Normal status, got %s", status)
		}
	})
}

// =============================================================================
// TestFormatValue - 格式化值测试
// =============================================================================

func TestFormatValue(t *testing.T) {
	evaluator := createTestMySQLEvaluator()

	tests := []struct {
		name       string
		value      float64
		metricName string
		expected   string
	}{
		{
			name:       "connection_usage - percentage",
			value:      75.234,
			metricName: "connection_usage",
			expected:   "75.2%",
		},
		{
			name:       "mgr_member_count - integer",
			value:      2,
			metricName: "mgr_member_count",
			expected:   "2",
		},
		{
			name:       "mgr_state_online - offline",
			value:      0,
			metricName: "mgr_state_online",
			expected:   "离线",
		},
		{
			name:       "mgr_state_online - online",
			value:      1,
			metricName: "mgr_state_online",
			expected:   "在线",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluator.formatValue(tt.value, tt.metricName)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
