package html

import (
	"fmt"
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
					"cpu_usage":      {Name: "cpu_usage", FormattedValue: "25.5%", Status: model.MetricStatusNormal},
					"memory_usage":   {Name: "memory_usage", FormattedValue: "45.2%", Status: model.MetricStatusNormal},
					"disk_usage_max": {Name: "disk_usage_max", FormattedValue: "60.1%", Status: model.MetricStatusNormal},
				},
			},
			{
				Hostname: "test-host-2",
				IP:       "192.168.1.2",
				OS:       "Linux",
				Status:   model.HostStatusWarning,
				Metrics: map[string]*model.MetricValue{
					"cpu_usage":      {Name: "cpu_usage", FormattedValue: "75.0%", Status: model.MetricStatusWarning},
					"memory_usage":   {Name: "memory_usage", FormattedValue: "50.0%", Status: model.MetricStatusNormal},
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

// ============================================================================
// MySQL HTML Report Tests
// ============================================================================

func TestWriter_WriteMySQLInspection_NilResult(t *testing.T) {
	w := NewWriter(nil, "")
	err := w.WriteMySQLInspection(nil, "test.html")
	if err == nil {
		t.Error("expected error for nil result")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("expected error message to mention nil, got: %s", err.Error())
	}
}

func TestWriter_WriteMySQLInspection_Success(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "mysql_report.html")

	// Create writer
	w := NewWriter(nil, "")

	// Create test result
	result := createTestMySQLInspectionResults()

	// Write report
	err := w.WriteMySQLInspection(result, outputPath)
	if err != nil {
		t.Fatalf("WriteMySQLInspection failed: %v", err)
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
		"MySQL 巡检报告",
		"172.18.182.91",
		"3306",
		"8.0.39",
		"MySQL 巡检概览",
		"MySQL 实例详情",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("expected content to contain '%s'", expected)
		}
	}
}

func TestWriter_WriteMySQLInspection_AddsHtmlExtension(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "mysql_report") // No extension

	w := NewWriter(nil, "")
	result := createTestMySQLInspectionResults()

	err := w.WriteMySQLInspection(result, outputPath)
	if err != nil {
		t.Fatalf("WriteMySQLInspection failed: %v", err)
	}

	// Verify .html extension was added
	expectedPath := outputPath + ".html"
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("file with .html extension was not created")
	}
}

func TestWriter_WriteMySQLInspection_WithAlerts(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "mysql_report_alerts.html")

	w := NewWriter(nil, "")
	result := createTestMySQLInspectionResultsWithAlerts()

	err := w.WriteMySQLInspection(result, outputPath)
	if err != nil {
		t.Fatalf("WriteMySQLInspection failed: %v", err)
	}

	// Read and verify content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	contentStr := string(content)
	expectedContent := []string{
		"MySQL 异常汇总",
		"连接使用率",
		"85.0%",
		"严重",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("expected content to contain '%s'", expected)
		}
	}
}

func TestWriter_WriteMySQLInspection_EmptyResult(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "mysql_empty_report.html")

	w := NewWriter(nil, "")
	result := &model.MySQLInspectionResults{
		InspectionTime: time.Now(),
		Duration:       time.Second,
		Summary: &model.MySQLInspectionSummary{
			TotalInstances:    0,
			NormalInstances:   0,
			WarningInstances:  0,
			CriticalInstances: 0,
			FailedInstances:   0,
		},
		AlertSummary: &model.MySQLAlertSummary{
			TotalAlerts:   0,
			WarningCount:  0,
			CriticalCount: 0,
		},
		Results: []*model.MySQLInspectionResult{},
		Alerts:  []*model.MySQLAlert{},
	}

	err := w.WriteMySQLInspection(result, outputPath)
	if err != nil {
		t.Fatalf("WriteMySQLInspection failed for empty result: %v", err)
	}

	// Verify file exists and contains basic structure
	content, _ := os.ReadFile(outputPath)
	if !strings.Contains(string(content), "MySQL 巡检概览") {
		t.Error("empty result should still contain basic structure")
	}
}

func TestPrepareMySQLTemplateData(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	w := NewWriter(loc, "")
	result := createTestMySQLInspectionResultsWithAlerts()

	data := w.prepareMySQLTemplateData(result)

	if data.Title != "MySQL 巡检报告" {
		t.Errorf("expected title 'MySQL 巡检报告', got '%s'", data.Title)
	}

	if len(data.Instances) != 2 {
		t.Errorf("expected 2 instances, got %d", len(data.Instances))
	}

	if len(data.Alerts) != 2 {
		t.Errorf("expected 2 alerts, got %d", len(data.Alerts))
	}

	// Check alerts are sorted (critical first)
	if data.Alerts[0].Level != "严重" {
		t.Error("alerts should be sorted with critical first")
	}
}

