// Package model provides data models for the inspection tool.
package model

import "strings"

// HostStatus represents the overall health status of a host.
type HostStatus string

const (
	HostStatusNormal   HostStatus = "normal"   // 正常
	HostStatusWarning  HostStatus = "warning"  // 警告
	HostStatusCritical HostStatus = "critical" // 严重
	HostStatusFailed   HostStatus = "failed"   // 采集失败
)

// DiskMountInfo represents disk usage information for a single mount point.
type DiskMountInfo struct {
	Path        string  `json:"path"`         // 挂载点路径
	Total       int64   `json:"total"`        // 总容量（bytes）
	Free        int64   `json:"free"`         // 可用空间（bytes）
	UsedPercent float64 `json:"used_percent"` // 使用率（%）
}

// HostMeta contains basic metadata about a host collected from N9E API.
type HostMeta struct {
	Ident         string          `json:"ident"`          // 原始标识符
	Hostname      string          `json:"hostname"`       // 主机名（从 ident 清理得到）
	IP            string          `json:"ip"`             // IP 地址
	OS            string          `json:"os"`             // 操作系统类型
	OSVersion     string          `json:"os_version"`     // 操作系统版本
	KernelVersion string          `json:"kernel_version"` // 内核版本
	CPUCores      int             `json:"cpu_cores"`      // CPU 核心数
	CPUModel      string          `json:"cpu_model"`      // CPU 型号
	MemoryTotal   int64           `json:"memory_total"`   // 内存总量（bytes）
	DiskMounts    []DiskMountInfo `json:"disk_mounts"`    // 磁盘挂载点列表
}

// CleanIdent extracts the hostname from an ident string.
// It handles the "hostname@IP" format by returning only the hostname part.
func CleanIdent(ident string) string {
	if idx := strings.Index(ident, "@"); idx > 0 {
		return ident[:idx]
	}
	return ident
}
