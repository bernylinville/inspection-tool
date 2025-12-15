// Package model provides data models for the inspection tool.
package model

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// =============================================================================
// MySQL 实例状态枚举
// =============================================================================

// MySQLInstanceStatus represents the health status of a MySQL instance.
type MySQLInstanceStatus string

const (
	MySQLStatusNormal   MySQLInstanceStatus = "normal"   // 正常
	MySQLStatusWarning  MySQLInstanceStatus = "warning"  // 警告
	MySQLStatusCritical MySQLInstanceStatus = "critical" // 严重
	MySQLStatusFailed   MySQLInstanceStatus = "failed"   // 采集失败
)

// IsHealthy returns true if the status is normal.
func (s MySQLInstanceStatus) IsHealthy() bool {
	return s == MySQLStatusNormal
}

// IsWarning returns true if the status is warning.
func (s MySQLInstanceStatus) IsWarning() bool {
	return s == MySQLStatusWarning
}

// IsCritical returns true if the status is critical.
func (s MySQLInstanceStatus) IsCritical() bool {
	return s == MySQLStatusCritical
}

// IsFailed returns true if the status is failed.
func (s MySQLInstanceStatus) IsFailed() bool {
	return s == MySQLStatusFailed
}

// =============================================================================
// MySQL 集群模式枚举
// =============================================================================

// MySQLClusterMode represents the MySQL cluster architecture mode.
type MySQLClusterMode string

const (
	ClusterModeMGR         MySQLClusterMode = "mgr"          // MySQL 8.0 MGR 模式
	ClusterModeDualMaster  MySQLClusterMode = "dual-master"  // 双主模式
	ClusterModeMasterSlave MySQLClusterMode = "master-slave" // 主从模式
)

// IsMGR returns true if the cluster mode is MGR.
func (m MySQLClusterMode) IsMGR() bool {
	return m == ClusterModeMGR
}

// IsDualMaster returns true if the cluster mode is dual-master.
func (m MySQLClusterMode) IsDualMaster() bool {
	return m == ClusterModeDualMaster
}

// IsMasterSlave returns true if the cluster mode is master-slave.
func (m MySQLClusterMode) IsMasterSlave() bool {
	return m == ClusterModeMasterSlave
}

// =============================================================================
// MySQL MGR 角色枚举
// =============================================================================

// MySQLMGRRole represents the role of a node in MGR cluster.
type MySQLMGRRole string

const (
	MGRRolePrimary   MySQLMGRRole = "PRIMARY"   // 主节点
	MGRRoleSecondary MySQLMGRRole = "SECONDARY" // 从节点
	MGRRoleUnknown   MySQLMGRRole = "UNKNOWN"   // 未知
)

// IsPrimary returns true if the role is PRIMARY.
func (r MySQLMGRRole) IsPrimary() bool {
	return r == MGRRolePrimary
}

// IsSecondary returns true if the role is SECONDARY.
func (r MySQLMGRRole) IsSecondary() bool {
	return r == MGRRoleSecondary
}

// =============================================================================
// MySQL 实例结构体
// =============================================================================

// MySQLInstance represents a MySQL database instance.
type MySQLInstance struct {
	Address       string           `json:"address"`        // 实例地址 (IP:Port)
	IP            string           `json:"ip"`             // IP 地址
	Port          int              `json:"port"`           // 端口号
	DatabaseType  string           `json:"database_type"`  // 数据库类型，固定为 "MySQL"
	Version       string           `json:"version"`        // 数据库版本 (如 8.0.39)
	InnoDBVersion string           `json:"innodb_version"` // InnoDB 版本
	ServerID      string           `json:"server_id"`      // Server ID
	ClusterMode   MySQLClusterMode `json:"cluster_mode"`   // 集群模式
}

// =============================================================================
// 辅助函数
// =============================================================================

// ParseAddress parses "IP:Port" format and returns IP and port.
// Returns error if the format is invalid.
func ParseAddress(address string) (ip string, port int, err error) {
	parts := strings.Split(address, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid address format: %s, expected IP:Port", address)
	}

	ip = strings.TrimSpace(parts[0])
	if ip == "" {
		return "", 0, fmt.Errorf("empty IP in address: %s", address)
	}

	portStr := strings.TrimSpace(parts[1])
	port, err = strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port in address %s: %w", address, err)
	}

	if port <= 0 || port > 65535 {
		return "", 0, fmt.Errorf("port out of range in address %s: %d", address, port)
	}

	return ip, port, nil
}

