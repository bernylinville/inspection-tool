//go:build ignore
// +build ignore

// This script generates a sample Excel report for manual verification.
// Run with: go run scripts/verify_excel.go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"inspection-tool/internal/model"
	"inspection-tool/internal/report/excel"
)

func main() {
	// Create test data
	result := createSampleData()

	// Create Excel writer
	tz, _ := time.LoadLocation("Asia/Shanghai")
	writer := excel.NewWriter(tz)

	// Generate report
	outputPath := filepath.Join(".", "sample_inspection_report.xlsx")
	if err := writer.Write(result, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating report: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Excel report generated: %s\n", outputPath)
	fmt.Println("\nReport contents:")
	fmt.Println("  - 巡检概览: Summary statistics")
	fmt.Println("  - 详细数据: Host details with metrics")
	fmt.Println("  - 异常汇总: Alerts sorted by severity")
	fmt.Println("\nPlease open the file to verify:")
	fmt.Println("  - Time is in Asia/Shanghai timezone")
	fmt.Println("  - Warning cells have yellow background")
	fmt.Println("  - Critical cells have red background")
	fmt.Println("  - Alerts are sorted (critical first)")
	fmt.Println("  - Disk paths are expanded dynamically")
}

func createSampleData() *model.InspectionResult {
	tz, _ := time.LoadLocation("Asia/Shanghai")
	inspectionTime := time.Now().In(tz)

	// Create hosts with various statuses
	hosts := []*model.HostResult{
		createNormalHost("web-server-01", "192.168.1.10", tz),
		createWarningHost("db-server-01", "192.168.1.20", tz),
		createCriticalHost("app-server-01", "192.168.1.30", tz),
		createFailedHost("monitor-01", "192.168.1.40", tz),
		createNormalHost("web-server-02", "192.168.1.11", tz),
	}

	// Collect all alerts
	var allAlerts []*model.Alert
	for _, host := range hosts {
		allAlerts = append(allAlerts, host.Alerts...)
	}

	return &model.InspectionResult{
		InspectionTime: inspectionTime,
		Duration:       3*time.Second + 500*time.Millisecond,
		Summary: &model.InspectionSummary{
			TotalHosts:    5,
			NormalHosts:   2,
			WarningHosts:  1,
			CriticalHosts: 1,
			FailedHosts:   1,
		},
		Hosts:  hosts,
		Alerts: allAlerts,
		AlertSummary: &model.AlertSummary{
			TotalAlerts:   5,
			WarningCount:  2,
			CriticalCount: 3,
		},
		Version: "1.0.0-dev",
	}
}

func createNormalHost(hostname, ip string, tz *time.Location) *model.HostResult {
	return &model.HostResult{
		Hostname:      hostname,
		IP:            ip,
		OS:            "Linux",
		OSVersion:     "CentOS 7.9.2009",
		KernelVersion: "3.10.0-1160.el7.x86_64",
		CPUCores:      8,
		CPUModel:      "Intel Xeon E5-2680 v4",
		MemoryTotal:   17179869184, // 16GB
		Status:        model.HostStatusNormal,
		CollectedAt:   time.Now().In(tz),
		Metrics: map[string]*model.MetricValue{
			"cpu_usage":     {Name: "cpu_usage", RawValue: 35.5, FormattedValue: "35.5%", Status: model.MetricStatusNormal},
			"memory_usage":  {Name: "memory_usage", RawValue: 45.2, FormattedValue: "45.2%", Status: model.MetricStatusNormal},
			"disk_usage_max": {Name: "disk_usage_max", RawValue: 52.0, FormattedValue: "52.0%", Status: model.MetricStatusNormal},
			"disk_usage:/":  {Name: "disk_usage:/", RawValue: 52.0, FormattedValue: "52.0%", Status: model.MetricStatusNormal, Labels: map[string]string{"path": "/"}},
			"disk_usage:/home": {Name: "disk_usage:/home", RawValue: 35.0, FormattedValue: "35.0%", Status: model.MetricStatusNormal, Labels: map[string]string{"path": "/home"}},
			"uptime":         {Name: "uptime", RawValue: 8640000, FormattedValue: "100天", Status: model.MetricStatusNormal},
			"load_1m":        {Name: "load_1m", RawValue: 1.2, FormattedValue: "1.20", Status: model.MetricStatusNormal},
			"load_per_core":  {Name: "load_per_core", RawValue: 0.15, FormattedValue: "0.15", Status: model.MetricStatusNormal},
			"processes_zombies": {Name: "processes_zombies", RawValue: 0, FormattedValue: "0", Status: model.MetricStatusNormal},
			"processes_total":   {Name: "processes_total", RawValue: 156, FormattedValue: "156", Status: model.MetricStatusNormal},
		},
		Alerts: nil,
	}
}

func createWarningHost(hostname, ip string, tz *time.Location) *model.HostResult {
	return &model.HostResult{
		Hostname:      hostname,
		IP:            ip,
		OS:            "Linux",
		OSVersion:     "Rocky Linux 9.0",
		KernelVersion: "5.14.0-70.el9.x86_64",
		CPUCores:      16,
		CPUModel:      "AMD EPYC 7543",
		MemoryTotal:   68719476736, // 64GB
		Status:        model.HostStatusWarning,
		CollectedAt:   time.Now().In(tz),
		Metrics: map[string]*model.MetricValue{
			"cpu_usage":     {Name: "cpu_usage", RawValue: 78.5, FormattedValue: "78.5%", Status: model.MetricStatusWarning},
			"memory_usage":  {Name: "memory_usage", RawValue: 65.0, FormattedValue: "65.0%", Status: model.MetricStatusNormal},
			"disk_usage_max": {Name: "disk_usage_max", RawValue: 75.0, FormattedValue: "75.0%", Status: model.MetricStatusWarning},
			"disk_usage:/":  {Name: "disk_usage:/", RawValue: 45.0, FormattedValue: "45.0%", Status: model.MetricStatusNormal, Labels: map[string]string{"path": "/"}},
			"disk_usage:/data": {Name: "disk_usage:/data", RawValue: 75.0, FormattedValue: "75.0%", Status: model.MetricStatusWarning, Labels: map[string]string{"path": "/data"}},
			"uptime":         {Name: "uptime", RawValue: 2592000, FormattedValue: "30天", Status: model.MetricStatusNormal},
			"load_1m":        {Name: "load_1m", RawValue: 8.5, FormattedValue: "8.50", Status: model.MetricStatusNormal},
			"load_per_core":  {Name: "load_per_core", RawValue: 0.53, FormattedValue: "0.53", Status: model.MetricStatusNormal},
			"processes_zombies": {Name: "processes_zombies", RawValue: 0, FormattedValue: "0", Status: model.MetricStatusNormal},
			"processes_total":   {Name: "processes_total", RawValue: 320, FormattedValue: "320", Status: model.MetricStatusNormal},
		},
		Alerts: []*model.Alert{
			{
				Hostname:          hostname,
				MetricName:        "cpu_usage",
				MetricDisplayName: "CPU利用率",
				CurrentValue:      78.5,
				FormattedValue:    "78.5%",
				WarningThreshold:  70.0,
				CriticalThreshold: 90.0,
				Level:             model.AlertLevelWarning,
				Message:           "CPU利用率 78.5% 超过警告阈值 70.0%",
			},
			{
				Hostname:          hostname,
				MetricName:        "disk_usage_max",
				MetricDisplayName: "磁盘最大利用率",
				CurrentValue:      75.0,
				FormattedValue:    "75.0%",
				WarningThreshold:  70.0,
				CriticalThreshold: 90.0,
				Level:             model.AlertLevelWarning,
				Message:           "磁盘最大利用率 75.0% 超过警告阈值 70.0%",
			},
		},
	}
}

func createCriticalHost(hostname, ip string, tz *time.Location) *model.HostResult {
	return &model.HostResult{
		Hostname:      hostname,
		IP:            ip,
		OS:            "Linux",
		OSVersion:     "Ubuntu 22.04.3 LTS",
		KernelVersion: "5.15.0-91-generic",
		CPUCores:      4,
		CPUModel:      "Intel Core i7-8700",
		MemoryTotal:   34359738368, // 32GB
		Status:        model.HostStatusCritical,
		CollectedAt:   time.Now().In(tz),
		Metrics: map[string]*model.MetricValue{
			"cpu_usage":     {Name: "cpu_usage", RawValue: 95.8, FormattedValue: "95.8%", Status: model.MetricStatusCritical},
			"memory_usage":  {Name: "memory_usage", RawValue: 92.5, FormattedValue: "92.5%", Status: model.MetricStatusCritical},
			"disk_usage_max": {Name: "disk_usage_max", RawValue: 88.0, FormattedValue: "88.0%", Status: model.MetricStatusWarning},
			"disk_usage:/":  {Name: "disk_usage:/", RawValue: 88.0, FormattedValue: "88.0%", Status: model.MetricStatusWarning, Labels: map[string]string{"path": "/"}},
			"disk_usage:/var": {Name: "disk_usage:/var", RawValue: 65.0, FormattedValue: "65.0%", Status: model.MetricStatusNormal, Labels: map[string]string{"path": "/var"}},
			"uptime":         {Name: "uptime", RawValue: 604800, FormattedValue: "7天", Status: model.MetricStatusNormal},
			"load_1m":        {Name: "load_1m", RawValue: 12.5, FormattedValue: "12.50", Status: model.MetricStatusNormal},
			"load_per_core":  {Name: "load_per_core", RawValue: 3.12, FormattedValue: "3.12", Status: model.MetricStatusCritical},
			"processes_zombies": {Name: "processes_zombies", RawValue: 5, FormattedValue: "5", Status: model.MetricStatusWarning},
			"processes_total":   {Name: "processes_total", RawValue: 512, FormattedValue: "512", Status: model.MetricStatusNormal},
		},
		Alerts: []*model.Alert{
			{
				Hostname:          hostname,
				MetricName:        "cpu_usage",
				MetricDisplayName: "CPU利用率",
				CurrentValue:      95.8,
				FormattedValue:    "95.8%",
				WarningThreshold:  70.0,
				CriticalThreshold: 90.0,
				Level:             model.AlertLevelCritical,
				Message:           "CPU利用率 95.8% 超过严重阈值 90.0%",
			},
			{
				Hostname:          hostname,
				MetricName:        "memory_usage",
				MetricDisplayName: "内存利用率",
				CurrentValue:      92.5,
				FormattedValue:    "92.5%",
				WarningThreshold:  70.0,
				CriticalThreshold: 90.0,
				Level:             model.AlertLevelCritical,
				Message:           "内存利用率 92.5% 超过严重阈值 90.0%",
			},
			{
				Hostname:          hostname,
				MetricName:        "load_per_core",
				MetricDisplayName: "每核负载",
				CurrentValue:      3.12,
				FormattedValue:    "3.12",
				WarningThreshold:  0.7,
				CriticalThreshold: 1.0,
				Level:             model.AlertLevelCritical,
				Message:           "每核负载 3.12 超过严重阈值 1.0",
			},
		},
	}
}

func createFailedHost(hostname, ip string, tz *time.Location) *model.HostResult {
	return &model.HostResult{
		Hostname:      hostname,
		IP:            ip,
		OS:            "",
		OSVersion:     "",
		KernelVersion: "",
		CPUCores:      0,
		MemoryTotal:   0,
		Status:        model.HostStatusFailed,
		CollectedAt:   time.Now().In(tz),
		Metrics:       map[string]*model.MetricValue{},
		Alerts:        nil,
		Error:         "connection timeout: failed to connect to host",
	}
}
