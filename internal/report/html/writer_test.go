package html

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"inspection-tool/internal/model"
)

func TestNewWriter(t *testing.T) {
	t.Run("nil timezone defaults to Asia/Shanghai", func(t *testing.T) {
		w := NewWriter(nil, "")
		if w.timezone == nil {
			t.Fatal("expected timezone to be set")
		}
		if w.timezone.String() != "Asia/Shanghai" {
			t.Errorf("expected timezone Asia/Shanghai, got %s", w.timezone.String())
		}
	})

	t.Run("custom timezone", func(t *testing.T) {
		loc, _ := time.LoadLocation("America/New_York")
		w := NewWriter(loc, "")
		if w.timezone != loc {
			t.Errorf("expected custom timezone")
		}
	})

	t.Run("with template path", func(t *testing.T) {
		w := NewWriter(nil, "/path/to/template.html")
		if w.templatePath != "/path/to/template.html" {
			t.Errorf("expected template path to be set")
		}
	})
}

func TestWriter_Format(t *testing.T) {
	w := NewWriter(nil, "")
	if w.Format() != "html" {
		t.Errorf("expected format 'html', got '%s'", w.Format())
	}
}

func TestWriter_Write_NilResult(t *testing.T) {
	w := NewWriter(nil, "")
	err := w.Write(nil, "test.html")
	if err == nil {
		t.Error("expected error for nil result")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("expected error message to mention nil, got: %s", err.Error())
	}
}

func TestWriter_Write_Success(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "test_report.html")

	// Create writer
	w := NewWriter(nil, "")

	// Create test result
	result := createTestResult()

	// Write report
	err := w.Write(result, outputPath)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("output file was not created")
	}

	// Read and verify content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	// Check for key content
	contentStr := string(content)
	expectedContent := []string{
		"<!DOCTYPE html>",
		"系统巡检报告",
		"test-host-1",
		"192.168.1.1",
		"正常",
		"巡检概览",
		"主机详情",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("expected content to contain '%s'", expected)
		}
	}
}

func TestWriter_Write_AddsHtmlExtension(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "test_report") // No extension

	w := NewWriter(nil, "")
	result := createTestResult()

	err := w.Write(result, outputPath)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify .html extension was added
	expectedPath := outputPath + ".html"
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("file with .html extension was not created")
	}
}

func TestWriter_LoadTemplate_Default(t *testing.T) {
	w := NewWriter(nil, "")
	tmpl, err := w.loadTemplate()
	if err != nil {
		t.Fatalf("failed to load default template: %v", err)
	}
	if tmpl == nil {
		t.Error("expected template to be loaded")
	}
}

func TestWriter_LoadTemplate_CustomNotFound(t *testing.T) {
	// Non-existent template path should fall back to default
	w := NewWriter(nil, "/nonexistent/path/template.html")
	tmpl, err := w.loadTemplate()
	if err != nil {
		t.Fatalf("failed to load template: %v", err)
	}
	if tmpl == nil {
		t.Error("expected template to be loaded (fallback to default)")
	}
}

func TestWriter_LoadTemplate_Custom(t *testing.T) {
	// Create a custom template
	tempDir := t.TempDir()
	customTemplate := filepath.Join(tempDir, "custom.html")

	customContent := `<!DOCTYPE html>
<html>
<head><title>{{.Title}}</title></head>
<body>
<h1>Custom Template</h1>
<p>Hosts: {{len .Hosts}}</p>
</body>
</html>`

	err := os.WriteFile(customTemplate, []byte(customContent), 0644)
	if err != nil {
		t.Fatalf("failed to create custom template: %v", err)
	}

	// Test loading custom template
	w := NewWriter(nil, customTemplate)
	tmpl, err := w.loadTemplate()
	if err != nil {
		t.Fatalf("failed to load custom template: %v", err)
	}
	if tmpl == nil {
		t.Error("expected custom template to be loaded")
	}

	// Test writing with custom template
	outputPath := filepath.Join(tempDir, "output.html")
	result := createTestResult()

	err = w.Write(result, outputPath)
	if err != nil {
		t.Fatalf("Write with custom template failed: %v", err)
	}

	// Verify custom template was used
	content, _ := os.ReadFile(outputPath)
	if !strings.Contains(string(content), "Custom Template") {
		t.Error("custom template was not used")
	}
}

func TestWriter_EmptyResult(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "empty_report.html")

	w := NewWriter(nil, "")
	emptyHosts := []*model.HostResult{}
	emptyAlerts := []*model.Alert{}
	result := &model.InspectionResult{
		InspectionTime: time.Now(),
		Duration:       time.Second,
		Summary:        model.NewInspectionSummary(emptyHosts),
		AlertSummary:   model.NewAlertSummary(emptyAlerts),
		Hosts:          emptyHosts,
		Alerts:         emptyAlerts,
	}

	err := w.Write(result, outputPath)
	if err != nil {
		t.Fatalf("Write failed for empty result: %v", err)
	}

	// Verify file exists and contains basic structure
	content, _ := os.ReadFile(outputPath)
	if !strings.Contains(string(content), "巡检概览") {
		t.Error("empty result should still contain basic structure")
	}
}