func TestConvertMySQLInstanceData(t *testing.T) {
	w := NewWriter(nil, "")
	instance := &model.MySQLInstance{
		Address:     "172.18.182.91:3306",
		IP:          "172.18.182.91",
		Port:        3306,
		Version:     "8.0.39",
		ServerID:    "1001",
		ClusterMode: model.ClusterModeMGR,
	}
	result := &model.MySQLInspectionResult{
		Instance:           instance,
		MaxConnections:     1000,
		CurrentConnections: 100,
		BinlogEnabled:      true,
		MGRStateOnline:     true,
		Status:             model.MySQLStatusNormal,
		Alerts:             []*model.MySQLAlert{},
	}

	data := w.convertMySQLInstanceData(result)

	if data.IP != "172.18.182.91" {
		t.Errorf("expected IP '172.18.182.91', got '%s'", data.IP)
	}
	if data.Port != 3306 {
		t.Errorf("expected port 3306, got %d", data.Port)
	}
	if data.Version != "8.0.39" {
		t.Errorf("expected version '8.0.39', got '%s'", data.Version)
	}
	if data.ClusterMode != "MGR" {
		t.Errorf("expected cluster mode 'MGR', got '%s'", data.ClusterMode)
	}
	if data.SyncStatus != "在线" {
		t.Errorf("expected sync status '在线', got '%s'", data.SyncStatus)
	}
	if data.BinlogEnabled != "启用" {
		t.Errorf("expected binlog '启用', got '%s'", data.BinlogEnabled)
	}
	if data.Status != "正常" {
		t.Errorf("expected status '正常', got '%s'", data.Status)
	}
	if data.StatusClass != "status-normal" {
		t.Errorf("expected status class 'status-normal', got '%s'", data.StatusClass)
	}
}

func TestConvertMySQLAlerts(t *testing.T) {
	w := NewWriter(nil, "")
	alerts := []*model.MySQLAlert{
		{
			Address:           "172.18.182.91:3306",
			MetricName:        "connection_usage",
			MetricDisplayName: "连接使用率",
			CurrentValue:      75.0,
			FormattedValue:    "75.0%",
			WarningThreshold:  70,
			CriticalThreshold: 90,
			Level:             model.AlertLevelWarning,
			Message:           "连接使用率达到警告阈值",
		},
		{
			Address:           "172.18.182.92:3306",
			MetricName:        "connection_usage",
			MetricDisplayName: "连接使用率",
			CurrentValue:      95.0,
			FormattedValue:    "95.0%",
			WarningThreshold:  70,
			CriticalThreshold: 90,
			Level:             model.AlertLevelCritical,
			Message:           "连接使用率达到严重阈值",
		},
	}

	converted := w.convertMySQLAlerts(alerts)

	// Should be sorted with critical first
	if len(converted) != 2 {
		t.Fatalf("expected 2 alerts, got %d", len(converted))
	}
	if converted[0].Level != "严重" {
		t.Error("critical alert should be first")
	}
	if converted[1].Level != "警告" {
		t.Error("warning alert should be second")
	}
}

func TestMySQLStatusText(t *testing.T) {
	tests := []struct {
		status   model.MySQLInstanceStatus
		expected string
	}{
		{model.MySQLStatusNormal, "正常"},
		{model.MySQLStatusWarning, "警告"},
		{model.MySQLStatusCritical, "严重"},
		{model.MySQLStatusFailed, "失败"},
		{model.MySQLInstanceStatus("unknown"), "未知"},
	}

	for _, tt := range tests {
		result := mysqlStatusText(tt.status)
		if result != tt.expected {
			t.Errorf("mysqlStatusText(%s) = %s, expected %s", tt.status, result, tt.expected)
		}
	}
}

func TestMySQLStatusClass(t *testing.T) {
	tests := []struct {
		status   model.MySQLInstanceStatus
		expected string
	}{
		{model.MySQLStatusNormal, "status-normal"},
		{model.MySQLStatusWarning, "status-warning"},
		{model.MySQLStatusCritical, "status-critical"},
		{model.MySQLStatusFailed, "status-failed"},
	}

	for _, tt := range tests {
		result := mysqlStatusClass(tt.status)
		if result != tt.expected {
			t.Errorf("mysqlStatusClass(%s) = %s, expected %s", tt.status, result, tt.expected)
		}
	}
}

func TestMySQLClusterModeText(t *testing.T) {
	tests := []struct {
		mode     model.MySQLClusterMode
		expected string
	}{
		{model.ClusterModeMGR, "MGR"},
		{model.ClusterModeDualMaster, "双主"},
		{model.ClusterModeMasterSlave, "主从"},
		{model.MySQLClusterMode("unknown"), "未知"},
	}

	for _, tt := range tests {
		result := mysqlClusterModeText(tt.mode)
		if result != tt.expected {
			t.Errorf("mysqlClusterModeText(%s) = %s, expected %s", tt.mode, result, tt.expected)
		}
	}
}

func TestGetMySQLSyncStatus(t *testing.T) {
	tests := []struct {
		name     string
		result   *model.MySQLInspectionResult
		expected string
	}{
		{
			name: "MGR online",
			result: &model.MySQLInspectionResult{
				Instance:       &model.MySQLInstance{ClusterMode: model.ClusterModeMGR},
				MGRStateOnline: true,
			},
			expected: "在线",
		},
		{
			name: "MGR offline",
			result: &model.MySQLInspectionResult{
				Instance:       &model.MySQLInstance{ClusterMode: model.ClusterModeMGR},
				MGRStateOnline: false,
			},
			expected: "离线",
		},
		{
			name: "Master-Slave sync OK",
			result: &model.MySQLInspectionResult{
				Instance:   &model.MySQLInstance{ClusterMode: model.ClusterModeMasterSlave},
				SyncStatus: true,
			},
			expected: "正常",
		},
		{
			name: "Master-Slave sync failed",
			result: &model.MySQLInspectionResult{
				Instance:   &model.MySQLInstance{ClusterMode: model.ClusterModeMasterSlave},
				SyncStatus: false,
			},
			expected: "异常",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getMySQLSyncStatus(tt.result)
			if result != tt.expected {
				t.Errorf("getMySQLSyncStatus() = %s, expected %s", result, tt.expected)
			}
		})
	}
}

