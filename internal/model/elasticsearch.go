// Package model provides data models for inspection tool.
package model

import (
	"fmt"
	"time"
)

// =============================================================================
// Elasticsearch 实例状态枚举
// =============================================================================

// ElasticsearchInstanceStatus represents health status of an Elasticsearch instance.
type ElasticsearchInstanceStatus string

const (
	ElasticsearchStatusNormal   ElasticsearchInstanceStatus = "normal"   // 正常
	ElasticsearchStatusWarning  ElasticsearchInstanceStatus = "warning"  // 警告
	ElasticsearchStatusCritical ElasticsearchInstanceStatus = "critical" // 严重
	ElasticsearchStatusFailed   ElasticsearchInstanceStatus = "failed"   // 采集失败
)

// IsHealthy returns true if status is normal.
func (s ElasticsearchInstanceStatus) IsHealthy() bool {
	return s == ElasticsearchStatusNormal
}

// IsWarning returns true if status is warning.
func (s ElasticsearchInstanceStatus) IsWarning() bool {
	return s == ElasticsearchStatusWarning
}

// IsCritical returns true if status is critical.
func (s ElasticsearchInstanceStatus) IsCritical() bool {
	return s == ElasticsearchStatusCritical
}

// IsFailed returns true if status is failed.
func (s ElasticsearchInstanceStatus) IsFailed() bool {
	return s == ElasticsearchStatusFailed
}

// =============================================================================
// Elasticsearch 节点角色枚举
// =============================================================================

// ElasticsearchRole represents the role of a node in Elasticsearch cluster.
type ElasticsearchRole string

const (
	ElasticsearchRoleMaster       ElasticsearchRole = "master"       // Master-eligible node
	ElasticsearchRoleData         ElasticsearchRole = "data"         // Data node
	ElasticsearchRoleCoordinating ElasticsearchRole = "coordinating" // Coordinating only node
	ElasticsearchRoleIngest       ElasticsearchRole = "ingest"       // Ingest node
	ElasticsearchRoleUnknown      ElasticsearchRole = "unknown"      // Unknown role
)

// IsMaster returns true if this is a master-eligible node.
func (r ElasticsearchRole) IsMaster() bool {
	return r == ElasticsearchRoleMaster
}

// IsData returns true if this is a data node.
func (r ElasticsearchRole) IsData() bool {
	return r == ElasticsearchRoleData
}

// =============================================================================
// Elasticsearch 实例结构体
// =============================================================================

// ElasticsearchInstance represents an Elasticsearch cluster instance.
// Note: Unlike MySQL/Redis which are single instances, ES represents
// an entire cluster discovered through monitoring metrics.
type ElasticsearchInstance struct {
	Address     string            `json:"address"`      // 实例地址 (IP:Port)
	IP          string            `json:"ip"`           // IP 地址
	Port        int               `json:"port"`         // 端口号
	Version     string            `json:"version"`      // ES 版本 (如 8.19.4)
	ClusterName string            `json:"cluster_name"` // 集群名称
	Role        ElasticsearchRole `json:"role"`         // 节点角色 (master/data/coordinating/ingest)
}

// NewElasticsearchInstance creates a new ElasticsearchInstance from address string.
// For ES, the address format is "http://IP:Port" (not "IP:Port").
// We extract the host label directly from monitoring metrics.
func NewElasticsearchInstance(address string) *ElasticsearchInstance {
	if address == "" {
		return nil
	}

	// For ES, address is already in "http://IP:Port" format
	// Store it directly without parsing
	return &ElasticsearchInstance{
		Address: address,
		IP:      address, // Will be updated in collector with actual IP
		Port:    9200,    // Default ES port
	}
}

// SetVersion sets ES version.
func (e *ElasticsearchInstance) SetVersion(version string) {
	e.Version = version
}

// SetClusterName sets ES cluster name.
func (e *ElasticsearchInstance) SetClusterName(clusterName string) {
	e.ClusterName = clusterName
}

// SetRole sets ES node role.
func (e *ElasticsearchInstance) SetRole(role ElasticsearchRole) {
	e.Role = role
}

// String returns a human-readable string representation of instance.
func (e *ElasticsearchInstance) String() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("Elasticsearch[%s] v%s (Cluster: %s, Role: %s)",
		e.Address, e.Version, e.ClusterName, e.Role)
}

// =============================================================================
// Elasticsearch 告警结构体
// =============================================================================

