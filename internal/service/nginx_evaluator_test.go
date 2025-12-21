package service

import (
	"testing"
	"time"

	"github.com/rs/zerolog"

	"inspection-tool/internal/config"
	"inspection-tool/internal/model"
)

// =============================================================================
// Test Helper Functions
// =============================================================================

// createTestNginxThresholds creates default Nginx threshold config for testing.
func createTestNginxThresholds() *config.NginxThresholds {
	return &config.NginxThresholds{
		ConnectionUsageWarning:   70,
		ConnectionUsageCritical:  90,
		LastErrorWarningMinutes:  60,
		LastErrorCriticalMinutes: 10,
	}
}

// createTestNginxEvaluatorMetricDefs creates test Nginx metric definitions.
func createTestNginxEvaluatorMetricDefs() []*model.NginxMetricDefinition {
	return []*model.NginxMetricDefinition{
		{Name: "nginx_up", DisplayName: "连接状态"},
		{Name: "connection_usage", DisplayName: "连接使用率"},
		{Name: "nginx_last_error_timestamp", DisplayName: "最后错误日志时间"},
		{Name: "nginx_error_page_4xx", DisplayName: "4xx 错误页配置"},
		{Name: "nginx_error_page_5xx", DisplayName: "5xx 错误页配置"},
		{Name: "nginx_non_root_user", DisplayName: "非 root 用户启动"},
		{Name: "nginx_upstream_check_status_code", DisplayName: "Upstream 后端状态"},
	}
}

// createTestNginxInspectionResult creates a test Nginx inspection result.
func createTestNginxInspectionResult(hostname string, port int) *model.NginxInspectionResult {
	instance := model.NewNginxInstance(hostname, port)
	result := model.NewNginxInspectionResult(instance)
	result.CollectedAt = time.Now()
	return result
}

// createTestNginxEvaluator creates a test Nginx evaluator.
func createTestNginxEvaluator() *NginxEvaluator {
	tz, _ := time.LoadLocation("Asia/Shanghai")
	return NewNginxEvaluator(
		createTestNginxThresholds(),
		createTestNginxEvaluatorMetricDefs(),
		tz,
		zerolog.Nop(),
	)
}

// =============================================================================
// TestNewNginxEvaluator - Constructor Tests
// =============================================================================