func TestHTMLBoolToText(t *testing.T) {
	if boolToText(true) != "启用" {
		t.Error("boolToText(true) should return '启用'")
	}
	if boolToText(false) != "禁用" {
		t.Error("boolToText(false) should return '禁用'")
	}
}

func TestFormatMySQLThreshold(t *testing.T) {
	tests := []struct {
		value      float64
		metricName string
		expected   string
	}{
		{70.0, "connection_usage", "70.0%"},
		{3.0, "mgr_member_count", "3"},
		{1.0, "mgr_state_online", "在线"},
		{0.0, "mgr_state_online", "离线"},
		{1.23, "other_metric", "1.23"},
	}

	for _, tt := range tests {
		result := formatMySQLThreshold(tt.value, tt.metricName)
		if result != tt.expected {
			t.Errorf("formatMySQLThreshold(%v, %s) = %s, expected %s", tt.value, tt.metricName, result, tt.expected)
		}
	}
}

// MySQL test helper functions

func createTestMySQLInspectionResults() *model.MySQLInspectionResults {
	instance1 := &model.MySQLInstance{
		Address:     "172.18.182.91:3306",
		IP:          "172.18.182.91",
		Port:        3306,
		Version:     "8.0.39",
		ServerID:    "1001",
		ClusterMode: model.ClusterModeMGR,
	}
	instance2 := &model.MySQLInstance{
		Address:     "172.18.182.92:3306",
		IP:          "172.18.182.92",
		Port:        3306,
		Version:     "8.0.39",
		ServerID:    "1002",
		ClusterMode: model.ClusterModeMGR,
	}

	return &model.MySQLInspectionResults{
		InspectionTime: time.Now(),
		Duration:       5 * time.Second,
		Summary: &model.MySQLInspectionSummary{
			TotalInstances:    2,
			NormalInstances:   2,
			WarningInstances:  0,
			CriticalInstances: 0,
			FailedInstances:   0,
		},
		AlertSummary: &model.MySQLAlertSummary{
			TotalAlerts:   0,
			WarningCount:  0,
			CriticalCount: 0,
		},
		Results: []*model.MySQLInspectionResult{
			{
				Instance:           instance1,
				ConnectionStatus:   true,
				MaxConnections:     1000,
				CurrentConnections: 100,
				BinlogEnabled:      true,
				MGRMemberCount:     3,
				MGRStateOnline:     true,
				Status:             model.MySQLStatusNormal,
				Alerts:             []*model.MySQLAlert{},
			},
			{
				Instance:           instance2,
				ConnectionStatus:   true,
				MaxConnections:     1000,
				CurrentConnections: 150,
				BinlogEnabled:      true,
				MGRMemberCount:     3,
				MGRStateOnline:     true,
				Status:             model.MySQLStatusNormal,
				Alerts:             []*model.MySQLAlert{},
			},
		},
		Alerts:  []*model.MySQLAlert{},
		Version: "1.0.0",
	}
}

func createTestMySQLInspectionResultsWithAlerts() *model.MySQLInspectionResults {
	result := createTestMySQLInspectionResults()

	// Add alerts
	alert1 := &model.MySQLAlert{
		Address:           "172.18.182.91:3306",
		MetricName:        "connection_usage",
		MetricDisplayName: "连接使用率",
		CurrentValue:      85.0,
		FormattedValue:    "85.0%",
		WarningThreshold:  70,
		CriticalThreshold: 90,
		Level:             model.AlertLevelWarning,
		Message:           "连接使用率达到警告阈值",
	}
	alert2 := &model.MySQLAlert{
		Address:           "172.18.182.92:3306",
		MetricName:        "connection_usage",
		MetricDisplayName: "连接使用率",
		CurrentValue:      95.0,
		FormattedValue:    "95.0%",
		WarningThreshold:  70,
		CriticalThreshold: 90,
		Level:             model.AlertLevelCritical,
		Message:           "连接使用率达到严重阈值",
	}

	result.Alerts = []*model.MySQLAlert{alert1, alert2}
	result.Results[0].Alerts = []*model.MySQLAlert{alert1}
	result.Results[0].Status = model.MySQLStatusWarning
	result.Results[1].Alerts = []*model.MySQLAlert{alert2}
	result.Results[1].Status = model.MySQLStatusCritical

	result.Summary.NormalInstances = 0
	result.Summary.WarningInstances = 1
	result.Summary.CriticalInstances = 1
	result.AlertSummary.TotalAlerts = 2
	result.AlertSummary.WarningCount = 1
	result.AlertSummary.CriticalCount = 1

	return result
}

// ============================================================================
// Redis HTML Report Tests
// ============================================================================

func TestWriter_WriteRedisInspection_NilResult(t *testing.T) {
	w := NewWriter(nil, "")
	err := w.WriteRedisInspection(nil, "test.html")
	if err == nil {
		t.Error("expected error for nil result")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("expected error message to mention nil, got: %s", err.Error())
	}
}

func TestWriter_WriteRedisInspection_Success(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "redis_report.html")

	// Create writer
	w := NewWriter(nil, "")

	// Create test result
	result := createTestRedisInspectionResults()

	// Write report
	err := w.WriteRedisInspection(result, outputPath)
	if err != nil {
		t.Fatalf("WriteRedisInspection failed: %v", err)
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
		"Redis 巡检报告",
		"192.18.102.2",
		"7000",
		"Redis 巡检概览",
		"Redis 实例详情",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("expected content to contain '%s'", expected)
		}
	}
}

