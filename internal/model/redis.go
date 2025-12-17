// Package model provides data models for the inspection tool.
package model

import (
	"fmt"
	"time"
)

// =============================================================================
// Redis 实例状态枚举
// =============================================================================

// RedisInstanceStatus represents the health status of a Redis instance.
type RedisInstanceStatus string

const (
	RedisStatusNormal   RedisInstanceStatus = "normal"   // 正常
	RedisStatusWarning  RedisInstanceStatus = "warning"  // 警告
	RedisStatusCritical RedisInstanceStatus = "critical" // 严重
	RedisStatusFailed   RedisInstanceStatus = "failed"   // 采集失败
)

// IsHealthy returns true if the status is normal.
func (s RedisInstanceStatus) IsHealthy() bool {
	return s == RedisStatusNormal
}

// IsWarning returns true if the status is warning.
func (s RedisInstanceStatus) IsWarning() bool {
	return s == RedisStatusWarning
}

// IsCritical returns true if the status is critical.
func (s RedisInstanceStatus) IsCritical() bool {
	return s == RedisStatusCritical
}

// IsFailed returns true if the status is failed.
func (s RedisInstanceStatus) IsFailed() bool {
	return s == RedisStatusFailed
}

// =============================================================================
// Redis 节点角色枚举
// =============================================================================

// RedisRole represents the role of a Redis instance in cluster mode.
type RedisRole string

const (
	RedisRoleMaster  RedisRole = "master"  // 主节点
	RedisRoleSlave   RedisRole = "slave"   // 从节点
	RedisRoleUnknown RedisRole = "unknown" // 未知角色
)

// IsMaster returns true if the role is master.
func (r RedisRole) IsMaster() bool {
	return r == RedisRoleMaster
}

// IsSlave returns true if the role is slave.
func (r RedisRole) IsSlave() bool {
	return r == RedisRoleSlave
}

// =============================================================================
// Redis 集群模式枚举
// =============================================================================

// RedisClusterMode represents the Redis cluster architecture mode.
type RedisClusterMode string

const (
	ClusterMode3M3S RedisClusterMode = "3m3s" // 3 主 3 从
	ClusterMode3M6S RedisClusterMode = "3m6s" // 3 主 6 从
)

// Is3M3S returns true if the cluster mode is 3m3s.
func (m RedisClusterMode) Is3M3S() bool {
	return m == ClusterMode3M3S
}

// Is3M6S returns true if the cluster mode is 3m6s.
func (m RedisClusterMode) Is3M6S() bool {
	return m == ClusterMode3M6S
}

// GetExpectedSlaveCount returns the expected number of slaves per master.
// Returns 1 for 3m3s, 2 for 3m6s, 0 for unknown.
func (m RedisClusterMode) GetExpectedSlaveCount() int {
	switch m {
	case ClusterMode3M3S:
		return 1
	case ClusterMode3M6S:
		return 2
	default:
		return 0
	}
}

// =============================================================================
// Redis 实例结构体
// =============================================================================

// RedisInstance represents a Redis instance (master or slave node).
type RedisInstance struct {
	Address         string           `json:"address"`          // 实例地址 (IP:Port)
	IP              string           `json:"ip"`               // IP 地址
	Port            int              `json:"port"`             // 端口号
	ApplicationType string           `json:"application_type"` // 应用类型，固定为 "Redis"
	Version         string           `json:"version"`          // Redis 版本（MVP 阶段显示 N/A）
	Role            RedisRole        `json:"role"`             // 节点角色 (master/slave)
	ClusterEnabled  bool             `json:"cluster_enabled"`  // 是否启用集群
}

// =============================================================================
// 构造函数和辅助方法
// =============================================================================

// NewRedisInstance creates a new RedisInstance from address string.
// The address should be in "IP:Port" format (e.g., "192.18.102.2:7000").
// Returns nil if the address is invalid.
func NewRedisInstance(address string) *RedisInstance {
	ip, port, err := ParseAddress(address)
	if err != nil {
		return nil
	}

	return &RedisInstance{
		Address:         address,
		IP:              ip,
		Port:            port,
		ApplicationType: "Redis",
		Role:            RedisRoleUnknown, // 默认未知，通过采集确定
		ClusterEnabled:  false,            // 默认未启用，通过采集确定
	}
}

