// Package model provides data models for the inspection tool.
package model

import (
	"fmt"
	"time"
)

// =============================================================================
// Nginx 实例状态枚举
// =============================================================================

// NginxInstanceStatus represents the health status of a Nginx instance.
type NginxInstanceStatus string

const (
	NginxStatusNormal   NginxInstanceStatus = "normal"   // 正常
	NginxStatusWarning  NginxInstanceStatus = "warning"  // 警告
	NginxStatusCritical NginxInstanceStatus = "critical" // 严重
	NginxStatusFailed   NginxInstanceStatus = "failed"   // 采集失败
)

// IsHealthy returns true if the status is normal.
func (s NginxInstanceStatus) IsHealthy() bool {
	return s == NginxStatusNormal
}

// IsWarning returns true if the status is warning.
func (s NginxInstanceStatus) IsWarning() bool {
	return s == NginxStatusWarning
}

// IsCritical returns true if the status is critical.
func (s NginxInstanceStatus) IsCritical() bool {
	return s == NginxStatusCritical
}

// IsFailed returns true if the status is failed.
func (s NginxInstanceStatus) IsFailed() bool {
	return s == NginxStatusFailed
}

// =============================================================================
// Nginx 实例结构体
// =============================================================================

// NginxInstance represents a Nginx/OpenResty instance.
type NginxInstance struct {
	Identifier      string `json:"identifier"`       // 唯一标识 (hostname:container 或 hostname:port)
	Hostname        string `json:"hostname"`         // 主机名 (agent_hostname)
	IP              string `json:"ip"`               // IP 地址（从 N9E 元信息获取）
	Port            int    `json:"port"`             // 监听端口
	Container       string `json:"container"`        // 容器名称（二进制部署时为空）
	ApplicationType string `json:"application_type"` // 应用类型 (nginx/openresty)
	Version         string `json:"version"`          // 版本号
	InstallPath     string `json:"install_path"`     // 安装路径
	ErrorLogPath    string `json:"error_log_path"`   // 错误日志路径
}

// =============================================================================
// 构造函数和辅助方法
// =============================================================================

// GenerateNginxIdentifier generates the unique identifier for a Nginx instance.
// If container is not empty, returns "hostname:container".
// Otherwise, returns "hostname:port".
func GenerateNginxIdentifier(hostname string, port int, container string) string {
	if container != "" {
		return fmt.Sprintf("%s:%s", hostname, container)
	}
	return fmt.Sprintf("%s:%d", hostname, port)
}

// NewNginxInstance creates a new NginxInstance for binary deployment.
// The identifier is generated from hostname and port.
func NewNginxInstance(hostname string, port int) *NginxInstance {
	if hostname == "" || port <= 0 || port > 65535 {
		return nil
	}

	return &NginxInstance{
		Identifier:      GenerateNginxIdentifier(hostname, port, ""),
		Hostname:        hostname,
		Port:            port,
		ApplicationType: "nginx", // 默认为 nginx
	}
}

// NewNginxInstanceWithContainer creates a new NginxInstance for container deployment.
// The identifier is generated from hostname and container name.
func NewNginxInstanceWithContainer(hostname, container string) *NginxInstance {
	if hostname == "" || container == "" {
		return nil
	}

	return &NginxInstance{
		Identifier:      GenerateNginxIdentifier(hostname, 0, container),
		Hostname:        hostname,
		Container:       container,
		ApplicationType: "nginx", // 默认为 nginx
	}
}

// SetApplicationType sets the application type (nginx or openresty).
func (n *NginxInstance) SetApplicationType(appType string) {
	n.ApplicationType = appType
}

// SetVersion sets the Nginx/OpenResty version.
func (n *NginxInstance) SetVersion(version string) {
	n.Version = version
}

// SetInstallPath sets the installation path.
func (n *NginxInstance) SetInstallPath(path string) {
	n.InstallPath = path
}

// SetErrorLogPath sets the error log path.
func (n *NginxInstance) SetErrorLogPath(path string) {
	n.ErrorLogPath = path
}

// SetIP sets the IP address.
func (n *NginxInstance) SetIP(ip string) {
	n.IP = ip
}

// IsContainerDeployment returns true if this is a container deployment.
func (n *NginxInstance) IsContainerDeployment() bool {
	return n.Container != ""
}

