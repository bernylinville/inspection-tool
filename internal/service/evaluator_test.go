package service

import (
	"testing"

	"github.com/rs/zerolog"

	"inspection-tool/internal/config"
	"inspection-tool/internal/model"
)

// Helper function to create default threshold config for testing
func createTestThresholds() *config.ThresholdsConfig {
	return &config.ThresholdsConfig{
		CPUUsage: config.ThresholdPair{
			Warning:  70,
			Critical: 90,
		},
		MemoryUsage: config.ThresholdPair{
			Warning:  70,
			Critical: 90,
		},
		DiskUsage: config.ThresholdPair{
			Warning:  70,
			Critical: 90,
		},
		ZombieProcesses: config.ThresholdPair{
			Warning:  1, // > 0 means >= 1
			Critical: 10,
		},
		LoadPerCore: config.ThresholdPair{
			Warning:  0.7,
			Critical: 1.0,
		},
	}
}

// Helper function to create test metric definitions
func createTestMetricDefs() []*model.MetricDefinition {
	return []*model.MetricDefinition{
		{Name: "cpu_usage", DisplayName: "CPU 利用率", Unit: "%"},
		{Name: "memory_usage", DisplayName: "内存利用率", Unit: "%"},
		{Name: "disk_usage", DisplayName: "磁盘利用率", Unit: "%"},
		{Name: "processes_zombies", DisplayName: "僵尸进程数", Unit: "个"},
		{Name: "load_per_core", DisplayName: "单核负载", Unit: ""},
		{Name: "uptime", DisplayName: "运行时间", Unit: "seconds"},
	}
}

// Helper function to create a test evaluator
func createTestEvaluator() *Evaluator {
	return NewEvaluator(
		createTestThresholds(),
		createTestMetricDefs(),
		zerolog.Nop(),
	)
}

// =============================================================================
// TestNewEvaluator - 构造函数测试
// =============================================================================

func TestNewEvaluator(t *testing.T) {
	thresholds := createTestThresholds()
	metrics := createTestMetricDefs()
	logger := zerolog.Nop()

	evaluator := NewEvaluator(thresholds, metrics, logger)

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
	if def, ok := evaluator.metricDefs["cpu_usage"]; !ok || def.DisplayName != "CPU 利用率" {
		t.Error("cpu_usage metric definition not accessible")
	}
}

func TestNewEvaluator_NilMetrics(t *testing.T) {
	evaluator := NewEvaluator(createTestThresholds(), nil, zerolog.Nop())

	if evaluator == nil {
		t.Fatal("expected non-nil evaluator even with nil metrics")
	}
	if len(evaluator.metricDefs) != 0 {
		t.Errorf("expected 0 metric definitions, got %d", len(evaluator.metricDefs))
	}
}

// =============================================================================
// CPU 评估测试
// =============================================================================

func TestEvaluator_CPUUsage_Normal(t *testing.T) {
	evaluator := createTestEvaluator()

	metrics := model.NewHostMetrics("server-01")
	metrics.SetMetric(&model.MetricValue{
		Name:     "cpu_usage",
		RawValue: 50.0, // < 70% warning threshold
	})

	result := evaluator.EvaluateHost("server-01", metrics)

	if result.Status != model.HostStatusNormal {
		t.Errorf("expected normal status, got %s", result.Status)
	}
	if len(result.Alerts) != 0 {
		t.Errorf("expected 0 alerts, got %d", len(result.Alerts))
	}

	// Verify metric status is updated
	cpuMetric := result.Metrics["cpu_usage"]
	if cpuMetric.Status != model.MetricStatusNormal {
		t.Errorf("expected metric status normal, got %s", cpuMetric.Status)
	}
}

func TestEvaluator_CPUUsage_Warning(t *testing.T) {
	evaluator := createTestEvaluator()

	metrics := model.NewHostMetrics("server-01")
	metrics.SetMetric(&model.MetricValue{
		Name:     "cpu_usage",
		RawValue: 75.0, // >= 70% warning, < 90% critical
	})

	result := evaluator.EvaluateHost("server-01", metrics)

	if result.Status != model.HostStatusWarning {
		t.Errorf("expected warning status, got %s", result.Status)
	}
	if len(result.Alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(result.Alerts))
	}

	alert := result.Alerts[0]
	if alert.Level != model.AlertLevelWarning {
		t.Errorf("expected warning level, got %s", alert.Level)
	}
	if alert.MetricName != "cpu_usage" {
		t.Errorf("expected metric name 'cpu_usage', got %s", alert.MetricName)
	}
	if alert.CurrentValue != 75.0 {
		t.Errorf("expected current value 75.0, got %.1f", alert.CurrentValue)
	}
	if alert.WarningThreshold != 70 {
		t.Errorf("expected warning threshold 70, got %.1f", alert.WarningThreshold)
	}
	if alert.CriticalThreshold != 90 {
		t.Errorf("expected critical threshold 90, got %.1f", alert.CriticalThreshold)
	}

	// Verify metric status is updated
	cpuMetric := result.Metrics["cpu_usage"]
	if cpuMetric.Status != model.MetricStatusWarning {
		t.Errorf("expected metric status warning, got %s", cpuMetric.Status)
	}
}