// ElasticsearchAlert represents a threshold violation alert for an Elasticsearch instance.
type ElasticsearchAlert struct {
	Address           string     `json:"address"`             // 实例地址 (IP:Port)
	MetricName        string     `json:"metric_name"`         // 指标名称
	MetricDisplayName string     `json:"metric_display_name"` // 指标中文显示名称
	CurrentValue      float64    `json:"current_value"`       // 当前值
	FormattedValue    string     `json:"formatted_value"`     // 格式化后的当前值
	WarningThreshold  float64    `json:"warning_threshold"`   // 警告阈值
	CriticalThreshold float64    `json:"critical_threshold"`  // 严重阈值
	Level             AlertLevel `json:"level"`               // 告警级别 (复用 alert.go 的 AlertLevel)
	Message           string     `json:"message"`             // 告警消息
}

// NewElasticsearchAlert creates a new ElasticsearchAlert with given parameters.
func NewElasticsearchAlert(address, metricName string, currentValue float64, level AlertLevel) *ElasticsearchAlert {
	return &ElasticsearchAlert{
		Address:      address,
		MetricName:   metricName,
		CurrentValue: currentValue,
		Level:        level,
	}
}

// IsWarning returns true if this alert is at warning level.
func (a *ElasticsearchAlert) IsWarning() bool {
	return a.Level == AlertLevelWarning
}

// IsCritical returns true if this alert is at critical level.
func (a *ElasticsearchAlert) IsCritical() bool {
	return a.Level == AlertLevelCritical
}

// =============================================================================
// Elasticsearch 指标值结构体
// =============================================================================

// ElasticsearchMetricValue represents a collected metric value for an Elasticsearch instance.
// It stores both numeric values and label-extracted string values (e.g., version).
type ElasticsearchMetricValue struct {
	Name           string            `json:"name"`                // 指标名称
	RawValue       float64           `json:"raw_value"`           // 原始数值
	FormattedValue string            `json:"formatted_value"`     // 格式化后的值
	StringValue    string            `json:"string_value"`        // 从标签提取的字符串值（version, cluster_name）
	Labels         map[string]string `json:"labels,omitempty"`    // 原始标签
	IsNA           bool              `json:"is_na"`               // 是否为 N/A
	Timestamp      int64             `json:"timestamp,omitempty"` // 采集时间戳
}

// =============================================================================
// Elasticsearch 巡检结果结构体
// =============================================================================

// ElasticsearchInspectionResult represents inspection result for a single Elasticsearch instance (cluster).
type ElasticsearchInspectionResult struct {
	// 实例元信息
	Instance *ElasticsearchInstance `json:"instance"`

	// 连接状态
	ConnectionStatus bool `json:"connection_status"` // elasticsearch_up = 1

	// 集群健康状态
	ClusterStatus      string `json:"cluster_status"`       // red/yellow/green
	RedIndicesCount    int    `json:"red_indices_count"`    // 红色索引数量
	YellowIndicesCount int    `json:"yellow_indices_count"` // 黄色索引数量
	UnassignedShards   int    `json:"unassigned_shards"`    // 未分配分片数
	ActiveShards       int    `json:"active_shards"`        // 活跃分片数
	NodeCount          int    `json:"node_count"`           // 集群节点数

	// 节点角色
	NodeRole ElasticsearchRole `json:"node_role"` // master/data/coordinating/ingest

	// 资源使用情况
	HeapMemoryPercent      float64 `json:"heap_memory_percent"`       // JVM 堆内存使用率
	CPUPercent             float64 `json:"cpu_percent"`               // CPU 使用率
	DiskUsagePercent       float64 `json:"disk_usage_percent"`        // 磁盘使用率
	FileHandleUsagePercent float64 `json:"file_handle_usage_percent"` // 文件句柄使用率

	// 稳定性指标
	CircuitBreakerTripped bool    `json:"circuit_breaker_tripped"` // 熔断器触发（5分钟内）
	ThreadPoolRejected    bool    `json:"thread_pool_rejected"`    // 线程池拒绝（5分钟内）
	GCDurationSeconds     float64 `json:"gc_duration_seconds"`     // GC 耗时（秒）

	// 运行时间
	Uptime int64 `json:"uptime"` // 运行时间（秒）

	// 整体状态和告警
	Status ElasticsearchInstanceStatus `json:"status"`
	Alerts []*ElasticsearchAlert       `json:"alerts,omitempty"`

	// 采集时间
	CollectedAt time.Time `json:"collected_at"`

	// 错误信息
	Error string `json:"error,omitempty"`

	// 指标集合 (key = metric name)
	Metrics map[string]*ElasticsearchMetricValue `json:"metrics,omitempty"`
}

// NewElasticsearchInspectionResult creates a new ElasticsearchInspectionResult from an ElasticsearchInstance.
func NewElasticsearchInspectionResult(instance *ElasticsearchInstance) *ElasticsearchInspectionResult {
	if instance == nil {
		return &ElasticsearchInspectionResult{
			Status: ElasticsearchStatusFailed,
			Alerts: make([]*ElasticsearchAlert, 0),
		}
	}
	return &ElasticsearchInspectionResult{
		Instance: instance,
		Status:   ElasticsearchStatusNormal,
		Alerts:   make([]*ElasticsearchAlert, 0),
	}
}

