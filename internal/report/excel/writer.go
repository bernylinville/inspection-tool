// Package excel provides Excel report generation for the inspection tool.
// It implements the report.ReportWriter interface to generate .xlsx files
// with inspection results, including summary, detailed data, and alerts.
package excel

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"inspection-tool/internal/model"
)

const (
	// Sheet names
	sheetSummary = "巡检概览"
	sheetDetail  = "详细数据"
	sheetAlerts  = "异常汇总"
	sheetMySQL       = "MySQL 巡检" // MySQL inspection sheet
	sheetMySQLAlerts = "MySQL 异常" // MySQL alerts sheet
	sheetRedis       = "Redis 巡检" // Redis inspection sheet
	sheetRedisAlerts = "Redis 异常" // Redis alerts sheet

	// Default sheet to remove
	defaultSheet = "Sheet1"

	// Colors for conditional formatting (RGB without #)
	colorWarningBg  = "FFEB9C" // Yellow background for warning
	colorWarningFg  = "9C6500" // Dark yellow text for warning
	colorCriticalBg = "FFC7CE" // Red background for critical
	colorCriticalFg = "9C0006" // Dark red text for critical
	colorHeaderBg   = "4472C4" // Blue background for header
	colorHeaderFg   = "FFFFFF" // White text for header
	colorNormalBg   = "C6EFCE" // Green background for normal
	colorNormalFg   = "006100" // Dark green text for normal

	// Column widths
	defaultColWidth = 15.0
	wideColWidth    = 25.0
	narrowColWidth  = 10.0
)

// Writer implements report.ReportWriter for Excel format.
type Writer struct {
	timezone *time.Location
}

// NewWriter creates a new Excel report writer.
// If timezone is nil, it defaults to Asia/Shanghai.
func NewWriter(timezone *time.Location) *Writer {
	if timezone == nil {
		timezone, _ = time.LoadLocation("Asia/Shanghai")
	}
	return &Writer{
		timezone: timezone,
	}
}

// Format returns the format identifier for this writer.
func (w *Writer) Format() string {
	return "excel"
}

// Write generates an Excel report from the inspection result.
func (w *Writer) Write(result *model.InspectionResult, outputPath string) error {
	if result == nil {
		return fmt.Errorf("inspection result is nil")
	}

	// Ensure output path has .xlsx extension
	if !strings.HasSuffix(strings.ToLower(outputPath), ".xlsx") {
		outputPath = outputPath + ".xlsx"
	}

	// Create new Excel file
	f := excelize.NewFile()
	defer f.Close()

	// Create worksheets
	if err := w.createSummarySheet(f, result); err != nil {
		return fmt.Errorf("failed to create summary sheet: %w", err)
	}

	if err := w.createDetailSheet(f, result); err != nil {
		return fmt.Errorf("failed to create detail sheet: %w", err)
	}

	if err := w.createAlertsSheet(f, result); err != nil {
		return fmt.Errorf("failed to create alerts sheet: %w", err)
	}

	// Remove default Sheet1
	if err := f.DeleteSheet(defaultSheet); err != nil {
		// Ignore error if sheet doesn't exist
	}

	// Set active sheet to summary
	idx, _ := f.GetSheetIndex(sheetSummary)
	f.SetActiveSheet(idx)

	// Ensure output directory exists
	dir := filepath.Dir(outputPath)
	if dir != "" && dir != "." {
		// Directory creation is handled by the caller
	}

	// Save the file
	if err := f.SaveAs(outputPath); err != nil {
		return fmt.Errorf("failed to save Excel file: %w", err)
	}

	return nil
}

// createSummarySheet creates the inspection summary worksheet.
func (w *Writer) createSummarySheet(f *excelize.File, result *model.InspectionResult) error {
	// Create sheet
	idx, err := f.NewSheet(sheetSummary)
	if err != nil {
		return err
	}
	f.SetActiveSheet(idx)

	// Create header style
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Size:  14,
			Color: colorHeaderFg,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{colorHeaderBg},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	if err != nil {
		return err
	}

	// Create title style
	titleStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
			Size: 18,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	if err != nil {
		return err
	}

	// Create value style
	valueStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Size: 12,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	if err != nil {
		return err
	}

	// Set column widths
	f.SetColWidth(sheetSummary, "A", "A", 20)
	f.SetColWidth(sheetSummary, "B", "B", 30)

	// Title
	f.MergeCell(sheetSummary, "A1", "B1")
	f.SetCellValue(sheetSummary, "A1", "系统巡检报告")
	f.SetCellStyle(sheetSummary, "A1", "B1", titleStyle)
	f.SetRowHeight(sheetSummary, 1, 30)

	// Summary data
	summaryData := []struct {
		label string
		value interface{}
	}{
		{"巡检时间", result.InspectionTime.In(w.timezone).Format("2006-01-02 15:04:05")},
		{"巡检耗时", formatDuration(result.Duration)},
		{"主机总数", result.Summary.TotalHosts},
		{"正常主机", result.Summary.NormalHosts},
		{"警告主机", result.Summary.WarningHosts},
		{"严重主机", result.Summary.CriticalHosts},
		{"失败主机", result.Summary.FailedHosts},
		{"告警总数", result.AlertSummary.TotalAlerts},
		{"警告告警", result.AlertSummary.WarningCount},
		{"严重告警", result.AlertSummary.CriticalCount},
	}

	if result.Version != "" {
		summaryData = append(summaryData, struct {
			label string
			value interface{}
		}{"工具版本", result.Version})
	}

	// Write summary data
	for i, item := range summaryData {
		row := i + 3 // Start from row 3
		f.SetCellValue(sheetSummary, fmt.Sprintf("A%d", row), item.label)
		f.SetCellValue(sheetSummary, fmt.Sprintf("B%d", row), item.value)
		f.SetCellStyle(sheetSummary, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), headerStyle)
		f.SetCellStyle(sheetSummary, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), valueStyle)
		f.SetRowHeight(sheetSummary, row, 22)
	}

	return nil
}

