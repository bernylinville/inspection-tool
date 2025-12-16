package excel

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"inspection-tool/internal/model"
)

func TestNewWriter(t *testing.T) {
	tests := []struct {
		name     string
		timezone *time.Location
		wantTZ   string
	}{
		{
			name:     "nil timezone defaults to Asia/Shanghai",
			timezone: nil,
			wantTZ:   "Asia/Shanghai",
		},
		{
			name:     "custom timezone",
			timezone: time.UTC,
			wantTZ:   "UTC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := NewWriter(tt.timezone)
			if w == nil {
				t.Fatal("NewWriter returned nil")
			}
			if w.timezone.String() != tt.wantTZ {
				t.Errorf("timezone = %v, want %v", w.timezone.String(), tt.wantTZ)
			}
		})
	}
}

func TestWriter_Format(t *testing.T) {
	w := NewWriter(nil)
	if got := w.Format(); got != "excel" {
		t.Errorf("Format() = %v, want %v", got, "excel")
	}
}

func TestWriter_Write_NilResult(t *testing.T) {
	w := NewWriter(nil)
	err := w.Write(nil, "test.xlsx")
	if err == nil {
		t.Error("Write() with nil result should return error")
	}
}

func TestWriter_Write_Success(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_report.xlsx")

	// Create test data
	result := createTestInspectionResult()

	// Write report
	w := NewWriter(nil)
	err := w.Write(result, outputPath)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}

	// Open and verify Excel file
	f, err := excelize.OpenFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to open Excel file: %v", err)
	}
	defer f.Close()

	// Verify sheets exist
	sheets := f.GetSheetList()
	expectedSheets := []string{sheetSummary, sheetDetail, sheetAlerts}
	for _, expected := range expectedSheets {
		found := false
		for _, s := range sheets {
			if s == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Sheet %q not found in Excel file", expected)
		}
	}

	// Verify default Sheet1 was removed
	for _, s := range sheets {
		if s == "Sheet1" {
			t.Error("Default Sheet1 should have been removed")
		}
	}
}

func TestWriter_Write_AddsXlsxExtension(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_report") // No extension

	result := createTestInspectionResult()
	w := NewWriter(nil)
	err := w.Write(result, outputPath)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Verify file with .xlsx extension exists
	expectedPath := outputPath + ".xlsx"
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("Output file should have .xlsx extension added")
	}
}

func TestWriter_SummarySheet(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_report.xlsx")

	result := createTestInspectionResult()
	w := NewWriter(nil)
	err := w.Write(result, outputPath)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	f, err := excelize.OpenFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to open Excel file: %v", err)
	}
	defer f.Close()

	// Verify summary content
	title, _ := f.GetCellValue(sheetSummary, "A1")
	if title != "系统巡检报告" {
		t.Errorf("Title = %q, want %q", title, "系统巡检报告")
	}

	// Verify host count
	totalHostsLabel, _ := f.GetCellValue(sheetSummary, "A5")
	if totalHostsLabel != "主机总数" {
		t.Errorf("Label = %q, want %q", totalHostsLabel, "主机总数")
	}
	totalHostsValue, _ := f.GetCellValue(sheetSummary, "B5")
	if totalHostsValue != "3" {
		t.Errorf("Total hosts = %q, want %q", totalHostsValue, "3")
	}
}

func TestWriter_DetailSheet(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_report.xlsx")

	result := createTestInspectionResult()
	w := NewWriter(nil)
	err := w.Write(result, outputPath)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	f, err := excelize.OpenFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to open Excel file: %v", err)
	}
	defer f.Close()

	// Verify header row
	hostname, _ := f.GetCellValue(sheetDetail, "A1")
	if hostname != "主机名" {
		t.Errorf("Header A1 = %q, want %q", hostname, "主机名")
	}

	// Verify first host data
	host1, _ := f.GetCellValue(sheetDetail, "A2")
	if host1 != "host-1" {
		t.Errorf("Host 1 = %q, want %q", host1, "host-1")
	}

	status1, _ := f.GetCellValue(sheetDetail, "C2")
	if status1 != "正常" {
		t.Errorf("Status 1 = %q, want %q", status1, "正常")
	}
}

