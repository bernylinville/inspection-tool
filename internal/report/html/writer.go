// Package html provides HTML report generation for the inspection tool.
// It implements the report.ReportWriter interface to generate .html files
// with inspection results, including summary, detailed data, and alerts.
package html

import (
	"embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"inspection-tool/internal/model"
)

//go:embed templates/*.html
var embeddedTemplates embed.FS

// Writer implements report.ReportWriter for HTML format.
type Writer struct {
	timezone     *time.Location
	templatePath string // User-defined template path (optional)
}

// TemplateData holds all data passed to the HTML template.
type TemplateData struct {
	Title          string
	InspectionTime string
	Duration       string
	Summary        *model.InspectionSummary
	AlertSummary   *model.AlertSummary
	Hosts          []*HostData
	Alerts         []*AlertData
	DiskPaths      []string
	Version        string
	GeneratedAt    string
}

// HostData represents host data formatted for template rendering.
type HostData struct {
	Hostname      string
	IP            string
	Status        string
	StatusClass   string
	OS            string
	OSVersion     string
	KernelVersion string
	CPUCores      int
	CPUModel      string
	MemoryTotal   string
	Metrics       map[string]*MetricData
	AlertCount    int
}

// MetricData represents metric data formatted for template rendering.
type MetricData struct {
	Name        string
	DisplayName string
	Value       string
	Status      string
	StatusClass string
	IsNA        bool
}

// AlertData represents alert data formatted for template rendering.
type AlertData struct {
	Hostname          string
	MetricName        string
	MetricDisplayName string
	CurrentValue      string
	WarningThreshold  string
	CriticalThreshold string
	Level             string
	LevelClass        string
	Message           string
}

// NewWriter creates a new HTML report writer.
// If timezone is nil, it defaults to Asia/Shanghai.
// If templatePath is empty, the embedded default template will be used.
func NewWriter(timezone *time.Location, templatePath string) *Writer {
	if timezone == nil {
		timezone, _ = time.LoadLocation("Asia/Shanghai")
	}
	return &Writer{
		timezone:     timezone,
		templatePath: templatePath,
	}
}

// Format returns the format identifier for this writer.
func (w *Writer) Format() string {
	return "html"
}

// Write generates an HTML report from the inspection result.
func (w *Writer) Write(result *model.InspectionResult, outputPath string) error {
	if result == nil {
		return fmt.Errorf("inspection result is nil")
	}

	// Ensure output path has .html extension
	if !strings.HasSuffix(strings.ToLower(outputPath), ".html") {
		outputPath = outputPath + ".html"
	}

	// Load template
	tmpl, err := w.loadTemplate()
	if err != nil {
		return fmt.Errorf("failed to load template: %w", err)
	}

	// Prepare template data
	data := w.prepareTemplateData(result)

	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Execute template
	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// loadTemplate loads the HTML template.
// It first tries to load a user-defined template, then falls back to the embedded default.
func (w *Writer) loadTemplate() (*template.Template, error) {
	// Define template functions
	funcMap := template.FuncMap{
		"formatSize":     formatSize,
		"formatDuration": formatDuration,
		"statusClass":    statusClass,
		"alertClass":     alertLevelClass,
	}

	// Try user-defined template first
	if w.templatePath != "" {
		if _, err := os.Stat(w.templatePath); err == nil {
			tmpl, err := template.New(filepath.Base(w.templatePath)).Funcs(funcMap).ParseFiles(w.templatePath)
			if err != nil {
				return nil, fmt.Errorf("failed to parse user template: %w", err)
			}
			return tmpl, nil
		}
		// User template not found, fall through to default
	}

	// Load embedded default template
	tmpl, err := template.New("default.html").Funcs(funcMap).ParseFS(embeddedTemplates, "templates/default.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse embedded template: %w", err)
	}
	return tmpl, nil
}

// prepareTemplateData converts InspectionResult to TemplateData for template rendering.
func (w *Writer) prepareTemplateData(result *model.InspectionResult) *TemplateData {
	// Collect unique disk paths
	diskPaths := w.collectDiskPaths(result.Hosts)

	// Convert hosts
	hosts := make([]*HostData, 0, len(result.Hosts))
	for _, host := range result.Hosts {
		hosts = append(hosts, w.convertHostData(host))
	}

	// Convert and sort alerts (critical first)
	alerts := w.convertAlerts(result.Alerts)

	return &TemplateData{
		Title:          "系统巡检报告",
		InspectionTime: result.InspectionTime.In(w.timezone).Format("2006-01-02 15:04:05"),
		Duration:       formatDuration(result.Duration),
		Summary:        result.Summary,
		AlertSummary:   result.AlertSummary,
		Hosts:          hosts,
		Alerts:         alerts,
		DiskPaths:      diskPaths,
		Version:        result.Version,
		GeneratedAt:    time.Now().In(w.timezone).Format("2006-01-02 15:04:05"),
	}
}