// createDetailSheet creates the detailed data worksheet.
func (w *Writer) createDetailSheet(f *excelize.File, result *model.InspectionResult) error {
	// Create sheet
	_, err := f.NewSheet(sheetDetail)
	if err != nil {
		return err
	}

	// Create styles
	headerStyle, err := w.createHeaderStyle(f)
	if err != nil {
		return err
	}

	warningStyle, err := w.createWarningStyle(f)
	if err != nil {
		return err
	}

	criticalStyle, err := w.createCriticalStyle(f)
	if err != nil {
		return err
	}

	normalStyle, err := w.createNormalStyle(f)
	if err != nil {
		return err
	}

	// Define headers
	headers := []string{
		"主机名", "IP地址", "状态", "操作系统", "系统版本", "内核版本",
		"CPU核心数", "CPU利用率", "内存利用率", "磁盘最大利用率",
		"运行时间", "1分钟负载", "每核负载", "僵尸进程", "总进程数",
	}

	// Get unique disk paths from all hosts
	diskPaths := w.collectDiskPaths(result.Hosts)
	for _, path := range diskPaths {
		headers = append(headers, fmt.Sprintf("磁盘:%s", path))
	}

	// Set column widths
	colWidths := map[string]float64{
		"A": 20, "B": 15, "C": 10, "D": 12, "E": 20, "F": 30,
		"G": 10, "H": 12, "I": 12, "J": 14,
		"K": 15, "L": 12, "M": 10, "N": 10, "O": 10,
	}
	for col, width := range colWidths {
		f.SetColWidth(sheetDetail, col, col, width)
	}

	// Set disk column widths
	for i := range diskPaths {
		col := columnName(16 + i) // Starting from column P
		f.SetColWidth(sheetDetail, col, col, 15)
	}

	// Write headers
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", columnName(i+1))
		f.SetCellValue(sheetDetail, cell, header)
		f.SetCellStyle(sheetDetail, cell, cell, headerStyle)
	}
	f.SetRowHeight(sheetDetail, 1, 25)

	// Freeze header row
	f.SetPanes(sheetDetail, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	// Write host data
	for i, host := range result.Hosts {
		row := i + 2 // Start from row 2
		rowStr := fmt.Sprintf("%d", row)

		// Basic info
		f.SetCellValue(sheetDetail, "A"+rowStr, host.Hostname)
		f.SetCellValue(sheetDetail, "B"+rowStr, host.IP)
		f.SetCellValue(sheetDetail, "C"+rowStr, statusText(host.Status))
		f.SetCellValue(sheetDetail, "D"+rowStr, host.OS)
		f.SetCellValue(sheetDetail, "E"+rowStr, host.OSVersion)
		f.SetCellValue(sheetDetail, "F"+rowStr, host.KernelVersion)
		f.SetCellValue(sheetDetail, "G"+rowStr, host.CPUCores)

		// Metrics
		w.setMetricCell(f, sheetDetail, "H"+rowStr, host.Metrics["cpu_usage"], warningStyle, criticalStyle, normalStyle)
		w.setMetricCell(f, sheetDetail, "I"+rowStr, host.Metrics["memory_usage"], warningStyle, criticalStyle, normalStyle)
		w.setMetricCell(f, sheetDetail, "J"+rowStr, host.Metrics["disk_usage_max"], warningStyle, criticalStyle, normalStyle)
		w.setMetricCell(f, sheetDetail, "K"+rowStr, host.Metrics["uptime"], 0, 0, 0)
		w.setMetricCell(f, sheetDetail, "L"+rowStr, host.Metrics["load_1m"], 0, 0, 0)
		w.setMetricCell(f, sheetDetail, "M"+rowStr, host.Metrics["load_per_core"], warningStyle, criticalStyle, normalStyle)
		w.setMetricCell(f, sheetDetail, "N"+rowStr, host.Metrics["processes_zombies"], warningStyle, criticalStyle, normalStyle)
		w.setMetricCell(f, sheetDetail, "O"+rowStr, host.Metrics["processes_total"], 0, 0, 0)

		// Disk usage by path
		for j, path := range diskPaths {
			col := columnName(16 + j)
			metricName := fmt.Sprintf("disk_usage:%s", path)
			w.setMetricCell(f, sheetDetail, col+rowStr, host.Metrics[metricName], warningStyle, criticalStyle, normalStyle)
		}

		// Apply status style to entire row
		statusStyle := w.getStatusStyle(host.Status, normalStyle, warningStyle, criticalStyle)
		if statusStyle > 0 {
			f.SetCellStyle(sheetDetail, "C"+rowStr, "C"+rowStr, statusStyle)
		}
	}

	return nil
}