func TestPrepareTemplateData(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	w := NewWriter(loc, "")
	result := createTestResultWithAlerts()

	data := w.prepareTemplateData(result)

	if data.Title != "系统巡检报告" {
		t.Errorf("expected title '系统巡检报告', got '%s'", data.Title)
	}

	if len(data.Hosts) != 2 {
		t.Errorf("expected 2 hosts, got %d", len(data.Hosts))
	}

	if len(data.Alerts) != 2 {
		t.Errorf("expected 2 alerts, got %d", len(data.Alerts))
	}

	// Check alerts are sorted (critical first)
	if data.Alerts[0].Level != "严重" {
		t.Error("alerts should be sorted with critical first")
	}
}

func TestCollectDiskPaths(t *testing.T) {
	w := NewWriter(nil, "")
	hosts := []*model.HostResult{
		{
			Metrics: map[string]*model.MetricValue{
				"disk_usage:/":     {Name: "disk_usage:/", FormattedValue: "50%"},
				"disk_usage:/home": {Name: "disk_usage:/home", FormattedValue: "30%"},
				"cpu_usage":        {Name: "cpu_usage", FormattedValue: "25%"},
			},
		},
		{
			Metrics: map[string]*model.MetricValue{
				"disk_usage:/":    {Name: "disk_usage:/", FormattedValue: "60%"},
				"disk_usage:/var": {Name: "disk_usage:/var", FormattedValue: "40%"},
			},
		},
	}

	paths := w.collectDiskPaths(hosts)

	if len(paths) != 3 {
		t.Errorf("expected 3 disk paths, got %d", len(paths))
	}

	// Check paths are sorted
	expected := []string{"/", "/home", "/var"}
	for i, path := range paths {
		if path != expected[i] {
			t.Errorf("expected path %s at index %d, got %s", expected[i], i, path)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{500 * time.Millisecond, "500ms"},
		{1500 * time.Millisecond, "1.5秒"},
		{90 * time.Second, "1.5分钟"},
		{2 * time.Hour, "2.0小时"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.duration)
		if result != tt.expected {
			t.Errorf("formatDuration(%v) = %s, expected %s", tt.duration, result, tt.expected)
		}
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.00 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{1024 * 1024 * 1024 * 1024, "1.00 TB"},
	}

	for _, tt := range tests {
		result := formatSize(tt.bytes)
		if result != tt.expected {
			t.Errorf("formatSize(%d) = %s, expected %s", tt.bytes, result, tt.expected)
		}
	}
}

func TestStatusText(t *testing.T) {
	tests := []struct {
		status   model.HostStatus
		expected string
	}{
		{model.HostStatusNormal, "正常"},
		{model.HostStatusWarning, "警告"},
		{model.HostStatusCritical, "严重"},
		{model.HostStatusFailed, "失败"},
		{model.HostStatus("unknown"), "未知"},
	}

	for _, tt := range tests {
		result := statusText(tt.status)
		if result != tt.expected {
			t.Errorf("statusText(%s) = %s, expected %s", tt.status, result, tt.expected)
		}
	}
}

func TestStatusClass(t *testing.T) {
	tests := []struct {
		status   model.HostStatus
		expected string
	}{
		{model.HostStatusNormal, "status-normal"},
		{model.HostStatusWarning, "status-warning"},
		{model.HostStatusCritical, "status-critical"},
		{model.HostStatusFailed, "status-failed"},
	}

	for _, tt := range tests {
		result := statusClass(tt.status)
		if result != tt.expected {
			t.Errorf("statusClass(%s) = %s, expected %s", tt.status, result, tt.expected)
		}
	}
}

func TestAlertLevelText(t *testing.T) {
	tests := []struct {
		level    model.AlertLevel
		expected string
	}{
		{model.AlertLevelNormal, "正常"},
		{model.AlertLevelWarning, "警告"},
		{model.AlertLevelCritical, "严重"},
	}

	for _, tt := range tests {
		result := alertLevelText(tt.level)
		if result != tt.expected {
			t.Errorf("alertLevelText(%s) = %s, expected %s", tt.level, result, tt.expected)
		}
	}
}

func TestAlertLevelClass(t *testing.T) {
	tests := []struct {
		level    model.AlertLevel
		expected string
	}{
		{model.AlertLevelNormal, "alert-normal"},
		{model.AlertLevelWarning, "alert-warning"},
		{model.AlertLevelCritical, "alert-critical"},
	}

	for _, tt := range tests {
		result := alertLevelClass(tt.level)
		if result != tt.expected {
			t.Errorf("alertLevelClass(%s) = %s, expected %s", tt.level, result, tt.expected)
		}
	}
}

func TestAlertLevelPriority(t *testing.T) {
	if alertLevelPriority(model.AlertLevelCritical) <= alertLevelPriority(model.AlertLevelWarning) {
		t.Error("critical should have higher priority than warning")
	}
	if alertLevelPriority(model.AlertLevelWarning) <= alertLevelPriority(model.AlertLevelNormal) {
		t.Error("warning should have higher priority than normal")
	}
}

func TestMetricStatusClass(t *testing.T) {
	tests := []struct {
		status   model.MetricStatus
		expected string
	}{
		{model.MetricStatusNormal, "metric-normal"},
		{model.MetricStatusWarning, "metric-warning"},
		{model.MetricStatusCritical, "metric-critical"},
		{model.MetricStatusPending, "metric-pending"},
	}

	for _, tt := range tests {
		result := metricStatusClass(tt.status)
		if result != tt.expected {
			t.Errorf("metricStatusClass(%s) = %s, expected %s", tt.status, result, tt.expected)
		}
	}
}

func TestFormatThreshold(t *testing.T) {
	tests := []struct {
		value      float64
		metricName string
		expected   string
	}{
		{70.0, "cpu_usage", "70.0%"},
		{90.0, "memory_usage", "90.0%"},
		{85.5, "disk_usage_max", "85.5%"},
		{0.75, "load_per_core", "0.75"},
		{5.0, "processes_zombies", "5"},
		{1.23, "other_metric", "1.23"},
	}

	for _, tt := range tests {
		result := formatThreshold(tt.value, tt.metricName)
		if result != tt.expected {
			t.Errorf("formatThreshold(%v, %s) = %s, expected %s", tt.value, tt.metricName, result, tt.expected)
		}
	}
}

// Helper functions

func createTestResult() *model.InspectionResult {
	return &model.InspectionResult{
		InspectionTime: time.Now(),
		Duration:       5 * time.Second,
		Summary: &model.InspectionSummary{
			TotalHosts:    2,
			NormalHosts:   1,
			WarningHosts:  1,
			CriticalHosts: 0,
			FailedHosts:   0,
		},
		AlertSummary: &model.AlertSummary{
			TotalAlerts:   1,
			WarningCount:  1,
			CriticalCount: 0,
		},
		Hosts: []*model.HostResult{
			{
				Hostname:      "test-host-1",
				IP:            "192.168.1.1",
				OS:            "Linux",
				OSVersion:     "CentOS 7.9",
				KernelVersion: "3.10.0",
				CPUCores:      4,
				Status:        model.HostStatusNormal,
				Metrics: map[string]*model.MetricValue{
					"cpu_usage":     {Name: "cpu_usage", FormattedValue: "25.5%", Status: model.MetricStatusNormal},
					"memory_usage":  {Name: "memory_usage", FormattedValue: "45.2%", Status: model.MetricStatusNormal},
					"disk_usage_max": {Name: "disk_usage_max", FormattedValue: "60.1%", Status: model.MetricStatusNormal},
				},
			},
			{
				Hostname: "test-host-2",
				IP:       "192.168.1.2",
				OS:       "Linux",
				Status:   model.HostStatusWarning,
				Metrics: map[string]*model.MetricValue{
					"cpu_usage":     {Name: "cpu_usage", FormattedValue: "75.0%", Status: model.MetricStatusWarning},
					"memory_usage":  {Name: "memory_usage", FormattedValue: "50.0%", Status: model.MetricStatusNormal},
					"disk_usage_max": {Name: "disk_usage_max", FormattedValue: "45.0%", Status: model.MetricStatusNormal},
				},
			},
		},
		Alerts: []*model.Alert{},
	}
}

func createTestResultWithAlerts() *model.InspectionResult {
	result := createTestResult()
	result.Alerts = []*model.Alert{
		{
			Hostname:          "test-host-1",
			MetricName:        "memory_usage",
			MetricDisplayName: "内存利用率",
			CurrentValue:      92.5,
			FormattedValue:    "92.5%",
			WarningThreshold:  70,
			CriticalThreshold: 90,
			Level:             model.AlertLevelCritical,
			Message:           "内存利用率达到严重阈值",
		},
		{
			Hostname:          "test-host-2",
			MetricName:        "cpu_usage",
			MetricDisplayName: "CPU利用率",
			CurrentValue:      75.0,
			FormattedValue:    "75.0%",
			WarningThreshold:  70,
			CriticalThreshold: 90,
			Level:             model.AlertLevelWarning,
			Message:           "CPU利用率达到警告阈值",
		},
	}
	result.AlertSummary.TotalAlerts = 2
	result.AlertSummary.WarningCount = 1
	result.AlertSummary.CriticalCount = 1
	return result
}