// AddAlert adds an alert to this instance and updates status accordingly.
func (r *ElasticsearchInspectionResult) AddAlert(alert *ElasticsearchAlert) {
	if alert == nil {
		return
	}
	r.Alerts = append(r.Alerts, alert)
	// Update instance status to most severe alert level
	if alert.Level == AlertLevelCritical {
		r.Status = ElasticsearchStatusCritical
	} else if alert.Level == AlertLevelWarning && r.Status != ElasticsearchStatusCritical {
		r.Status = ElasticsearchStatusWarning
	}
}

// HasAlerts returns true if this instance has any alerts.
func (r *ElasticsearchInspectionResult) HasAlerts() bool {
	return len(r.Alerts) > 0
}

// GetAddress returns instance address, or empty string if instance is nil.
func (r *ElasticsearchInspectionResult) GetAddress() string {
	if r.Instance == nil {
		return ""
	}
	return r.Instance.Address
}

// SetMetric adds or updates a metric value for this instance.
func (r *ElasticsearchInspectionResult) SetMetric(value *ElasticsearchMetricValue) {
	if r.Metrics == nil {
		r.Metrics = make(map[string]*ElasticsearchMetricValue)
	}
	r.Metrics[value.Name] = value
}

// GetMetric retrieves a metric value by name, returns nil if not found.
func (r *ElasticsearchInspectionResult) GetMetric(name string) *ElasticsearchMetricValue {
	if r.Metrics == nil {
		return nil
	}
	return r.Metrics[name]
}

// =============================================================================
// Elasticsearch 巡检摘要与结果集合
// =============================================================================

// ElasticsearchInspectionSummary provides aggregated statistics about Elasticsearch inspection.
type ElasticsearchInspectionSummary struct {
	TotalInstances    int `json:"total_instances"`    // 实例总数
	NormalInstances   int `json:"normal_instances"`   // 正常实例数
	WarningInstances  int `json:"warning_instances"`  // 警告实例数
	CriticalInstances int `json:"critical_instances"` // 严重实例数
	FailedInstances   int `json:"failed_instances"`   // 采集失败实例数
}

// NewElasticsearchInspectionSummary creates a new ElasticsearchInspectionSummary from inspection results.
func NewElasticsearchInspectionSummary(results []*ElasticsearchInspectionResult) *ElasticsearchInspectionSummary {
	summary := &ElasticsearchInspectionSummary{}
	for _, result := range results {
		if result == nil {
			continue
		}
		summary.TotalInstances++
		switch result.Status {
		case ElasticsearchStatusNormal:
			summary.NormalInstances++
		case ElasticsearchStatusWarning:
			summary.WarningInstances++
		case ElasticsearchStatusCritical:
			summary.CriticalInstances++
		case ElasticsearchStatusFailed:
			summary.FailedInstances++
		}
	}
	return summary
}

// ElasticsearchAlertSummary provides aggregated alert statistics for Elasticsearch inspection.
type ElasticsearchAlertSummary struct {
	TotalAlerts   int `json:"total_alerts"`   // 告警总数
	WarningCount  int `json:"warning_count"`  // 警告级别数量
	CriticalCount int `json:"critical_count"` // 严重级别数量
}