// createAlertsSheet creates the alerts summary worksheet.
func (w *Writer) createAlertsSheet(f *excelize.File, result *model.InspectionResult) error {
	// Create sheet
	_, err := f.NewSheet(sheetAlerts)
	if err != nil {
		return err
	}

	// Create styles
	headerStyle, err := w.createHeaderStyle(f)
	if err != nil {
		return err
	}

	warningStyle, err := w.createWarningStyle(f)
	if err != nil {
		return err
	}

	criticalStyle, err := w.createCriticalStyle(f)
	if err != nil {
		return err
	}

	// Define headers
	headers := []string{"主机名", "告警级别", "指标名称", "当前值", "警告阈值", "严重阈值", "告警消息"}

	// Set column widths
	colWidths := []float64{20, 12, 15, 15, 12, 12, 40}
	for i, width := range colWidths {
		col := columnName(i + 1)
		f.SetColWidth(sheetAlerts, col, col, width)
	}

	// Write headers
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", columnName(i+1))
		f.SetCellValue(sheetAlerts, cell, header)
		f.SetCellStyle(sheetAlerts, cell, cell, headerStyle)
	}
	f.SetRowHeight(sheetAlerts, 1, 25)

	// Freeze header row
	f.SetPanes(sheetAlerts, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	// Sort alerts by level (critical first) then by hostname
	alerts := make([]*model.Alert, len(result.Alerts))
	copy(alerts, result.Alerts)
	sort.Slice(alerts, func(i, j int) bool {
		if alerts[i].Level != alerts[j].Level {
			return alertLevelPriority(alerts[i].Level) > alertLevelPriority(alerts[j].Level)
		}
		return alerts[i].Hostname < alerts[j].Hostname
	})

	// Write alert data
	for i, alert := range alerts {
		row := i + 2
		rowStr := fmt.Sprintf("%d", row)

		f.SetCellValue(sheetAlerts, "A"+rowStr, alert.Hostname)
		f.SetCellValue(sheetAlerts, "B"+rowStr, alertLevelText(alert.Level))
		f.SetCellValue(sheetAlerts, "C"+rowStr, alert.MetricDisplayName)
		f.SetCellValue(sheetAlerts, "D"+rowStr, alert.FormattedValue)
		f.SetCellValue(sheetAlerts, "E"+rowStr, formatThreshold(alert.WarningThreshold, alert.MetricName))
		f.SetCellValue(sheetAlerts, "F"+rowStr, formatThreshold(alert.CriticalThreshold, alert.MetricName))
		f.SetCellValue(sheetAlerts, "G"+rowStr, alert.Message)

		// Apply style based on alert level
		var style int
		if alert.Level == model.AlertLevelCritical {
			style = criticalStyle
		} else if alert.Level == model.AlertLevelWarning {
			style = warningStyle
		}
		if style > 0 {
			f.SetCellStyle(sheetAlerts, "B"+rowStr, "B"+rowStr, style)
		}
	}

	return nil
}

// Helper functions

func (w *Writer) createHeaderStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Size:  11,
			Color: colorHeaderFg,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{colorHeaderBg},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
}

func (w *Writer) createWarningStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Color: colorWarningFg,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{colorWarningBg},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
}

func (w *Writer) createCriticalStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Color: colorCriticalFg,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{colorCriticalBg},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
}

func (w *Writer) createNormalStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Color: colorNormalFg,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{colorNormalBg},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
}

func (w *Writer) setMetricCell(f *excelize.File, sheet, cell string, metric *model.MetricValue, warningStyle, criticalStyle, normalStyle int) {
	if metric == nil || metric.IsNA {
		f.SetCellValue(sheet, cell, "N/A")
		return
	}

	f.SetCellValue(sheet, cell, metric.FormattedValue)

	// Apply style based on metric status
	var style int
	switch metric.Status {
	case model.MetricStatusCritical:
		style = criticalStyle
	case model.MetricStatusWarning:
		style = warningStyle
	case model.MetricStatusNormal:
		// Only apply normal style if styles are provided
		if normalStyle > 0 && warningStyle > 0 && criticalStyle > 0 {
			// Don't apply normal style to avoid visual clutter
		}
	}
	if style > 0 {
		f.SetCellStyle(sheet, cell, cell, style)
	}
}

func (w *Writer) collectDiskPaths(hosts []*model.HostResult) []string {
	pathSet := make(map[string]bool)
	for _, host := range hosts {
		for name := range host.Metrics {
			if strings.HasPrefix(name, "disk_usage:") {
				path := strings.TrimPrefix(name, "disk_usage:")
				pathSet[path] = true
			}
		}
	}

	paths := make([]string, 0, len(pathSet))
	for path := range pathSet {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func (w *Writer) getStatusStyle(status model.HostStatus, normalStyle, warningStyle, criticalStyle int) int {
	switch status {
	case model.HostStatusCritical:
		return criticalStyle
	case model.HostStatusWarning:
		return warningStyle
	case model.HostStatusNormal:
		return normalStyle
	default:
		return 0
	}
}

// columnName converts a 1-based column index to Excel column name (A, B, ..., Z, AA, AB, ...).
func columnName(index int) string {
	result := ""
	for index > 0 {
		index--
		result = string(rune('A'+index%26)) + result
		index /= 26
	}
	return result
}

// formatDuration formats a duration in a human-readable format.
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1f秒", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1f分钟", d.Minutes())
	}
	return fmt.Sprintf("%.1f小时", d.Hours())
}

// statusText converts host status to Chinese text.
func statusText(status model.HostStatus) string {
	switch status {
	case model.HostStatusNormal:
		return "正常"
	case model.HostStatusWarning:
		return "警告"
	case model.HostStatusCritical:
		return "严重"
	case model.HostStatusFailed:
		return "失败"
	default:
		return "未知"
	}
}

// alertLevelText converts alert level to Chinese text.
func alertLevelText(level model.AlertLevel) string {
	switch level {
	case model.AlertLevelNormal:
		return "正常"
	case model.AlertLevelWarning:
		return "警告"
	case model.AlertLevelCritical:
		return "严重"
	default:
		return "未知"
	}
}

// alertLevelPriority returns a numeric priority for sorting (higher = more severe).
func alertLevelPriority(level model.AlertLevel) int {
	switch level {
	case model.AlertLevelCritical:
		return 2
	case model.AlertLevelWarning:
		return 1
	default:
		return 0
	}
}

// formatThreshold formats a threshold value based on metric type.
func formatThreshold(value float64, metricName string) string {
	switch metricName {
	case "cpu_usage", "memory_usage", "disk_usage_max":
		return fmt.Sprintf("%.1f%%", value)
	case "load_per_core":
		return fmt.Sprintf("%.2f", value)
	case "processes_zombies":
		return fmt.Sprintf("%.0f", value)
	default:
		return fmt.Sprintf("%.2f", value)
	}
}

// ============================================================================
// MySQL Report Helper Functions
// ============================================================================

