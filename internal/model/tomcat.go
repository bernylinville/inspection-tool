package model

import (
	"fmt"
	"time"
)

// =============================================================================
// Tomcat 实例状态枚举
// =============================================================================

type TomcatInstanceStatus string

const (
	TomcatStatusNormal   TomcatInstanceStatus = "normal"
	TomcatStatusWarning  TomcatInstanceStatus = "warning"
	TomcatStatusCritical TomcatInstanceStatus = "critical"
	TomcatStatusFailed   TomcatInstanceStatus = "failed"
)

func (s TomcatInstanceStatus) IsHealthy() bool {
	return s == TomcatStatusNormal
}

func (s TomcatInstanceStatus) IsWarning() bool {
	return s == TomcatStatusWarning
}

func (s TomcatInstanceStatus) IsCritical() bool {
	return s == TomcatStatusCritical
}

func (s TomcatInstanceStatus) IsFailed() bool {
	return s == TomcatStatusFailed
}

// =============================================================================
// Tomcat 实例结构体
// =============================================================================

type TomcatInstance struct {
	Identifier      string `json:"identifier"`
	Hostname        string `json:"hostname"`
	IP              string `json:"ip"`
	Port            int    `json:"port"`
	Container       string `json:"container"`
	ApplicationType string `json:"application_type"`
	Version         string `json:"version"`
	InstallPath     string `json:"install_path"`
	LogPath         string `json:"log_path"`
	JVMConfig       string `json:"jvm_config"`
}

func GenerateTomcatIdentifier(hostname string, port int, container string) string {
	if container != "" {
		return fmt.Sprintf("%s:%s", hostname, container)
	}
	return fmt.Sprintf("%s:%d", hostname, port)
}

func NewTomcatInstance(hostname string, port int) *TomcatInstance {
	return &TomcatInstance{
		Identifier:      GenerateTomcatIdentifier(hostname, port, ""),
		Hostname:        hostname,
		Port:            port,
		Container:       "",
		ApplicationType: "tomcat",
	}
}

func NewTomcatInstanceWithContainer(hostname, container string) *TomcatInstance {
	return &TomcatInstance{
		Identifier:      GenerateTomcatIdentifier(hostname, 0, container),
		Hostname:        hostname,
		Container:       container,
		Port:            0,
		ApplicationType: "tomcat",
	}
}

func (i *TomcatInstance) SetIP(ip string) {
	if i == nil {
		return
	}
	i.IP = ip
}

func (i *TomcatInstance) SetApplicationType(appType string) {
	if i == nil {
		return
	}
	i.ApplicationType = appType
}

func (i *TomcatInstance) SetVersion(version string) {
	if i == nil {
		return
	}
	i.Version = version
}

func (i *TomcatInstance) SetInstallPath(path string) {
	if i == nil {
		return
	}
	i.InstallPath = path
}

func (i *TomcatInstance) SetLogPath(path string) {
	if i == nil {
		return
	}
	i.LogPath = path
}

func (i *TomcatInstance) SetJVMConfig(config string) {
	if i == nil {
		return
	}
	i.JVMConfig = config
}

func (i *TomcatInstance) IsContainerDeployment() bool {
	return i != nil && i.Container != ""
}

func (i *TomcatInstance) String() string {
	if i == nil {
		return "TomcatInstance(nil)"
	}
	if i.IsContainerDeployment() {
		return fmt.Sprintf("TomcatInstance(%s [container: %s])", i.Hostname, i.Container)
	}
	return fmt.Sprintf("TomcatInstance(%s:%d)", i.Hostname, i.Port)
}

// =============================================================================
// Tomcat 告警结构体
// =============================================================================

type TomcatAlert struct {
	Identifier        string     `json:"identifier"`
	MetricName        string     `json:"metric_name"`
	MetricDisplayName string     `json:"metric_display_name"`
	CurrentValue      float64    `json:"current_value"`
	FormattedValue    string     `json:"formatted_value"`
	WarningThreshold  float64    `json:"warning_threshold"`
	CriticalThreshold float64    `json:"critical_threshold"`
	Level             AlertLevel `json:"level"`
	Message           string     `json:"message"`
}

func NewTomcatAlert(identifier, metricName string, currentValue float64, level AlertLevel) *TomcatAlert {
	return &TomcatAlert{
		Identifier:   identifier,
		MetricName:   metricName,
		CurrentValue: currentValue,
		Level:        level,
	}
}

