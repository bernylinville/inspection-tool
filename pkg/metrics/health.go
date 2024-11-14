package metrics

import (
	"fmt"
	"strings"
)

// HealthStatusType 定义健康状态类型
type HealthStatusType string

// 定义健康状态常量
const (
	HealthStatusHealthy  HealthStatusType = "healthy"
	HealthStatusWarning  HealthStatusType = "warning"
	HealthStatusCritical HealthStatusType = "critical"
)

// HealthStatus 定义健康状态
type HealthStatus struct {
	Status    HealthStatusType // healthy, warning, critical
	Message   string
	Threshold struct {
		CPU    float64
		Memory float64
		Disk   float64
		Load   float64
	}
}

// String 实现 Stringer 接口
func (h HealthStatusType) String() string {
	return string(h)
}

// CheckHealth 检查主机健康状态
func CheckHealth(data MetricData) HealthStatus {
	status := HealthStatus{
		Status: HealthStatusHealthy,
		Threshold: struct {
			CPU    float64
			Memory float64
			Disk   float64
			Load   float64
		}{
			CPU:    80,
			Memory: 85,
			Disk:   90,
			Load:   5,
		},
	}

	var messages []string

	if data.CPU > status.Threshold.CPU {
		messages = append(messages, fmt.Sprintf("CPU使用率过高: %.2f%%", data.CPU))
		status.Status = HealthStatusWarning
	}

	if data.Memory > status.Threshold.Memory {
		messages = append(messages, fmt.Sprintf("内存使用率过高: %.2f%%", data.Memory))
		status.Status = HealthStatusWarning
	}

	if data.DiskUsage > status.Threshold.Disk {
		messages = append(messages, fmt.Sprintf("磁盘使用率过高: %.2f%%", data.DiskUsage))
		status.Status = HealthStatusCritical
	}

	if data.SystemLoad5 > status.Threshold.Load {
		messages = append(messages, fmt.Sprintf("系统负载过高: %.2f", data.SystemLoad5))
		status.Status = HealthStatusWarning
	}

	status.Message = strings.Join(messages, "; ")
	return status
}