// mysqlStatusText converts MySQL instance status to Chinese text.
func mysqlStatusText(status model.MySQLInstanceStatus) string {
	switch status {
	case model.MySQLStatusNormal:
		return "正常"
	case model.MySQLStatusWarning:
		return "警告"
	case model.MySQLStatusCritical:
		return "严重"
	case model.MySQLStatusFailed:
		return "失败"
	default:
		return "未知"
	}
}

// mysqlClusterModeText converts MySQL cluster mode to Chinese text.
func mysqlClusterModeText(mode model.MySQLClusterMode) string {
	switch mode {
	case model.ClusterModeMGR:
		return "MGR"
	case model.ClusterModeDualMaster:
		return "双主"
	case model.ClusterModeMasterSlave:
		return "主从"
	default:
		return "未知"
	}
}

// boolToText converts boolean to Chinese text (启用/禁用).
func boolToText(b bool) string {
	if b {
		return "启用"
	}
	return "禁用"
}

// getMySQLSyncStatus returns sync status text based on cluster mode.
func (w *Writer) getMySQLSyncStatus(r *model.MySQLInspectionResult) string {
	if r.Instance.ClusterMode.IsMGR() {
		if r.MGRStateOnline {
			return "在线"
		}
		return "离线"
	}
	if r.SyncStatus {
		return "正常"
	}
	return "异常"
}

// formatMySQLThreshold formats a MySQL alert threshold value based on metric type.
func formatMySQLThreshold(value float64, metricName string) string {
	switch metricName {
	case "connection_usage":
		return fmt.Sprintf("%.1f%%", value)
	case "mgr_member_count":
		return fmt.Sprintf("%.0f", value)
	case "mgr_state_online":
		if value > 0 {
			return "在线"
		}
		return "离线"
	default:
		return fmt.Sprintf("%.2f", value)
	}
}

// ============================================================================
// MySQL Report Methods
// ============================================================================

// WriteMySQLInspection generates an Excel report for MySQL inspection results.
func (w *Writer) WriteMySQLInspection(result *model.MySQLInspectionResults, outputPath string) error {
	if result == nil {
		return fmt.Errorf("MySQL inspection result is nil")
	}

	// Ensure output path has .xlsx extension
	if !strings.HasSuffix(strings.ToLower(outputPath), ".xlsx") {
		outputPath = outputPath + ".xlsx"
	}

	// Create new Excel file
	f := excelize.NewFile()
	defer f.Close()

	// Create MySQL sheet
	if err := w.createMySQLSheet(f, result); err != nil {
		return fmt.Errorf("failed to create MySQL sheet: %w", err)
	}

	// Create MySQL alerts sheet
	if err := w.createMySQLAlertsSheet(f, result); err != nil {
		return fmt.Errorf("failed to create MySQL alerts sheet: %w", err)
	}

	// Remove default Sheet1
	if err := f.DeleteSheet(defaultSheet); err != nil {
		// Ignore error if sheet doesn't exist
	}

	// Set active sheet to MySQL
	idx, _ := f.GetSheetIndex(sheetMySQL)
	f.SetActiveSheet(idx)

	// Save the file
	if err := f.SaveAs(outputPath); err != nil {
		return fmt.Errorf("failed to save Excel file: %w", err)
	}

	return nil
}

// createMySQLSheet creates the MySQL inspection data worksheet.
func (w *Writer) createMySQLSheet(f *excelize.File, result *model.MySQLInspectionResults) error {
	// Create sheet
	_, err := f.NewSheet(sheetMySQL)
	if err != nil {
		return err
	}

	// Create styles
	headerStyle, err := w.createHeaderStyle(f)
	if err != nil {
		return err
	}

	warningStyle, err := w.createWarningStyle(f)
	if err != nil {
		return err
	}

	criticalStyle, err := w.createCriticalStyle(f)
	if err != nil {
		return err
	}

	normalStyle, err := w.createNormalStyle(f)
	if err != nil {
		return err
	}

	// Define headers
	headers := []string{
		"巡检时间", "IP地址", "端口", "数据库版本", "Server ID",
		"集群模式", "同步状态", "最大连接数", "当前连接数", "Binlog状态", "整体状态",
	}

	// Set column widths
	colWidths := map[string]float64{
		"A": 20, // 巡检时间
		"B": 15, // IP地址
		"C": 8,  // 端口
		"D": 12, // 数据库版本
		"E": 12, // Server ID
		"F": 12, // 集群模式
		"G": 10, // 同步状态
		"H": 12, // 最大连接数
		"I": 12, // 当前连接数
		"J": 12, // Binlog状态
		"K": 10, // 整体状态
	}
	for col, width := range colWidths {
		f.SetColWidth(sheetMySQL, col, col, width)
	}

	// Write headers
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", columnName(i+1))
		f.SetCellValue(sheetMySQL, cell, header)
		f.SetCellStyle(sheetMySQL, cell, cell, headerStyle)
	}
	f.SetRowHeight(sheetMySQL, 1, 25)

	// Freeze header row
	f.SetPanes(sheetMySQL, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	// Write MySQL instance data
	for i, r := range result.Results {
		row := i + 2 // Start from row 2
		rowStr := fmt.Sprintf("%d", row)

		// A: 巡检时间
		f.SetCellValue(sheetMySQL, "A"+rowStr, result.InspectionTime.In(w.timezone).Format("2006-01-02 15:04:05"))
		// B: IP地址
		f.SetCellValue(sheetMySQL, "B"+rowStr, r.Instance.IP)
		// C: 端口
		f.SetCellValue(sheetMySQL, "C"+rowStr, r.Instance.Port)
		// D: 数据库版本
		f.SetCellValue(sheetMySQL, "D"+rowStr, r.Instance.Version)
		// E: Server ID
		f.SetCellValue(sheetMySQL, "E"+rowStr, r.Instance.ServerID)
		// F: 集群模式
		f.SetCellValue(sheetMySQL, "F"+rowStr, mysqlClusterModeText(r.Instance.ClusterMode))
		// G: 同步状态
		f.SetCellValue(sheetMySQL, "G"+rowStr, w.getMySQLSyncStatus(r))
		// H: 最大连接数
		f.SetCellValue(sheetMySQL, "H"+rowStr, r.MaxConnections)
		// I: 当前连接数
		f.SetCellValue(sheetMySQL, "I"+rowStr, r.CurrentConnections)
		// J: Binlog状态
		f.SetCellValue(sheetMySQL, "J"+rowStr, boolToText(r.BinlogEnabled))
		// K: 整体状态
		f.SetCellValue(sheetMySQL, "K"+rowStr, mysqlStatusText(r.Status))

		// Apply conditional format to status column
		statusCell := "K" + rowStr
		switch r.Status {
		case model.MySQLStatusCritical:
			f.SetCellStyle(sheetMySQL, statusCell, statusCell, criticalStyle)
		case model.MySQLStatusWarning:
			f.SetCellStyle(sheetMySQL, statusCell, statusCell, warningStyle)
		case model.MySQLStatusNormal:
			f.SetCellStyle(sheetMySQL, statusCell, statusCell, normalStyle)
		}
	}

	return nil
}