func TestWriter_AlertsSheet(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_report.xlsx")

	result := createTestInspectionResult()
	w := NewWriter(nil)
	err := w.Write(result, outputPath)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	f, err := excelize.OpenFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to open Excel file: %v", err)
	}
	defer f.Close()

	// Verify header row
	hostname, _ := f.GetCellValue(sheetAlerts, "A1")
	if hostname != "主机名" {
		t.Errorf("Header A1 = %q, want %q", hostname, "主机名")
	}

	// Verify alert data (sorted by severity - critical first)
	firstAlertHost, _ := f.GetCellValue(sheetAlerts, "A2")
	if firstAlertHost != "host-3" {
		t.Errorf("First alert host = %q, want %q (critical should be first)", firstAlertHost, "host-3")
	}

	firstAlertLevel, _ := f.GetCellValue(sheetAlerts, "B2")
	if firstAlertLevel != "严重" {
		t.Errorf("First alert level = %q, want %q", firstAlertLevel, "严重")
	}
}

func TestWriter_EmptyResult(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_report.xlsx")

	result := &model.InspectionResult{
		InspectionTime: time.Now(),
		Duration:       time.Second,
		Summary:        &model.InspectionSummary{},
		AlertSummary:   &model.AlertSummary{},
		Hosts:          []*model.HostResult{},
		Alerts:         []*model.Alert{},
	}

	w := NewWriter(nil)
	err := w.Write(result, outputPath)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Verify file exists and is valid
	f, err := excelize.OpenFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to open Excel file: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) != 3 {
		t.Errorf("Expected 3 sheets, got %d", len(sheets))
	}
}

// Helper functions