func (a *TomcatAlert) IsWarning() bool {
	return a != nil && a.Level == AlertLevelWarning
}

func (a *TomcatAlert) IsCritical() bool {
	return a != nil && a.Level == AlertLevelCritical
}

// =============================================================================
// Tomcat 指标值结构体
// =============================================================================

type TomcatMetricValue struct {
	Name           string            `json:"name"`
	RawValue       float64           `json:"raw_value"`
	StringValue    string            `json:"string_value,omitempty"` // 标签提取的字符串值
	FormattedValue string            `json:"formatted_value"`
	IsNA           bool              `json:"is_na"`
	Timestamp      int64             `json:"timestamp"`
	Labels         map[string]string `json:"labels,omitempty"`
}

// =============================================================================
// Tomcat 巡检结果结构体
// =============================================================================

type TomcatInspectionResult struct {
	Instance               *TomcatInstance         `json:"instance"`
	Up                     bool                    `json:"up"`
	Connections            int                     `json:"connections"`
	UptimeSeconds          int64                   `json:"uptime_seconds"`
	UptimeFormatted        string                  `json:"uptime_formatted"`
	LastErrorTimestamp     int64                   `json:"last_error_timestamp"`
	LastErrorTimeFormatted string                  `json:"last_error_time_formatted"`
	NonRootUser            bool                    `json:"non_root_user"`
	PID                    int                     `json:"-"`
	Metrics                map[string]*TomcatMetricValue `json:"-"` // 指标映射（内部使用，不序列化）
	Status                 TomcatInstanceStatus    `json:"status"`
	Alerts                 []*TomcatAlert          `json:"alerts,omitempty"`
	CollectedAt            time.Time               `json:"collected_at"`
	Error                  string                  `json:"error,omitempty"`
}

func NewTomcatInspectionResult(instance *TomcatInstance) *TomcatInspectionResult {
	return &TomcatInspectionResult{
		Instance:           instance,
		Status:             TomcatStatusNormal,
		Alerts:             make([]*TomcatAlert, 0),
		LastErrorTimestamp: 0,
	}
}

func (r *TomcatInspectionResult) AddAlert(alert *TomcatAlert) {
	if r == nil || alert == nil {
		return
	}
	r.Alerts = append(r.Alerts, alert)
}

func (r *TomcatInspectionResult) HasAlerts() bool {
	return r != nil && len(r.Alerts) > 0
}