func TestWriter_WriteRedisInspection_AddsHtmlExtension(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "redis_report") // No extension

	w := NewWriter(nil, "")
	result := createTestRedisInspectionResults()

	err := w.WriteRedisInspection(result, outputPath)
	if err != nil {
		t.Fatalf("WriteRedisInspection failed: %v", err)
	}

	// Verify .html extension was added
	expectedPath := outputPath + ".html"
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("file with .html extension was not created")
	}
}

func TestWriter_WriteRedisInspection_WithAlerts(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "redis_report_alerts.html")

	w := NewWriter(nil, "")
	result := createTestRedisInspectionResultsWithAlerts()

	err := w.WriteRedisInspection(result, outputPath)
	if err != nil {
		t.Fatalf("WriteRedisInspection failed: %v", err)
	}

	// Read and verify content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	contentStr := string(content)
	expectedContent := []string{
		"Redis 异常汇总",
		"连接使用率",
		"85.0%",
		"严重",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("expected content to contain '%s'", expected)
		}
	}
}

func TestWriter_WriteRedisInspection_EmptyResult(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "redis_empty_report.html")

	w := NewWriter(nil, "")
	result := &model.RedisInspectionResults{
		InspectionTime: time.Now(),
		Duration:       time.Second,
		Summary: &model.RedisInspectionSummary{
			TotalInstances:    0,
			NormalInstances:   0,
			WarningInstances:  0,
			CriticalInstances: 0,
			FailedInstances:   0,
		},
		AlertSummary: &model.RedisAlertSummary{
			TotalAlerts:   0,
			WarningCount:  0,
			CriticalCount: 0,
		},
		Results: []*model.RedisInspectionResult{},
		Alerts:  []*model.RedisAlert{},
	}

	err := w.WriteRedisInspection(result, outputPath)
	if err != nil {
		t.Fatalf("WriteRedisInspection failed for empty result: %v", err)
	}

	// Verify file exists and contains basic structure
	content, _ := os.ReadFile(outputPath)
	if !strings.Contains(string(content), "Redis 巡检概览") {
		t.Error("empty result should still contain basic structure")
	}
}

func TestPrepareRedisTemplateData(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	w := NewWriter(loc, "")
	result := createTestRedisInspectionResultsWithAlerts()

	data := w.prepareRedisTemplateData(result)

	if data.Title != "Redis 巡检报告" {
		t.Errorf("expected title 'Redis 巡检报告', got '%s'", data.Title)
	}

	if len(data.Instances) != 2 {
		t.Errorf("expected 2 instances, got %d", len(data.Instances))
	}

	if len(data.Alerts) != 2 {
		t.Errorf("expected 2 alerts, got %d", len(data.Alerts))
	}

	// Check alerts are sorted (critical first)
	if data.Alerts[0].Level != "严重" {
		t.Error("alerts should be sorted with critical first")
	}
}

func TestConvertRedisInstanceData_Master(t *testing.T) {
	w := NewWriter(nil, "")
	instance := &model.RedisInstance{
		Address: "192.18.102.2:7000",
		IP:      "192.18.102.2",
		Port:    7000,
		Version: "",
		Role:    model.RedisRoleMaster,
	}
	result := &model.RedisInspectionResult{
		Instance:         instance,
		ConnectionStatus: true,
		ClusterEnabled:   true,
		MaxClients:       10000,
		ConnectedClients: 100,
		ConnectedSlaves:  2,
		MasterLinkStatus: true,
		MasterPort:       0,
		ReplicationLag:   0,
		Status:           model.RedisStatusNormal,
		Alerts:           []*model.RedisAlert{},
	}

	data := w.convertRedisInstanceData(result)

	if data.IP != "192.18.102.2" {
		t.Errorf("expected IP '192.18.102.2', got '%s'", data.IP)
	}
	if data.Port != 7000 {
		t.Errorf("expected port 7000, got %d", data.Port)
	}
	if data.Version != "N/A" {
		t.Errorf("expected version 'N/A', got '%s'", data.Version)
	}
	if data.Role != "主" {
		t.Errorf("expected role '主', got '%s'", data.Role)
	}
	if data.ClusterEnabled != "启用" {
		t.Errorf("expected cluster enabled '启用', got '%s'", data.ClusterEnabled)
	}
	if data.ConnectionStatus != "正常" {
		t.Errorf("expected connection status '正常', got '%s'", data.ConnectionStatus)
	}
	// Master node should show N/A for slave-specific fields
	if data.MasterLinkStatus != "N/A" {
		t.Errorf("expected master link status 'N/A' for master node, got '%s'", data.MasterLinkStatus)
	}
	if data.MasterPort != "N/A" {
		t.Errorf("expected master port 'N/A' for master node, got '%s'", data.MasterPort)
	}
	if data.ReplicationLag != "N/A" {
		t.Errorf("expected replication lag 'N/A' for master node, got '%s'", data.ReplicationLag)
	}
	if data.Status != "正常" {
		t.Errorf("expected status '正常', got '%s'", data.Status)
	}
	if data.StatusClass != "status-normal" {
		t.Errorf("expected status class 'status-normal', got '%s'", data.StatusClass)
	}
}

