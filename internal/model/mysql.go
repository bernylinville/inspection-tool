// Package model provides data models for the inspection tool.
package model

import (
	"fmt"
	"strconv"
	"strings"
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