// NewMySQLInstance creates a new MySQLInstance from address string.
// The address should be in "IP:Port" format.
// Returns nil if the address is invalid.
func NewMySQLInstance(address string) *MySQLInstance {
	ip, port, err := ParseAddress(address)
	if err != nil {
		return nil
	}

	return &MySQLInstance{
		Address:      address,
		IP:           ip,
		Port:         port,
		DatabaseType: "MySQL",
	}
}

// NewMySQLInstanceWithClusterMode creates a new MySQLInstance with cluster mode.
func NewMySQLInstanceWithClusterMode(address string, clusterMode MySQLClusterMode) *MySQLInstance {
	instance := NewMySQLInstance(address)
	if instance != nil {
		instance.ClusterMode = clusterMode
	}
	return instance
}

// SetVersion sets the MySQL version and optionally InnoDB version.
func (m *MySQLInstance) SetVersion(version string, innodbVersion string) {
	m.Version = version
	m.InnoDBVersion = innodbVersion
}

// SetServerID sets the server ID.
func (m *MySQLInstance) SetServerID(serverID string) {
	m.ServerID = serverID
}

// String returns a human-readable string representation of the instance.
func (m *MySQLInstance) String() string {
	if m == nil {
		return "<nil>"
	}
	return fmt.Sprintf("MySQL[%s] v%s (ServerID: %s, Mode: %s)",
		m.Address, m.Version, m.ServerID, m.ClusterMode)
}

// =============================================================================
// MySQL 告警结构体
// =============================================================================

