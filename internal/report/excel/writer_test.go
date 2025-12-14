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