func TestEvaluator_CPUUsage_Critical(t *testing.T) {
	evaluator := createTestEvaluator()

	metrics := model.NewHostMetrics("server-01")
	metrics.SetMetric(&model.MetricValue{
		Name:     "cpu_usage",
		RawValue: 95.0, // >= 90% critical threshold
	})

	result := evaluator.EvaluateHost("server-01", metrics)

	if result.Status != model.HostStatusCritical {
		t.Errorf("expected critical status, got %s", result.Status)
	}
	if len(result.Alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(result.Alerts))
	}

	alert := result.Alerts[0]
	if alert.Level != model.AlertLevelCritical {
		t.Errorf("expected critical level, got %s", alert.Level)
	}
	if alert.MetricDisplayName != "CPU 利用率" {
		t.Errorf("expected display name 'CPU 利用率', got %s", alert.MetricDisplayName)
	}

	// Verify metric status is updated
	cpuMetric := result.Metrics["cpu_usage"]
	if cpuMetric.Status != model.MetricStatusCritical {
		t.Errorf("expected metric status critical, got %s", cpuMetric.Status)
	}
}

func TestEvaluator_CPUUsage_ExactThreshold(t *testing.T) {
	evaluator := createTestEvaluator()

	// Test exact warning threshold
	t.Run("ExactWarning", func(t *testing.T) {
		metrics := model.NewHostMetrics("server-01")
		metrics.SetMetric(&model.MetricValue{
			Name:     "cpu_usage",
			RawValue: 70.0, // == warning threshold
		})

		result := evaluator.EvaluateHost("server-01", metrics)
		if result.Status != model.HostStatusWarning {
			t.Errorf("at exact warning threshold, expected warning status, got %s", result.Status)
		}
	})

	// Test exact critical threshold
	t.Run("ExactCritical", func(t *testing.T) {
		metrics := model.NewHostMetrics("server-01")
		metrics.SetMetric(&model.MetricValue{
			Name:     "cpu_usage",
			RawValue: 90.0, // == critical threshold
		})

		result := evaluator.EvaluateHost("server-01", metrics)
		if result.Status != model.HostStatusCritical {
			t.Errorf("at exact critical threshold, expected critical status, got %s", result.Status)
		}
	})
}

// =============================================================================
// 内存评估测试
// =============================================================================

func TestEvaluator_MemoryUsage_Thresholds(t *testing.T) {
	evaluator := createTestEvaluator()

	tests := []struct {
		name           string
		value          float64
		expectedStatus model.HostStatus
		expectedLevel  model.AlertLevel
	}{
		{"Normal", 50.0, model.HostStatusNormal, model.AlertLevelNormal},
		{"Warning", 75.0, model.HostStatusWarning, model.AlertLevelWarning},
		{"Critical", 95.0, model.HostStatusCritical, model.AlertLevelCritical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := model.NewHostMetrics("server-01")
			metrics.SetMetric(&model.MetricValue{
				Name:     "memory_usage",
				RawValue: tt.value,
			})

			result := evaluator.EvaluateHost("server-01", metrics)

			if result.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, result.Status)
			}

			if tt.expectedLevel == model.AlertLevelNormal {
				if len(result.Alerts) != 0 {
					t.Errorf("expected 0 alerts for normal, got %d", len(result.Alerts))
				}
			} else {
				if len(result.Alerts) != 1 {
					t.Fatalf("expected 1 alert, got %d", len(result.Alerts))
				}
				if result.Alerts[0].Level != tt.expectedLevel {
					t.Errorf("expected level %s, got %s", tt.expectedLevel, result.Alerts[0].Level)
				}
			}
		})
	}
}

// =============================================================================
// 磁盘评估测试
// =============================================================================