func TestConvertRedisInstanceData_Slave(t *testing.T) {
	w := NewWriter(nil, "")
	instance := &model.RedisInstance{
		Address: "192.18.102.2:7001",
		IP:      "192.18.102.2",
		Port:    7001,
		Version: "",
		Role:    model.RedisRoleSlave,
	}
	result := &model.RedisInspectionResult{
		Instance:         instance,
		ConnectionStatus: true,
		ClusterEnabled:   true,
		MaxClients:       10000,
		ConnectedClients: 50,
		ConnectedSlaves:  0,
		MasterLinkStatus: true,
		MasterPort:       7000,
		ReplicationLag:   1024,
		Status:           model.RedisStatusNormal,
		Alerts:           []*model.RedisAlert{},
	}

	data := w.convertRedisInstanceData(result)

	if data.Role != "从" {
		t.Errorf("expected role '从', got '%s'", data.Role)
	}
	// Slave node should show actual values for slave-specific fields
	if data.MasterLinkStatus != "正常" {
		t.Errorf("expected master link status '正常', got '%s'", data.MasterLinkStatus)
	}
	if data.MasterPort != "7000" {
		t.Errorf("expected master port '7000', got '%s'", data.MasterPort)
	}
	if data.ReplicationLag != "1.00 KB" {
		t.Errorf("expected replication lag '1.00 KB', got '%s'", data.ReplicationLag)
	}
}

func TestConvertRedisAlerts(t *testing.T) {
	w := NewWriter(nil, "")
	alerts := []*model.RedisAlert{
		{
			Address:           "192.18.102.2:7000",
			MetricName:        "connection_usage",
			MetricDisplayName: "连接使用率",
			CurrentValue:      75.0,
			FormattedValue:    "75.0%",
			WarningThreshold:  70,
			CriticalThreshold: 90,
			Level:             model.AlertLevelWarning,
			Message:           "连接使用率达到警告阈值",
		},
		{
			Address:           "192.18.102.3:7000",
			MetricName:        "connection_usage",
			MetricDisplayName: "连接使用率",
			CurrentValue:      95.0,
			FormattedValue:    "95.0%",
			WarningThreshold:  70,
			CriticalThreshold: 90,
			Level:             model.AlertLevelCritical,
			Message:           "连接使用率达到严重阈值",
		},
	}

	converted := w.convertRedisAlerts(alerts)

	// Should be sorted with critical first
	if len(converted) != 2 {
		t.Fatalf("expected 2 alerts, got %d", len(converted))
	}
	if converted[0].Level != "严重" {
		t.Error("critical alert should be first")
	}
	if converted[1].Level != "警告" {
		t.Error("warning alert should be second")
	}
}

func TestRedisStatusText(t *testing.T) {
	tests := []struct {
		status   model.RedisInstanceStatus
		expected string
	}{
		{model.RedisStatusNormal, "正常"},
		{model.RedisStatusWarning, "警告"},
		{model.RedisStatusCritical, "严重"},
		{model.RedisStatusFailed, "失败"},
		{model.RedisInstanceStatus("unknown"), "未知"},
	}

	for _, tt := range tests {
		result := redisStatusText(tt.status)
		if result != tt.expected {
			t.Errorf("redisStatusText(%s) = %s, expected %s", tt.status, result, tt.expected)
		}
	}
}

func TestRedisStatusClass(t *testing.T) {
	tests := []struct {
		status   model.RedisInstanceStatus
		expected string
	}{
		{model.RedisStatusNormal, "status-normal"},
		{model.RedisStatusWarning, "status-warning"},
		{model.RedisStatusCritical, "status-critical"},
		{model.RedisStatusFailed, "status-failed"},
	}

	for _, tt := range tests {
		result := redisStatusClass(tt.status)
		if result != tt.expected {
			t.Errorf("redisStatusClass(%s) = %s, expected %s", tt.status, result, tt.expected)
		}
	}
}

func TestRedisRoleText(t *testing.T) {
	tests := []struct {
		role     model.RedisRole
		expected string
	}{
		{model.RedisRoleMaster, "主"},
		{model.RedisRoleSlave, "从"},
		{model.RedisRoleUnknown, "未知"},
		{model.RedisRole("other"), "未知"},
	}

	for _, tt := range tests {
		result := redisRoleText(tt.role)
		if result != tt.expected {
			t.Errorf("redisRoleText(%s) = %s, expected %s", tt.role, result, tt.expected)
		}
	}
}

func TestGetRedisLinkStatus(t *testing.T) {
	tests := []struct {
		name     string
		result   *model.RedisInspectionResult
		expected string
	}{
		{
			name: "Master node returns N/A",
			result: &model.RedisInspectionResult{
				Instance:         &model.RedisInstance{Role: model.RedisRoleMaster},
				MasterLinkStatus: true,
			},
			expected: "N/A",
		},
		{
			name: "Slave link up",
			result: &model.RedisInspectionResult{
				Instance:         &model.RedisInstance{Role: model.RedisRoleSlave},
				MasterLinkStatus: true,
			},
			expected: "正常",
		},
		{
			name: "Slave link down",
			result: &model.RedisInspectionResult{
				Instance:         &model.RedisInstance{Role: model.RedisRoleSlave},
				MasterLinkStatus: false,
			},
			expected: "异常",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getRedisLinkStatus(tt.result)
			if result != tt.expected {
				t.Errorf("getRedisLinkStatus() = %s, expected %s", result, tt.expected)
			}
		})
	}
}