// String returns a human-readable string representation of the instance.
func (n *NginxInstance) String() string {
	if n == nil {
		return "<nil>"
	}
	if n.Container != "" {
		return fmt.Sprintf("Nginx[%s] %s v%s (Container: %s)",
			n.Identifier, n.ApplicationType, n.Version, n.Container)
	}
	return fmt.Sprintf("Nginx[%s] %s v%s (Port: %d)",
		n.Identifier, n.ApplicationType, n.Version, n.Port)
}

// =============================================================================
// Nginx Upstream 状态结构体
// =============================================================================

// NginxUpstreamStatus represents the health status of a Nginx upstream backend.
type NginxUpstreamStatus struct {
	UpstreamName   string `json:"upstream_name"`   // upstream 标签（upstream 组名）
	BackendAddress string `json:"backend_address"` // name 标签 (IP:Port)
	Status         bool   `json:"status"`          // status_code=1 为 true
	RiseCount      int    `json:"rise_count"`      // 连续成功次数
	FallCount      int    `json:"fall_count"`      // 连续失败次数
}

// NewNginxUpstreamStatus creates a new NginxUpstreamStatus.
func NewNginxUpstreamStatus(upstreamName, backendAddress string, status bool) *NginxUpstreamStatus {
	return &NginxUpstreamStatus{
		UpstreamName:   upstreamName,
		BackendAddress: backendAddress,
		Status:         status,
	}
}

// IsHealthy returns true if the upstream backend is healthy (status=true).
func (u *NginxUpstreamStatus) IsHealthy() bool {
	return u.Status
}

// =============================================================================
// Nginx 告警结构体
// =============================================================================

// NginxAlert represents a threshold violation alert for a Nginx instance.
type NginxAlert struct {
	Identifier        string     `json:"identifier"`          // 实例唯一标识
	MetricName        string     `json:"metric_name"`         // 指标名称
	MetricDisplayName string     `json:"metric_display_name"` // 指标中文显示名称
	CurrentValue      float64    `json:"current_value"`       // 当前值
	FormattedValue    string     `json:"formatted_value"`     // 格式化后的当前值
	WarningThreshold  float64    `json:"warning_threshold"`   // 警告阈值
	CriticalThreshold float64    `json:"critical_threshold"`  // 严重阈值
	Level             AlertLevel `json:"level"`               // 告警级别 (复用 alert.go 的 AlertLevel)
	Message           string     `json:"message"`             // 告警消息
}

// NewNginxAlert creates a new NginxAlert with the given parameters.
func NewNginxAlert(identifier, metricName string, currentValue float64, level AlertLevel) *NginxAlert {
	return &NginxAlert{
		Identifier:   identifier,
		MetricName:   metricName,
		CurrentValue: currentValue,
		Level:        level,
	}
}

// IsWarning returns true if this alert is at warning level.
func (a *NginxAlert) IsWarning() bool {
	return a.Level == AlertLevelWarning
}

// IsCritical returns true if this alert is at critical level.
func (a *NginxAlert) IsCritical() bool {
	return a.Level == AlertLevelCritical
}

// =============================================================================
// Nginx 指标值结构体
// =============================================================================

// NginxMetricValue represents a collected metric value for a Nginx instance.
// It stores both numeric values and label-extracted string values.
type NginxMetricValue struct {
	Name           string            `json:"name"`                // 指标名称
	RawValue       float64           `json:"raw_value"`           // 原始数值
	FormattedValue string            `json:"formatted_value"`     // 格式化后的值
	StringValue    string            `json:"string_value"`        // 从标签提取的字符串值
	Labels         map[string]string `json:"labels,omitempty"`    // 原始标签
	IsNA           bool              `json:"is_na"`               // 是否为 N/A
	Timestamp      int64             `json:"timestamp,omitempty"` // 采集时间戳
}

// =============================================================================
// Nginx 巡检结果结构体
// =============================================================================