func TestEvaluator_DiskUsage_UseMaxValue(t *testing.T) {
	evaluator := createTestEvaluator()

	// disk_usage_max is used for alert evaluation
	metrics := model.NewHostMetrics("server-01")
	metrics.SetMetric(&model.MetricValue{
		Name:     "disk_usage_max",
		RawValue: 85.0, // Above warning threshold
	})

	result := evaluator.EvaluateHost("server-01", metrics)

	if result.Status != model.HostStatusWarning {
		t.Errorf("expected warning status for disk_usage_max=85, got %s", result.Status)
	}
	if len(result.Alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(result.Alerts))
	}

	alert := result.Alerts[0]
	if alert.MetricName != "disk_usage_max" {
		t.Errorf("expected metric name 'disk_usage_max', got %s", alert.MetricName)
	}
	if alert.MetricDisplayName != "磁盘利用率" {
		t.Errorf("expected display name '磁盘利用率', got %s", alert.MetricDisplayName)
	}
}

func TestEvaluator_DiskUsage_ExpandedMetrics_NoAlert(t *testing.T) {
	evaluator := createTestEvaluator()

	// Expanded metrics (disk_usage:/home) should NOT generate alerts
	// They are for display purposes only
	metrics := model.NewHostMetrics("server-01")
	metrics.SetMetric(&model.MetricValue{
		Name:     "disk_usage:/home",
		RawValue: 95.0, // Above critical threshold, but expanded metric
	})

	result := evaluator.EvaluateHost("server-01", metrics)

	if len(result.Alerts) != 0 {
		t.Errorf("expected 0 alerts for expanded metric, got %d", len(result.Alerts))
	}

	// Status should still reflect the severity (for display), but no alert is generated
	// Note: expanded metrics get their status set but don't trigger alerts
}

func TestEvaluator_DiskUsage_MultipleDisks(t *testing.T) {
	evaluator := createTestEvaluator()

	// Real scenario: multiple expanded metrics + aggregated max
	metrics := model.NewHostMetrics("server-01")
	metrics.SetMetric(&model.MetricValue{
		Name:     "disk_usage:/",
		RawValue: 45.0,
	})
	metrics.SetMetric(&model.MetricValue{
		Name:     "disk_usage:/home",
		RawValue: 65.0,
	})
	metrics.SetMetric(&model.MetricValue{
		Name:     "disk_usage:/var",
		RawValue: 92.0, // Critical!
	})
	metrics.SetMetric(&model.MetricValue{
		Name:     "disk_usage_max",
		RawValue: 92.0, // Aggregated max - this triggers the alert
	})

	result := evaluator.EvaluateHost("server-01", metrics)

	// Only disk_usage_max should trigger an alert
	if len(result.Alerts) != 1 {
		t.Errorf("expected 1 alert (from disk_usage_max), got %d", len(result.Alerts))
	}

	if result.Status != model.HostStatusCritical {
		t.Errorf("expected critical status, got %s", result.Status)
	}
}

// =============================================================================
// 负载评估测试
// =============================================================================

func TestEvaluator_LoadPerCore_Thresholds(t *testing.T) {
	evaluator := createTestEvaluator()

	tests := []struct {
		name           string
		value          float64
		expectedStatus model.HostStatus
	}{
		{"Normal", 0.5, model.HostStatusNormal},
		{"Warning", 0.8, model.HostStatusWarning},
		{"Critical", 1.2, model.HostStatusCritical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := model.NewHostMetrics("server-01")
			metrics.SetMetric(&model.MetricValue{
				Name:     "load_per_core",
				RawValue: tt.value,
			})

			result := evaluator.EvaluateHost("server-01", metrics)

			if result.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, result.Status)
			}
		})
	}
}

// =============================================================================
// 僵尸进程评估测试
// =============================================================================

func TestEvaluator_ZombieProcesses_Thresholds(t *testing.T) {
	evaluator := createTestEvaluator()

	tests := []struct {
		name           string
		value          float64
		expectedStatus model.HostStatus
		alertCount     int
	}{
		{"NoZombies", 0.0, model.HostStatusNormal, 0},   // == 0, threshold is > 0
		{"Warning", 5.0, model.HostStatusWarning, 1},    // > 0 warning, < 10 critical
		{"Critical", 15.0, model.HostStatusCritical, 1}, // >= 10 critical
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := model.NewHostMetrics("server-01")
			metrics.SetMetric(&model.MetricValue{
				Name:     "processes_zombies",
				RawValue: tt.value,
			})

			result := evaluator.EvaluateHost("server-01", metrics)

			if result.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, result.Status)
			}
			if len(result.Alerts) != tt.alertCount {
				t.Errorf("expected %d alerts, got %d", tt.alertCount, len(result.Alerts))
			}
		})
	}
}