func TestGetRedisReplicationLag(t *testing.T) {
	tests := []struct {
		name     string
		result   *model.RedisInspectionResult
		expected string
	}{
		{
			name: "Master node returns N/A",
			result: &model.RedisInspectionResult{
				Instance:       &model.RedisInstance{Role: model.RedisRoleMaster},
				ReplicationLag: 1024,
			},
			expected: "N/A",
		},
		{
			name: "Slave zero lag",
			result: &model.RedisInspectionResult{
				Instance:       &model.RedisInstance{Role: model.RedisRoleSlave},
				ReplicationLag: 0,
			},
			expected: "0 B",
		},
		{
			name: "Slave with lag",
			result: &model.RedisInspectionResult{
				Instance:       &model.RedisInstance{Role: model.RedisRoleSlave},
				ReplicationLag: 1024 * 1024,
			},
			expected: "1.00 MB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getRedisReplicationLag(tt.result)
			if result != tt.expected {
				t.Errorf("getRedisReplicationLag() = %s, expected %s", result, tt.expected)
			}
		})
	}
}

func TestGetRedisMasterPort(t *testing.T) {
	tests := []struct {
		name     string
		result   *model.RedisInspectionResult
		expected string
	}{
		{
			name: "Master node returns N/A",
			result: &model.RedisInspectionResult{
				Instance:   &model.RedisInstance{Role: model.RedisRoleMaster},
				MasterPort: 7000,
			},
			expected: "N/A",
		},
		{
			name: "Slave with zero master port",
			result: &model.RedisInspectionResult{
				Instance:   &model.RedisInstance{Role: model.RedisRoleSlave},
				MasterPort: 0,
			},
			expected: "N/A",
		},
		{
			name: "Slave with valid master port",
			result: &model.RedisInspectionResult{
				Instance:   &model.RedisInstance{Role: model.RedisRoleSlave},
				MasterPort: 7000,
			},
			expected: "7000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getRedisMasterPort(tt.result)
			if result != tt.expected {
				t.Errorf("getRedisMasterPort() = %s, expected %s", result, tt.expected)
			}
		})
	}
}

func TestFormatRedisConnectionUsage(t *testing.T) {
	tests := []struct {
		name     string
		result   *model.RedisInspectionResult
		expected string
	}{
		{
			name: "Zero max clients returns N/A",
			result: &model.RedisInspectionResult{
				MaxClients:       0,
				ConnectedClients: 100,
			},
			expected: "N/A",
		},
		{
			name: "Normal usage",
			result: &model.RedisInspectionResult{
				MaxClients:       10000,
				ConnectedClients: 1000,
			},
			expected: "10.0%",
		},
		{
			name: "High usage",
			result: &model.RedisInspectionResult{
				MaxClients:       1000,
				ConnectedClients: 900,
			},
			expected: "90.0%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRedisConnectionUsage(tt.result)
			if result != tt.expected {
				t.Errorf("formatRedisConnectionUsage() = %s, expected %s", result, tt.expected)
			}
		})
	}
}

func TestFormatRedisThreshold(t *testing.T) {
	tests := []struct {
		value      float64
		metricName string
		expected   string
	}{
		{70.0, "connection_usage", "70.0%"},
		{1048576.0, "replication_lag", "1.00 MB"},
		{1.0, "master_link_status", "正常"},
		{0.0, "master_link_status", "异常"},
		{1.23, "other_metric", "1.23"},
	}

	for _, tt := range tests {
		result := formatRedisThreshold(tt.value, tt.metricName)
		if result != tt.expected {
			t.Errorf("formatRedisThreshold(%v, %s) = %s, expected %s", tt.value, tt.metricName, result, tt.expected)
		}
	}
}

func TestWriter_WriteCombined_WithRedis(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "combined_with_redis.html")

	w := NewWriter(nil, "")
	hostResult := createTestResult()
	mysqlResult := createTestMySQLInspectionResults()
	redisResult := createTestRedisInspectionResults()

	err := w.WriteCombined(hostResult, mysqlResult, redisResult, outputPath)
	if err != nil {
		t.Fatalf("WriteCombined with Redis failed: %v", err)
	}

	// Read and verify content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	contentStr := string(content)
	expectedContent := []string{
		"系统巡检报告",
		"主机巡检",
		"MySQL 数据库巡检",
		"Redis 数据库巡检",
		"Redis 巡检概览",
		"Redis 实例详情",
		"192.18.102.2",
		"7000",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("expected content to contain '%s'", expected)
		}
	}
}

func TestWriter_WriteCombined_OnlyRedis(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "only_redis.html")

	w := NewWriter(nil, "")
	redisResult := createTestRedisInspectionResults()

	err := w.WriteCombined(nil, nil, redisResult, outputPath)
	if err != nil {
		t.Fatalf("WriteCombined with only Redis failed: %v", err)
	}

	// Read and verify content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	contentStr := string(content)

	// Should contain Redis section
	if !strings.Contains(contentStr, "Redis 数据库巡检") {
		t.Error("expected Redis section in report")
	}

	// Should NOT contain Host or MySQL sections
	if strings.Contains(contentStr, "主机巡检") {
		t.Error("should not contain Host section when only Redis is provided")
	}
	if strings.Contains(contentStr, "MySQL 数据库巡检") {
		t.Error("should not contain MySQL section when only Redis is provided")
	}
}

// Redis test helper functions