// createMySQLAlertsSheet creates the MySQL alerts summary worksheet.
func (w *Writer) createMySQLAlertsSheet(f *excelize.File, result *model.MySQLInspectionResults) error {
	// Create sheet
	_, err := f.NewSheet(sheetMySQLAlerts)
	if err != nil {
		return err
	}

	// Create styles
	headerStyle, err := w.createHeaderStyle(f)
	if err != nil {
		return err
	}

	warningStyle, err := w.createWarningStyle(f)
	if err != nil {
		return err
	}

	criticalStyle, err := w.createCriticalStyle(f)
	if err != nil {
		return err
	}

	// Define headers
	headers := []string{"实例地址", "告警级别", "指标名称", "当前值", "警告阈值", "严重阈值", "告警消息"}

	// Set column widths
	colWidths := []float64{20, 12, 15, 15, 12, 12, 40}
	for i, width := range colWidths {
		col := columnName(i + 1)
		f.SetColWidth(sheetMySQLAlerts, col, col, width)
	}

	// Write headers
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", columnName(i+1))
		f.SetCellValue(sheetMySQLAlerts, cell, header)
		f.SetCellStyle(sheetMySQLAlerts, cell, cell, headerStyle)
	}
	f.SetRowHeight(sheetMySQLAlerts, 1, 25)

	// Freeze header row
	f.SetPanes(sheetMySQLAlerts, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	// Sort alerts by level (critical first) then by address
	alerts := make([]*model.MySQLAlert, len(result.Alerts))
	copy(alerts, result.Alerts)
	sort.Slice(alerts, func(i, j int) bool {
		if alerts[i].Level != alerts[j].Level {
			return alertLevelPriority(alerts[i].Level) > alertLevelPriority(alerts[j].Level)
		}
		return alerts[i].Address < alerts[j].Address
	})

	// Write alert data
	for i, alert := range alerts {
		row := i + 2
		rowStr := fmt.Sprintf("%d", row)

		f.SetCellValue(sheetMySQLAlerts, "A"+rowStr, alert.Address)
		f.SetCellValue(sheetMySQLAlerts, "B"+rowStr, alertLevelText(alert.Level))
		f.SetCellValue(sheetMySQLAlerts, "C"+rowStr, alert.MetricDisplayName)
		f.SetCellValue(sheetMySQLAlerts, "D"+rowStr, alert.FormattedValue)
		f.SetCellValue(sheetMySQLAlerts, "E"+rowStr, formatMySQLThreshold(alert.WarningThreshold, alert.MetricName))
		f.SetCellValue(sheetMySQLAlerts, "F"+rowStr, formatMySQLThreshold(alert.CriticalThreshold, alert.MetricName))
		f.SetCellValue(sheetMySQLAlerts, "G"+rowStr, alert.Message)

		// Apply style based on alert level
		var style int
		if alert.Level == model.AlertLevelCritical {
			style = criticalStyle
		} else if alert.Level == model.AlertLevelWarning {
			style = warningStyle
		}
		if style > 0 {
			f.SetCellStyle(sheetMySQLAlerts, "B"+rowStr, "B"+rowStr, style)
		}
	}

	return nil
}

// AppendMySQLInspection appends MySQL inspection data to an existing Excel file.
// This method opens an existing file and adds MySQL-specific worksheets.
func (w *Writer) AppendMySQLInspection(result *model.MySQLInspectionResults, existingPath string) error {
	if result == nil {
		return fmt.Errorf("MySQL inspection result is nil")
	}

	// Ensure path has .xlsx extension
	if !strings.HasSuffix(strings.ToLower(existingPath), ".xlsx") {
		existingPath = existingPath + ".xlsx"
	}

	// Open existing Excel file
	f, err := excelize.OpenFile(existingPath)
	if err != nil {
		return fmt.Errorf("failed to open existing file: %w", err)
	}
	defer f.Close()

	// Add MySQL inspection worksheet (reuse existing method)
	if err := w.createMySQLSheet(f, result); err != nil {
		return fmt.Errorf("failed to create MySQL sheet: %w", err)
	}

	// Add MySQL alerts worksheet (reuse existing method)
	if err := w.createMySQLAlertsSheet(f, result); err != nil {
		return fmt.Errorf("failed to create MySQL alerts sheet: %w", err)
	}

	// Save the file
	if err := f.Save(); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

// ============================================================================
// Redis Report Helper Functions
// ============================================================================

// redisStatusText converts Redis instance status to Chinese text.
func redisStatusText(status model.RedisInstanceStatus) string {
	switch status {
	case model.RedisStatusNormal:
		return "正常"
	case model.RedisStatusWarning:
		return "警告"
	case model.RedisStatusCritical:
		return "严重"
	case model.RedisStatusFailed:
		return "失败"
	default:
		return "未知"
	}
}

// redisRoleText converts Redis role to Chinese text.
func redisRoleText(role model.RedisRole) string {
	switch role {
	case model.RedisRoleMaster:
		return "主"
	case model.RedisRoleSlave:
		return "从"
	default:
		return "未知"
	}
}

// redisBoolText converts boolean to Chinese text for display (是/否).
func redisBoolText(b bool) string {
	if b {
		return "是"
	}
	return "否"
}

// formatReplicationLag formats replication lag in bytes to human-readable format.
func formatReplicationLag(lag int64) string {
	if lag <= 0 {
		return "0 B"
	}
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case lag >= GB:
		return fmt.Sprintf("%.2f GB", float64(lag)/float64(GB))
	case lag >= MB:
		return fmt.Sprintf("%.2f MB", float64(lag)/float64(MB))
	case lag >= KB:
		return fmt.Sprintf("%.2f KB", float64(lag)/float64(KB))
	default:
		return fmt.Sprintf("%d B", lag)
	}
}

