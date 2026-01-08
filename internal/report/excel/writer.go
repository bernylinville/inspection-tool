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
	sheetSummary       = "巡检概览"
	sheetBaselineCheck = "基线检查"
	sheetDetail        = "详细数据"
	sheetAlerts        = "异常汇总"
	sheetMySQL         = "MySQL 巡检"
	sheetRedis         = "Redis 巡检"
	sheetNginx         = "Nginx 巡检"
	sheetTomcat        = "Tomcat 巡检"

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

	if err := w.createBaselineCheckSheet(f, result); err != nil {
		return fmt.Errorf("failed to create baseline check sheet: %w", err)
	}

	if err := w.createDetailSheet(f, result); err != nil {
		return fmt.Errorf("failed to create detail sheet: %w", err)
	}

	unifiedAlerts := w.collectUnifiedAlerts(result, nil, nil, nil, nil)
	if len(unifiedAlerts) > 0 {
		if err := w.createUnifiedAlertsSheet(f, unifiedAlerts); err != nil {
			return fmt.Errorf("failed to create unified alerts sheet: %w", err)
		}
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
		return fmt.Errorf("create sheet %s: %w", sheetSummary, err)
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
		return fmt.Errorf("create header style for %s: %w", sheetSummary, err)
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
		return fmt.Errorf("create title style for %s: %w", sheetSummary, err)
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
		return fmt.Errorf("create value style for %s: %w", sheetSummary, err)
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
		return fmt.Errorf("create sheet %s: %w", sheetDetail, err)
	}

	// Create styles
	headerStyle, err := w.createHeaderStyle(f)
	if err != nil {
		return fmt.Errorf("create header style for %s: %w", sheetDetail, err)
	}

	warningStyle, err := w.createWarningStyle(f)
	if err != nil {
		return fmt.Errorf("create warning style for %s: %w", sheetDetail, err)
	}

	criticalStyle, err := w.createCriticalStyle(f)
	if err != nil {
		return fmt.Errorf("create critical style for %s: %w", sheetDetail, err)
	}

	normalStyle, err := w.createNormalStyle(f)
	if err != nil {
		return fmt.Errorf("create normal style for %s: %w", sheetDetail, err)
	}

	headers := []string{
		"主机名", "IP地址", "状态", "内核版本",
		"CPU利用率", "内存利用率", "内存空闲", "磁盘最大利用率",
		"运行时间", "NTP时间偏差", "僵尸进程", "打开句柄数", "句柄最大值",
	}

	diskPaths := w.collectDiskPaths(result.Hosts)
	for _, path := range diskPaths {
		headers = append(headers, fmt.Sprintf("磁盘:%s", path))
	}

	colWidths := map[string]float64{
		"A": 20, "B": 15, "C": 10, "D": 25,
		"E": 12, "F": 12, "G": 12, "H": 14,
		"I": 15, "J": 14, "K": 10, "L": 12, "M": 12,
	}
	for col, width := range colWidths {
		f.SetColWidth(sheetDetail, col, col, width)
	}

	for i := range diskPaths {
		col := columnName(14 + i)
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

		f.SetCellValue(sheetDetail, "A"+rowStr, host.Hostname)
		f.SetCellValue(sheetDetail, "B"+rowStr, host.IP)
		f.SetCellValue(sheetDetail, "C"+rowStr, statusText(host.Status))
		f.SetCellValue(sheetDetail, "D"+rowStr, host.KernelVersion)

		w.setMetricCell(f, sheetDetail, "E"+rowStr, host.Metrics["cpu_usage"], warningStyle, criticalStyle, normalStyle)
		w.setMetricCell(f, sheetDetail, "F"+rowStr, host.Metrics["memory_usage"], warningStyle, criticalStyle, normalStyle)
		w.setMemoryFreeCell(f, sheetDetail, "G"+rowStr, host.Metrics["memory_available"])
		w.setMetricCell(f, sheetDetail, "H"+rowStr, host.Metrics["disk_usage_max"], warningStyle, criticalStyle, normalStyle)
		w.setMetricCell(f, sheetDetail, "I"+rowStr, host.Metrics["uptime"], 0, 0, 0)
		w.setMetricCell(f, sheetDetail, "J"+rowStr, host.Metrics["ntp_offset"], warningStyle, criticalStyle, normalStyle)
		w.setMetricCell(f, sheetDetail, "K"+rowStr, host.Metrics["processes_zombies"], warningStyle, criticalStyle, normalStyle)
		w.setMetricCell(f, sheetDetail, "L"+rowStr, host.Metrics["open_files"], 0, 0, 0)
		w.setMetricCell(f, sheetDetail, "M"+rowStr, host.Metrics["max_files"], 0, 0, 0)

		for j, path := range diskPaths {
			col := columnName(14 + j)
			metricName := fmt.Sprintf("disk_usage:%s", path)
			w.setMetricCell(f, sheetDetail, col+rowStr, host.Metrics[metricName], warningStyle, criticalStyle, normalStyle)
		}

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
		return fmt.Errorf("create sheet %s: %w", sheetAlerts, err)
	}

	// Create styles
	headerStyle, err := w.createHeaderStyle(f)
	if err != nil {
		return fmt.Errorf("create header style for %s: %w", sheetAlerts, err)
	}

	warningStyle, err := w.createWarningStyle(f)
	if err != nil {
		return fmt.Errorf("create warning style for %s: %w", sheetAlerts, err)
	}

	criticalStyle, err := w.createCriticalStyle(f)
	if err != nil {
		return fmt.Errorf("create critical style for %s: %w", sheetAlerts, err)
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

var sysctlParamNames = []string{
	"net.ipv4.ip_local_port_range_min",
	"net.ipv4.ip_local_port_range_max",
	"net.netfilter.nf_conntrack_max",
	"net.ipv4.tcp_max_tw_buckets",
	"net.netfilter.nf_conntrack_tcp_timeout_fin_wait",
	"net.netfilter.nf_conntrack_tcp_timeout_time_wait",
	"net.netfilter.nf_conntrack_tcp_timeout_close_wait",
	"net.netfilter.nf_conntrack_tcp_timeout_established",
	"net.ipv4.tcp_tw_reuse",
	"net.ipv4.tcp_timestamps",
}

var sysctlDisplayNames = []string{
	"端口范围(最小)",
	"端口范围(最大)",
	"连接跟踪最大",
	"TIME_WAIT桶数",
	"FIN_WAIT超时",
	"TIME_WAIT超时",
	"CLOSE_WAIT超时",
	"ESTABLISHED超时",
	"tcp_tw_reuse",
	"tcp_timestamps",
}

func (w *Writer) createBaselineCheckSheet(f *excelize.File, result *model.InspectionResult) error {
	_, err := f.NewSheet(sheetBaselineCheck)
	if err != nil {
		return fmt.Errorf("create sheet %s: %w", sheetBaselineCheck, err)
	}

	headerStyle, err := w.createHeaderStyle(f)
	if err != nil {
		return fmt.Errorf("create header style for %s: %w", sheetBaselineCheck, err)
	}

	warningStyle, err := w.createWarningStyle(f)
	if err != nil {
		return fmt.Errorf("create warning style for %s: %w", sheetBaselineCheck, err)
	}

	criticalStyle, err := w.createCriticalStyle(f)
	if err != nil {
		return fmt.Errorf("create critical style for %s: %w", sheetBaselineCheck, err)
	}

	normalStyle, err := w.createNormalStyle(f)
	if err != nil {
		return fmt.Errorf("create normal style for %s: %w", sheetBaselineCheck, err)
	}

	headers := []string{
		"巡检时间", "主机名", "IP地址", "操作系统", "内核版本", "运行时间",
		"密码过期", "密码策略", "文件句柄", "公网访问",
	}
	headers = append(headers, sysctlDisplayNames...)

	colWidths := []float64{
		20, 20, 15, 25, 30, 15,
		30, 40, 15, 10,
		12, 12, 14, 14, 12, 12, 12, 12, 12, 12,
	}

	for i, width := range colWidths {
		col := columnName(i + 1)
		f.SetColWidth(sheetBaselineCheck, col, col, width)
	}

	for i, header := range headers {
		cell := fmt.Sprintf("%s1", columnName(i+1))
		f.SetCellValue(sheetBaselineCheck, cell, header)
		f.SetCellStyle(sheetBaselineCheck, cell, cell, headerStyle)
	}
	f.SetRowHeight(sheetBaselineCheck, 1, 25)

	f.SetPanes(sheetBaselineCheck, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	inspectionTimeStr := result.InspectionTime.In(w.timezone).Format("2006-01-02 15:04:05")

	for i, host := range result.Hosts {
		row := i + 2
		rowStr := fmt.Sprintf("%d", row)

		f.SetCellValue(sheetBaselineCheck, "A"+rowStr, inspectionTimeStr)
		f.SetCellValue(sheetBaselineCheck, "B"+rowStr, host.Hostname)
		f.SetCellValue(sheetBaselineCheck, "C"+rowStr, host.IP)
		f.SetCellValue(sheetBaselineCheck, "D"+rowStr, fmt.Sprintf("%s %s", host.OS, host.OSVersion))
		f.SetCellValue(sheetBaselineCheck, "E"+rowStr, host.KernelVersion)
		w.setMetricCell(f, sheetBaselineCheck, "F"+rowStr, host.Metrics["uptime"], 0, 0, 0)

		w.setExpandedMetricCell(f, sheetBaselineCheck, "G"+rowStr, "password_expiry", "user", host.Metrics)
		w.setExpandedMetricCell(f, sheetBaselineCheck, "H"+rowStr, "password_policy", "param", host.Metrics)
		w.setFileHandleCell(f, sheetBaselineCheck, "I"+rowStr, host.Metrics["open_files"], host.Metrics["max_files"], warningStyle, criticalStyle)
		w.setPublicNetworkCell(f, sheetBaselineCheck, "J"+rowStr, host.Metrics["public_network"], normalStyle, criticalStyle)

		for j, paramName := range sysctlParamNames {
			col := columnName(11 + j)
			metricName := fmt.Sprintf("sysctl_params:%s", paramName)
			w.setSysctlCell(f, sheetBaselineCheck, col+rowStr, host.Metrics[metricName])
		}
	}

	return nil
}

func (w *Writer) setFileHandleCell(f *excelize.File, sheet, cell string, openFiles, maxFiles *model.MetricValue, warningStyle, criticalStyle int) {
	if openFiles == nil || openFiles.IsNA || maxFiles == nil || maxFiles.IsNA {
		f.SetCellValue(sheet, cell, "N/A")
		return
	}
	usage := fmt.Sprintf("%.0f / %.0f", openFiles.RawValue, maxFiles.RawValue)
	f.SetCellValue(sheet, cell, usage)

	if maxFiles.RawValue > 0 {
		usagePercent := (openFiles.RawValue / maxFiles.RawValue) * 100
		if usagePercent >= 90 {
			f.SetCellStyle(sheet, cell, cell, criticalStyle)
		} else if usagePercent >= 70 {
			f.SetCellStyle(sheet, cell, cell, warningStyle)
		}
	}
}

func (w *Writer) setPublicNetworkCell(f *excelize.File, sheet, cell string, metric *model.MetricValue, normalStyle, criticalStyle int) {
	if metric == nil || metric.IsNA {
		f.SetCellValue(sheet, cell, "N/A")
		return
	}
	if metric.RawValue == 1 {
		f.SetCellValue(sheet, cell, "成功")
		f.SetCellStyle(sheet, cell, cell, criticalStyle)
	} else {
		f.SetCellValue(sheet, cell, "失败")
		f.SetCellStyle(sheet, cell, cell, normalStyle)
	}
}

func (w *Writer) setSysctlCell(f *excelize.File, sheet, cell string, metric *model.MetricValue) {
	if metric == nil || metric.IsNA {
		f.SetCellValue(sheet, cell, "N/A")
		return
	}
	f.SetCellValue(sheet, cell, fmt.Sprintf("%.0f", metric.RawValue))
}

func (w *Writer) createUnifiedAlertsSheet(f *excelize.File, unifiedAlerts []*model.UnifiedAlert) error {
	_, err := f.NewSheet(sheetAlerts)
	if err != nil {
		return fmt.Errorf("create sheet %s: %w", sheetAlerts, err)
	}

	headerStyle, err := w.createHeaderStyle(f)
	if err != nil {
		return fmt.Errorf("create header style for %s: %w", sheetAlerts, err)
	}

	warningStyle, err := w.createWarningStyle(f)
	if err != nil {
		return fmt.Errorf("create warning style for %s: %w", sheetAlerts, err)
	}

	criticalStyle, err := w.createCriticalStyle(f)
	if err != nil {
		return fmt.Errorf("create critical style for %s: %w", sheetAlerts, err)
	}

	headers := []string{"来源类型", "实例标识", "告警级别", "指标名称", "当前值", "警告阈值", "严重阈值", "告警消息"}

	colWidths := []float64{12, 25, 10, 18, 15, 12, 12, 45}
	for i, width := range colWidths {
		col := columnName(i + 1)
		f.SetColWidth(sheetAlerts, col, col, width)
	}

	for i, header := range headers {
		cell := fmt.Sprintf("%s1", columnName(i+1))
		f.SetCellValue(sheetAlerts, cell, header)
		f.SetCellStyle(sheetAlerts, cell, cell, headerStyle)
	}
	f.SetRowHeight(sheetAlerts, 1, 25)

	f.SetPanes(sheetAlerts, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	sort.Slice(unifiedAlerts, func(i, j int) bool {
		if unifiedAlerts[i].Level != unifiedAlerts[j].Level {
			return alertLevelPriority(unifiedAlerts[i].Level) > alertLevelPriority(unifiedAlerts[j].Level)
		}
		if unifiedAlerts[i].SourceType != unifiedAlerts[j].SourceType {
			return unifiedAlerts[i].SourceType < unifiedAlerts[j].SourceType
		}
		return unifiedAlerts[i].Identifier < unifiedAlerts[j].Identifier
	})

	for i, alert := range unifiedAlerts {
		row := i + 2
		rowStr := fmt.Sprintf("%d", row)

		f.SetCellValue(sheetAlerts, "A"+rowStr, string(alert.SourceType))
		f.SetCellValue(sheetAlerts, "B"+rowStr, alert.Identifier)
		f.SetCellValue(sheetAlerts, "C"+rowStr, alertLevelText(alert.Level))
		f.SetCellValue(sheetAlerts, "D"+rowStr, alert.MetricDisplayName)
		f.SetCellValue(sheetAlerts, "E"+rowStr, alert.FormattedValue)
		f.SetCellValue(sheetAlerts, "F"+rowStr, formatThreshold(alert.WarningThreshold, alert.MetricName))
		f.SetCellValue(sheetAlerts, "G"+rowStr, formatThreshold(alert.CriticalThreshold, alert.MetricName))
		f.SetCellValue(sheetAlerts, "H"+rowStr, alert.Message)

		var style int
		if alert.Level == model.AlertLevelCritical {
			style = criticalStyle
		} else if alert.Level == model.AlertLevelWarning {
			style = warningStyle
		}
		if style > 0 {
			f.SetCellStyle(sheetAlerts, "C"+rowStr, "C"+rowStr, style)
		}

		sourceStyle := w.getSourceTypeStyle(f, alert.SourceType)
		if sourceStyle > 0 {
			f.SetCellStyle(sheetAlerts, "A"+rowStr, "A"+rowStr, sourceStyle)
		}
	}

	return nil
}

func (w *Writer) getSourceTypeStyle(f *excelize.File, sourceType model.AlertSourceType) int {
	var bgColor string
	switch sourceType {
	case model.AlertSourceHost:
		bgColor = "D6E3F8"
	case model.AlertSourceMySQL:
		bgColor = "D5E8D4"
	case model.AlertSourceRedis:
		bgColor = "F8D7DA"
	case model.AlertSourceNginx:
		bgColor = "D4EDDA"
	case model.AlertSourceTomcat:
		bgColor = "FFE8CC"
	default:
		return 0
	}

	style, err := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{bgColor},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	if err != nil {
		return 0
	}
	return style
}

func (w *Writer) collectUnifiedAlerts(hostResult *model.InspectionResult, mysqlResult *model.MySQLInspectionResults, redisResult *model.RedisInspectionResults, nginxResult *model.NginxInspectionResults, tomcatResult *model.TomcatInspectionResults) []*model.UnifiedAlert {
	var alerts []*model.UnifiedAlert

	if hostResult != nil {
		for _, alert := range hostResult.Alerts {
			if u := model.NewUnifiedAlertFromHostAlert(alert); u != nil {
				alerts = append(alerts, u)
			}
		}
	}

	if mysqlResult != nil {
		for _, alert := range mysqlResult.Alerts {
			if u := model.NewUnifiedAlertFromMySQLAlert(alert); u != nil {
				alerts = append(alerts, u)
			}
		}
	}

	if redisResult != nil {
		for _, alert := range redisResult.Alerts {
			if u := model.NewUnifiedAlertFromRedisAlert(alert); u != nil {
				alerts = append(alerts, u)
			}
		}
	}

	if nginxResult != nil {
		for _, alert := range nginxResult.Alerts {
			if u := w.convertNginxAlert(alert); u != nil {
				alerts = append(alerts, u)
			}
		}
	}

	if tomcatResult != nil {
		for _, alert := range tomcatResult.Alerts {
			if u := w.convertTomcatAlert(alert); u != nil {
				alerts = append(alerts, u)
			}
		}
	}

	return alerts
}

func (w *Writer) convertNginxAlert(alert *model.NginxAlert) *model.UnifiedAlert {
	if alert == nil {
		return nil
	}
	return &model.UnifiedAlert{
		SourceType:        model.AlertSourceNginx,
		Identifier:        alert.Identifier,
		Level:             alert.Level,
		MetricName:        alert.MetricName,
		MetricDisplayName: alert.MetricDisplayName,
		CurrentValue:      alert.CurrentValue,
		FormattedValue:    alert.FormattedValue,
		WarningThreshold:  alert.WarningThreshold,
		CriticalThreshold: alert.CriticalThreshold,
		Message:           alert.Message,
	}
}

func (w *Writer) convertTomcatAlert(alert *model.TomcatAlert) *model.UnifiedAlert {
	if alert == nil {
		return nil
	}
	return &model.UnifiedAlert{
		SourceType:        model.AlertSourceTomcat,
		Identifier:        alert.Identifier,
		Level:             alert.Level,
		MetricName:        alert.MetricName,
		MetricDisplayName: alert.MetricDisplayName,
		CurrentValue:      alert.CurrentValue,
		FormattedValue:    alert.FormattedValue,
		WarningThreshold:  alert.WarningThreshold,
		CriticalThreshold: alert.CriticalThreshold,
		Message:           alert.Message,
	}
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

	var style int
	switch metric.Status {
	case model.MetricStatusCritical:
		style = criticalStyle
	case model.MetricStatusWarning:
		style = warningStyle
	case model.MetricStatusNormal:
	}
	if style > 0 {
		f.SetCellStyle(sheet, cell, cell, style)
	}
}

func (w *Writer) setMemoryFreeCell(f *excelize.File, sheet, cell string, metric *model.MetricValue) {
	if metric == nil || metric.IsNA {
		f.SetCellValue(sheet, cell, "N/A")
		return
	}
	f.SetCellValue(sheet, cell, metric.FormattedValue)
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

func (w *Writer) setSecurityMetricCell(f *excelize.File, sheet, cell string, metric *model.MetricValue, _ map[string]*model.MetricValue) {
	if metric == nil || metric.IsNA {
		f.SetCellValue(sheet, cell, "N/A")
		return
	}
	if metric.RawValue == 1 {
		f.SetCellValue(sheet, cell, "成功")
	} else {
		f.SetCellValue(sheet, cell, "失败")
	}
}

func (w *Writer) setExpandedMetricCell(f *excelize.File, sheet, cell, metricPrefix, labelName string, metrics map[string]*model.MetricValue) {
	var parts []string
	for name, mv := range metrics {
		if strings.HasPrefix(name, metricPrefix+":") && mv != nil && !mv.IsNA {
			labelValue := strings.TrimPrefix(name, metricPrefix+":")
			switch metricPrefix {
			case "password_expiry":
				if mv.RawValue == -1 {
					parts = append(parts, fmt.Sprintf("%s:永不过期", labelValue))
				} else if mv.RawValue == -2 {
					parts = append(parts, fmt.Sprintf("%s:无法获取", labelValue))
				} else {
					parts = append(parts, fmt.Sprintf("%s:%.0f天", labelValue, mv.RawValue))
				}
			case "password_policy":
				parts = append(parts, fmt.Sprintf("%s=%.0f", labelValue, mv.RawValue))
			case "sysctl_params":
				parts = append(parts, fmt.Sprintf("%s=%.0f", labelValue, mv.RawValue))
			default:
				parts = append(parts, fmt.Sprintf("%s:%.2f", labelValue, mv.RawValue))
			}
		}
	}
	if len(parts) == 0 {
		f.SetCellValue(sheet, cell, "N/A")
		return
	}
	sort.Strings(parts)
	f.SetCellValue(sheet, cell, strings.Join(parts, ", "))
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

func (w *Writer) formatBinlogExpireDays(seconds int) string {
	if seconds <= 0 {
		return "N/A"
	}
	days := seconds / 86400
	return fmt.Sprintf("%d", days)
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

	if !strings.HasSuffix(strings.ToLower(outputPath), ".xlsx") {
		outputPath = outputPath + ".xlsx"
	}

	f := excelize.NewFile()
	defer f.Close()

	if err := w.createMySQLSheet(f, result); err != nil {
		return fmt.Errorf("failed to create MySQL sheet: %w", err)
	}

	if err := f.DeleteSheet(defaultSheet); err != nil {
	}

	idx, _ := f.GetSheetIndex(sheetMySQL)
	f.SetActiveSheet(idx)

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
		return fmt.Errorf("create sheet %s: %w", sheetMySQL, err)
	}

	// Create styles
	headerStyle, err := w.createHeaderStyle(f)
	if err != nil {
		return fmt.Errorf("create header style for %s: %w", sheetMySQL, err)
	}

	warningStyle, err := w.createWarningStyle(f)
	if err != nil {
		return fmt.Errorf("create warning style for %s: %w", sheetMySQL, err)
	}

	criticalStyle, err := w.createCriticalStyle(f)
	if err != nil {
		return fmt.Errorf("create critical style for %s: %w", sheetMySQL, err)
	}

	normalStyle, err := w.createNormalStyle(f)
	if err != nil {
		return fmt.Errorf("create normal style for %s: %w", sheetMySQL, err)
	}

	headers := []string{
		"巡检时间", "IP地址", "端口", "数据库版本", "Server ID",
		"集群模式", "同步状态", "最大连接数", "当前连接数",
		"慢查询日志", "Binlog状态", "Binlog保留(天)", "非root用户", "远程连接用户", "整体状态",
	}

	colWidths := map[string]float64{
		"A": 20, "B": 15, "C": 8, "D": 12, "E": 12,
		"F": 12, "G": 10, "H": 12, "I": 12,
		"J": 10, "K": 10, "L": 14, "M": 12, "N": 20, "O": 10,
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
		row := i + 2
		rowStr := fmt.Sprintf("%d", row)

		f.SetCellValue(sheetMySQL, "A"+rowStr, result.InspectionTime.In(w.timezone).Format("2006-01-02 15:04:05"))
		f.SetCellValue(sheetMySQL, "B"+rowStr, r.Instance.IP)
		f.SetCellValue(sheetMySQL, "C"+rowStr, r.Instance.Port)
		f.SetCellValue(sheetMySQL, "D"+rowStr, r.Instance.Version)
		f.SetCellValue(sheetMySQL, "E"+rowStr, r.Instance.ServerID)
		f.SetCellValue(sheetMySQL, "F"+rowStr, mysqlClusterModeText(r.Instance.ClusterMode))
		f.SetCellValue(sheetMySQL, "G"+rowStr, w.getMySQLSyncStatus(r))
		f.SetCellValue(sheetMySQL, "H"+rowStr, r.MaxConnections)
		f.SetCellValue(sheetMySQL, "I"+rowStr, r.CurrentConnections)
		f.SetCellValue(sheetMySQL, "J"+rowStr, boolToText(r.SlowQueryLogEnabled))
		f.SetCellValue(sheetMySQL, "K"+rowStr, boolToText(r.BinlogEnabled))
		f.SetCellValue(sheetMySQL, "L"+rowStr, w.formatBinlogExpireDays(r.BinlogExpireSeconds))
		f.SetCellValue(sheetMySQL, "M"+rowStr, r.NonRootUser)

		// 远程连接用户列
		if r.RemoteUsersCount > 0 && len(r.RemoteUsers) > 0 {
			f.SetCellValue(sheetMySQL, "N"+rowStr, strings.Join(r.RemoteUsers, ", "))
			// 有远程用户时显示警告色
			f.SetCellStyle(sheetMySQL, "N"+rowStr, "N"+rowStr, warningStyle)
		} else if r.RemoteUsersCount > 0 {
			f.SetCellValue(sheetMySQL, "N"+rowStr, fmt.Sprintf("%d 个用户", r.RemoteUsersCount))
			f.SetCellStyle(sheetMySQL, "N"+rowStr, "N"+rowStr, warningStyle)
		} else {
			f.SetCellValue(sheetMySQL, "N"+rowStr, "无")
		}

		f.SetCellValue(sheetMySQL, "O"+rowStr, mysqlStatusText(r.Status))

		statusCell := "O" + rowStr
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

// AppendMySQLInspection appends MySQL inspection data to an existing Excel file.
// This method opens an existing file and adds MySQL-specific worksheets.
func (w *Writer) AppendMySQLInspection(result *model.MySQLInspectionResults, existingPath string) error {
	if result == nil {
		return fmt.Errorf("MySQL inspection result is nil")
	}

	if !strings.HasSuffix(strings.ToLower(existingPath), ".xlsx") {
		existingPath = existingPath + ".xlsx"
	}

	f, err := excelize.OpenFile(existingPath)
	if err != nil {
		return fmt.Errorf("failed to open existing file: %w", err)
	}
	defer f.Close()

	if err := w.createMySQLSheet(f, result); err != nil {
		return fmt.Errorf("failed to create MySQL sheet: %w", err)
	}

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

func (w *Writer) getMasterHostText(r *model.RedisInspectionResult) string {
	if r.Instance == nil || r.Instance.Role.IsMaster() {
		return "N/A"
	}
	if r.MasterHost == "" {
		return "N/A"
	}
	return r.MasterHost
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

	if !strings.HasSuffix(strings.ToLower(outputPath), ".xlsx") {
		outputPath = outputPath + ".xlsx"
	}

	f := excelize.NewFile()
	defer f.Close()

	if err := w.createRedisSheet(f, result); err != nil {
		return fmt.Errorf("failed to create Redis sheet: %w", err)
	}

	if err := f.DeleteSheet(defaultSheet); err != nil {
	}

	idx, _ := f.GetSheetIndex(sheetRedis)
	f.SetActiveSheet(idx)

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
		return fmt.Errorf("create sheet %s: %w", sheetRedis, err)
	}

	// Create styles
	headerStyle, err := w.createHeaderStyle(f)
	if err != nil {
		return fmt.Errorf("create header style for %s: %w", sheetRedis, err)
	}

	warningStyle, err := w.createWarningStyle(f)
	if err != nil {
		return fmt.Errorf("create warning style for %s: %w", sheetRedis, err)
	}

	criticalStyle, err := w.createCriticalStyle(f)
	if err != nil {
		return fmt.Errorf("create critical style for %s: %w", sheetRedis, err)
	}

	normalStyle, err := w.createNormalStyle(f)
	if err != nil {
		return fmt.Errorf("create normal style for %s: %w", sheetRedis, err)
	}

	headers := []string{
		"巡检时间", "IP地址", "端口", "应用类型", "Redis版本",
		"普通用户启动", "连接状态", "集群模式", "主从链接状态",
		"节点角色", "Master节点IP", "复制延迟", "最大连接数", "整体状态",
	}

	colWidths := map[string]float64{
		"A": 18, "B": 15, "C": 8, "D": 8, "E": 12,
		"F": 12, "G": 10, "H": 10, "I": 12,
		"J": 10, "K": 15, "L": 12, "M": 10, "N": 10,
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
		row := i + 2
		rowStr := fmt.Sprintf("%d", row)

		f.SetCellValue(sheetRedis, "A"+rowStr, result.InspectionTime.In(w.timezone).Format("2006-01-02 15:04:05"))
		if r.Instance != nil {
			f.SetCellValue(sheetRedis, "B"+rowStr, r.Instance.IP)
			f.SetCellValue(sheetRedis, "C"+rowStr, r.Instance.Port)
		}
		f.SetCellValue(sheetRedis, "D"+rowStr, "Redis")
		if r.Instance != nil && r.Instance.Version != "" {
			f.SetCellValue(sheetRedis, "E"+rowStr, r.Instance.Version)
		} else {
			f.SetCellValue(sheetRedis, "E"+rowStr, "N/A")
		}
		f.SetCellValue(sheetRedis, "F"+rowStr, r.NonRootUser)
		f.SetCellValue(sheetRedis, "G"+rowStr, redisBoolText(r.ConnectionStatus))
		f.SetCellValue(sheetRedis, "H"+rowStr, redisBoolText(r.ClusterEnabled))
		f.SetCellValue(sheetRedis, "I"+rowStr, w.getMasterLinkStatusText(r))
		if r.Instance != nil {
			f.SetCellValue(sheetRedis, "J"+rowStr, redisRoleText(r.Instance.Role))
		} else {
			f.SetCellValue(sheetRedis, "J"+rowStr, "未知")
		}
		f.SetCellValue(sheetRedis, "K"+rowStr, w.getMasterHostText(r))
		f.SetCellValue(sheetRedis, "L"+rowStr, w.getReplicationLagText(r))
		f.SetCellValue(sheetRedis, "M"+rowStr, r.MaxClients)
		f.SetCellValue(sheetRedis, "N"+rowStr, redisStatusText(r.Status))

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

// AppendRedisInspection appends Redis inspection data to an existing Excel file.
// This method opens an existing file and adds Redis-specific worksheets.
// If multiple clusters are detected, it creates separate sheets for each cluster.
func (w *Writer) AppendRedisInspection(result *model.RedisInspectionResults, existingPath string) error {
	if result == nil {
		return fmt.Errorf("Redis inspection result is nil")
	}

	if !strings.HasSuffix(strings.ToLower(existingPath), ".xlsx") {
		existingPath = existingPath + ".xlsx"
	}

	f, err := excelize.OpenFile(existingPath)
	if err != nil {
		return fmt.Errorf("failed to open existing file: %w", err)
	}
	defer f.Close()

	if result.HasMultipleClusters() {
		for _, cluster := range result.Clusters {
			if err := w.createRedisClusterSheet(f, cluster, result.InspectionTime); err != nil {
				return fmt.Errorf("failed to create Redis cluster sheet for %s: %w", cluster.ID, err)
			}
		}
	} else {
		if err := w.createRedisSheet(f, result); err != nil {
			return fmt.Errorf("failed to create Redis sheet: %w", err)
		}
	}

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
		return fmt.Errorf("create sheet %s: %w", sheetName, err)
	}

	// Create styles
	headerStyle, err := w.createHeaderStyle(f)
	if err != nil {
		return fmt.Errorf("create header style for %s: %w", sheetName, err)
	}

	warningStyle, err := w.createWarningStyle(f)
	if err != nil {
		return fmt.Errorf("create warning style for %s: %w", sheetName, err)
	}

	criticalStyle, err := w.createCriticalStyle(f)
	if err != nil {
		return fmt.Errorf("create critical style for %s: %w", sheetName, err)
	}

	normalStyle, err := w.createNormalStyle(f)
	if err != nil {
		return fmt.Errorf("create normal style for %s: %w", sheetName, err)
	}

	headers := []string{
		"巡检时间", "IP地址", "端口", "应用类型", "Redis版本",
		"普通用户启动", "连接状态", "集群模式", "主从链接状态",
		"节点角色", "Master节点IP", "复制延迟", "最大连接数", "整体状态",
	}

	colWidths := map[string]float64{
		"A": 18, "B": 15, "C": 8, "D": 8, "E": 12,
		"F": 12, "G": 10, "H": 10, "I": 12,
		"J": 10, "K": 15, "L": 12, "M": 10, "N": 10,
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

		f.SetCellValue(sheetName, "A"+rowStr, inspectionTime.In(w.timezone).Format("2006-01-02 15:04:05"))
		if r.Instance != nil {
			f.SetCellValue(sheetName, "B"+rowStr, r.Instance.IP)
			f.SetCellValue(sheetName, "C"+rowStr, r.Instance.Port)
		}
		f.SetCellValue(sheetName, "D"+rowStr, "Redis")
		if r.Instance != nil && r.Instance.Version != "" {
			f.SetCellValue(sheetName, "E"+rowStr, r.Instance.Version)
		} else {
			f.SetCellValue(sheetName, "E"+rowStr, "N/A")
		}
		f.SetCellValue(sheetName, "F"+rowStr, r.NonRootUser)
		f.SetCellValue(sheetName, "G"+rowStr, redisBoolText(r.ConnectionStatus))
		f.SetCellValue(sheetName, "H"+rowStr, redisBoolText(r.ClusterEnabled))
		f.SetCellValue(sheetName, "I"+rowStr, w.getMasterLinkStatusText(r))
		if r.Instance != nil {
			f.SetCellValue(sheetName, "J"+rowStr, redisRoleText(r.Instance.Role))
		} else {
			f.SetCellValue(sheetName, "J"+rowStr, "未知")
		}
		f.SetCellValue(sheetName, "K"+rowStr, w.getMasterHostText(r))
		f.SetCellValue(sheetName, "L"+rowStr, w.getReplicationLagText(r))
		f.SetCellValue(sheetName, "M"+rowStr, r.MaxClients)
		f.SetCellValue(sheetName, "N"+rowStr, redisStatusText(r.Status))

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

// WriteCombined generates an Excel report combining Host, MySQL, Redis, Nginx, and Tomcat inspection results.
func (w *Writer) WriteCombined(hostResult *model.InspectionResult, mysqlResult *model.MySQLInspectionResults, redisResult *model.RedisInspectionResults, nginxResult *model.NginxInspectionResults, tomcatResult *model.TomcatInspectionResults, outputPath string) error {
	if hostResult == nil && mysqlResult == nil && redisResult == nil && nginxResult == nil && tomcatResult == nil {
		return fmt.Errorf("all inspection results are nil")
	}

	if !strings.HasSuffix(strings.ToLower(outputPath), ".xlsx") {
		outputPath = outputPath + ".xlsx"
	}

	f := excelize.NewFile()
	defer f.Close()

	if hostResult != nil {
		if err := w.createSummarySheet(f, hostResult); err != nil {
			return fmt.Errorf("failed to create summary sheet: %w", err)
		}
		if err := w.createBaselineCheckSheet(f, hostResult); err != nil {
			return fmt.Errorf("failed to create baseline check sheet: %w", err)
		}
		if err := w.createDetailSheet(f, hostResult); err != nil {
			return fmt.Errorf("failed to create detail sheet: %w", err)
		}
	}

	if mysqlResult != nil {
		if err := w.createMySQLSheet(f, mysqlResult); err != nil {
			return fmt.Errorf("failed to create MySQL sheet: %w", err)
		}
	}

	if redisResult != nil {
		if err := w.createRedisSheet(f, redisResult); err != nil {
			return fmt.Errorf("failed to create Redis sheet: %w", err)
		}
	}

	if nginxResult != nil {
		if err := w.createNginxSheet(f, nginxResult); err != nil {
			return fmt.Errorf("failed to create Nginx sheet: %w", err)
		}
	}

	if tomcatResult != nil {
		if err := w.createTomcatSheet(f, tomcatResult); err != nil {
			return fmt.Errorf("failed to create Tomcat sheet: %w", err)
		}
	}

	unifiedAlerts := w.collectUnifiedAlerts(hostResult, mysqlResult, redisResult, nginxResult, tomcatResult)
	if len(unifiedAlerts) > 0 {
		if err := w.createUnifiedAlertsSheet(f, unifiedAlerts); err != nil {
			return fmt.Errorf("failed to create unified alerts sheet: %w", err)
		}
	}

	if err := f.DeleteSheet(defaultSheet); err != nil {
	}

	activeSheet := sheetSummary
	if hostResult == nil {
		if mysqlResult != nil {
			activeSheet = sheetMySQL
		} else if redisResult != nil {
			activeSheet = sheetRedis
		} else if nginxResult != nil {
			activeSheet = sheetNginx
		} else if tomcatResult != nil {
			activeSheet = sheetTomcat
		}
	}
	idx, _ := f.GetSheetIndex(activeSheet)
	f.SetActiveSheet(idx)

	if err := f.SaveAs(outputPath); err != nil {
		return fmt.Errorf("failed to save Excel file: %w", err)
	}

	return nil
}

// createNginxSheet creates the Nginx inspection sheet.
func (w *Writer) createNginxSheet(f *excelize.File, result *model.NginxInspectionResults) error {
	// Create sheet
	_, err := f.NewSheet(sheetNginx)
	if err != nil {
		return fmt.Errorf("create sheet %s: %w", sheetNginx, err)
	}

	// Create styles
	headerStyle, err := w.createHeaderStyle(f)
	if err != nil {
		return fmt.Errorf("create header style for %s: %w", sheetNginx, err)
	}

	warningStyle, err := w.createWarningStyle(f)
	if err != nil {
		return fmt.Errorf("create warning style for %s: %w", sheetNginx, err)
	}

	criticalStyle, err := w.createCriticalStyle(f)
	if err != nil {
		return fmt.Errorf("create critical style for %s: %w", sheetNginx, err)
	}

	normalStyle, err := w.createNormalStyle(f)
	if err != nil {
		return fmt.Errorf("create normal style for %s: %w", sheetNginx, err)
	}

	headers := []string{
		"巡检时间", "IP地址", "端口", "版本", "运行状态",
		"非root用户", "活跃连接数", "Worker进程数", "Worker连接数",
		"5xx错误页", "整体状态",
	}

	colWidths := map[string]float64{
		"A": 20, "B": 15, "C": 8, "D": 15, "E": 10,
		"F": 12, "G": 12, "H": 12, "I": 12,
		"J": 10, "K": 10,
	}
	for col, width := range colWidths {
		f.SetColWidth(sheetNginx, col, col, width)
	}

	sheetName := sheetNginx
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", columnName(i+1))
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	// Freeze header row
	f.SetPanes(sheetName, &excelize.Panes{
		Freeze:      true,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	// Write Nginx instance data
	for i, r := range result.Results {
		row := i + 2
		rowStr := fmt.Sprintf("%d", row)

		f.SetCellValue(sheetName, "A"+rowStr, result.InspectionTime.In(w.timezone).Format("2006-01-02 15:04:05"))
		if r.Instance != nil {
			f.SetCellValue(sheetName, "B"+rowStr, r.Instance.IP)
			f.SetCellValue(sheetName, "C"+rowStr, r.Instance.Port)
			f.SetCellValue(sheetName, "D"+rowStr, r.Instance.Version)
		}
		if r.Up {
			f.SetCellValue(sheetName, "E"+rowStr, "运行")
		} else {
			f.SetCellValue(sheetName, "E"+rowStr, "停止")
		}
		if r.NonRootUser {
			f.SetCellValue(sheetName, "F"+rowStr, "是")
		} else {
			f.SetCellValue(sheetName, "F"+rowStr, "否")
		}
		f.SetCellValue(sheetName, "G"+rowStr, r.ActiveConnections)
		f.SetCellValue(sheetName, "H"+rowStr, r.WorkerProcesses)
		f.SetCellValue(sheetName, "I"+rowStr, r.WorkerConnections)
		if r.ErrorPage5xxConfigured {
			f.SetCellValue(sheetName, "J"+rowStr, "已配置")
		} else {
			f.SetCellValue(sheetName, "J"+rowStr, "未配置")
		}
		f.SetCellValue(sheetName, "K"+rowStr, nginxStatusText(r.Status))

		statusCell := "K" + rowStr
		switch r.Status {
		case model.NginxStatusCritical:
			f.SetCellStyle(sheetName, statusCell, statusCell, criticalStyle)
		case model.NginxStatusWarning:
			f.SetCellStyle(sheetName, statusCell, statusCell, warningStyle)
		case model.NginxStatusNormal:
			f.SetCellStyle(sheetName, statusCell, statusCell, normalStyle)
		}
	}

	return nil
}

// nginxStatusText converts NginxInstanceStatus to Chinese text.
func nginxStatusText(status model.NginxInstanceStatus) string {
	switch status {
	case model.NginxStatusNormal:
		return "正常"
	case model.NginxStatusWarning:
		return "警告"
	case model.NginxStatusCritical:
		return "严重"
	case model.NginxStatusFailed:
		return "失败"
	default:
		return "未知"
	}
}

// formatNginxThreshold formats threshold value for display.
func formatNginxThreshold(value float64) string {
	if value == 0 {
		return "N/A"
	}
	return fmt.Sprintf("%.1f", value)
}

// =============================================================================
// Tomcat Report Helper Functions
// ============================================================================

// tomcatStatusText converts Tomcat instance status to Chinese text.
func tomcatStatusText(status model.TomcatInstanceStatus) string {
	switch status {
	case model.TomcatStatusNormal:
		return "正常"
	case model.TomcatStatusWarning:
		return "警告"
	case model.TomcatStatusCritical:
		return "严重"
	case model.TomcatStatusFailed:
		return "失败"
	default:
		return "未知"
	}
}

// tomcatBoolToText converts boolean to Chinese text (是/否).
func tomcatBoolToText(b bool) string {
	if b {
		return "是"
	}
	return "否"
}

// getTomcatPortOrContainer returns container name if container deployment,
// otherwise returns port number.
func getTomcatPortOrContainer(r *model.TomcatInspectionResult) string {
	if r.Instance == nil {
		return ""
	}
	if r.Instance.IsContainerDeployment() {
		return r.Instance.Container
	}
	return fmt.Sprintf("%d", r.Instance.Port)
}

// formatTomcatThreshold formats a Tomcat alert threshold value.
func formatTomcatThreshold(value float64, metricName string) string {
	switch metricName {
	case "last_error_timestamp":
		// Time-based thresholds (in minutes)
		return fmt.Sprintf("%.0f分钟", value)
	default:
		return fmt.Sprintf("%.2f", value)
	}
}

// WriteNginxInspection generates an Excel report for Nginx inspection results.
func (w *Writer) WriteNginxInspection(result *model.NginxInspectionResults, outputPath string) error {
	if result == nil {
		return fmt.Errorf("Nginx inspection result is nil")
	}

	if !strings.HasSuffix(strings.ToLower(outputPath), ".xlsx") {
		outputPath = outputPath + ".xlsx"
	}

	f := excelize.NewFile()
	defer f.Close()

	if err := w.createNginxSheet(f, result); err != nil {
		return fmt.Errorf("failed to create Nginx sheet: %w", err)
	}

	if err := f.DeleteSheet(defaultSheet); err != nil {
	}

	idx, _ := f.GetSheetIndex(sheetNginx)
	f.SetActiveSheet(idx)

	if err := f.SaveAs(outputPath); err != nil {
		return fmt.Errorf("failed to save Excel file: %w", err)
	}

	return nil
}

// AppendNginxInspection appends Nginx inspection data to an existing Excel file.
func (w *Writer) AppendNginxInspection(result *model.NginxInspectionResults, existingPath string) error {
	if result == nil {
		return fmt.Errorf("Nginx inspection result is nil")
	}

	f, err := excelize.OpenFile(existingPath)
	if err != nil {
		return fmt.Errorf("failed to open existing Excel file: %w", err)
	}
	defer f.Close()

	if err := w.createNginxSheet(f, result); err != nil {
		return fmt.Errorf("failed to create Nginx sheet: %w", err)
	}

	if err := f.Save(); err != nil {
		return fmt.Errorf("failed to save Excel file: %w", err)
	}

	return nil
}

// createTomcatSheet creates the Tomcat inspection worksheet.
func (w *Writer) createTomcatSheet(f *excelize.File, result *model.TomcatInspectionResults) error {
	if result == nil || len(result.Results) == 0 {
		return nil
	}

	// Create sheet
	_, err := f.NewSheet(sheetTomcat)
	if err != nil {
		return fmt.Errorf("create sheet %s: %w", sheetTomcat, err)
	}

	// Create styles
	headerStyle, err := w.createHeaderStyle(f)
	if err != nil {
		return fmt.Errorf("create header style for %s: %w", sheetTomcat, err)
	}

	warningStyle, err := w.createWarningStyle(f)
	if err != nil {
		return fmt.Errorf("create warning style for %s: %w", sheetTomcat, err)
	}

	criticalStyle, err := w.createCriticalStyle(f)
	if err != nil {
		return fmt.Errorf("create critical style for %s: %w", sheetTomcat, err)
	}

	normalStyle, err := w.createNormalStyle(f)
	if err != nil {
		return fmt.Errorf("create normal style for %s: %w", sheetTomcat, err)
	}

	// Define headers (15 columns)
	headers := []string{
		"巡检时间", "主机名", "IP地址", "应用类型", "端口", "容器名",
		"版本", "安装路径", "日志路径", "JVM配置",
		"连接数", "运行时长", "非root用户", "最近错误时间", "整体状态",
	}

	// Set column widths
	colWidths := map[string]float64{
		"A": 20, "B": 18, "C": 15, "D": 12, "E": 10, "F": 18,
		"G": 12, "H": 30, "I": 30, "J": 25, "K": 12, "L": 18,
		"M": 14, "N": 20, "O": 12,
	}

	for col, width := range colWidths {
		f.SetColWidth(sheetTomcat, col, col, width)
	}

	// Write headers
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", columnName(i+1))
		f.SetCellValue(sheetTomcat, cell, header)
		f.SetCellStyle(sheetTomcat, cell, cell, headerStyle)
	}

	// Freeze header row
	f.SetPanes(sheetTomcat, &excelize.Panes{Freeze: true, YSplit: 1})

	// Write data rows
	for i, r := range result.Results {
		row := i + 2
		inspectionTime := result.InspectionTime.In(w.timezone).Format("2006-01-02 15:04:05")

		f.SetCellValue(sheetTomcat, "A"+fmt.Sprint(row), inspectionTime)
		f.SetCellValue(sheetTomcat, "B"+fmt.Sprint(row), r.Instance.Hostname)
		f.SetCellValue(sheetTomcat, "C"+fmt.Sprint(row), r.Instance.IP)
		f.SetCellValue(sheetTomcat, "D"+fmt.Sprint(row), r.Instance.ApplicationType)
		f.SetCellValue(sheetTomcat, "E"+fmt.Sprint(row), r.Instance.Port)
		f.SetCellValue(sheetTomcat, "F"+fmt.Sprint(row), r.Instance.Container)
		f.SetCellValue(sheetTomcat, "G"+fmt.Sprint(row), r.Instance.Version)
		f.SetCellValue(sheetTomcat, "H"+fmt.Sprint(row), r.Instance.InstallPath)
		f.SetCellValue(sheetTomcat, "I"+fmt.Sprint(row), r.Instance.LogPath)
		f.SetCellValue(sheetTomcat, "J"+fmt.Sprint(row), r.Instance.JVMConfig)
		f.SetCellValue(sheetTomcat, "K"+fmt.Sprint(row), r.Connections)
		f.SetCellValue(sheetTomcat, "L"+fmt.Sprint(row), r.UptimeFormatted)
		f.SetCellValue(sheetTomcat, "M"+fmt.Sprint(row), tomcatBoolToText(r.NonRootUser))
		f.SetCellValue(sheetTomcat, "N"+fmt.Sprint(row), r.LastErrorTimeFormatted)

		// Status column with conditional formatting
		statusCell := "O" + fmt.Sprint(row)
		statusText := tomcatStatusText(r.Status)
		f.SetCellValue(sheetTomcat, statusCell, statusText)

		switch r.Status {
		case model.TomcatStatusCritical:
			f.SetCellStyle(sheetTomcat, statusCell, statusCell, criticalStyle)
		case model.TomcatStatusWarning:
			f.SetCellStyle(sheetTomcat, statusCell, statusCell, warningStyle)
		case model.TomcatStatusNormal:
			f.SetCellStyle(sheetTomcat, statusCell, statusCell, normalStyle)
		}
	}

	return nil
}

// WriteTomcatInspection generates a standalone Excel report for Tomcat inspection.
func (w *Writer) WriteTomcatInspection(result *model.TomcatInspectionResults, outputPath string) error {
	if result == nil {
		return fmt.Errorf("tomcat inspection result is nil")
	}

	if !strings.HasSuffix(strings.ToLower(outputPath), ".xlsx") {
		outputPath = outputPath + ".xlsx"
	}

	f := excelize.NewFile()
	defer f.Close()

	if err := w.createTomcatSheet(f, result); err != nil {
		return fmt.Errorf("failed to create Tomcat sheet: %w", err)
	}

	if err := f.DeleteSheet(defaultSheet); err != nil {
	}

	idx, _ := f.GetSheetIndex(sheetTomcat)
	f.SetActiveSheet(idx)

	return f.SaveAs(outputPath)
}

// AppendTomcatInspection appends Tomcat sheets to an existing Excel file.
func (w *Writer) AppendTomcatInspection(result *model.TomcatInspectionResults, existingPath string) error {
	if result == nil {
		return fmt.Errorf("tomcat inspection result is nil")
	}

	f, err := excelize.OpenFile(existingPath)
	if err != nil {
		return fmt.Errorf("failed to open existing file: %w", err)
	}
	defer f.Close()

	if err := w.createTomcatSheet(f, result); err != nil {
		return fmt.Errorf("failed to create Tomcat sheet: %w", err)
	}

	return f.Save()
}