func createTestRedisInspectionResults() *model.RedisInspectionResults {
	instance1 := &model.RedisInstance{
		Address: "192.18.102.2:7000",
		IP:      "192.18.102.2",
		Port:    7000,
		Version: "",
		Role:    model.RedisRoleMaster,
	}
	instance2 := &model.RedisInstance{
		Address: "192.18.102.2:7001",
		IP:      "192.18.102.2",
		Port:    7001,
		Version: "",
		Role:    model.RedisRoleSlave,
	}

	return &model.RedisInspectionResults{
		InspectionTime: time.Now(),
		Duration:       5 * time.Second,
		Summary: &model.RedisInspectionSummary{
			TotalInstances:    2,
			NormalInstances:   2,
			WarningInstances:  0,
			CriticalInstances: 0,
			FailedInstances:   0,
		},
		AlertSummary: &model.RedisAlertSummary{
			TotalAlerts:   0,
			WarningCount:  0,
			CriticalCount: 0,
		},
		Results: []*model.RedisInspectionResult{
			{
				Instance:         instance1,
				ConnectionStatus: true,
				ClusterEnabled:   true,
				MaxClients:       10000,
				ConnectedClients: 100,
				ConnectedSlaves:  2,
				MasterLinkStatus: true,
				MasterPort:       0,
				ReplicationLag:   0,
				Status:           model.RedisStatusNormal,
				Alerts:           []*model.RedisAlert{},
			},
			{
				Instance:         instance2,
				ConnectionStatus: true,
				ClusterEnabled:   true,
				MaxClients:       10000,
				ConnectedClients: 50,
				ConnectedSlaves:  0,
				MasterLinkStatus: true,
				MasterPort:       7000,
				ReplicationLag:   0,
				Status:           model.RedisStatusNormal,
				Alerts:           []*model.RedisAlert{},
			},
		},
		Alerts:  []*model.RedisAlert{},
		Version: "1.0.0",
	}
}

func createTestRedisInspectionResultsWithAlerts() *model.RedisInspectionResults {
	result := createTestRedisInspectionResults()

	// Add alerts
	alert1 := &model.RedisAlert{
		Address:           "192.18.102.2:7000",
		MetricName:        "connection_usage",
		MetricDisplayName: "连接使用率",
		CurrentValue:      85.0,
		FormattedValue:    "85.0%",
		WarningThreshold:  70,
		CriticalThreshold: 90,
		Level:             model.AlertLevelWarning,
		Message:           "连接使用率达到警告阈值",
	}
	alert2 := &model.RedisAlert{
		Address:           "192.18.102.3:7000",
		MetricName:        "connection_usage",
		MetricDisplayName: "连接使用率",
		CurrentValue:      95.0,
		FormattedValue:    "95.0%",
		WarningThreshold:  70,
		CriticalThreshold: 90,
		Level:             model.AlertLevelCritical,
		Message:           "连接使用率达到严重阈值",
	}

	result.Alerts = []*model.RedisAlert{alert1, alert2}
	result.Results[0].Alerts = []*model.RedisAlert{alert1}
	result.Results[0].Status = model.RedisStatusWarning

	result.Summary.NormalInstances = 1
	result.Summary.WarningInstances = 1
	result.Summary.CriticalInstances = 0
	result.AlertSummary.TotalAlerts = 2
	result.AlertSummary.WarningCount = 1
	result.AlertSummary.CriticalCount = 1

	return result
}

// ============================================================================
// Redis Multi-Cluster Tests (陕西项目场景)
// ============================================================================

func TestWriter_WriteCombined_MultipleRedisClusters(t *testing.T) {
	w := NewWriter(time.UTC, "")

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "multi_cluster_report.html")

	// Create multi-cluster results
	redisResult := createTestRedisMultiClusterResults()

	err := w.WriteCombined(nil, nil, redisResult, outputPath)
	if err != nil {
		t.Fatalf("WriteCombined failed: %v", err)
	}

	// Read generated file
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	contentStr := string(content)

	// Should contain both cluster sections
	if !strings.Contains(contentStr, "Redis 集群 - 192.18.102") {
		t.Error("expected cluster 192.18.102 section in report")
	}
	if !strings.Contains(contentStr, "Redis 集群 - 192.18.107") {
		t.Error("expected cluster 192.18.107 section in report")
	}

	// Should have cluster-specific tables (check for redis-cluster-table-0, redis-cluster-table-1)
	if !strings.Contains(contentStr, "redis-cluster-table-0") {
		t.Error("expected first cluster table ID in report")
	}
	if !strings.Contains(contentStr, "redis-cluster-table-1") {
		t.Error("expected second cluster table ID in report")
	}
}