// convertHostData converts a HostResult to HostData for template rendering.
func (w *Writer) convertHostData(host *model.HostResult) *HostData {
	metrics := make(map[string]*MetricData)
	for name, metric := range host.Metrics {
		metrics[name] = w.convertMetricData(metric)
	}

	return &HostData{
		Hostname:      host.Hostname,
		IP:            host.IP,
		Status:        statusText(host.Status),
		StatusClass:   statusClass(host.Status),
		OS:            host.OS,
		OSVersion:     host.OSVersion,
		KernelVersion: host.KernelVersion,
		CPUCores:      host.CPUCores,
		CPUModel:      host.CPUModel,
		MemoryTotal:   formatSize(host.MemoryTotal),
		Metrics:       metrics,
		AlertCount:    len(host.Alerts),
	}
}

// convertMetricData converts a MetricValue to MetricData for template rendering.
func (w *Writer) convertMetricData(metric *model.MetricValue) *MetricData {
	if metric == nil {
		return &MetricData{
			Value:       "N/A",
			IsNA:        true,
			StatusClass: "",
		}
	}

	return &MetricData{
		Name:        metric.Name,
		Value:       metric.FormattedValue,
		Status:      string(metric.Status),
		StatusClass: metricStatusClass(metric.Status),
		IsNA:        metric.IsNA,
	}
}

// convertAlerts converts and sorts alerts for template rendering.
func (w *Writer) convertAlerts(alerts []*model.Alert) []*AlertData {
	// Make a copy for sorting
	sortedAlerts := make([]*model.Alert, len(alerts))
	copy(sortedAlerts, alerts)

	// Sort by level (critical first) then by hostname
	sort.Slice(sortedAlerts, func(i, j int) bool {
		if sortedAlerts[i].Level != sortedAlerts[j].Level {
			return alertLevelPriority(sortedAlerts[i].Level) > alertLevelPriority(sortedAlerts[j].Level)
		}
		return sortedAlerts[i].Hostname < sortedAlerts[j].Hostname
	})

	// Convert to AlertData
	result := make([]*AlertData, 0, len(sortedAlerts))
	for _, alert := range sortedAlerts {
		result = append(result, &AlertData{
			Hostname:          alert.Hostname,
			MetricName:        alert.MetricName,
			MetricDisplayName: alert.MetricDisplayName,
			CurrentValue:      alert.FormattedValue,
			WarningThreshold:  formatThreshold(alert.WarningThreshold, alert.MetricName),
			CriticalThreshold: formatThreshold(alert.CriticalThreshold, alert.MetricName),
			Level:             alertLevelText(alert.Level),
			LevelClass:        alertLevelClass(alert.Level),
			Message:           alert.Message,
		})
	}
	return result
}

// collectDiskPaths collects unique disk paths from all hosts.
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

// Helper functions

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

// formatSize formats bytes to human-readable size.
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
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

// statusClass returns the CSS class for a host status.
func statusClass(status model.HostStatus) string {
	switch status {
	case model.HostStatusNormal:
		return "status-normal"
	case model.HostStatusWarning:
		return "status-warning"
	case model.HostStatusCritical:
		return "status-critical"
	case model.HostStatusFailed:
		return "status-failed"
	default:
		return ""
	}
}

// metricStatusClass returns the CSS class for a metric status.
func metricStatusClass(status model.MetricStatus) string {
	switch status {
	case model.MetricStatusNormal:
		return "metric-normal"
	case model.MetricStatusWarning:
		return "metric-warning"
	case model.MetricStatusCritical:
		return "metric-critical"
	case model.MetricStatusPending:
		return "metric-pending"
	default:
		return ""
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

// alertLevelClass returns the CSS class for an alert level.
func alertLevelClass(level model.AlertLevel) string {
	switch level {
	case model.AlertLevelNormal:
		return "alert-normal"
	case model.AlertLevelWarning:
		return "alert-warning"
	case model.AlertLevelCritical:
		return "alert-critical"
	default:
		return ""
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