// NewElasticsearchAlertSummary creates a new ElasticsearchAlertSummary from a list of alerts.
func NewElasticsearchAlertSummary(alerts []*ElasticsearchAlert) *ElasticsearchAlertSummary {
	summary := &ElasticsearchAlertSummary{}
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

// ElasticsearchInspectionResults represents complete result of Elasticsearch inspection.
type ElasticsearchInspectionResults struct {
	// 巡检时间信息
	InspectionTime time.Time     `json:"inspection_time"` // 巡检开始时间（Asia/Shanghai）
	Duration       time.Duration `json:"duration"`        // 巡检耗时

	// 巡检摘要
	Summary *ElasticsearchInspectionSummary `json:"summary"` // 摘要统计

	// 实例结果
	Results []*ElasticsearchInspectionResult `json:"results"` // 实例巡检结果列表

	// 告警汇总
	Alerts       []*ElasticsearchAlert      `json:"alerts"`        // 所有告警列表
	AlertSummary *ElasticsearchAlertSummary `json:"alert_summary"` // 告警摘要统计

	// 元数据
	Version string `json:"version,omitempty"` // 工具版本号
}

// NewElasticsearchInspectionResults creates a new ElasticsearchInspectionResults with given inspection time.
func NewElasticsearchInspectionResults(inspectionTime time.Time) *ElasticsearchInspectionResults {
	return &ElasticsearchInspectionResults{
		InspectionTime: inspectionTime,
		Results:        make([]*ElasticsearchInspectionResult, 0),
		Alerts:         make([]*ElasticsearchAlert, 0),
	}
}

// AddResult adds an instance result to the inspection.
func (r *ElasticsearchInspectionResults) AddResult(result *ElasticsearchInspectionResult) {
	if result == nil {
		return
	}
	r.Results = append(r.Results, result)
	// Collect all alerts from this instance
	r.Alerts = append(r.Alerts, result.Alerts...)
}

// Finalize calculates summaries after all instances have been added.
// This should be called after all instances are processed.
func (r *ElasticsearchInspectionResults) Finalize(endTime time.Time) {
	r.Duration = endTime.Sub(r.InspectionTime)
	r.Summary = NewElasticsearchInspectionSummary(r.Results)
	r.AlertSummary = NewElasticsearchAlertSummary(r.Alerts)
}

// GetResultByAddress finds an instance result by address.
func (r *ElasticsearchInspectionResults) GetResultByAddress(address string) *ElasticsearchInspectionResult {
	for _, result := range r.Results {
		if result != nil && result.GetAddress() == address {
			return result
		}
	}
	return nil
}

// GetCriticalResults returns all instances with critical status.
func (r *ElasticsearchInspectionResults) GetCriticalResults() []*ElasticsearchInspectionResult {
	var critical []*ElasticsearchInspectionResult
	for _, result := range r.Results {
		if result != nil && result.Status == ElasticsearchStatusCritical {
			critical = append(critical, result)
		}
	}
	return critical
}

// GetWarningResults returns all instances with warning status.
func (r *ElasticsearchInspectionResults) GetWarningResults() []*ElasticsearchInspectionResult {
	var warning []*ElasticsearchInspectionResult
	for _, result := range r.Results {
		if result != nil && result.Status == ElasticsearchStatusWarning {
			warning = append(warning, result)
		}
	}
	return warning
}

// GetFailedResults returns all instances that failed collection.
func (r *ElasticsearchInspectionResults) GetFailedResults() []*ElasticsearchInspectionResult {
	var failed []*ElasticsearchInspectionResult
	for _, result := range r.Results {
		if result != nil && result.Status == ElasticsearchStatusFailed {
			failed = append(failed, result)
		}
	}
	return failed
}

// HasCritical returns true if any instance has critical status.
func (r *ElasticsearchInspectionResults) HasCritical() bool {
	return r.Summary != nil && r.Summary.CriticalInstances > 0
}

// HasWarning returns true if any instance has warning status.
func (r *ElasticsearchInspectionResults) HasWarning() bool {
	return r.Summary != nil && r.Summary.WarningInstances > 0
}

// HasAlerts returns true if there are any alerts.
func (r *ElasticsearchInspectionResults) HasAlerts() bool {
	return len(r.Alerts) > 0
}

// =============================================================================
// Elasticsearch 指标定义结构体
// =============================================================================

// ElasticsearchMetricDefinition defines an Elasticsearch metric to be collected.
// This struct maps to YAML configuration in configs/elasticsearch-metrics.yaml.
type ElasticsearchMetricDefinition struct {
	Name         string `yaml:"name"`          // 指标唯一标识
	DisplayName  string `yaml:"display_name"`  // 中文显示名称
	Query        string `yaml:"query"`         // PromQL 查询表达式
	Category     string `yaml:"category"`      // 分类 (cluster, node, resource, stability)
	LabelExtract string `yaml:"label_extract"` // 从指标标签提取值（可选，如 version, cluster_name）
	Format       string `yaml:"format"`        // 格式化类型（可选：size, duration, percent）
	Status       string `yaml:"status"`        // 状态（pending=待实现）
	Note         string `yaml:"note"`          // 备注说明
}

// IsPending returns true if this metric is not yet implemented.
// A metric is considered pending if its status is "pending" or if it has no query.
func (m *ElasticsearchMetricDefinition) IsPending() bool {
	return m.Status == "pending" || m.Query == ""
}

// HasLabelExtract returns true if this metric extracts value from a label.
func (m *ElasticsearchMetricDefinition) HasLabelExtract() bool {
	return m.LabelExtract != ""
}

// GetDisplayName returns display name, or name if display name is empty.
func (m *ElasticsearchMetricDefinition) GetDisplayName() string {
	if m.DisplayName != "" {
		return m.DisplayName
	}
	return m.Name
}

// ElasticsearchMetricsConfig represents the root structure of elasticsearch-metrics.yaml file.
// This struct is used by config.LoadElasticsearchMetrics to parse the YAML configuration.
type ElasticsearchMetricsConfig struct {
	Metrics []*ElasticsearchMetricDefinition `yaml:"elasticsearch_metrics" json:"elasticsearch_metrics"` // Elasticsearch 指标定义列表
}