func TestNewNginxEvaluator(t *testing.T) {
	thresholds := createTestNginxThresholds()
	metrics := createTestNginxEvaluatorMetricDefs()
	tz, _ := time.LoadLocation("Asia/Shanghai")
	logger := zerolog.Nop()

	evaluator := NewNginxEvaluator(thresholds, metrics, tz, logger)

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

func TestNewNginxEvaluator_NilTimezone(t *testing.T) {
	evaluator := NewNginxEvaluator(
		createTestNginxThresholds(),
		createTestNginxEvaluatorMetricDefs(),
		nil, // nil timezone should default to Asia/Shanghai
		zerolog.Nop(),
	)

	if evaluator.timezone == nil {
		t.Fatal("expected non-nil timezone")
	}
	if evaluator.timezone.String() != "Asia/Shanghai" {
		t.Errorf("expected Asia/Shanghai timezone, got %s", evaluator.timezone.String())
	}
}

// =============================================================================
// TestEvaluateConnectionStatus - Connection Status Tests
// =============================================================================

func TestEvaluateConnectionStatus(t *testing.T) {
	evaluator := createTestNginxEvaluator()

	tests := []struct {
		name        string
		up          bool
		expectAlert bool
	}{
		{
			name:        "nginx_up=1 - Normal",
			up:          true,
			expectAlert: false,
		},
		{
			name:        "nginx_up=0 - Critical",
			up:          false,
			expectAlert: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createTestNginxInspectionResult("GX-NM-NGX-01", 80)
			result.Up = tt.up

			alert := evaluator.evaluateConnectionStatus(result)

			if tt.expectAlert {
				if alert == nil {
					t.Fatal("expected alert but got nil")
				}
				if alert.Level != model.AlertLevelCritical {
					t.Errorf("expected level Critical, got %s", alert.Level)
				}
				if alert.MetricName != "nginx_up" {
					t.Errorf("expected metric name 'nginx_up', got %s", alert.MetricName)
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
// TestEvaluateConnectionUsage - Connection Usage Tests
// =============================================================================

func TestNginxEvaluateConnectionUsage(t *testing.T) {
	evaluator := createTestNginxEvaluator()

	tests := []struct {
		name             string
		workerProcesses  int
		workerConnections int
		activeConnections int
		expectedLevel    model.AlertLevel
		expectAlert      bool
	}{
		{
			name:              "75% usage - Warning",
			workerProcesses:   4,
			workerConnections: 1000,
			activeConnections: 3000, // 3000 / 4000 = 75%
			expectedLevel:     model.AlertLevelWarning,
			expectAlert:       true,
		},
		{
			name:              "95% usage - Critical",
			workerProcesses:   4,
			workerConnections: 1000,
			activeConnections: 3800, // 3800 / 4000 = 95%
			expectedLevel:     model.AlertLevelCritical,
			expectAlert:       true,
		},
		{
			name:              "50% usage - Normal",
			workerProcesses:   4,
			workerConnections: 1000,
			activeConnections: 2000, // 2000 / 4000 = 50%
			expectAlert:       false,
		},
		{
			name:              "Exactly 70% - Warning",
			workerProcesses:   4,
			workerConnections: 1000,
			activeConnections: 2800, // 2800 / 4000 = 70%
			expectedLevel:     model.AlertLevelWarning,
			expectAlert:       true,
		},
		{
			name:              "Exactly 90% - Critical",
			workerProcesses:   4,
			workerConnections: 1000,
			activeConnections: 3600, // 3600 / 4000 = 90%
			expectedLevel:     model.AlertLevelCritical,
			expectAlert:       true,
		},
		{
			name:              "Cannot calculate (workerProcesses=0) - No alert",
			workerProcesses:   0,
			workerConnections: 1000,
			activeConnections: 500,
			expectAlert:       false,
		},
		{
			name:              "Cannot calculate (workerConnections=0) - No alert",
			workerProcesses:   4,
			workerConnections: 0,
			activeConnections: 500,
			expectAlert:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createTestNginxInspectionResult("GX-NM-NGX-01", 80)
			result.WorkerProcesses = tt.workerProcesses
			result.WorkerConnections = tt.workerConnections
			result.ActiveConnections = tt.activeConnections
			result.CalculateConnectionUsagePercent()

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
// TestEvaluateLastErrorTime - Last Error Time Tests
// =============================================================================

func TestEvaluateLastErrorTime(t *testing.T) {
	evaluator := createTestNginxEvaluator()
	now := time.Now().Unix()

	tests := []struct {
		name           string
		timestamp      int64
		expectedLevel  model.AlertLevel
		expectAlert    bool
	}{
		{
			name:        "No error (timestamp=0) - Normal",
			timestamp:   0,
			expectAlert: false,
		},
		{
			name:          "Error 5 minutes ago - Critical",
			timestamp:     now - 5*60, // 5 minutes ago
			expectedLevel: model.AlertLevelCritical,
			expectAlert:   true,
		},
		{
			name:          "Error 30 minutes ago - Warning",
			timestamp:     now - 30*60, // 30 minutes ago
			expectedLevel: model.AlertLevelWarning,
			expectAlert:   true,
		},
		{
			name:        "Error 2 hours ago - Normal",
			timestamp:   now - 120*60, // 2 hours ago
			expectAlert: false,
		},
		{
			name:          "Error exactly 10 minutes ago - Critical boundary",
			timestamp:     now - 10*60,
			expectedLevel: model.AlertLevelCritical,
			expectAlert:   true,
		},
		{
			name:          "Error exactly 60 minutes ago - Warning boundary",
			timestamp:     now - 60*60,
			expectedLevel: model.AlertLevelWarning,
			expectAlert:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createTestNginxInspectionResult("GX-NM-NGX-01", 80)
			result.LastErrorTimestamp = tt.timestamp

			alert := evaluator.evaluateLastErrorTime(result)

			if tt.expectAlert {
				if alert == nil {
					t.Fatal("expected alert but got nil")
				}
				if alert.Level != tt.expectedLevel {
					t.Errorf("expected level %s, got %s", tt.expectedLevel, alert.Level)
				}
				if alert.MetricName != "last_error_time" {
					t.Errorf("expected metric name 'last_error_time', got %s", alert.MetricName)
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
// TestEvaluateErrorPage4xx - Error Page 4xx Tests
// =============================================================================

func TestEvaluateErrorPage4xx(t *testing.T) {
	evaluator := createTestNginxEvaluator()

	tests := []struct {
		name        string
		configured  bool
		hasMetric   bool
		expectAlert bool
	}{
		{
			name:        "4xx error page configured - Normal",
			configured:  true,
			hasMetric:   true,
			expectAlert: false,
		},
		{
			name:        "4xx error page not configured - Critical",
			configured:  false,
			hasMetric:   true,
			expectAlert: true,
		},
		{
			name:        "Metric not collected - No alert",
			configured:  false,
			hasMetric:   false,
			expectAlert: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createTestNginxInspectionResult("GX-NM-NGX-01", 80)
			result.ErrorPage4xxConfigured = tt.configured

			if tt.hasMetric {
				mv := &model.NginxMetricValue{
					Name:     "nginx_error_page_4xx",
					RawValue: 0,
					IsNA:     false,
				}
				if tt.configured {
					mv.RawValue = 1
				}
				result.SetMetric(mv)
			}

			alert := evaluator.evaluateErrorPage4xx(result)

			if tt.expectAlert {
				if alert == nil {
					t.Fatal("expected alert but got nil")
				}
				if alert.Level != model.AlertLevelCritical {
					t.Errorf("expected level Critical, got %s", alert.Level)
				}
				if alert.MetricName != "error_page_4xx" {
					t.Errorf("expected metric name 'error_page_4xx', got %s", alert.MetricName)
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
// TestEvaluateErrorPage5xx - Error Page 5xx Tests
// =============================================================================

func TestEvaluateErrorPage5xx(t *testing.T) {
	evaluator := createTestNginxEvaluator()

	tests := []struct {
		name        string
		configured  bool
		hasMetric   bool
		expectAlert bool
	}{
		{
			name:        "5xx error page configured - Normal",
			configured:  true,
			hasMetric:   true,
			expectAlert: false,
		},
		{
			name:        "5xx error page not configured - Critical",
			configured:  false,
			hasMetric:   true,
			expectAlert: true,
		},
		{
			name:        "Metric not collected - No alert",
			configured:  false,
			hasMetric:   false,
			expectAlert: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createTestNginxInspectionResult("GX-NM-NGX-01", 80)
			result.ErrorPage5xxConfigured = tt.configured

			if tt.hasMetric {
				mv := &model.NginxMetricValue{
					Name:     "nginx_error_page_5xx",
					RawValue: 0,
					IsNA:     false,
				}
				if tt.configured {
					mv.RawValue = 1
				}
				result.SetMetric(mv)
			}

			alert := evaluator.evaluateErrorPage5xx(result)

			if tt.expectAlert {
				if alert == nil {
					t.Fatal("expected alert but got nil")
				}
				if alert.Level != model.AlertLevelCritical {
					t.Errorf("expected level Critical, got %s", alert.Level)
				}
				if alert.MetricName != "error_page_5xx" {
					t.Errorf("expected metric name 'error_page_5xx', got %s", alert.MetricName)
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
// TestEvaluateNonRootUser - Non-Root User Tests
// =============================================================================

func TestEvaluateNonRootUser(t *testing.T) {
	evaluator := createTestNginxEvaluator()

	tests := []struct {
		name        string
		nonRoot     bool
		hasMetric   bool
		expectAlert bool
	}{
		{
			name:        "Non-root user - Normal",
			nonRoot:     true,
			hasMetric:   true,
			expectAlert: false,
		},
		{
			name:        "Root user - Critical",
			nonRoot:     false,
			hasMetric:   true,
			expectAlert: true,
		},
		{
			name:        "Metric not collected - No alert",
			nonRoot:     false,
			hasMetric:   false,
			expectAlert: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createTestNginxInspectionResult("GX-NM-NGX-01", 80)
			result.NonRootUser = tt.nonRoot

			if tt.hasMetric {
				mv := &model.NginxMetricValue{
					Name:     "nginx_non_root_user",
					RawValue: 0,
					IsNA:     false,
				}
				if tt.nonRoot {
					mv.RawValue = 1
				}
				result.SetMetric(mv)
			}

			alert := evaluator.evaluateNonRootUser(result)

			if tt.expectAlert {
				if alert == nil {
					t.Fatal("expected alert but got nil")
				}
				if alert.Level != model.AlertLevelCritical {
					t.Errorf("expected level Critical, got %s", alert.Level)
				}
				if alert.MetricName != "non_root_user" {
					t.Errorf("expected metric name 'non_root_user', got %s", alert.MetricName)
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
// TestEvaluateUpstreamStatus - Upstream Status Tests
// =============================================================================

func TestEvaluateUpstreamStatus(t *testing.T) {
	evaluator := createTestNginxEvaluator()

	t.Run("All backends healthy - No alerts", func(t *testing.T) {
		result := createTestNginxInspectionResult("GX-NM-NGX-01", 80)
		result.UpstreamStatus = []model.NginxUpstreamStatus{
			{UpstreamName: "backend", BackendAddress: "172.18.182.91:8080", Status: true},
			{UpstreamName: "backend", BackendAddress: "172.18.182.92:8080", Status: true},
		}

		alerts := evaluator.evaluateUpstreamStatus(result)

		if len(alerts) != 0 {
			t.Errorf("expected 0 alerts, got %d", len(alerts))
		}
	})

	t.Run("One unhealthy backend - One Critical alert", func(t *testing.T) {
		result := createTestNginxInspectionResult("GX-NM-NGX-01", 80)
		result.UpstreamStatus = []model.NginxUpstreamStatus{
			{UpstreamName: "backend", BackendAddress: "172.18.182.91:8080", Status: true},
			{UpstreamName: "backend", BackendAddress: "172.18.182.92:8080", Status: false, FallCount: 3},
		}

		alerts := evaluator.evaluateUpstreamStatus(result)

		if len(alerts) != 1 {
			t.Fatalf("expected 1 alert, got %d", len(alerts))
		}
		if alerts[0].Level != model.AlertLevelCritical {
			t.Errorf("expected Critical level, got %s", alerts[0].Level)
		}
		if !contains(alerts[0].Message, "172.18.182.92:8080") {
			t.Errorf("expected message to contain backend address, got: %s", alerts[0].Message)
		}
	})

	t.Run("Multiple unhealthy backends - Multiple alerts", func(t *testing.T) {
		result := createTestNginxInspectionResult("GX-NM-NGX-01", 80)
		result.UpstreamStatus = []model.NginxUpstreamStatus{
			{UpstreamName: "backend", BackendAddress: "172.18.182.91:8080", Status: false, FallCount: 5},
			{UpstreamName: "backend", BackendAddress: "172.18.182.92:8080", Status: false, FallCount: 3},
		}

		alerts := evaluator.evaluateUpstreamStatus(result)

		if len(alerts) != 2 {
			t.Errorf("expected 2 alerts, got %d", len(alerts))
		}
		for _, alert := range alerts {
			if alert.Level != model.AlertLevelCritical {
				t.Errorf("expected Critical level, got %s", alert.Level)
			}
		}
	})

	t.Run("No upstream configured - No alerts", func(t *testing.T) {
		result := createTestNginxInspectionResult("GX-NM-NGX-01", 80)
		result.UpstreamStatus = []model.NginxUpstreamStatus{}

		alerts := evaluator.evaluateUpstreamStatus(result)

		if len(alerts) != 0 {
			t.Errorf("expected 0 alerts, got %d", len(alerts))
		}
	})
}

// =============================================================================
// TestEvaluate - Single Instance Evaluation Tests
// =============================================================================

func TestNginxEvaluate(t *testing.T) {
	evaluator := createTestNginxEvaluator()

	t.Run("Normal instance with no alerts", func(t *testing.T) {
		result := createTestNginxInspectionResult("GX-NM-NGX-01", 80)
		result.Up = true
		result.WorkerProcesses = 4
		result.WorkerConnections = 1024
		result.ActiveConnections = 1000 // ~24%
		result.CalculateConnectionUsagePercent()
		result.LastErrorTimestamp = 0 // No error
		result.ErrorPage4xxConfigured = true
		result.ErrorPage5xxConfigured = true
		result.NonRootUser = true

		// Set metrics
		result.SetMetric(&model.NginxMetricValue{Name: "nginx_error_page_4xx", RawValue: 1})
		result.SetMetric(&model.NginxMetricValue{Name: "nginx_error_page_5xx", RawValue: 1})
		result.SetMetric(&model.NginxMetricValue{Name: "nginx_non_root_user", RawValue: 1})

		evalResult := evaluator.Evaluate(result)

		if evalResult.Status != model.NginxStatusNormal {
			t.Errorf("expected status Normal, got %s", evalResult.Status)
		}
		if len(evalResult.Alerts) != 0 {
			t.Errorf("expected 0 alerts, got %d", len(evalResult.Alerts))
		}
		if result.Status != model.NginxStatusNormal {
			t.Error("result.Status should be updated to Normal")
		}
	})

	t.Run("Instance with connection usage warning", func(t *testing.T) {
		result := createTestNginxInspectionResult("GX-NM-NGX-02", 80)
		result.Up = true
		result.WorkerProcesses = 4
		result.WorkerConnections = 1024
		result.ActiveConnections = 3200 // 78%
		result.CalculateConnectionUsagePercent()

		evalResult := evaluator.Evaluate(result)

		if evalResult.Status != model.NginxStatusWarning {
			t.Errorf("expected status Warning, got %s", evalResult.Status)
		}
		if len(evalResult.Alerts) != 1 {
			t.Fatalf("expected 1 alert, got %d", len(evalResult.Alerts))
		}
		if evalResult.Alerts[0].Level != model.AlertLevelWarning {
			t.Errorf("expected alert level Warning, got %s", evalResult.Alerts[0].Level)
		}
	})

	t.Run("Instance with multiple critical alerts", func(t *testing.T) {
		result := createTestNginxInspectionResult("GX-NM-NGX-03", 80)
		result.Up = false // Critical: nginx down
		result.WorkerProcesses = 4
		result.WorkerConnections = 1024
		result.ActiveConnections = 3900 // 95% Critical
		result.CalculateConnectionUsagePercent()
		result.LastErrorTimestamp = time.Now().Unix() - 5*60 // 5 min ago - Critical
		result.NonRootUser = false                           // Root user - Critical
		result.SetMetric(&model.NginxMetricValue{Name: "nginx_non_root_user", RawValue: 0})

		evalResult := evaluator.Evaluate(result)

		if evalResult.Status != model.NginxStatusCritical {
			t.Errorf("expected status Critical, got %s", evalResult.Status)
		}
		if len(evalResult.Alerts) < 3 {
			t.Errorf("expected at least 3 alerts, got %d", len(evalResult.Alerts))
		}
		// All should be critical
		for _, alert := range evalResult.Alerts {
			if alert.Level != model.AlertLevelCritical {
				t.Errorf("expected all alerts to be Critical, got %s for %s", alert.Level, alert.MetricName)
			}
		}
	})

	t.Run("Failed instance - skip evaluation", func(t *testing.T) {
		result := createTestNginxInspectionResult("GX-NM-NGX-04", 80)
		result.Error = "connection failed"

		evalResult := evaluator.Evaluate(result)

		if evalResult.Status != model.NginxStatusFailed {
			t.Errorf("expected status Failed, got %s", evalResult.Status)
		}
		if len(evalResult.Alerts) != 0 {
			t.Errorf("expected 0 alerts for failed instance, got %d", len(evalResult.Alerts))
		}
	})
}

// =============================================================================
// TestEvaluateAll - Batch Evaluation Tests
// =============================================================================

func TestNginxEvaluateAll(t *testing.T) {
	evaluator := createTestNginxEvaluator()

	results := map[string]*model.NginxInspectionResult{
		"GX-NM-NGX-01:80": createTestNginxInspectionResult("GX-NM-NGX-01", 80),
		"GX-NM-NGX-02:80": createTestNginxInspectionResult("GX-NM-NGX-02", 80),
		"GX-NM-NGX-03:80": createTestNginxInspectionResult("GX-NM-NGX-03", 80),
	}

	// Instance 1: Normal
	results["GX-NM-NGX-01:80"].Up = true
	results["GX-NM-NGX-01:80"].WorkerProcesses = 4
	results["GX-NM-NGX-01:80"].WorkerConnections = 1024
	results["GX-NM-NGX-01:80"].ActiveConnections = 1000
	results["GX-NM-NGX-01:80"].CalculateConnectionUsagePercent()

	// Instance 2: Warning
	results["GX-NM-NGX-02:80"].Up = true
	results["GX-NM-NGX-02:80"].WorkerProcesses = 4
	results["GX-NM-NGX-02:80"].WorkerConnections = 1024
	results["GX-NM-NGX-02:80"].ActiveConnections = 3200 // ~78%
	results["GX-NM-NGX-02:80"].CalculateConnectionUsagePercent()

	// Instance 3: Critical
	results["GX-NM-NGX-03:80"].Up = false

	evalResults := evaluator.EvaluateAll(results)

	if len(evalResults) != 3 {
		t.Fatalf("expected 3 evaluation results, got %d", len(evalResults))
	}

	// Count statuses
	var normalCount, warningCount, criticalCount int
	for _, er := range evalResults {
		switch er.Status {
		case model.NginxStatusNormal:
			normalCount++
		case model.NginxStatusWarning:
			warningCount++
		case model.NginxStatusCritical:
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
// TestDetermineInstanceStatus - Status Aggregation Tests
// =============================================================================

func TestNginxDetermineInstanceStatus(t *testing.T) {
	evaluator := createTestNginxEvaluator()

	t.Run("Only warning alerts", func(t *testing.T) {
		alerts := []*model.NginxAlert{
			{Level: model.AlertLevelWarning},
			{Level: model.AlertLevelWarning},
		}
		status := evaluator.determineInstanceStatus(alerts)
		if status != model.NginxStatusWarning {
			t.Errorf("expected Warning status, got %s", status)
		}
	})

	t.Run("Only critical alerts", func(t *testing.T) {
		alerts := []*model.NginxAlert{
			{Level: model.AlertLevelCritical},
		}
		status := evaluator.determineInstanceStatus(alerts)
		if status != model.NginxStatusCritical {
			t.Errorf("expected Critical status, got %s", status)
		}
	})

	t.Run("Mixed warning and critical alerts - should be Critical", func(t *testing.T) {
		alerts := []*model.NginxAlert{
			{Level: model.AlertLevelWarning},
			{Level: model.AlertLevelCritical},
			{Level: model.AlertLevelWarning},
		}
		status := evaluator.determineInstanceStatus(alerts)
		if status != model.NginxStatusCritical {
			t.Errorf("expected Critical status (highest priority), got %s", status)
		}
	})

	t.Run("No alerts - should be Normal", func(t *testing.T) {
		alerts := []*model.NginxAlert{}
		status := evaluator.determineInstanceStatus(alerts)
		if status != model.NginxStatusNormal {
			t.Errorf("expected Normal status, got %s", status)
		}
	})
}

// =============================================================================
// TestFormatValue - Value Formatting Tests
// =============================================================================

func TestFormatValue_Nginx(t *testing.T) {
	evaluator := createTestNginxEvaluator()

	tests := []struct {
		name       string
		value      float64
		metricName string
		expected   string
	}{
		{
			name:       "nginx_up - running",
			value:      1,
			metricName: "nginx_up",
			expected:   "运行",
		},
		{
			name:       "nginx_up - stopped",
			value:      0,
			metricName: "nginx_up",
			expected:   "停止",
		},
		{
			name:       "connection_usage - percentage",
			value:      75.234,
			metricName: "connection_usage",
			expected:   "75.2%",
		},
		{
			name:       "last_error_time - minutes",
			value:      15,
			metricName: "last_error_time",
			expected:   "15 分钟前",
		},
		{
			name:       "error_page_4xx - not configured",
			value:      0,
			metricName: "error_page_4xx",
			expected:   "未配置",
		},
		{
			name:       "error_page_5xx - configured",
			value:      1,
			metricName: "error_page_5xx",
			expected:   "已配置",
		},
		{
			name:       "non_root_user - root",
			value:      0,
			metricName: "non_root_user",
			expected:   "root 用户",
		},
		{
			name:       "non_root_user - non-root",
			value:      1,
			metricName: "non_root_user",
			expected:   "普通用户",
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

// =============================================================================
// TestGetThresholds - Threshold Retrieval Tests
// =============================================================================

func TestGetThresholds(t *testing.T) {
	evaluator := createTestNginxEvaluator()

	tests := []struct {
		name             string
		metricName       string
		expectedWarning  float64
		expectedCritical float64
	}{
		{
			name:             "connection_usage",
			metricName:       "connection_usage",
			expectedWarning:  70,
			expectedCritical: 90,
		},
		{
			name:             "last_error_time",
			metricName:       "last_error_time",
			expectedWarning:  60, // minutes
			expectedCritical: 10, // minutes (inverted: warning > critical)
		},
		{
			name:             "nginx_up",
			metricName:       "nginx_up",
			expectedWarning:  1,
			expectedCritical: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warning, critical := evaluator.getThresholds(tt.metricName)
			if warning != tt.expectedWarning {
				t.Errorf("expected warning threshold %f, got %f", tt.expectedWarning, warning)
			}
			if critical != tt.expectedCritical {
				t.Errorf("expected critical threshold %f, got %f", tt.expectedCritical, critical)
			}
		})
	}
}

// =============================================================================
// TestGetDisplayName - Display Name Tests
// =============================================================================

func TestGetDisplayName(t *testing.T) {
	evaluator := createTestNginxEvaluator()

	tests := []struct {
		name       string
		metricName string
		expected   string
	}{
		{
			name:       "connection_usage custom display",
			metricName: "connection_usage",
			expected:   "连接使用率",
		},
		{
			name:       "nginx_up from metric defs",
			metricName: "nginx_up",
			expected:   "连接状态",
		},
		{
			name:       "unknown metric returns name",
			metricName: "unknown_metric",
			expected:   "unknown_metric",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluator.getDisplayName(tt.metricName)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