// NginxInspectionResult represents the inspection result for a single Nginx instance.
type NginxInspectionResult struct {
	// 实例元信息
	Instance *NginxInstance `json:"instance"`

	// 运行状态
	Up bool `json:"up"` // nginx_up = 1

	// 连接相关
	ActiveConnections      int     `json:"active_connections"`       // 活跃连接数
	WorkerProcesses        int     `json:"worker_processes"`         // worker 进程数
	WorkerConnections      int     `json:"worker_connections"`       // 单 worker 最大连接数
	ConnectionUsagePercent float64 `json:"connection_usage_percent"` // 连接使用率（-1 表示无法计算）

	// 错误页配置
	ErrorPage4xxConfigured bool `json:"error_page_4xx_configured"` // 4xx 错误页配置
	ErrorPage5xxConfigured bool `json:"error_page_5xx_configured"` // 5xx 错误页配置

	// 错误日志
	LastErrorTimestamp     int64  `json:"last_error_timestamp"`      // 最近错误日志时间戳（0 表示无错误）
	LastErrorTimeFormatted string `json:"last_error_time_formatted"` // 格式化显示

	// 安全配置
	NonRootUser bool `json:"non_root_user"` // 是否非 root 用户启动

	// Upstream 后端状态
	UpstreamStatus []NginxUpstreamStatus `json:"upstream_status"` // 后端健康状态（可为空数组）

	// 整体状态和告警
	Status NginxInstanceStatus `json:"status"`
	Alerts []*NginxAlert       `json:"alerts,omitempty"`

	// 采集时间
	CollectedAt time.Time `json:"collected_at"`

	// 错误信息
	Error string `json:"error,omitempty"`

	// 指标集合 (key = metric name)
	Metrics map[string]*NginxMetricValue `json:"metrics,omitempty"`
}

// NewNginxInspectionResult creates a new NginxInspectionResult from a NginxInstance.
func NewNginxInspectionResult(instance *NginxInstance) *NginxInspectionResult {
	if instance == nil {
		return &NginxInspectionResult{
			Status:                 NginxStatusFailed,
			Alerts:                 make([]*NginxAlert, 0),
			UpstreamStatus:         make([]NginxUpstreamStatus, 0),
			ConnectionUsagePercent: -1, // 无法计算
		}
	}
	return &NginxInspectionResult{
		Instance:               instance,
		Status:                 NginxStatusNormal,
		Alerts:                 make([]*NginxAlert, 0),
		UpstreamStatus:         make([]NginxUpstreamStatus, 0),
		ConnectionUsagePercent: -1, // 初始为无法计算，采集后更新
	}
}

// AddAlert adds an alert to this instance and updates the status accordingly.
func (r *NginxInspectionResult) AddAlert(alert *NginxAlert) {
	if alert == nil {
		return
	}
	r.Alerts = append(r.Alerts, alert)
	// Update instance status to the most severe alert level
	if alert.Level == AlertLevelCritical {
		r.Status = NginxStatusCritical
	} else if alert.Level == AlertLevelWarning && r.Status != NginxStatusCritical {
		r.Status = NginxStatusWarning
	}
}

// HasAlerts returns true if this instance has any alerts.
func (r *NginxInspectionResult) HasAlerts() bool {
	return len(r.Alerts) > 0
}

// GetConnectionUsagePercent calculates and returns the connection usage percentage.
// Returns -1 if WorkerProcesses or WorkerConnections is 0 (cannot calculate).
// Formula: ActiveConnections / (WorkerProcesses * WorkerConnections) * 100
func (r *NginxInspectionResult) GetConnectionUsagePercent() float64 {
	if r.WorkerProcesses == 0 || r.WorkerConnections == 0 {
		return -1
	}
	maxConns := r.WorkerProcesses * r.WorkerConnections
	return float64(r.ActiveConnections) / float64(maxConns) * 100
}

// CalculateConnectionUsagePercent calculates and stores the connection usage percentage.
// Should be called after WorkerProcesses, WorkerConnections, and ActiveConnections are set.
func (r *NginxInspectionResult) CalculateConnectionUsagePercent() {
	r.ConnectionUsagePercent = r.GetConnectionUsagePercent()
}

// GetIdentifier returns the instance identifier, or empty string if instance is nil.
func (r *NginxInspectionResult) GetIdentifier() string {
	if r.Instance == nil {
		return ""
	}
	return r.Instance.Identifier
}

// SetMetric adds or updates a metric value for this instance.
func (r *NginxInspectionResult) SetMetric(value *NginxMetricValue) {
	if r.Metrics == nil {
		r.Metrics = make(map[string]*NginxMetricValue)
	}
	r.Metrics[value.Name] = value
}

// GetMetric retrieves a metric value by name, returns nil if not found.
func (r *NginxInspectionResult) GetMetric(name string) *NginxMetricValue {
	if r.Metrics == nil {
		return nil
	}
	return r.Metrics[name]
}

// AddUpstreamStatus adds an upstream backend status to this instance.
func (r *NginxInspectionResult) AddUpstreamStatus(status NginxUpstreamStatus) {
	r.UpstreamStatus = append(r.UpstreamStatus, status)
}