// =============================================================================
// 待定项（N/A）测试
// =============================================================================

func TestEvaluator_PendingMetric_Skipped(t *testing.T) {
	evaluator := createTestEvaluator()

	metrics := model.NewHostMetrics("server-01")
	metrics.SetMetric(model.NewNAMetricValue("ntp_check")) // IsNA = true

	result := evaluator.EvaluateHost("server-01", metrics)

	if result.Status != model.HostStatusNormal {
		t.Errorf("expected normal status for N/A metric, got %s", result.Status)
	}
	if len(result.Alerts) != 0 {
		t.Errorf("expected 0 alerts for N/A metric, got %d", len(result.Alerts))
	}

	// Verify status is set to pending
	ntpMetric := result.Metrics["ntp_check"]
	if ntpMetric.Status != model.MetricStatusPending {
		t.Errorf("expected pending status for N/A metric, got %s", ntpMetric.Status)
	}
}

func TestEvaluator_MixedMetrics_NormalAndNA(t *testing.T) {
	evaluator := createTestEvaluator()

	metrics := model.NewHostMetrics("server-01")
	metrics.SetMetric(&model.MetricValue{
		Name:     "cpu_usage",
		RawValue: 50.0,
	})
	metrics.SetMetric(model.NewNAMetricValue("ntp_check"))
	metrics.SetMetric(model.NewNAMetricValue("public_network"))

	result := evaluator.EvaluateHost("server-01", metrics)

	if result.Status != model.HostStatusNormal {
		t.Errorf("expected normal status, got %s", result.Status)
	}
	if len(result.Alerts) != 0 {
		t.Errorf("expected 0 alerts, got %d", len(result.Alerts))
	}
}

// =============================================================================
// 主机状态评估测试
// =============================================================================

func TestEvaluator_HostStatus_MostSevere(t *testing.T) {
	evaluator := createTestEvaluator()

	// Multiple alerts of different levels - status should be the most severe
	metrics := model.NewHostMetrics("server-01")
	metrics.SetMetric(&model.MetricValue{
		Name:     "cpu_usage",
		RawValue: 75.0, // Warning
	})
	metrics.SetMetric(&model.MetricValue{
		Name:     "memory_usage",
		RawValue: 95.0, // Critical
	})
	metrics.SetMetric(&model.MetricValue{
		Name:     "disk_usage_max",
		RawValue: 50.0, // Normal
	})

	result := evaluator.EvaluateHost("server-01", metrics)

	if result.Status != model.HostStatusCritical {
		t.Errorf("expected critical status (most severe), got %s", result.Status)
	}
	if len(result.Alerts) != 2 {
		t.Errorf("expected 2 alerts, got %d", len(result.Alerts))
	}
}

func TestEvaluator_HostStatus_Normal(t *testing.T) {
	evaluator := createTestEvaluator()

	metrics := model.NewHostMetrics("server-01")
	metrics.SetMetric(&model.MetricValue{
		Name:     "cpu_usage",
		RawValue: 50.0,
	})
	metrics.SetMetric(&model.MetricValue{
		Name:     "memory_usage",
		RawValue: 50.0,
	})
	metrics.SetMetric(&model.MetricValue{
		Name:     "disk_usage_max",
		RawValue: 50.0,
	})

	result := evaluator.EvaluateHost("server-01", metrics)

	if result.Status != model.HostStatusNormal {
		t.Errorf("expected normal status, got %s", result.Status)
	}
	if len(result.Alerts) != 0 {
		t.Errorf("expected 0 alerts, got %d", len(result.Alerts))
	}
}

func TestEvaluator_HostStatus_OnlyWarnings(t *testing.T) {
	evaluator := createTestEvaluator()

	metrics := model.NewHostMetrics("server-01")
	metrics.SetMetric(&model.MetricValue{
		Name:     "cpu_usage",
		RawValue: 75.0, // Warning
	})
	metrics.SetMetric(&model.MetricValue{
		Name:     "memory_usage",
		RawValue: 80.0, // Warning
	})

	result := evaluator.EvaluateHost("server-01", metrics)

	if result.Status != model.HostStatusWarning {
		t.Errorf("expected warning status, got %s", result.Status)
	}
}