func createTestInspectionResult() *model.InspectionResult {
	tz, _ := time.LoadLocation("Asia/Shanghai")
	inspectionTime := time.Date(2025, 12, 13, 10, 0, 0, 0, tz)

	// Create hosts
	host1 := &model.HostResult{
		Hostname:      "host-1",
		IP:            "192.168.1.1",
		OS:            "Linux",
		OSVersion:     "CentOS 7.9",
		KernelVersion: "3.10.0-1160.el7",
		CPUCores:      4,
		Status:        model.HostStatusNormal,
		Metrics: map[string]*model.MetricValue{
			"cpu_usage": {
				Name:           "cpu_usage",
				RawValue:       45.5,
				FormattedValue: "45.5%",
				Status:         model.MetricStatusNormal,
			},
			"memory_usage": {
				Name:           "memory_usage",
				RawValue:       60.0,
				FormattedValue: "60.0%",
				Status:         model.MetricStatusNormal,
			},
			"disk_usage_max": {
				Name:           "disk_usage_max",
				RawValue:       55.0,
				FormattedValue: "55.0%",
				Status:         model.MetricStatusNormal,
			},
			"disk_usage:/": {
				Name:           "disk_usage:/",
				RawValue:       55.0,
				FormattedValue: "55.0%",
				Status:         model.MetricStatusNormal,
				Labels:         map[string]string{"path": "/"},
			},
		},
	}

	host2 := &model.HostResult{
		Hostname:      "host-2",
		IP:            "192.168.1.2",
		OS:            "Linux",
		OSVersion:     "Ubuntu 22.04",
		KernelVersion: "5.15.0-generic",
		CPUCores:      8,
		Status:        model.HostStatusWarning,
		Metrics: map[string]*model.MetricValue{
			"cpu_usage": {
				Name:           "cpu_usage",
				RawValue:       75.0,
				FormattedValue: "75.0%",
				Status:         model.MetricStatusWarning,
			},
			"memory_usage": {
				Name:           "memory_usage",
				RawValue:       65.0,
				FormattedValue: "65.0%",
				Status:         model.MetricStatusNormal,
			},
			"disk_usage_max": {
				Name:           "disk_usage_max",
				RawValue:       50.0,
				FormattedValue: "50.0%",
				Status:         model.MetricStatusNormal,
			},
		},
		Alerts: []*model.Alert{
			{
				Hostname:          "host-2",
				MetricName:        "cpu_usage",
				MetricDisplayName: "CPU利用率",
				CurrentValue:      75.0,
				FormattedValue:    "75.0%",
				WarningThreshold:  70.0,
				CriticalThreshold: 90.0,
				Level:             model.AlertLevelWarning,
				Message:           "CPU利用率 75.0% 超过警告阈值 70.0%",
			},
		},
	}

	host3 := &model.HostResult{
		Hostname:      "host-3",
		IP:            "192.168.1.3",
		OS:            "Linux",
		OSVersion:     "Rocky 9.0",
		KernelVersion: "5.14.0-70.el9",
		CPUCores:      16,
		Status:        model.HostStatusCritical,
		Metrics: map[string]*model.MetricValue{
			"cpu_usage": {
				Name:           "cpu_usage",
				RawValue:       95.0,
				FormattedValue: "95.0%",
				Status:         model.MetricStatusCritical,
			},
			"memory_usage": {
				Name:           "memory_usage",
				RawValue:       92.0,
				FormattedValue: "92.0%",
				Status:         model.MetricStatusCritical,
			},
			"disk_usage_max": {
				Name:           "disk_usage_max",
				RawValue:       88.0,
				FormattedValue: "88.0%",
				Status:         model.MetricStatusWarning,
			},
			"disk_usage:/": {
				Name:           "disk_usage:/",
				RawValue:       88.0,
				FormattedValue: "88.0%",
				Status:         model.MetricStatusWarning,
				Labels:         map[string]string{"path": "/"},
			},
			"disk_usage:/home": {
				Name:           "disk_usage:/home",
				RawValue:       45.0,
				FormattedValue: "45.0%",
				Status:         model.MetricStatusNormal,
				Labels:         map[string]string{"path": "/home"},
			},
		},
		Alerts: []*model.Alert{
			{
				Hostname:          "host-3",
				MetricName:        "cpu_usage",
				MetricDisplayName: "CPU利用率",
				CurrentValue:      95.0,
				FormattedValue:    "95.0%",
				WarningThreshold:  70.0,
				CriticalThreshold: 90.0,
				Level:             model.AlertLevelCritical,
				Message:           "CPU利用率 95.0% 超过严重阈值 90.0%",
			},
			{
				Hostname:          "host-3",
				MetricName:        "memory_usage",
				MetricDisplayName: "内存利用率",
				CurrentValue:      92.0,
				FormattedValue:    "92.0%",
				WarningThreshold:  70.0,
				CriticalThreshold: 90.0,
				Level:             model.AlertLevelCritical,
				Message:           "内存利用率 92.0% 超过严重阈值 90.0%",
			},
		},
	}

	// Collect all alerts
	allAlerts := make([]*model.Alert, 0)
	allAlerts = append(allAlerts, host2.Alerts...)
	allAlerts = append(allAlerts, host3.Alerts...)

	return &model.InspectionResult{
		InspectionTime: inspectionTime,
		Duration:       5 * time.Second,
		Summary: &model.InspectionSummary{
			TotalHosts:    3,
			NormalHosts:   1,
			WarningHosts:  1,
			CriticalHosts: 1,
			FailedHosts:   0,
		},
		Hosts:  []*model.HostResult{host1, host2, host3},
		Alerts: allAlerts,
		AlertSummary: &model.AlertSummary{
			TotalAlerts:   3,
			WarningCount:  1,
			CriticalCount: 2,
		},
		Version: "1.0.0-test",
	}
}