// HasUpstreamStatus returns true if this instance has any upstream status records.
func (r *NginxInspectionResult) HasUpstreamStatus() bool {
	return len(r.UpstreamStatus) > 0
}

// GetUnhealthyUpstreams returns all unhealthy upstream backends.
func (r *NginxInspectionResult) GetUnhealthyUpstreams() []NginxUpstreamStatus {
	var unhealthy []NginxUpstreamStatus
	for _, u := range r.UpstreamStatus {
		if !u.Status {
			unhealthy = append(unhealthy, u)
		}
	}
	return unhealthy
}

// FormatLastErrorTime formats the last error timestamp to a human-readable string.
// Returns "无错误" if the timestamp is 0 (no error).
// Otherwise, returns the time formatted as "2006-01-02 15:04:05".
func (r *NginxInspectionResult) FormatLastErrorTime(loc *time.Location) string {
	if r.LastErrorTimestamp == 0 {
		return "无错误"
	}
	t := time.Unix(r.LastErrorTimestamp, 0)
	if loc != nil {
		t = t.In(loc)
	}
	return t.Format("2006-01-02 15:04:05")
}

// =============================================================================
// Nginx 巡检摘要与结果集合
// =============================================================================

// NginxInspectionSummary provides aggregated statistics about the Nginx inspection.
type NginxInspectionSummary struct {
	TotalInstances    int `json:"total_instances"`    // 实例总数
	NormalInstances   int `json:"normal_instances"`   // 正常实例数
	WarningInstances  int `json:"warning_instances"`  // 警告实例数
	CriticalInstances int `json:"critical_instances"` // 严重实例数
	FailedInstances   int `json:"failed_instances"`   // 采集失败实例数
}

// NewNginxInspectionSummary creates a new NginxInspectionSummary from inspection results.
func NewNginxInspectionSummary(results []*NginxInspectionResult) *NginxInspectionSummary {
	summary := &NginxInspectionSummary{}
	for _, result := range results {
		if result == nil {
			continue
		}
		summary.TotalInstances++
		switch result.Status {
		case NginxStatusNormal:
			summary.NormalInstances++
		case NginxStatusWarning:
			summary.WarningInstances++
		case NginxStatusCritical:
			summary.CriticalInstances++
		case NginxStatusFailed:
			summary.FailedInstances++
		}
	}
	return summary
}

// NginxAlertSummary provides aggregated alert statistics for Nginx inspection.
type NginxAlertSummary struct {
	TotalAlerts   int `json:"total_alerts"`   // 告警总数
	WarningCount  int `json:"warning_count"`  // 警告级别数量
	CriticalCount int `json:"critical_count"` // 严重级别数量
}

// NewNginxAlertSummary creates a new NginxAlertSummary from a list of alerts.
func NewNginxAlertSummary(alerts []*NginxAlert) *NginxAlertSummary {
	summary := &NginxAlertSummary{}
	for _, alert := range alerts {
		if alert == nil {
			continue
		}
		summary.TotalAlerts++
		switch alert.Level {
		case AlertLevelWarning:
			summary.WarningCount++
		case AlertLevelCritical:
			summary.CriticalCount++
		}
	}
	return summary
}

// NginxInspectionResults represents the complete result of Nginx inspection.
type NginxInspectionResults struct {
	// 巡检时间信息
	InspectionTime time.Time     `json:"inspection_time"` // 巡检开始时间（Asia/Shanghai）
	Duration       time.Duration `json:"duration"`        // 巡检耗时

	// 巡检摘要
	Summary *NginxInspectionSummary `json:"summary"` // 摘要统计

	// 实例结果
	Results []*NginxInspectionResult `json:"results"` // 实例巡检结果列表

	// 告警汇总
	Alerts       []*NginxAlert      `json:"alerts"`        // 所有告警列表
	AlertSummary *NginxAlertSummary `json:"alert_summary"` // 告警摘要统计

	// 元数据
	Version string `json:"version,omitempty"` // 工具版本号
}

// NewNginxInspectionResults creates a new NginxInspectionResults with the given inspection time.
func NewNginxInspectionResults(inspectionTime time.Time) *NginxInspectionResults {
	return &NginxInspectionResults{
		InspectionTime: inspectionTime,
		Results:        make([]*NginxInspectionResult, 0),
		Alerts:         make([]*NginxAlert, 0),
	}
}