// NewRedisInstanceWithRole creates a new RedisInstance with specified role.
func NewRedisInstanceWithRole(address string, role RedisRole) *RedisInstance {
	instance := NewRedisInstance(address)
	if instance != nil {
		instance.Role = role
	}
	return instance
}

// SetVersion sets the Redis version.
func (r *RedisInstance) SetVersion(version string) {
	r.Version = version
}

// SetClusterEnabled sets the cluster mode status.
func (r *RedisInstance) SetClusterEnabled(enabled bool) {
	r.ClusterEnabled = enabled
}

// String returns a human-readable string representation of the instance.
func (r *RedisInstance) String() string {
	if r == nil {
		return "<nil>"
	}
	return fmt.Sprintf("Redis[%s] Role=%s Version=%s Cluster=%t",
		r.Address, r.Role, r.Version, r.ClusterEnabled)
}

// =============================================================================
// Redis 告警结构体
// =============================================================================

// RedisAlert represents a threshold violation alert for a Redis instance.
type RedisAlert struct {
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

// NewRedisAlert creates a new RedisAlert with the given parameters.
func NewRedisAlert(address, metricName string, currentValue float64, level AlertLevel) *RedisAlert {
	return &RedisAlert{
		Address:      address,
		MetricName:   metricName,
		CurrentValue: currentValue,
		Level:        level,
	}
}

// IsWarning returns true if this alert is at warning level.
func (a *RedisAlert) IsWarning() bool {
	return a.Level == AlertLevelWarning
}

// IsCritical returns true if this alert is at critical level.
func (a *RedisAlert) IsCritical() bool {
	return a.Level == AlertLevelCritical
}

// =============================================================================
// Redis 指标值结构体
// =============================================================================

// RedisMetricValue represents a collected metric value for a Redis instance.
type RedisMetricValue struct {
	Name           string            `json:"name"`                // 指标名称
	RawValue       float64           `json:"raw_value"`           // 原始数值
	FormattedValue string            `json:"formatted_value"`     // 格式化后的值
	StringValue    string            `json:"string_value"`        // 从标签提取的字符串值
	Labels         map[string]string `json:"labels,omitempty"`    // 原始标签
	IsNA           bool              `json:"is_na"`               // 是否为 N/A
	Timestamp      int64             `json:"timestamp,omitempty"` // 采集时间戳
}

// =============================================================================
// Redis 巡检结果结构体
// =============================================================================

// RedisInspectionResult represents the inspection result for a single Redis instance.
type RedisInspectionResult struct {
	// 实例元信息
	Instance *RedisInstance `json:"instance"`

	// 连接状态
	ConnectionStatus bool `json:"connection_status"` // redis_up = 1

	// 集群相关
	ClusterEnabled bool   `json:"cluster_enabled"` // redis_cluster_enabled = 1
	ClusterState   string `json:"cluster_state"`   // 集群状态描述

	// 复制相关 (仅 slave 节点)
	MasterLinkStatus bool  `json:"master_link_status"` // redis_master_link_status = 1
	MasterReplOffset int64 `json:"master_repl_offset"` // 已知的 master 复制偏移量
	SlaveReplOffset  int64 `json:"slave_repl_offset"`  // slave 复制偏移量
	ReplicationLag   int64 `json:"replication_lag"`    // 复制延迟（字节）
	MasterPort       int   `json:"master_port"`        // 对应的 master 端口

	// 连接数
	MaxClients       int `json:"max_clients"`       // 最大连接数
	ConnectedClients int `json:"connected_clients"` // 当前连接数

	// master 节点
	ConnectedSlaves int `json:"connected_slaves"` // 连接的 slave 数量

	// 运行时间
	Uptime int64 `json:"uptime"` // 秒

	// 待实现项 (MVP 阶段显示 N/A)
	NonRootUser string `json:"non_root_user"`

	// 整体状态和告警
	Status RedisInstanceStatus `json:"status"`
	Alerts []*RedisAlert       `json:"alerts,omitempty"`

	// 采集时间
	CollectedAt time.Time `json:"collected_at"`

	// 错误信息
	Error string `json:"error,omitempty"`

	// 指标集合 (key = metric name)
	Metrics map[string]*RedisMetricValue `json:"metrics,omitempty"`
}

// NewRedisInspectionResult creates a new RedisInspectionResult from a RedisInstance.
func NewRedisInspectionResult(instance *RedisInstance) *RedisInspectionResult {
	if instance == nil {
		return &RedisInspectionResult{
			Status:      RedisStatusFailed,
			NonRootUser: "N/A",
			Alerts:      make([]*RedisAlert, 0),
		}
	}
	return &RedisInspectionResult{
		Instance:    instance,
		Status:      RedisStatusNormal,
		NonRootUser: "N/A", // MVP 阶段固定为 N/A
		Alerts:      make([]*RedisAlert, 0),
	}
}

// AddAlert adds an alert to this instance and updates the status accordingly.
func (r *RedisInspectionResult) AddAlert(alert *RedisAlert) {
	if alert == nil {
		return
	}
	r.Alerts = append(r.Alerts, alert)
	// Update instance status to the most severe alert level
	if alert.Level == AlertLevelCritical {
		r.Status = RedisStatusCritical
	} else if alert.Level == AlertLevelWarning && r.Status != RedisStatusCritical {
		r.Status = RedisStatusWarning
	}
}

// HasAlerts returns true if this instance has any alerts.
func (r *RedisInspectionResult) HasAlerts() bool {
	return len(r.Alerts) > 0
}

// GetConnectionUsagePercent calculates the connection usage percentage.
// Returns 0 if MaxClients is 0 to avoid division by zero.
func (r *RedisInspectionResult) GetConnectionUsagePercent() float64 {
	if r.MaxClients == 0 {
		return 0
	}
	return float64(r.ConnectedClients) / float64(r.MaxClients) * 100
}

// GetAddress returns the instance address, or empty string if instance is nil.
func (r *RedisInspectionResult) GetAddress() string {
	if r.Instance == nil {
		return ""
	}
	return r.Instance.Address
}

// SetMetric adds or updates a metric value for this instance.
func (r *RedisInspectionResult) SetMetric(value *RedisMetricValue) {
	if r.Metrics == nil {
		r.Metrics = make(map[string]*RedisMetricValue)
	}
	r.Metrics[value.Name] = value
}

// GetMetric retrieves a metric value by name, returns nil if not found.
func (r *RedisInspectionResult) GetMetric(name string) *RedisMetricValue {
	if r.Metrics == nil {
		return nil
	}
	return r.Metrics[name]
}

// =============================================================================
// Redis 巡检摘要与结果集合
// =============================================================================

// RedisInspectionSummary provides aggregated statistics about the Redis inspection.
type RedisInspectionSummary struct {
	TotalInstances    int `json:"total_instances"`    // 实例总数
	NormalInstances   int `json:"normal_instances"`   // 正常实例数
	WarningInstances  int `json:"warning_instances"`  // 警告实例数
	CriticalInstances int `json:"critical_instances"` // 严重实例数
	FailedInstances   int `json:"failed_instances"`   // 采集失败实例数
}

// NewRedisInspectionSummary creates a new RedisInspectionSummary from inspection results.
func NewRedisInspectionSummary(results []*RedisInspectionResult) *RedisInspectionSummary {
	summary := &RedisInspectionSummary{}
	for _, result := range results {
		if result == nil {
			continue
		}
		summary.TotalInstances++
		switch result.Status {
		case RedisStatusNormal:
			summary.NormalInstances++
		case RedisStatusWarning:
			summary.WarningInstances++
		case RedisStatusCritical:
			summary.CriticalInstances++
		case RedisStatusFailed:
			summary.FailedInstances++
		}
	}
	return summary
}

// RedisAlertSummary provides aggregated alert statistics for Redis inspection.
type RedisAlertSummary struct {
	TotalAlerts   int `json:"total_alerts"`   // 告警总数
	WarningCount  int `json:"warning_count"`  // 警告级别数量
	CriticalCount int `json:"critical_count"` // 严重级别数量
}

// NewRedisAlertSummary creates a new RedisAlertSummary from a list of alerts.
func NewRedisAlertSummary(alerts []*RedisAlert) *RedisAlertSummary {
	summary := &RedisAlertSummary{}
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

// RedisInspectionResults represents the complete result of Redis inspection.
type RedisInspectionResults struct {
	// 巡检时间信息
	InspectionTime time.Time     `json:"inspection_time"` // 巡检开始时间（Asia/Shanghai）
	Duration       time.Duration `json:"duration"`        // 巡检耗时

	// 巡检摘要
	Summary *RedisInspectionSummary `json:"summary"` // 摘要统计

	// 实例结果
	Results []*RedisInspectionResult `json:"results"` // 实例巡检结果列表

	// 告警汇总
	Alerts       []*RedisAlert      `json:"alerts"`        // 所有告警列表
	AlertSummary *RedisAlertSummary `json:"alert_summary"` // 告警摘要统计

	// 元数据
	Version string `json:"version,omitempty"` // 工具版本号
}

// NewRedisInspectionResults creates a new RedisInspectionResults with the given inspection time.
func NewRedisInspectionResults(inspectionTime time.Time) *RedisInspectionResults {
	return &RedisInspectionResults{
		InspectionTime: inspectionTime,
		Results:        make([]*RedisInspectionResult, 0),
		Alerts:         make([]*RedisAlert, 0),
	}
}

// AddResult adds an instance result to the inspection.
func (r *RedisInspectionResults) AddResult(result *RedisInspectionResult) {
	if result == nil {
		return
	}
	r.Results = append(r.Results, result)
	// Collect all alerts from this instance
	r.Alerts = append(r.Alerts, result.Alerts...)
}

// Finalize calculates summaries after all instances have been added.
// This should be called after all instances are processed.
func (r *RedisInspectionResults) Finalize(endTime time.Time) {
	r.Duration = endTime.Sub(r.InspectionTime)
	r.Summary = NewRedisInspectionSummary(r.Results)
	r.AlertSummary = NewRedisAlertSummary(r.Alerts)
}

// GetResultByAddress finds an instance result by address.
func (r *RedisInspectionResults) GetResultByAddress(address string) *RedisInspectionResult {
	for _, result := range r.Results {
		if result != nil && result.GetAddress() == address {
			return result
		}
	}
	return nil
}

// GetCriticalResults returns all instances with critical status.
func (r *RedisInspectionResults) GetCriticalResults() []*RedisInspectionResult {
	var critical []*RedisInspectionResult
	for _, result := range r.Results {
		if result != nil && result.Status == RedisStatusCritical {
			critical = append(critical, result)
		}
	}
	return critical
}

// GetWarningResults returns all instances with warning status.
func (r *RedisInspectionResults) GetWarningResults() []*RedisInspectionResult {
	var warning []*RedisInspectionResult
	for _, result := range r.Results {
		if result != nil && result.Status == RedisStatusWarning {
			warning = append(warning, result)
		}
	}
	return warning
}

// GetFailedResults returns all instances that failed collection.
func (r *RedisInspectionResults) GetFailedResults() []*RedisInspectionResult {
	var failed []*RedisInspectionResult
	for _, result := range r.Results {
		if result != nil && result.Status == RedisStatusFailed {
			failed = append(failed, result)
		}
	}
	return failed
}

// HasCritical returns true if any instance has critical status.
func (r *RedisInspectionResults) HasCritical() bool {
	return r.Summary != nil && r.Summary.CriticalInstances > 0
}

// HasWarning returns true if any instance has warning status.
func (r *RedisInspectionResults) HasWarning() bool {
	return r.Summary != nil && r.Summary.WarningInstances > 0
}

// HasAlerts returns true if there are any alerts.
func (r *RedisInspectionResults) HasAlerts() bool {
	return len(r.Alerts) > 0
}

// =============================================================================
// Redis Metric Definition
// =============================================================================

// RedisMetricDefinition defines a Redis metric from redis-metrics.yaml.
type RedisMetricDefinition struct {
	Name        string `yaml:"name"`         // Metric unique identifier
	DisplayName string `yaml:"display_name"` // Chinese display name
	Query       string `yaml:"query"`        // PromQL query expression
	Category    string `yaml:"category"`     // Category: connection, cluster, replication, status, info, security
	Format      string `yaml:"format"`       // Format type: size, duration, percent (optional)
	Status      string `yaml:"status"`       // Status: pending = not yet implemented (optional)
	Note        string `yaml:"note"`         // Note/description
}

// IsPending returns true if this metric is not yet implemented.
// A metric is considered pending if its status is "pending" or if it has no query.
func (m *RedisMetricDefinition) IsPending() bool {
	return m.Status == "pending" || m.Query == ""
}