func TestColumnName(t *testing.T) {
	tests := []struct {
		index int
		want  string
	}{
		{1, "A"},
		{2, "B"},
		{26, "Z"},
		{27, "AA"},
		{28, "AB"},
		{52, "AZ"},
		{53, "BA"},
		{702, "ZZ"},
		{703, "AAA"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := columnName(tt.index); got != tt.want {
				t.Errorf("columnName(%d) = %q, want %q", tt.index, got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{500 * time.Millisecond, "500ms"},
		{5 * time.Second, "5.0秒"},
		{90 * time.Second, "1.5分钟"},
		{2 * time.Hour, "2.0小时"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := formatDuration(tt.duration); got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

func TestStatusText(t *testing.T) {
	tests := []struct {
		status model.HostStatus
		want   string
	}{
		{model.HostStatusNormal, "正常"},
		{model.HostStatusWarning, "警告"},
		{model.HostStatusCritical, "严重"},
		{model.HostStatusFailed, "失败"},
		{model.HostStatus("unknown"), "未知"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := statusText(tt.status); got != tt.want {
				t.Errorf("statusText(%v) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestAlertLevelPriority(t *testing.T) {
	tests := []struct {
		level model.AlertLevel
		want  int
	}{
		{model.AlertLevelCritical, 2},
		{model.AlertLevelWarning, 1},
		{model.AlertLevelNormal, 0},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			if got := alertLevelPriority(tt.level); got != tt.want {
				t.Errorf("alertLevelPriority(%v) = %d, want %d", tt.level, got, tt.want)
			}
		})
	}
}

func TestWriter_CollectDiskPaths(t *testing.T) {
	w := NewWriter(nil)
	hosts := []*model.HostResult{
		{
			Metrics: map[string]*model.MetricValue{
				"cpu_usage":        {Name: "cpu_usage"},
				"disk_usage:/":     {Name: "disk_usage:/"},
				"disk_usage:/home": {Name: "disk_usage:/home"},
			},
		},
		{
			Metrics: map[string]*model.MetricValue{
				"disk_usage:/":    {Name: "disk_usage:/"},
				"disk_usage:/var": {Name: "disk_usage:/var"},
			},
		},
	}

	paths := w.collectDiskPaths(hosts)

	// Should be sorted and unique
	expected := []string{"/", "/home", "/var"}
	if len(paths) != len(expected) {
		t.Errorf("collectDiskPaths returned %d paths, want %d", len(paths), len(expected))
	}
	for i, p := range paths {
		if p != expected[i] {
			t.Errorf("paths[%d] = %q, want %q", i, p, expected[i])
		}
	}
}

// ============================================================================
// MySQL Report Tests
// ============================================================================

func TestWriter_WriteMySQLInspection_NilResult(t *testing.T) {
	w := NewWriter(nil)
	err := w.WriteMySQLInspection(nil, "test.xlsx")
	if err == nil {
		t.Error("WriteMySQLInspection() with nil result should return error")
	}
}

func TestWriter_WriteMySQLInspection_Success(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_mysql_report.xlsx")

	result := createTestMySQLInspectionResults()

	w := NewWriter(nil)
	err := w.WriteMySQLInspection(result, outputPath)
	if err != nil {
		t.Fatalf("WriteMySQLInspection() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}

	// Open and verify Excel file
	f, err := excelize.OpenFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to open Excel file: %v", err)
	}
	defer f.Close()

	// Verify MySQL sheet exists
	sheets := f.GetSheetList()
	found := false
	for _, s := range sheets {
		if s == sheetMySQL {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Sheet %q not found in Excel file", sheetMySQL)
	}

	// Verify default Sheet1 was removed
	for _, s := range sheets {
		if s == "Sheet1" {
			t.Error("Default Sheet1 should have been removed")
		}
	}
}

func TestWriter_WriteMySQLInspection_AddsXlsxExtension(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_mysql_report") // No extension

	result := createTestMySQLInspectionResults()
	w := NewWriter(nil)
	err := w.WriteMySQLInspection(result, outputPath)
	if err != nil {
		t.Fatalf("WriteMySQLInspection() error = %v", err)
	}

	// Verify file with .xlsx extension exists
	expectedPath := outputPath + ".xlsx"
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("Output file should have .xlsx extension added")
	}
}

func TestWriter_MySQLSheet_Headers(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_mysql_report.xlsx")

	result := createTestMySQLInspectionResults()
	w := NewWriter(nil)
	err := w.WriteMySQLInspection(result, outputPath)
	if err != nil {
		t.Fatalf("WriteMySQLInspection() error = %v", err)
	}

	f, err := excelize.OpenFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to open Excel file: %v", err)
	}
	defer f.Close()

	// Verify all 11 headers
	expectedHeaders := []struct {
		cell   string
		header string
	}{
		{"A1", "巡检时间"},
		{"B1", "IP地址"},
		{"C1", "端口"},
		{"D1", "数据库版本"},
		{"E1", "Server ID"},
		{"F1", "集群模式"},
		{"G1", "同步状态"},
		{"H1", "最大连接数"},
		{"I1", "当前连接数"},
		{"J1", "Binlog状态"},
		{"K1", "整体状态"},
	}

	for _, eh := range expectedHeaders {
		value, _ := f.GetCellValue(sheetMySQL, eh.cell)
		if value != eh.header {
			t.Errorf("Header %s = %q, want %q", eh.cell, value, eh.header)
		}
	}
}

func TestWriter_MySQLSheet_DataMapping(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_mysql_report.xlsx")

	result := createTestMySQLInspectionResults()
	w := NewWriter(nil)
	err := w.WriteMySQLInspection(result, outputPath)
	if err != nil {
		t.Fatalf("WriteMySQLInspection() error = %v", err)
	}

	f, err := excelize.OpenFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to open Excel file: %v", err)
	}
	defer f.Close()

	// Verify first row data (row 2, since row 1 is header)
	tests := []struct {
		cell     string
		expected string
	}{
		{"B2", "172.18.182.91"},       // IP地址
		{"C2", "3306"},                // 端口
		{"D2", "8.0.39"},              // 数据库版本
		{"E2", "91"},                  // Server ID
		{"F2", "MGR"},                 // 集群模式
		{"G2", "在线"},                  // 同步状态
		{"H2", "1000"},                // 最大连接数
		{"I2", "100"},                 // 当前连接数
		{"J2", "启用"},                  // Binlog状态
		{"K2", "正常"},                  // 整体状态
	}

	for _, tt := range tests {
		value, _ := f.GetCellValue(sheetMySQL, tt.cell)
		if value != tt.expected {
			t.Errorf("Cell %s = %q, want %q", tt.cell, value, tt.expected)
		}
	}
}

func TestWriter_MySQLSheet_ConditionalFormat(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_mysql_report.xlsx")

	result := createTestMySQLInspectionResults()
	w := NewWriter(nil)
	err := w.WriteMySQLInspection(result, outputPath)
	if err != nil {
		t.Fatalf("WriteMySQLInspection() error = %v", err)
	}

	f, err := excelize.OpenFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to open Excel file: %v", err)
	}
	defer f.Close()

	// Verify status column values and order (normal, warning, critical)
	expectedStatuses := []struct {
		cell   string
		status string
	}{
		{"K2", "正常"},
		{"K3", "警告"},
		{"K4", "严重"},
	}

	for _, es := range expectedStatuses {
		value, _ := f.GetCellValue(sheetMySQL, es.cell)
		if value != es.status {
			t.Errorf("Status at %s = %q, want %q", es.cell, value, es.status)
		}
	}
}

// Helper function to create test MySQL inspection results
func createTestMySQLInspectionResults() *model.MySQLInspectionResults {
	tz, _ := time.LoadLocation("Asia/Shanghai")
	inspectionTime := time.Date(2025, 12, 16, 10, 0, 0, 0, tz)

	// Create alerts for warning and critical instances
	warningAlert := &model.MySQLAlert{
		Address:           "172.18.182.92:3306",
		MetricName:        "connection_usage",
		MetricDisplayName: "连接使用率",
		CurrentValue:      80.0,
		FormattedValue:    "80.0%",
		WarningThreshold:  70.0,
		CriticalThreshold: 90.0,
		Level:             model.AlertLevelWarning,
		Message:           "连接使用率 80.0% 超过警告阈值 70.0%",
	}

	criticalAlert := &model.MySQLAlert{
		Address:           "172.18.182.93:3306",
		MetricName:        "mgr_state_online",
		MetricDisplayName: "MGR 在线状态",
		CurrentValue:      0,
		FormattedValue:    "离线",
		WarningThreshold:  0,
		CriticalThreshold: 1,
		Level:             model.AlertLevelCritical,
		Message:           "MGR 节点离线",
	}

	return &model.MySQLInspectionResults{
		InspectionTime: inspectionTime,
		Duration:       2 * time.Second,
		Summary: &model.MySQLInspectionSummary{
			TotalInstances:    3,
			NormalInstances:   1,
			WarningInstances:  1,
			CriticalInstances: 1,
		},
		Results: []*model.MySQLInspectionResult{
			// Normal instance
			{
				Instance: &model.MySQLInstance{
					Address:     "172.18.182.91:3306",
					IP:          "172.18.182.91",
					Port:        3306,
					Version:     "8.0.39",
					ServerID:    "91",
					ClusterMode: model.ClusterModeMGR,
				},
				MaxConnections:     1000,
				CurrentConnections: 100,
				MGRStateOnline:     true,
				BinlogEnabled:      true,
				Status:             model.MySQLStatusNormal,
			},
			// Warning instance (high connection usage)
			{
				Instance: &model.MySQLInstance{
					Address:     "172.18.182.92:3306",
					IP:          "172.18.182.92",
					Port:        3306,
					Version:     "8.0.39",
					ServerID:    "92",
					ClusterMode: model.ClusterModeMGR,
				},
				MaxConnections:     1000,
				CurrentConnections: 800,
				MGRStateOnline:     true,
				BinlogEnabled:      true,
				Status:             model.MySQLStatusWarning,
				Alerts:             []*model.MySQLAlert{warningAlert},
			},
			// Critical instance (MGR offline)
			{
				Instance: &model.MySQLInstance{
					Address:     "172.18.182.93:3306",
					IP:          "172.18.182.93",
					Port:        3306,
					Version:     "8.0.39",
					ServerID:    "93",
					ClusterMode: model.ClusterModeMGR,
				},
				MaxConnections:     1000,
				CurrentConnections: 50,
				MGRStateOnline:     false,
				BinlogEnabled:      true,
				Status:             model.MySQLStatusCritical,
				Alerts:             []*model.MySQLAlert{criticalAlert},
			},
		},
		Alerts: []*model.MySQLAlert{warningAlert, criticalAlert},
		AlertSummary: &model.MySQLAlertSummary{
			TotalAlerts:   2,
			WarningCount:  1,
			CriticalCount: 1,
		},
		Version: "1.0.0-test",
	}
}

// Test MySQL helper functions
func TestMySQLStatusText(t *testing.T) {
	tests := []struct {
		status model.MySQLInstanceStatus
		want   string
	}{
		{model.MySQLStatusNormal, "正常"},
		{model.MySQLStatusWarning, "警告"},
		{model.MySQLStatusCritical, "严重"},
		{model.MySQLStatusFailed, "失败"},
		{model.MySQLInstanceStatus("unknown"), "未知"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := mysqlStatusText(tt.status); got != tt.want {
				t.Errorf("mysqlStatusText(%v) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestMySQLClusterModeText(t *testing.T) {
	tests := []struct {
		mode model.MySQLClusterMode
		want string
	}{
		{model.ClusterModeMGR, "MGR"},
		{model.ClusterModeDualMaster, "双主"},
		{model.ClusterModeMasterSlave, "主从"},
		{model.MySQLClusterMode("unknown"), "未知"},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			if got := mysqlClusterModeText(tt.mode); got != tt.want {
				t.Errorf("mysqlClusterModeText(%v) = %q, want %q", tt.mode, got, tt.want)
			}
		})
	}
}

func TestBoolToText(t *testing.T) {
	tests := []struct {
		value bool
		want  string
	}{
		{true, "启用"},
		{false, "禁用"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := boolToText(tt.value); got != tt.want {
				t.Errorf("boolToText(%v) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestGetMySQLSyncStatus(t *testing.T) {
	w := NewWriter(nil)

	tests := []struct {
		name   string
		result *model.MySQLInspectionResult
		want   string
	}{
		{
			name: "MGR online",
			result: &model.MySQLInspectionResult{
				Instance:       &model.MySQLInstance{ClusterMode: model.ClusterModeMGR},
				MGRStateOnline: true,
			},
			want: "在线",
		},
		{
			name: "MGR offline",
			result: &model.MySQLInspectionResult{
				Instance:       &model.MySQLInstance{ClusterMode: model.ClusterModeMGR},
				MGRStateOnline: false,
			},
			want: "离线",
		},
		{
			name: "Master-Slave sync OK",
			result: &model.MySQLInspectionResult{
				Instance:   &model.MySQLInstance{ClusterMode: model.ClusterModeMasterSlave},
				SyncStatus: true,
			},
			want: "正常",
		},
		{
			name: "Master-Slave sync failed",
			result: &model.MySQLInspectionResult{
				Instance:   &model.MySQLInstance{ClusterMode: model.ClusterModeMasterSlave},
				SyncStatus: false,
			},
			want: "异常",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := w.getMySQLSyncStatus(tt.result); got != tt.want {
				t.Errorf("getMySQLSyncStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ============================================================================
// MySQL Alerts Sheet Tests
// ============================================================================

func TestWriter_WriteMySQLInspection_AlertsSheetExists(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_mysql_report.xlsx")

	result := createTestMySQLInspectionResults()

	w := NewWriter(nil)
	err := w.WriteMySQLInspection(result, outputPath)
	if err != nil {
		t.Fatalf("WriteMySQLInspection() error = %v", err)
	}

	// Open and verify Excel file
	f, err := excelize.OpenFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to open Excel file: %v", err)
	}
	defer f.Close()

	// Verify MySQL alerts sheet exists
	sheets := f.GetSheetList()
	found := false
	for _, s := range sheets {
		if s == sheetMySQLAlerts {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Sheet %q not found in Excel file, got sheets: %v", sheetMySQLAlerts, sheets)
	}
}

func TestWriter_MySQLAlertsSheet_Headers(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_mysql_report.xlsx")

	result := createTestMySQLInspectionResults()
	w := NewWriter(nil)
	err := w.WriteMySQLInspection(result, outputPath)
	if err != nil {
		t.Fatalf("WriteMySQLInspection() error = %v", err)
	}

	f, err := excelize.OpenFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to open Excel file: %v", err)
	}
	defer f.Close()

	// Verify all 7 headers
	expectedHeaders := []struct {
		cell   string
		header string
	}{
		{"A1", "实例地址"},
		{"B1", "告警级别"},
		{"C1", "指标名称"},
		{"D1", "当前值"},
		{"E1", "警告阈值"},
		{"F1", "严重阈值"},
		{"G1", "告警消息"},
	}

	for _, eh := range expectedHeaders {
		value, _ := f.GetCellValue(sheetMySQLAlerts, eh.cell)
		if value != eh.header {
			t.Errorf("Header %s = %q, want %q", eh.cell, value, eh.header)
		}
	}
}

func TestWriter_MySQLAlertsSheet_DataMapping(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_mysql_report.xlsx")

	result := createTestMySQLInspectionResults()
	w := NewWriter(nil)
	err := w.WriteMySQLInspection(result, outputPath)
	if err != nil {
		t.Fatalf("WriteMySQLInspection() error = %v", err)
	}

	f, err := excelize.OpenFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to open Excel file: %v", err)
	}
	defer f.Close()

	// Row 2 should be the critical alert (sorted first by severity)
	// Critical alert: 172.18.182.93:3306, mgr_state_online
	tests := []struct {
		cell     string
		expected string
	}{
		{"A2", "172.18.182.93:3306"}, // 实例地址 (critical first)
		{"B2", "严重"},                  // 告警级别
		{"C2", "MGR 在线状态"},           // 指标名称
		{"D2", "离线"},                  // 当前值
		{"G2", "MGR 节点离线"},           // 告警消息
	}

	for _, tt := range tests {
		value, _ := f.GetCellValue(sheetMySQLAlerts, tt.cell)
		if value != tt.expected {
			t.Errorf("Cell %s = %q, want %q", tt.cell, value, tt.expected)
		}
	}
}

func TestWriter_MySQLAlertsSheet_SortBySeverity(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_mysql_report.xlsx")

	result := createTestMySQLInspectionResults()
	w := NewWriter(nil)
	err := w.WriteMySQLInspection(result, outputPath)
	if err != nil {
		t.Fatalf("WriteMySQLInspection() error = %v", err)
	}

	f, err := excelize.OpenFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to open Excel file: %v", err)
	}
	defer f.Close()

	// Verify alerts are sorted by severity (critical first)
	// Row 2: Critical alert
	level2, _ := f.GetCellValue(sheetMySQLAlerts, "B2")
	if level2 != "严重" {
		t.Errorf("First alert level = %q, want %q (critical should be first)", level2, "严重")
	}

	// Row 3: Warning alert
	level3, _ := f.GetCellValue(sheetMySQLAlerts, "B3")
	if level3 != "警告" {
		t.Errorf("Second alert level = %q, want %q", level3, "警告")
	}
}

func TestWriter_MySQLAlertsSheet_EmptyAlerts(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_mysql_report.xlsx")

	// Create result with no alerts
	tz, _ := time.LoadLocation("Asia/Shanghai")
	result := &model.MySQLInspectionResults{
		InspectionTime: time.Now().In(tz),
		Duration:       time.Second,
		Summary: &model.MySQLInspectionSummary{
			TotalInstances:  1,
			NormalInstances: 1,
		},
		Results: []*model.MySQLInspectionResult{
			{
				Instance: &model.MySQLInstance{
					Address:     "172.18.182.91:3306",
					IP:          "172.18.182.91",
					Port:        3306,
					ClusterMode: model.ClusterModeMGR,
				},
				Status: model.MySQLStatusNormal,
			},
		},
		Alerts:       []*model.MySQLAlert{}, // Empty alerts
		AlertSummary: &model.MySQLAlertSummary{},
	}

	w := NewWriter(nil)
	err := w.WriteMySQLInspection(result, outputPath)
	if err != nil {
		t.Fatalf("WriteMySQLInspection() error = %v", err)
	}

	// Verify file exists and sheet is created (with only headers)
	f, err := excelize.OpenFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to open Excel file: %v", err)
	}
	defer f.Close()

	// Sheet should exist
	sheets := f.GetSheetList()
	found := false
	for _, s := range sheets {
		if s == sheetMySQLAlerts {
			found = true
			break
		}
	}
	if !found {
		t.Error("MySQL alerts sheet should exist even with empty alerts")
	}

	// Headers should be present
	header, _ := f.GetCellValue(sheetMySQLAlerts, "A1")
	if header != "实例地址" {
		t.Errorf("Header A1 = %q, want %q", header, "实例地址")
	}

	// Row 2 should be empty (no data)
	row2, _ := f.GetCellValue(sheetMySQLAlerts, "A2")
	if row2 != "" {
		t.Errorf("Row 2 should be empty for empty alerts, got %q", row2)
	}
}

func TestFormatMySQLThreshold(t *testing.T) {
	tests := []struct {
		name       string
		value      float64
		metricName string
		want       string
	}{
		{
			name:       "connection_usage percentage",
			value:      70.0,
			metricName: "connection_usage",
			want:       "70.0%",
		},
		{
			name:       "mgr_member_count integer",
			value:      3.0,
			metricName: "mgr_member_count",
			want:       "3",
		},
		{
			name:       "mgr_state_online online",
			value:      1.0,
			metricName: "mgr_state_online",
			want:       "在线",
		},
		{
			name:       "mgr_state_online offline",
			value:      0.0,
			metricName: "mgr_state_online",
			want:       "离线",
		},
		{
			name:       "default format",
			value:      1.234,
			metricName: "unknown_metric",
			want:       "1.23",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatMySQLThreshold(tt.value, tt.metricName); got != tt.want {
				t.Errorf("formatMySQLThreshold(%v, %q) = %q, want %q", tt.value, tt.metricName, got, tt.want)
			}
		})
	}
}