// formatRedisThreshold formats a Redis alert threshold value based on metric type.
func formatRedisThreshold(value float64, metricName string) string {
	switch metricName {
	case "connection_usage":
		return fmt.Sprintf("%.1f%%", value)
	case "replication_lag":
		return formatReplicationLag(int64(value))
	case "master_link_status":
		if value > 0 {
			return "正常"
		}
		return "断开"
	default:
		return fmt.Sprintf("%.2f", value)
	}
}

// getMasterLinkStatusText returns master link status text based on role.
func (w *Writer) getMasterLinkStatusText(r *model.RedisInspectionResult) string {
	if r.Instance == nil || r.Instance.Role.IsMaster() {
		return "N/A"
	}
	return redisBoolText(r.MasterLinkStatus)
}

// getMasterPortText returns master port text (N/A for master nodes).
func (w *Writer) getMasterPortText(r *model.RedisInspectionResult) string {
	if r.Instance == nil || r.Instance.Role.IsMaster() {
		return "N/A"
	}
	if r.MasterPort == 0 {
		return "N/A"
	}
	return fmt.Sprintf("%d", r.MasterPort)
}

// getReplicationLagText returns replication lag text (N/A for master nodes).
func (w *Writer) getReplicationLagText(r *model.RedisInspectionResult) string {
	if r.Instance == nil || r.Instance.Role.IsMaster() {
		return "N/A"
	}
	return formatReplicationLag(r.ReplicationLag)
}

// ============================================================================
// Redis Report Methods
// ============================================================================

// WriteRedisInspection generates an Excel report for Redis inspection results.
func (w *Writer) WriteRedisInspection(result *model.RedisInspectionResults, outputPath string) error {
	if result == nil {
		return fmt.Errorf("Redis inspection result is nil")
	}

	// Ensure output path has .xlsx extension
	if !strings.HasSuffix(strings.ToLower(outputPath), ".xlsx") {
		outputPath = outputPath + ".xlsx"
	}

	// Create new Excel file
	f := excelize.NewFile()
	defer f.Close()

	// Create Redis sheet
	if err := w.createRedisSheet(f, result); err != nil {
		return fmt.Errorf("failed to create Redis sheet: %w", err)
	}

	// Create Redis alerts sheet
	if err := w.createRedisAlertsSheet(f, result); err != nil {
		return fmt.Errorf("failed to create Redis alerts sheet: %w", err)
	}

	// Remove default Sheet1
	if err := f.DeleteSheet(defaultSheet); err != nil {
		// Ignore error if sheet doesn't exist
	}

	// Set active sheet to Redis
	idx, _ := f.GetSheetIndex(sheetRedis)
	f.SetActiveSheet(idx)

	// Save the file
	if err := f.SaveAs(outputPath); err != nil {
		return fmt.Errorf("failed to save Excel file: %w", err)
	}

	return nil
}

