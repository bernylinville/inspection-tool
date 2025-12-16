// Package model provides data models for the inspection tool.
package model

import (
	"fmt"
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