// =============================================================================
// 多主机评估测试
// =============================================================================

func TestEvaluator_EvaluateAll_MultipleHosts(t *testing.T) {
	evaluator := createTestEvaluator()

	hostMetrics := map[string]*model.HostMetrics{
		"server-01": {
			Hostname: "server-01",
			Metrics: map[string]*model.MetricValue{
				"cpu_usage": {Name: "cpu_usage", RawValue: 50.0}, // Normal
			},
		},
		"server-02": {
			Hostname: "server-02",
			Metrics: map[string]*model.MetricValue{
				"cpu_usage": {Name: "cpu_usage", RawValue: 75.0}, // Warning
			},
		},
		"server-03": {
			Hostname: "server-03",
			Metrics: map[string]*model.MetricValue{
				"cpu_usage": {Name: "cpu_usage", RawValue: 95.0}, // Critical
			},
		},
	}

	result := evaluator.EvaluateAll(hostMetrics)

	if len(result.HostResults) != 3 {
		t.Errorf("expected 3 host results, got %d", len(result.HostResults))
	}
	if len(result.Alerts) != 2 {
		t.Errorf("expected 2 alerts (warning + critical), got %d", len(result.Alerts))
	}
	if result.Summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if result.Summary.TotalAlerts != 2 {
		t.Errorf("expected 2 total alerts in summary, got %d", result.Summary.TotalAlerts)
	}
	if result.Summary.WarningCount != 1 {
		t.Errorf("expected 1 warning in summary, got %d", result.Summary.WarningCount)
	}
	if result.Summary.CriticalCount != 1 {
		t.Errorf("expected 1 critical in summary, got %d", result.Summary.CriticalCount)
	}
}

func TestEvaluator_EvaluateAll_EmptyInput(t *testing.T) {
	evaluator := createTestEvaluator()

	result := evaluator.EvaluateAll(map[string]*model.HostMetrics{})

	if len(result.HostResults) != 0 {
		t.Errorf("expected 0 host results, got %d", len(result.HostResults))
	}
	if len(result.Alerts) != 0 {
		t.Errorf("expected 0 alerts, got %d", len(result.Alerts))
	}
	if result.Summary.TotalAlerts != 0 {
		t.Errorf("expected 0 total alerts in summary, got %d", result.Summary.TotalAlerts)
	}
}

// =============================================================================
// 边界条件测试
// =============================================================================

func TestEvaluator_EmptyMetrics(t *testing.T) {
	evaluator := createTestEvaluator()

	metrics := model.NewHostMetrics("server-01")
	// No metrics added

	result := evaluator.EvaluateHost("server-01", metrics)

	if result.Status != model.HostStatusNormal {
		t.Errorf("expected normal status for empty metrics, got %s", result.Status)
	}
	if len(result.Alerts) != 0 {
		t.Errorf("expected 0 alerts for empty metrics, got %d", len(result.Alerts))
	}
}

func TestEvaluator_NilHostMetrics(t *testing.T) {
	evaluator := createTestEvaluator()

	result := evaluator.EvaluateHost("server-01", nil)

	if result.Status != model.HostStatusNormal {
		t.Errorf("expected normal status for nil metrics, got %s", result.Status)
	}
	if len(result.Alerts) != 0 {
		t.Errorf("expected 0 alerts for nil metrics, got %d", len(result.Alerts))
	}
}

func TestEvaluator_ThresholdNotConfigured(t *testing.T) {
	evaluator := createTestEvaluator()

	// Metric without configured threshold
	metrics := model.NewHostMetrics("server-01")
	metrics.SetMetric(&model.MetricValue{
		Name:     "uptime", // No threshold configured for uptime
		RawValue: 86400.0,  // 1 day in seconds
	})

	result := evaluator.EvaluateHost("server-01", metrics)

	if result.Status != model.HostStatusNormal {
		t.Errorf("expected normal status for unconfigured threshold, got %s", result.Status)
	}
	if len(result.Alerts) != 0 {
		t.Errorf("expected 0 alerts for unconfigured threshold, got %d", len(result.Alerts))
	}
}

func TestEvaluator_NilThresholds(t *testing.T) {
	evaluator := NewEvaluator(nil, createTestMetricDefs(), zerolog.Nop())

	metrics := model.NewHostMetrics("server-01")
	metrics.SetMetric(&model.MetricValue{
		Name:     "cpu_usage",
		RawValue: 95.0,
	})

	result := evaluator.EvaluateHost("server-01", metrics)

	// With nil thresholds, no alerts should be generated
	if len(result.Alerts) != 0 {
		t.Errorf("expected 0 alerts with nil thresholds, got %d", len(result.Alerts))
	}
}