// createRedisSheet creates the Redis inspection data worksheet.
func (w *Writer) createRedisSheet(f *excelize.File, result *model.RedisInspectionResults) error {
	// Create sheet
	_, err := f.NewSheet(sheetRedis)
	if err != nil {
		return err
	}

	// Create styles
	headerStyle, err := w.createHeaderStyle(f)
	if err != nil {
		return err
	}

	warningStyle, err := w.createWarningStyle(f)
	if err != nil {
		return err
	}

	criticalStyle, err := w.createCriticalStyle(f)
	if err != nil {
		return err
	}

	normalStyle, err := w.createNormalStyle(f)
	if err != nil {
		return err
	}

	// Define headers
	headers := []string{
		"巡检时间", "IP地址", "端口", "应用类型", "Redis版本",
		"是否普通用户启动", "连接状态", "集群模式", "主从链接状态",
		"节点角色", "Master端口", "复制延迟", "最大连接数", "整体状态",
	}

	// Set column widths
	colWidths := map[string]float64{
		"A": 18, // 巡检时间
		"B": 15, // IP地址
		"C": 8,  // 端口
		"D": 8,  // 应用类型
		"E": 12, // Redis版本
		"F": 15, // 是否普通用户启动
		"G": 10, // 连接状态
		"H": 10, // 集群模式
		"I": 12, // 主从链接状态
		"J": 10, // 节点角色
		"K": 10, // Master端口
		"L": 12, // 复制延迟
		"M": 10, // 最大连接数
		"N": 10, // 整体状态
	}
	for col, width := range colWidths {
		f.SetColWidth(sheetRedis, col, col, width)
	}

	// Write headers
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", columnName(i+1))
		f.SetCellValue(sheetRedis, cell, header)
		f.SetCellStyle(sheetRedis, cell, cell, headerStyle)
	}
	f.SetRowHeight(sheetRedis, 1, 25)

	// Freeze header row
	f.SetPanes(sheetRedis, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	// Write Redis instance data
	for i, r := range result.Results {
		row := i + 2 // Start from row 2
		rowStr := fmt.Sprintf("%d", row)

		// A: 巡检时间
		f.SetCellValue(sheetRedis, "A"+rowStr, result.InspectionTime.In(w.timezone).Format("2006-01-02 15:04:05"))
		// B: IP地址
		if r.Instance != nil {
			f.SetCellValue(sheetRedis, "B"+rowStr, r.Instance.IP)
		}
		// C: 端口
		if r.Instance != nil {
			f.SetCellValue(sheetRedis, "C"+rowStr, r.Instance.Port)
		}
		// D: 应用类型
		f.SetCellValue(sheetRedis, "D"+rowStr, "Redis")
		// E: Redis版本
		if r.Instance != nil && r.Instance.Version != "" {
			f.SetCellValue(sheetRedis, "E"+rowStr, r.Instance.Version)
		} else {
			f.SetCellValue(sheetRedis, "E"+rowStr, "N/A")
		}
		// F: 是否普通用户启动
		f.SetCellValue(sheetRedis, "F"+rowStr, r.NonRootUser)
		// G: 连接状态
		f.SetCellValue(sheetRedis, "G"+rowStr, redisBoolText(r.ConnectionStatus))
		// H: 集群模式
		f.SetCellValue(sheetRedis, "H"+rowStr, redisBoolText(r.ClusterEnabled))
		// I: 主从链接状态
		f.SetCellValue(sheetRedis, "I"+rowStr, w.getMasterLinkStatusText(r))
		// J: 节点角色
		if r.Instance != nil {
			f.SetCellValue(sheetRedis, "J"+rowStr, redisRoleText(r.Instance.Role))
		} else {
			f.SetCellValue(sheetRedis, "J"+rowStr, "未知")
		}
		// K: Master端口
		f.SetCellValue(sheetRedis, "K"+rowStr, w.getMasterPortText(r))
		// L: 复制延迟
		f.SetCellValue(sheetRedis, "L"+rowStr, w.getReplicationLagText(r))
		// M: 最大连接数
		f.SetCellValue(sheetRedis, "M"+rowStr, r.MaxClients)
		// N: 整体状态
		f.SetCellValue(sheetRedis, "N"+rowStr, redisStatusText(r.Status))

		// Apply conditional format to status column
		statusCell := "N" + rowStr
		switch r.Status {
		case model.RedisStatusCritical:
			f.SetCellStyle(sheetRedis, statusCell, statusCell, criticalStyle)
		case model.RedisStatusWarning:
			f.SetCellStyle(sheetRedis, statusCell, statusCell, warningStyle)
		case model.RedisStatusNormal:
			f.SetCellStyle(sheetRedis, statusCell, statusCell, normalStyle)
		}
	}

	return nil
}

// createRedisAlertsSheet creates the Redis alerts summary worksheet.
func (w *Writer) createRedisAlertsSheet(f *excelize.File, result *model.RedisInspectionResults) error {
	// Create sheet
	_, err := f.NewSheet(sheetRedisAlerts)
	if err != nil {
		return err
	}

	// Create styles
	headerStyle, err := w.createHeaderStyle(f)
	if err != nil {
		return err
	}

	warningStyle, err := w.createWarningStyle(f)
	if err != nil {
		return err
	}

	criticalStyle, err := w.createCriticalStyle(f)
	if err != nil {
		return err
	}

	// Define headers
	headers := []string{"实例地址", "告警级别", "指标名称", "当前值", "警告阈值", "严重阈值", "告警消息"}

	// Set column widths
	colWidths := []float64{20, 12, 15, 15, 12, 12, 40}
	for i, width := range colWidths {
		col := columnName(i + 1)
		f.SetColWidth(sheetRedisAlerts, col, col, width)
	}

	// Write headers
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", columnName(i+1))
		f.SetCellValue(sheetRedisAlerts, cell, header)
		f.SetCellStyle(sheetRedisAlerts, cell, cell, headerStyle)
	}
	f.SetRowHeight(sheetRedisAlerts, 1, 25)

	// Freeze header row
	f.SetPanes(sheetRedisAlerts, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	// Sort alerts by level (critical first) then by address
	alerts := make([]*model.RedisAlert, len(result.Alerts))
	copy(alerts, result.Alerts)
	sort.Slice(alerts, func(i, j int) bool {
		if alerts[i].Level != alerts[j].Level {
			return alertLevelPriority(alerts[i].Level) > alertLevelPriority(alerts[j].Level)
		}
		return alerts[i].Address < alerts[j].Address
	})

	// Write alert data
	for i, alert := range alerts {
		row := i + 2
		rowStr := fmt.Sprintf("%d", row)

		f.SetCellValue(sheetRedisAlerts, "A"+rowStr, alert.Address)
		f.SetCellValue(sheetRedisAlerts, "B"+rowStr, alertLevelText(alert.Level))
		f.SetCellValue(sheetRedisAlerts, "C"+rowStr, alert.MetricDisplayName)
		f.SetCellValue(sheetRedisAlerts, "D"+rowStr, alert.FormattedValue)
		f.SetCellValue(sheetRedisAlerts, "E"+rowStr, formatRedisThreshold(alert.WarningThreshold, alert.MetricName))
		f.SetCellValue(sheetRedisAlerts, "F"+rowStr, formatRedisThreshold(alert.CriticalThreshold, alert.MetricName))
		f.SetCellValue(sheetRedisAlerts, "G"+rowStr, alert.Message)

		// Apply style based on alert level
		var style int
		if alert.Level == model.AlertLevelCritical {
			style = criticalStyle
		} else if alert.Level == model.AlertLevelWarning {
			style = warningStyle
		}
		if style > 0 {
			f.SetCellStyle(sheetRedisAlerts, "B"+rowStr, "B"+rowStr, style)
		}
	}

	return nil
}