func (r *TomcatInspectionResult) FormatUptime(loc *time.Location) string {
	if r == nil {
		return "N/A"
	}
	if loc == nil {
		loc = time.UTC
	}

	seconds := r.UptimeSeconds
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	if days > 0 {
		return fmt.Sprintf("%d天 %02d:%02d:%02d", days, hours, minutes, secs)
	}
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

func (r *TomcatInspectionResult) FormatLastErrorTime(loc *time.Location) string {
	if r == nil {
		return "N/A"
	}
	if loc == nil {
		loc = time.UTC
	}

	if r.LastErrorTimestamp == 0 {
		return "无错误"
	}

	t := time.Unix(r.LastErrorTimestamp, 0).In(loc)
	return t.Format("2006-01-02 15:04:05")
}

func (r *TomcatInspectionResult) GetIdentifier() string {
	if r == nil || r.Instance == nil {
		return ""
	}
	return r.Instance.Identifier
}

func (r *TomcatInspectionResult) SetMetric(mv *TomcatMetricValue) {
	if r == nil || mv == nil {
		return
	}
	if r.Metrics == nil {
		r.Metrics = make(map[string]*TomcatMetricValue)
	}
	r.Metrics[mv.Name] = mv
}

func (r *TomcatInspectionResult) GetMetric(name string) *TomcatMetricValue {
	if r == nil || r.Metrics == nil {
		return nil
	}
	return r.Metrics[name]
}

// =============================================================================
// Tomcat 巡检摘要结构体
// =============================================================================

type TomcatInspectionSummary struct {
	TotalInstances    int `json:"total_instances"`
	NormalInstances   int `json:"normal_instances"`
	WarningInstances  int `json:"warning_instances"`
	CriticalInstances int `json:"critical_instances"`
	FailedInstances   int `json:"failed_instances"`
}

func NewTomcatInspectionSummary(results []*TomcatInspectionResult) *TomcatInspectionSummary {
	summary := &TomcatInspectionSummary{
		TotalInstances: len(results),
	}

	for _, result := range results {
		if result == nil {
			continue
		}

		switch result.Status {
		case TomcatStatusNormal:
			summary.NormalInstances++
		case TomcatStatusWarning:
			summary.WarningInstances++
		case TomcatStatusCritical:
			summary.CriticalInstances++
		case TomcatStatusFailed:
			summary.FailedInstances++
		}
	}

	return summary
}

// =============================================================================
// Tomcat 告警摘要结构体
// =============================================================================

type TomcatAlertSummary struct {
	TotalAlerts   int `json:"total_alerts"`
	WarningCount  int `json:"warning_count"`
	CriticalCount int `json:"critical_count"`
}

func NewTomcatAlertSummary(alerts []*TomcatAlert) *TomcatAlertSummary {
	summary := &TomcatAlertSummary{
		TotalAlerts: len(alerts),
	}

	for _, alert := range alerts {
		if alert == nil {
			continue
		}

		switch alert.Level {
		case AlertLevelWarning:
			summary.WarningCount++
		case AlertLevelCritical:
			summary.CriticalCount++
		}
	}

	return summary
}

// =============================================================================
// Tomcat 完整巡检结果容器
// =============================================================================

type TomcatInspectionResults struct {
	InspectionTime time.Time                `json:"inspection_time"`
	Duration       time.Duration            `json:"duration"`
	Summary        *TomcatInspectionSummary `json:"summary"`
	Results        []*TomcatInspectionResult `json:"results"`
	Alerts         []*TomcatAlert           `json:"alerts"`
	AlertSummary   *TomcatAlertSummary      `json:"alert_summary"`
	Version        string                   `json:"version,omitempty"`
}

func NewTomcatInspectionResults(inspectionTime time.Time) *TomcatInspectionResults {
	return &TomcatInspectionResults{
		InspectionTime: inspectionTime,
		Results:        make([]*TomcatInspectionResult, 0),
		Alerts:         make([]*TomcatAlert, 0),
	}
}

func (r *TomcatInspectionResults) AddResult(result *TomcatInspectionResult) {
	if r == nil || result == nil {
		return
	}
	r.Results = append(r.Results, result)

	if result.HasAlerts() {
		r.Alerts = append(r.Alerts, result.Alerts...)
	}
}

func (r *TomcatInspectionResults) Finalize(endTime time.Time) {
	if r == nil {
		return
	}

	r.Duration = endTime.Sub(r.InspectionTime)
	r.Summary = NewTomcatInspectionSummary(r.Results)
	r.AlertSummary = NewTomcatAlertSummary(r.Alerts)
}

func (r *TomcatInspectionResults) GetResultByIdentifier(identifier string) *TomcatInspectionResult {
	if r == nil {
		return nil
	}

	for _, result := range r.Results {
		if result != nil && result.GetIdentifier() == identifier {
			return result
		}
	}
	return nil
}

func (r *TomcatInspectionResults) GetCriticalResults() []*TomcatInspectionResult {
	if r == nil {
		return nil
	}

	var critical []*TomcatInspectionResult
	for _, result := range r.Results {
		if result != nil && result.Status.IsCritical() {
			critical = append(critical, result)
		}
	}
	return critical
}

func (r *TomcatInspectionResults) GetWarningResults() []*TomcatInspectionResult {
	if r == nil {
		return nil
	}

	var warnings []*TomcatInspectionResult
	for _, result := range r.Results {
		if result != nil && result.Status.IsWarning() {
			warnings = append(warnings, result)
		}
	}
	return warnings
}

func (r *TomcatInspectionResults) GetFailedResults() []*TomcatInspectionResult {
	if r == nil {
		return nil
	}

	var failed []*TomcatInspectionResult
	for _, result := range r.Results {
		if result != nil && result.Status.IsFailed() {
			failed = append(failed, result)
		}
	}
	return failed
}

func (r *TomcatInspectionResults) HasCritical() bool {
	return r != nil && r.Summary != nil && r.Summary.CriticalInstances > 0
}

func (r *TomcatInspectionResults) HasWarning() bool {
	return r != nil && r.Summary != nil && r.Summary.WarningInstances > 0
}

func (r *TomcatInspectionResults) HasAlerts() bool {
	return r != nil && r.AlertSummary != nil && r.AlertSummary.TotalAlerts > 0
}