func TestWriter_WriteCombined_SingleClusterNoMultiClusterSections(t *testing.T) {
	w := NewWriter(time.UTC, "")

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "single_cluster_report.html")

	// Create single-cluster results (all same network segment)
	redisResult := createTestRedisInspectionResults()

	err := w.WriteCombined(nil, nil, redisResult, outputPath)
	if err != nil {
		t.Fatalf("WriteCombined failed: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	contentStr := string(content)

	// Should have single Redis table (not cluster-specific)
	if !strings.Contains(contentStr, "redis-table") {
		t.Error("expected single redis-table ID for single cluster")
	}

	// Should NOT have cluster-specific table IDs
	if strings.Contains(contentStr, "redis-cluster-table-") {
		t.Error("should not have cluster-specific table IDs for single cluster")
	}

	// Should NOT have cluster section headers
	if strings.Contains(contentStr, "Redis 集群 - ") {
		t.Error("should not have cluster section headers for single cluster")
	}
}

func TestWriter_ConvertRedisClusterData(t *testing.T) {
	w := NewWriter(time.UTC, "")

	cluster := &model.RedisCluster{
		ID:   "192.18.102",
		Name: "Redis 集群 - 192.18.102",
		Summary: &model.RedisInspectionSummary{
			TotalInstances:    6,
			NormalInstances:   5,
			WarningInstances:  1,
			CriticalInstances: 0,
			FailedInstances:   0,
		},
		AlertSummary: &model.RedisAlertSummary{
			TotalAlerts:   1,
			WarningCount:  1,
			CriticalCount: 0,
		},
		Instances: []*model.RedisInspectionResult{
			{
				Instance: &model.RedisInstance{
					Address: "192.18.102.2:7000",
					IP:      "192.18.102.2",
					Port:    7000,
					Role:    model.RedisRoleMaster,
				},
				Status: model.RedisStatusNormal,
			},
		},
		Alerts: []*model.RedisAlert{},
	}

	clusterData := w.convertRedisClusterData(cluster)

	if clusterData == nil {
		t.Fatal("convertRedisClusterData returned nil")
	}
	if clusterData.ID != "192.18.102" {
		t.Errorf("expected ID '192.18.102', got %q", clusterData.ID)
	}
	if clusterData.Name != "Redis 集群 - 192.18.102" {
		t.Errorf("expected Name 'Redis 集群 - 192.18.102', got %q", clusterData.Name)
	}
	if len(clusterData.Instances) != 1 {
		t.Errorf("expected 1 instance, got %d", len(clusterData.Instances))
	}
	if clusterData.Summary.TotalInstances != 6 {
		t.Errorf("expected TotalInstances 6, got %d", clusterData.Summary.TotalInstances)
	}
}

func TestWriter_ConvertRedisClusterData_Nil(t *testing.T) {
	w := NewWriter(time.UTC, "")

	clusterData := w.convertRedisClusterData(nil)

	if clusterData != nil {
		t.Error("expected nil for nil input")
	}
}

// createTestRedisMultiClusterResults creates test data with 2 clusters (陕西项目场景)
func createTestRedisMultiClusterResults() *model.RedisInspectionResults {
	results := &model.RedisInspectionResults{
		InspectionTime: time.Now(),
		Duration:       5 * time.Second,
		Results:        make([]*model.RedisInspectionResult, 0, 12),
	}

	// Cluster 1: 192.18.102.x - 3 masters, 3 slaves
	for i := 2; i <= 4; i++ {
		ip := fmt.Sprintf("192.18.102.%d", i)
		// Master
		results.Results = append(results.Results, &model.RedisInspectionResult{
			Instance: &model.RedisInstance{
				Address: fmt.Sprintf("%s:7000", ip),
				IP:      ip,
				Port:    7000,
				Role:    model.RedisRoleMaster,
			},
			ConnectionStatus: true,
			ClusterEnabled:   true,
			MaxClients:       10000,
			ConnectedClients: 100,
			ConnectedSlaves:  1,
			Status:           model.RedisStatusNormal,
		})
		// Slave
		results.Results = append(results.Results, &model.RedisInspectionResult{
			Instance: &model.RedisInstance{
				Address: fmt.Sprintf("%s:7001", ip),
				IP:      ip,
				Port:    7001,
				Role:    model.RedisRoleSlave,
			},
			ConnectionStatus: true,
			ClusterEnabled:   true,
			MasterLinkStatus: true,
			MasterPort:       7000,
			MaxClients:       10000,
			ConnectedClients: 50,
			Status:           model.RedisStatusNormal,
		})
	}

	// Cluster 2: 192.18.107.x - 3 masters, 3 slaves
	for i := 5; i <= 7; i++ {
		ip := fmt.Sprintf("192.18.107.%d", i)
		// Master
		results.Results = append(results.Results, &model.RedisInspectionResult{
			Instance: &model.RedisInstance{
				Address: fmt.Sprintf("%s:7000", ip),
				IP:      ip,
				Port:    7000,
				Role:    model.RedisRoleMaster,
			},
			ConnectionStatus: true,
			ClusterEnabled:   true,
			MaxClients:       10000,
			ConnectedClients: 200,
			ConnectedSlaves:  1,
			Status:           model.RedisStatusNormal,
		})
		// Slave
		results.Results = append(results.Results, &model.RedisInspectionResult{
			Instance: &model.RedisInstance{
				Address: fmt.Sprintf("%s:7001", ip),
				IP:      ip,
				Port:    7001,
				Role:    model.RedisRoleSlave,
			},
			ConnectionStatus: true,
			ClusterEnabled:   true,
			MasterLinkStatus: true,
			MasterPort:       7000,
			MaxClients:       10000,
			ConnectedClients: 100,
			Status:           model.RedisStatusNormal,
		})
	}

	// Call GroupByClusters to populate Clusters field
	results.GroupByClusters()

	results.Summary = &model.RedisInspectionSummary{
		TotalInstances:    12,
		NormalInstances:   12,
		WarningInstances:  0,
		CriticalInstances: 0,
		FailedInstances:   0,
	}
	results.AlertSummary = &model.RedisAlertSummary{
		TotalAlerts:   0,
		WarningCount:  0,
		CriticalCount: 0,
	}
	results.Version = "1.0.0-test"

	return results
}