// AddResult adds an instance result to the inspection.
func (r *NginxInspectionResults) AddResult(result *NginxInspectionResult) {
	if result == nil {
		return
	}
	r.Results = append(r.Results, result)
	// Collect all alerts from this instance
	r.Alerts = append(r.Alerts, result.Alerts...)
}

// Finalize calculates summaries after all instances have been added.
// This should be called after all instances are processed.
func (r *NginxInspectionResults) Finalize(endTime time.Time) {
	r.Duration = endTime.Sub(r.InspectionTime)
	r.Summary = NewNginxInspectionSummary(r.Results)
	r.AlertSummary = NewNginxAlertSummary(r.Alerts)
}

// GetResultByIdentifier finds an instance result by identifier.
func (r *NginxInspectionResults) GetResultByIdentifier(identifier string) *NginxInspectionResult {
	for _, result := range r.Results {
		if result != nil && result.GetIdentifier() == identifier {
			return result
		}
	}
	return nil
}

// GetCriticalResults returns all instances with critical status.
func (r *NginxInspectionResults) GetCriticalResults() []*NginxInspectionResult {
	var critical []*NginxInspectionResult
	for _, result := range r.Results {
		if result != nil && result.Status == NginxStatusCritical {
			critical = append(critical, result)
		}
	}
	return critical
}

// GetWarningResults returns all instances with warning status.
func (r *NginxInspectionResults) GetWarningResults() []*NginxInspectionResult {
	var warning []*NginxInspectionResult
	for _, result := range r.Results {
		if result != nil && result.Status == NginxStatusWarning {
			warning = append(warning, result)
		}
	}
	return warning
}

// GetFailedResults returns all instances that failed collection.
func (r *NginxInspectionResults) GetFailedResults() []*NginxInspectionResult {
	var failed []*NginxInspectionResult
	for _, result := range r.Results {
		if result != nil && result.Status == NginxStatusFailed {
			failed = append(failed, result)
		}
	}
	return failed
}

// HasCritical returns true if any instance has critical status.
func (r *NginxInspectionResults) HasCritical() bool {
	return r.Summary != nil && r.Summary.CriticalInstances > 0
}

// HasWarning returns true if any instance has warning status.
func (r *NginxInspectionResults) HasWarning() bool {
	return r.Summary != nil && r.Summary.WarningInstances > 0
}

// HasAlerts returns true if there are any alerts.
func (r *NginxInspectionResults) HasAlerts() bool {
	return len(r.Alerts) > 0
}

// =============================================================================
// Nginx 指标定义结构体
// =============================================================================

// NginxMetricDefinition defines a Nginx metric to be collected.
// This struct maps to the YAML configuration in configs/nginx-metrics.yaml.
type NginxMetricDefinition struct {
	Name         string   `yaml:"name" json:"name"`                    // 指标唯一标识
	DisplayName  string   `yaml:"display_name" json:"display_name"`    // 中文显示名称
	Query        string   `yaml:"query" json:"query"`                  // PromQL 查询表达式
	Category     string   `yaml:"category" json:"category"`            // 分类 (connection, info, config, log, security, upstream)
	LabelExtract []string `yaml:"label_extract" json:"label_extract"`  // 从指标标签提取值（可选）
	Format       string   `yaml:"format" json:"format"`                // 格式化类型（可选：size, duration, percent, timestamp）
	Status       string   `yaml:"status" json:"status"`                // 状态（pending=待实现）
	Note         string   `yaml:"note" json:"note"`                    // 备注说明
}

// IsPending returns true if this metric is not yet implemented.
// A metric is considered pending if its status is "pending" or if it has no query.
func (m *NginxMetricDefinition) IsPending() bool {
	return m.Status == "pending" || m.Query == ""
}

// HasLabelExtract returns true if this metric extracts values from labels.
func (m *NginxMetricDefinition) HasLabelExtract() bool {
	return len(m.LabelExtract) > 0
}

// GetDisplayName returns the display name, or the name if display name is empty.
func (m *NginxMetricDefinition) GetDisplayName() string {
	if m.DisplayName != "" {
		return m.DisplayName
	}
	return m.Name
}

// NginxMetricsConfig represents the root structure of nginx-metrics.yaml file.
// This struct is used by config.LoadNginxMetrics to parse the YAML configuration.
type NginxMetricsConfig struct {
	Metrics []*NginxMetricDefinition `yaml:"nginx_metrics" json:"nginx_metrics"` // Nginx 指标定义列表
}