// AppendRedisInspection appends Redis inspection data to an existing Excel file.
// This method opens an existing file and adds Redis-specific worksheets.
// If multiple clusters are detected, it creates separate sheets for each cluster.
func (w *Writer) AppendRedisInspection(result *model.RedisInspectionResults, existingPath string) error {
	if result == nil {
		return fmt.Errorf("Redis inspection result is nil")
	}

	// Ensure path has .xlsx extension
	if !strings.HasSuffix(strings.ToLower(existingPath), ".xlsx") {
		existingPath = existingPath + ".xlsx"
	}

	// Open existing Excel file
	f, err := excelize.OpenFile(existingPath)
	if err != nil {
		return fmt.Errorf("failed to open existing file: %w", err)
	}
	defer f.Close()

	// Check if multiple clusters exist
	if result.HasMultipleClusters() {
		// Create separate sheet for each cluster
		for _, cluster := range result.Clusters {
			if err := w.createRedisClusterSheet(f, cluster, result.InspectionTime); err != nil {
				return fmt.Errorf("failed to create Redis cluster sheet for %s: %w", cluster.ID, err)
			}
		}
		// Create a combined alerts sheet for all clusters
		if err := w.createRedisAlertsSheet(f, result); err != nil {
			return fmt.Errorf("failed to create Redis alerts sheet: %w", err)
		}
	} else {
		// Single cluster: use original flat display
		if err := w.createRedisSheet(f, result); err != nil {
			return fmt.Errorf("failed to create Redis sheet: %w", err)
		}
		if err := w.createRedisAlertsSheet(f, result); err != nil {
			return fmt.Errorf("failed to create Redis alerts sheet: %w", err)
		}
	}

	// Save the file
	if err := f.Save(); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

// createRedisClusterSheet creates a Redis inspection worksheet for a specific cluster.
// Sheet name format: "Redis-{网段ID}", e.g., "Redis-192.18.102"
func (w *Writer) createRedisClusterSheet(f *excelize.File, cluster *model.RedisCluster, inspectionTime time.Time) error {
	if cluster == nil {
		return fmt.Errorf("cluster is nil")
	}

	// Sheet name: Redis-{网段}
	sheetName := fmt.Sprintf("Redis-%s", cluster.ID)

	// Create sheet
	_, err := f.NewSheet(sheetName)
	if err != nil {
		return err
	}

	// Create styles
	headerStyle, err := w.createHeaderStyle(f)
	if err != nil {
		return err
	}

	warningStyle, err := w.createWarningStyle(f)
	if err != nil {
		return err
	}

	criticalStyle, err := w.createCriticalStyle(f)
	if err != nil {
		return err
	}

	normalStyle, err := w.createNormalStyle(f)
	if err != nil {
		return err
	}

	// Define headers (same as createRedisSheet)
	headers := []string{
		"巡检时间", "IP地址", "端口", "应用类型", "Redis版本",
		"是否普通用户启动", "连接状态", "集群模式", "主从链接状态",
		"节点角色", "Master端口", "复制延迟", "最大连接数", "整体状态",
	}

	// Set column widths
	colWidths := map[string]float64{
		"A": 18, "B": 15, "C": 8, "D": 8, "E": 12,
		"F": 15, "G": 10, "H": 10, "I": 12,
		"J": 10, "K": 10, "L": 12, "M": 10, "N": 10,
	}
	for col, width := range colWidths {
		f.SetColWidth(sheetName, col, col, width)
	}

	// Write headers
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", columnName(i+1))
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}
	f.SetRowHeight(sheetName, 1, 25)

	// Freeze header row
	f.SetPanes(sheetName, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	// Write Redis instance data for this cluster
	for i, r := range cluster.Instances {
		row := i + 2
		rowStr := fmt.Sprintf("%d", row)

		// A: 巡检时间
		f.SetCellValue(sheetName, "A"+rowStr, inspectionTime.In(w.timezone).Format("2006-01-02 15:04:05"))
		// B: IP地址
		if r.Instance != nil {
			f.SetCellValue(sheetName, "B"+rowStr, r.Instance.IP)
		}
		// C: 端口
		if r.Instance != nil {
			f.SetCellValue(sheetName, "C"+rowStr, r.Instance.Port)
		}
		// D: 应用类型
		f.SetCellValue(sheetName, "D"+rowStr, "Redis")
		// E: Redis版本
		if r.Instance != nil && r.Instance.Version != "" {
			f.SetCellValue(sheetName, "E"+rowStr, r.Instance.Version)
		} else {
			f.SetCellValue(sheetName, "E"+rowStr, "N/A")
		}
		// F: 是否普通用户启动
		f.SetCellValue(sheetName, "F"+rowStr, r.NonRootUser)
		// G: 连接状态
		f.SetCellValue(sheetName, "G"+rowStr, redisBoolText(r.ConnectionStatus))
		// H: 集群模式
		f.SetCellValue(sheetName, "H"+rowStr, redisBoolText(r.ClusterEnabled))
		// I: 主从链接状态
		f.SetCellValue(sheetName, "I"+rowStr, w.getMasterLinkStatusText(r))
		// J: 节点角色
		if r.Instance != nil {
			f.SetCellValue(sheetName, "J"+rowStr, redisRoleText(r.Instance.Role))
		} else {
			f.SetCellValue(sheetName, "J"+rowStr, "未知")
		}
		// K: Master端口
		f.SetCellValue(sheetName, "K"+rowStr, w.getMasterPortText(r))
		// L: 复制延迟
		f.SetCellValue(sheetName, "L"+rowStr, w.getReplicationLagText(r))
		// M: 最大连接数
		f.SetCellValue(sheetName, "M"+rowStr, r.MaxClients)
		// N: 整体状态
		f.SetCellValue(sheetName, "N"+rowStr, redisStatusText(r.Status))

		// Apply conditional format to status column
		statusCell := "N" + rowStr
		switch r.Status {
		case model.RedisStatusCritical:
			f.SetCellStyle(sheetName, statusCell, statusCell, criticalStyle)
		case model.RedisStatusWarning:
			f.SetCellStyle(sheetName, statusCell, statusCell, warningStyle)
		case model.RedisStatusNormal:
			f.SetCellStyle(sheetName, statusCell, statusCell, normalStyle)
		}
	}

	return nil
}