func TestEvaluator_NilMetricValue(t *testing.T) {
	evaluator := createTestEvaluator()

	metrics := &model.HostMetrics{
		Hostname: "server-01",
		Metrics: map[string]*model.MetricValue{
			"cpu_usage": nil, // nil metric value
		},
	}

	result := evaluator.EvaluateHost("server-01", metrics)

	// Should handle nil gracefully
	if result.Status != model.HostStatusNormal {
		t.Errorf("expected normal status with nil metric value, got %s", result.Status)
	}
}

// =============================================================================
// 辅助函数测试
// =============================================================================

func TestEvaluator_getMetricDisplayName(t *testing.T) {
	evaluator := createTestEvaluator()

	tests := []struct {
		metricName      string
		expectedDisplay string
	}{
		{"cpu_usage", "CPU 利用率"},
		{"memory_usage", "内存利用率"},
		{"disk_usage_max", "磁盘利用率"},
		{"disk_usage:/home", "磁盘利用率"},
		{"unknown_metric", "unknown_metric"}, // Fallback
	}

	for _, tt := range tests {
		t.Run(tt.metricName, func(t *testing.T) {
			displayName := evaluator.getMetricDisplayName(tt.metricName)
			if displayName != tt.expectedDisplay {
				t.Errorf("expected display name '%s', got '%s'", tt.expectedDisplay, displayName)
			}
		})
	}
}

func TestEvaluator_getThreshold(t *testing.T) {
	evaluator := createTestEvaluator()

	tests := []struct {
		metricName    string
		expectedValid bool
	}{
		{"cpu_usage", true},
		{"memory_usage", true},
		{"disk_usage_max", true},
		{"processes_zombies", true},
		{"load_per_core", true},
		{"uptime", false},  // No threshold configured
		{"unknown", false}, // Unknown metric
	}

	for _, tt := range tests {
		t.Run(tt.metricName, func(t *testing.T) {
			threshold := evaluator.getThreshold(tt.metricName)
			if tt.expectedValid && threshold == nil {
				t.Errorf("expected valid threshold for %s", tt.metricName)
			}
			if !tt.expectedValid && threshold != nil {
				t.Errorf("expected nil threshold for %s", tt.metricName)
			}
		})
	}
}

func TestEvaluator_buildAlertMessage(t *testing.T) {
	evaluator := createTestEvaluator()
	threshold := &config.ThresholdPair{Warning: 70, Critical: 90}

	// Warning message with percentage
	msg := evaluator.buildAlertMessage("cpu_usage", 75.0, model.AlertLevelWarning, threshold)
	if msg == "" {
		t.Error("expected non-empty alert message")
	}
	// Should contain level and value
	if !contains(msg, "警告") {
		t.Errorf("warning message should contain '警告', got: %s", msg)
	}

	// Critical message
	msg = evaluator.buildAlertMessage("cpu_usage", 95.0, model.AlertLevelCritical, threshold)
	if !contains(msg, "严重") {
		t.Errorf("critical message should contain '严重', got: %s", msg)
	}
}

func TestEvaluator_determineHostStatus(t *testing.T) {
	evaluator := createTestEvaluator()

	tests := []struct {
		name           string
		alerts         []*model.Alert
		expectedStatus model.HostStatus
	}{
		{
			"NoAlerts",
			[]*model.Alert{},
			model.HostStatusNormal,
		},
		{
			"OnlyWarning",
			[]*model.Alert{{Level: model.AlertLevelWarning}},
			model.HostStatusWarning,
		},
		{
			"OnlyCritical",
			[]*model.Alert{{Level: model.AlertLevelCritical}},
			model.HostStatusCritical,
		},
		{
			"MixedWarningAndCritical",
			[]*model.Alert{
				{Level: model.AlertLevelWarning},
				{Level: model.AlertLevelCritical},
			},
			model.HostStatusCritical, // Most severe
		},
		{
			"WithNilAlert",
			[]*model.Alert{nil, {Level: model.AlertLevelWarning}},
			model.HostStatusWarning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := evaluator.determineHostStatus(tt.alerts)
			if status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, status)
			}
		})
	}
}

// Helper function - uses contains from collector_test.go (same package)