// MySQLAlert represents a threshold violation alert for a MySQL instance.
type MySQLAlert struct {
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

// NewMySQLAlert creates a new MySQLAlert with the given parameters.
func NewMySQLAlert(address, metricName string, currentValue float64, level AlertLevel) *MySQLAlert {
	return &MySQLAlert{
		Address:      address,
		MetricName:   metricName,
		CurrentValue: currentValue,
		Level:        level,
	}
}

// IsWarning returns true if this alert is at warning level.
func (a *MySQLAlert) IsWarning() bool {
	return a.Level == AlertLevelWarning
}

// IsCritical returns true if this alert is at critical level.
func (a *MySQLAlert) IsCritical() bool {
	return a.Level == AlertLevelCritical
}

// =============================================================================
// MySQL 巡检结果结构体
// =============================================================================

// MySQLInspectionResult represents the inspection result for a single MySQL instance.
type MySQLInspectionResult struct {
	// 实例元信息
	Instance *MySQLInstance `json:"instance"`

	// 连接状态
	ConnectionStatus bool `json:"connection_status"` // mysql_up = 1

	// 复制相关 (MGR 模式 SlaveRunning 显示 N/A)
	SlaveRunning bool `json:"slave_running"` // Slave 线程状态
	SyncStatus   bool `json:"sync_status"`   // 同步是否正常

	// 慢查询日志
	SlowQueryLogEnabled bool   `json:"slow_query_log_enabled"`
	SlowQueryLogPath    string `json:"slow_query_log_path"`

	// 连接数
	MaxConnections     int `json:"max_connections"`
	CurrentConnections int `json:"current_connections"`

	// Binlog 配置
	BinlogEnabled       bool `json:"binlog_enabled"`
	BinlogExpireSeconds int  `json:"binlog_expire_seconds"`

	// MGR 专属字段 (仅 MGR 模式有效)
	MGRMemberCount int          `json:"mgr_member_count"`
	MGRRole        MySQLMGRRole `json:"mgr_role"`
	MGRStateOnline bool         `json:"mgr_state_online"`

	// 待实现项 (MVP 阶段显示 N/A)
	NonRootUser string `json:"non_root_user"`

	// 运行时间
	Uptime int64 `json:"uptime"` // 秒

	// 整体状态和告警
	Status MySQLInstanceStatus `json:"status"`
	Alerts []*MySQLAlert       `json:"alerts,omitempty"`

	// 采集时间
	CollectedAt time.Time `json:"collected_at"`

	// 错误信息
	Error string `json:"error,omitempty"`
}

// NewMySQLInspectionResult creates a new MySQLInspectionResult from a MySQLInstance.
func NewMySQLInspectionResult(instance *MySQLInstance) *MySQLInspectionResult {
	if instance == nil {
		return &MySQLInspectionResult{
			Status:      MySQLStatusFailed,
			NonRootUser: "N/A",
			Alerts:      make([]*MySQLAlert, 0),
		}
	}
	return &MySQLInspectionResult{
		Instance:    instance,
		Status:      MySQLStatusNormal,
		NonRootUser: "N/A", // MVP 阶段固定为 N/A
		Alerts:      make([]*MySQLAlert, 0),
	}
}

// AddAlert adds an alert to this instance and updates the status accordingly.
func (r *MySQLInspectionResult) AddAlert(alert *MySQLAlert) {
	if alert == nil {
		return
	}
	r.Alerts = append(r.Alerts, alert)
	// Update instance status to the most severe alert level
	if alert.Level == AlertLevelCritical {
		r.Status = MySQLStatusCritical
	} else if alert.Level == AlertLevelWarning && r.Status != MySQLStatusCritical {
		r.Status = MySQLStatusWarning
	}
}

// HasAlerts returns true if this instance has any alerts.
func (r *MySQLInspectionResult) HasAlerts() bool {
	return len(r.Alerts) > 0
}

// GetConnectionUsagePercent calculates the connection usage percentage.
// Returns 0 if MaxConnections is 0 to avoid division by zero.
func (r *MySQLInspectionResult) GetConnectionUsagePercent() float64 {
	if r.MaxConnections == 0 {
		return 0
	}
	return float64(r.CurrentConnections) / float64(r.MaxConnections) * 100
}

// GetAddress returns the instance address, or empty string if instance is nil.
func (r *MySQLInspectionResult) GetAddress() string {
	if r.Instance == nil {
		return ""
	}
	return r.Instance.Address
}

// =============================================================================
// MySQL 巡检摘要与结果集合
// =============================================================================

// MySQLInspectionSummary provides aggregated statistics about the MySQL inspection.
type MySQLInspectionSummary struct {
	TotalInstances    int `json:"total_instances"`    // 实例总数
	NormalInstances   int `json:"normal_instances"`   // 正常实例数
	WarningInstances  int `json:"warning_instances"`  // 警告实例数
	CriticalInstances int `json:"critical_instances"` // 严重实例数
	FailedInstances   int `json:"failed_instances"`   // 采集失败实例数
}

// NewMySQLInspectionSummary creates a new MySQLInspectionSummary from inspection results.
func NewMySQLInspectionSummary(results []*MySQLInspectionResult) *MySQLInspectionSummary {
	summary := &MySQLInspectionSummary{}
	for _, result := range results {
		if result == nil {
			continue
		}
		summary.TotalInstances++
		switch result.Status {
		case MySQLStatusNormal:
			summary.NormalInstances++
		case MySQLStatusWarning:
			summary.WarningInstances++
		case MySQLStatusCritical:
			summary.CriticalInstances++
		case MySQLStatusFailed:
			summary.FailedInstances++
		}
	}
	return summary
}

// MySQLAlertSummary provides aggregated alert statistics for MySQL inspection.
type MySQLAlertSummary struct {
	TotalAlerts   int `json:"total_alerts"`   // 告警总数
	WarningCount  int `json:"warning_count"`  // 警告级别数量
	CriticalCount int `json:"critical_count"` // 严重级别数量
}

// NewMySQLAlertSummary creates a new MySQLAlertSummary from a list of alerts.
func NewMySQLAlertSummary(alerts []*MySQLAlert) *MySQLAlertSummary {
	summary := &MySQLAlertSummary{}
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

// MySQLInspectionResults represents the complete result of MySQL inspection.
type MySQLInspectionResults struct {
	// 巡检时间信息
	InspectionTime time.Time     `json:"inspection_time"` // 巡检开始时间（Asia/Shanghai）
	Duration       time.Duration `json:"duration"`        // 巡检耗时

	// 巡检摘要
	Summary *MySQLInspectionSummary `json:"summary"` // 摘要统计

	// 实例结果
	Results []*MySQLInspectionResult `json:"results"` // 实例巡检结果列表

	// 告警汇总
	Alerts       []*MySQLAlert      `json:"alerts"`        // 所有告警列表
	AlertSummary *MySQLAlertSummary `json:"alert_summary"` // 告警摘要统计

	// 元数据
	Version string `json:"version,omitempty"` // 工具版本号
}

// NewMySQLInspectionResults creates a new MySQLInspectionResults with the given inspection time.
func NewMySQLInspectionResults(inspectionTime time.Time) *MySQLInspectionResults {
	return &MySQLInspectionResults{
		InspectionTime: inspectionTime,
		Results:        make([]*MySQLInspectionResult, 0),
		Alerts:         make([]*MySQLAlert, 0),
	}
}

// AddResult adds an instance result to the inspection.
func (r *MySQLInspectionResults) AddResult(result *MySQLInspectionResult) {
	if result == nil {
		return
	}
	r.Results = append(r.Results, result)
	// Collect all alerts from this instance
	r.Alerts = append(r.Alerts, result.Alerts...)
}

// Finalize calculates summaries after all instances have been added.
// This should be called after all instances are processed.
func (r *MySQLInspectionResults) Finalize(endTime time.Time) {
	r.Duration = endTime.Sub(r.InspectionTime)
	r.Summary = NewMySQLInspectionSummary(r.Results)
	r.AlertSummary = NewMySQLAlertSummary(r.Alerts)
}

// GetResultByAddress finds an instance result by address.
func (r *MySQLInspectionResults) GetResultByAddress(address string) *MySQLInspectionResult {
	for _, result := range r.Results {
		if result != nil && result.GetAddress() == address {
			return result
		}
	}
	return nil
}

// GetCriticalResults returns all instances with critical status.
func (r *MySQLInspectionResults) GetCriticalResults() []*MySQLInspectionResult {
	var critical []*MySQLInspectionResult
	for _, result := range r.Results {
		if result != nil && result.Status == MySQLStatusCritical {
			critical = append(critical, result)
		}
	}
	return critical
}

// GetWarningResults returns all instances with warning status.
func (r *MySQLInspectionResults) GetWarningResults() []*MySQLInspectionResult {
	var warning []*MySQLInspectionResult
	for _, result := range r.Results {
		if result != nil && result.Status == MySQLStatusWarning {
			warning = append(warning, result)
		}
	}
	return warning
}

// GetFailedResults returns all instances that failed collection.
func (r *MySQLInspectionResults) GetFailedResults() []*MySQLInspectionResult {
	var failed []*MySQLInspectionResult
	for _, result := range r.Results {
		if result != nil && result.Status == MySQLStatusFailed {
			failed = append(failed, result)
		}
	}
	return failed
}

// HasCritical returns true if any instance has critical status.
func (r *MySQLInspectionResults) HasCritical() bool {
	return r.Summary != nil && r.Summary.CriticalInstances > 0
}

// HasWarning returns true if any instance has warning status.
func (r *MySQLInspectionResults) HasWarning() bool {
	return r.Summary != nil && r.Summary.WarningInstances > 0
}

// HasAlerts returns true if there are any alerts.
func (r *MySQLInspectionResults) HasAlerts() bool {
	return len(r.Alerts) > 0
}

// =============================================================================
// MySQL 指标定义结构体
// =============================================================================

// MySQLMetricDefinition defines a MySQL metric to be collected.
// This struct maps to the YAML configuration in configs/mysql-metrics.yaml.
type MySQLMetricDefinition struct {
	Name         string `yaml:"name"`          // 指标唯一标识
	DisplayName  string `yaml:"display_name"`  // 中文显示名称
	Query        string `yaml:"query"`         // PromQL 查询表达式
	Category     string `yaml:"category"`      // 分类 (connection, info, mgr, binlog, log, status, replication, security)
	ClusterMode  string `yaml:"cluster_mode"`  // 适用的集群模式（可选：mgr, dual-master, master-slave）
	LabelExtract string `yaml:"label_extract"` // 从指标标签提取值（可选，如 version, member_id）
	Format       string `yaml:"format"`        // 格式化类型（可选：size, duration, percent）
	Status       string `yaml:"status"`        // 状态（pending=待实现）
	Note         string `yaml:"note"`          // 备注说明
}

// IsPending returns true if this metric is not yet implemented.
// A metric is considered pending if its status is "pending" or if it has no query.
func (m *MySQLMetricDefinition) IsPending() bool {
	return m.Status == "pending" || m.Query == ""
}

// HasLabelExtract returns true if this metric extracts value from a label.
func (m *MySQLMetricDefinition) HasLabelExtract() bool {
	return m.LabelExtract != ""
}

// IsForClusterMode checks if this metric is applicable for the given cluster mode.
// Returns true if the metric has no cluster mode restriction (applies to all modes),
// or if the metric's cluster mode matches the given mode.
func (m *MySQLMetricDefinition) IsForClusterMode(mode MySQLClusterMode) bool {
	if m.ClusterMode == "" {
		return true // Applies to all cluster modes
	}
	return m.ClusterMode == string(mode)
}

// GetDisplayName returns the display name, or the name if display name is empty.
func (m *MySQLMetricDefinition) GetDisplayName() string {
	if m.DisplayName != "" {
		return m.DisplayName
	}
	return m.Name
}
