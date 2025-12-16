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

// ============================================================================
// MySQL Report Data Structures
// ============================================================================

// MySQLTemplateData holds MySQL inspection data for template rendering.
type MySQLTemplateData struct {
	Title          string
	InspectionTime string
	Duration       string
	Summary        *model.MySQLInspectionSummary
	AlertSummary   *model.MySQLAlertSummary
	Instances      []*MySQLInstanceData
	Alerts         []*MySQLAlertData
	Version        string
	GeneratedAt    string
}

// MySQLInstanceData represents MySQL instance data formatted for template.
type MySQLInstanceData struct {
	Address            string
	IP                 string
	Port               int
	Version            string
	ServerID           string
	ClusterMode        string
	SyncStatus         string
	MaxConnections     int
	CurrentConnections int
	BinlogEnabled      string
	Status             string
	StatusClass        string
	AlertCount         int
}

// MySQLAlertData represents MySQL alert data formatted for template.
type MySQLAlertData struct {
	Address           string
	MetricName        string
	MetricDisplayName string
	CurrentValue      string
	WarningThreshold  string
	CriticalThreshold string
	Level             string
	LevelClass        string
	Message           string
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

// mysqlStatusClass returns the CSS class for MySQL instance status.
func mysqlStatusClass(status model.MySQLInstanceStatus) string {
	switch status {
	case model.MySQLStatusNormal:
		return "status-normal"
	case model.MySQLStatusWarning:
		return "status-warning"
	case model.MySQLStatusCritical:
		return "status-critical"
	case model.MySQLStatusFailed:
		return "status-failed"
	default:
		return ""
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

// getMySQLSyncStatus returns sync status text based on cluster mode.
func getMySQLSyncStatus(r *model.MySQLInspectionResult) string {
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

// boolToText converts boolean to Chinese text (启用/禁用).
func boolToText(b bool) string {
	if b {
		return "启用"
	}
	return "禁用"
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

// WriteMySQLInspection generates an HTML report for MySQL inspection results.
func (w *Writer) WriteMySQLInspection(result *model.MySQLInspectionResults, outputPath string) error {
	if result == nil {
		return fmt.Errorf("MySQL inspection result is nil")
	}

	// Ensure output path has .html extension
	if !strings.HasSuffix(strings.ToLower(outputPath), ".html") {
		outputPath = outputPath + ".html"
	}

	// Load MySQL template
	tmpl, err := w.loadMySQLTemplate()
	if err != nil {
		return fmt.Errorf("failed to load MySQL template: %w", err)
	}

	// Prepare template data
	data := w.prepareMySQLTemplateData(result)

	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Execute template
	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute MySQL template: %w", err)
	}

	return nil
}

// loadMySQLTemplate loads the MySQL HTML template.
func (w *Writer) loadMySQLTemplate() (*template.Template, error) {
	// Define template functions
	funcMap := template.FuncMap{
		"formatSize":     formatSize,
		"formatDuration": formatDuration,
		"statusClass":    statusClass,
		"alertClass":     alertLevelClass,
	}

	// Load embedded MySQL template
	tmpl, err := template.New("mysql.html").Funcs(funcMap).ParseFS(embeddedTemplates, "templates/mysql.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse embedded MySQL template: %w", err)
	}
	return tmpl, nil
}

// prepareMySQLTemplateData converts MySQLInspectionResults to MySQLTemplateData for template rendering.
func (w *Writer) prepareMySQLTemplateData(result *model.MySQLInspectionResults) *MySQLTemplateData {
	// Convert instances
	instances := make([]*MySQLInstanceData, 0, len(result.Results))
	for _, r := range result.Results {
		instances = append(instances, w.convertMySQLInstanceData(r))
	}

	// Convert and sort alerts (critical first)
	alerts := w.convertMySQLAlerts(result.Alerts)

	return &MySQLTemplateData{
		Title:          "MySQL 巡检报告",
		InspectionTime: result.InspectionTime.In(w.timezone).Format("2006-01-02 15:04:05"),
		Duration:       formatDuration(result.Duration),
		Summary:        result.Summary,
		AlertSummary:   result.AlertSummary,
		Instances:      instances,
		Alerts:         alerts,
		Version:        result.Version,
		GeneratedAt:    time.Now().In(w.timezone).Format("2006-01-02 15:04:05"),
	}
}

// convertMySQLInstanceData converts a MySQLInspectionResult to MySQLInstanceData for template rendering.
func (w *Writer) convertMySQLInstanceData(r *model.MySQLInspectionResult) *MySQLInstanceData {
	return &MySQLInstanceData{
		Address:            r.GetAddress(),
		IP:                 r.Instance.IP,
		Port:               r.Instance.Port,
		Version:            r.Instance.Version,
		ServerID:           r.Instance.ServerID,
		ClusterMode:        mysqlClusterModeText(r.Instance.ClusterMode),
		SyncStatus:         getMySQLSyncStatus(r),
		MaxConnections:     r.MaxConnections,
		CurrentConnections: r.CurrentConnections,
		BinlogEnabled:      boolToText(r.BinlogEnabled),
		Status:             mysqlStatusText(r.Status),
		StatusClass:        mysqlStatusClass(r.Status),
		AlertCount:         len(r.Alerts),
	}
}

// convertMySQLAlerts converts and sorts MySQL alerts for template rendering.
func (w *Writer) convertMySQLAlerts(alerts []*model.MySQLAlert) []*MySQLAlertData {
	// Make a copy for sorting
	sortedAlerts := make([]*model.MySQLAlert, len(alerts))
	copy(sortedAlerts, alerts)

	// Sort by level (critical first) then by address
	sort.Slice(sortedAlerts, func(i, j int) bool {
		if sortedAlerts[i].Level != sortedAlerts[j].Level {
			return alertLevelPriority(sortedAlerts[i].Level) > alertLevelPriority(sortedAlerts[j].Level)
		}
		return sortedAlerts[i].Address < sortedAlerts[j].Address
	})

	// Convert to MySQLAlertData
	result := make([]*MySQLAlertData, 0, len(sortedAlerts))
	for _, alert := range sortedAlerts {
		result = append(result, &MySQLAlertData{
			Address:           alert.Address,
			MetricName:        alert.MetricName,
			MetricDisplayName: alert.MetricDisplayName,
			CurrentValue:      alert.FormattedValue,
			WarningThreshold:  formatMySQLThreshold(alert.WarningThreshold, alert.MetricName),
			CriticalThreshold: formatMySQLThreshold(alert.CriticalThreshold, alert.MetricName),
			Level:             alertLevelText(alert.Level),
			LevelClass:        alertLevelClass(alert.Level),
			Message:           alert.Message,
		})
	}
	return result
}

// ============================================================================
// Combined Report (Host + MySQL) Data Structures and Methods
// ============================================================================

// CombinedTemplateData holds both Host and MySQL inspection data for combined template.
type CombinedTemplateData struct {
	Title          string
	InspectionTime string
	Duration       string
	// Host data
	HasHost          bool
	HostSummary      *model.InspectionSummary
	HostAlertSummary *model.AlertSummary
	Hosts            []*HostData
	HostAlerts       []*AlertData
	DiskPaths        []string
	// MySQL data
	HasMySQL          bool
	MySQLSummary      *model.MySQLInspectionSummary
	MySQLAlertSummary *model.MySQLAlertSummary
	MySQLInstances    []*MySQLInstanceData
	MySQLAlerts       []*MySQLAlertData
	// Common
	Version     string
	GeneratedAt string
}

// WriteCombined generates an HTML report combining Host and MySQL inspection results.
func (w *Writer) WriteCombined(hostResult *model.InspectionResult, mysqlResult *model.MySQLInspectionResults, outputPath string) error {
	// At least one result must be present
	if hostResult == nil && mysqlResult == nil {
		return fmt.Errorf("both host and MySQL inspection results are nil")
	}

	// Ensure output path has .html extension
	if !strings.HasSuffix(strings.ToLower(outputPath), ".html") {
		outputPath = outputPath + ".html"
	}

	// Load combined template
	tmpl, err := w.loadCombinedTemplate()
	if err != nil {
		return fmt.Errorf("failed to load combined template: %w", err)
	}

	// Prepare combined template data
	data := w.prepareCombinedTemplateData(hostResult, mysqlResult)

	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Execute template
	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute combined template: %w", err)
	}

	return nil
}

// loadCombinedTemplate loads the combined HTML template.
func (w *Writer) loadCombinedTemplate() (*template.Template, error) {
	// Define template functions
	funcMap := template.FuncMap{
		"formatSize":     formatSize,
		"formatDuration": formatDuration,
		"statusClass":    statusClass,
		"alertClass":     alertLevelClass,
	}

	// Load embedded combined template
	tmpl, err := template.New("combined.html").Funcs(funcMap).ParseFS(embeddedTemplates, "templates/combined.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse embedded combined template: %w", err)
	}
	return tmpl, nil
}

// prepareCombinedTemplateData prepares data for the combined template.
func (w *Writer) prepareCombinedTemplateData(hostResult *model.InspectionResult, mysqlResult *model.MySQLInspectionResults) *CombinedTemplateData {
	data := &CombinedTemplateData{
		Title:       "系统巡检报告",
		GeneratedAt: time.Now().In(w.timezone).Format("2006-01-02 15:04:05"),
	}

	// Determine inspection time and duration from available results
	if hostResult != nil {
		data.InspectionTime = hostResult.InspectionTime.In(w.timezone).Format("2006-01-02 15:04:05")
		data.Duration = formatDuration(hostResult.Duration)
		data.Version = hostResult.Version
	} else if mysqlResult != nil {
		data.InspectionTime = mysqlResult.InspectionTime.In(w.timezone).Format("2006-01-02 15:04:05")
		data.Duration = formatDuration(mysqlResult.Duration)
		data.Version = mysqlResult.Version
	}

	// Fill Host data if available
	if hostResult != nil {
		data.HasHost = true
		data.HostSummary = hostResult.Summary
		data.HostAlertSummary = hostResult.AlertSummary
		data.DiskPaths = w.collectDiskPaths(hostResult.Hosts)

		// Convert hosts
		hosts := make([]*HostData, 0, len(hostResult.Hosts))
		for _, host := range hostResult.Hosts {
			hosts = append(hosts, w.convertHostData(host))
		}
		data.Hosts = hosts

		// Convert host alerts
		data.HostAlerts = w.convertAlerts(hostResult.Alerts)
	}

	// Fill MySQL data if available
	if mysqlResult != nil {
		data.HasMySQL = true
		data.MySQLSummary = mysqlResult.Summary
		data.MySQLAlertSummary = mysqlResult.AlertSummary

		// Convert MySQL instances
		instances := make([]*MySQLInstanceData, 0, len(mysqlResult.Results))
		for _, r := range mysqlResult.Results {
			instances = append(instances, w.convertMySQLInstanceData(r))
		}
		data.MySQLInstances = instances

		// Convert MySQL alerts
		data.MySQLAlerts = w.convertMySQLAlerts(mysqlResult.Alerts)
	}

	return data
}
