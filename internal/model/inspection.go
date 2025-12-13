// Package model provides data models for the inspection tool.
package model

import "time"

// InspectionSummary provides aggregated statistics about the inspection.
type InspectionSummary struct {
	TotalHosts    int `json:"total_hosts"`    // 主机总数
	NormalHosts   int `json:"normal_hosts"`   // 正常主机数
	WarningHosts  int `json:"warning_hosts"`  // 警告主机数
	CriticalHosts int `json:"critical_hosts"` // 严重主机数
	FailedHosts   int `json:"failed_hosts"`   // 采集失败主机数
}

// NewInspectionSummary creates a new InspectionSummary from host results.
func NewInspectionSummary(hosts []*HostResult) *InspectionSummary {
	summary := &InspectionSummary{}
	for _, host := range hosts {
		if host == nil {
			continue
		}
		summary.TotalHosts++
		switch host.Status {
		case HostStatusNormal:
			summary.NormalHosts++
		case HostStatusWarning:
			summary.WarningHosts++
		case HostStatusCritical:
			summary.CriticalHosts++
		case HostStatusFailed:
			summary.FailedHosts++
		}
	}
	return summary
}

// HostResult represents the inspection result for a single host.
type HostResult struct {
	// 基础信息
	Hostname      string     `json:"hostname"`       // 主机名
	IP            string     `json:"ip"`             // IP 地址
	OS            string     `json:"os"`             // 操作系统类型
	OSVersion     string     `json:"os_version"`     // 操作系统版本
	KernelVersion string     `json:"kernel_version"` // 内核版本
	CPUCores      int        `json:"cpu_cores"`      // CPU 核心数
	CPUModel      string     `json:"cpu_model"`      // CPU 型号
	MemoryTotal   int64      `json:"memory_total"`   // 内存总量（bytes）
	Status        HostStatus `json:"status"`         // 整体状态

	// 指标数据
	Metrics map[string]*MetricValue `json:"metrics"` // 指标集合，key = 指标名称

	// 告警信息
	Alerts []*Alert `json:"alerts,omitempty"` // 该主机的告警列表

	// 时间信息
	CollectedAt time.Time `json:"collected_at"` // 采集时间（Asia/Shanghai）

	// 错误信息
	Error string `json:"error,omitempty"` // 采集错误信息
}

// NewHostResult creates a new HostResult from HostMeta.
func NewHostResult(meta *HostMeta) *HostResult {
	if meta == nil {
		return &HostResult{
			Status:  HostStatusFailed,
			Metrics: make(map[string]*MetricValue),
		}
	}
	return &HostResult{
		Hostname:      meta.Hostname,
		IP:            meta.IP,
		OS:            meta.OS,
		OSVersion:     meta.OSVersion,
		KernelVersion: meta.KernelVersion,
		CPUCores:      meta.CPUCores,
		CPUModel:      meta.CPUModel,
		MemoryTotal:   meta.MemoryTotal,
		Status:        HostStatusNormal,
		Metrics:       make(map[string]*MetricValue),
		Alerts:        make([]*Alert, 0),
	}
}

// SetMetric adds or updates a metric value for this host.
func (r *HostResult) SetMetric(value *MetricValue) {
	if r.Metrics == nil {
		r.Metrics = make(map[string]*MetricValue)
	}
	if value != nil {
		r.Metrics[value.Name] = value
	}
}

// GetMetric retrieves a metric value by name, returns nil if not found.
func (r *HostResult) GetMetric(name string) *MetricValue {
	if r.Metrics == nil {
		return nil
	}
	return r.Metrics[name]
}

// AddAlert adds an alert to this host and updates the status accordingly.
func (r *HostResult) AddAlert(alert *Alert) {
	if alert == nil {
		return
	}
	r.Alerts = append(r.Alerts, alert)
	// Update host status to the most severe alert level
	if alert.Level == AlertLevelCritical {
		r.Status = HostStatusCritical
	} else if alert.Level == AlertLevelWarning && r.Status != HostStatusCritical {
		r.Status = HostStatusWarning
	}
}

// HasAlerts returns true if this host has any alerts.
func (r *HostResult) HasAlerts() bool {
	return len(r.Alerts) > 0
}

// InspectionResult represents the complete result of a system inspection.
type InspectionResult struct {
	// 巡检时间信息
	InspectionTime time.Time     `json:"inspection_time"` // 巡检开始时间（Asia/Shanghai）
	Duration       time.Duration `json:"duration"`        // 巡检耗时

	// 巡检摘要
	Summary *InspectionSummary `json:"summary"` // 摘要统计

	// 主机结果
	Hosts []*HostResult `json:"hosts"` // 主机巡检结果列表

	// 告警汇总
	Alerts       []*Alert      `json:"alerts"`        // 所有告警列表
	AlertSummary *AlertSummary `json:"alert_summary"` // 告警摘要统计

	// 元数据
	Version string `json:"version,omitempty"` // 工具版本号
}

// NewInspectionResult creates a new InspectionResult with the given inspection time.
func NewInspectionResult(inspectionTime time.Time) *InspectionResult {
	return &InspectionResult{
		InspectionTime: inspectionTime,
		Hosts:          make([]*HostResult, 0),
		Alerts:         make([]*Alert, 0),
	}
}

// AddHost adds a host result to the inspection.
func (r *InspectionResult) AddHost(host *HostResult) {
	if host == nil {
		return
	}
	r.Hosts = append(r.Hosts, host)
	// Collect all alerts from this host
	r.Alerts = append(r.Alerts, host.Alerts...)
}

// Finalize calculates summaries after all hosts have been added.
// This should be called after all hosts are processed.
func (r *InspectionResult) Finalize(endTime time.Time) {
	r.Duration = endTime.Sub(r.InspectionTime)
	r.Summary = NewInspectionSummary(r.Hosts)
	r.AlertSummary = NewAlertSummary(r.Alerts)
}

// GetHostByName finds a host result by hostname.
func (r *InspectionResult) GetHostByName(hostname string) *HostResult {
	for _, host := range r.Hosts {
		if host != nil && host.Hostname == hostname {
			return host
		}
	}
	return nil
}

// GetCriticalHosts returns all hosts with critical status.
func (r *InspectionResult) GetCriticalHosts() []*HostResult {
	var critical []*HostResult
	for _, host := range r.Hosts {
		if host != nil && host.Status == HostStatusCritical {
			critical = append(critical, host)
		}
	}
	return critical
}

// GetWarningHosts returns all hosts with warning status.
func (r *InspectionResult) GetWarningHosts() []*HostResult {
	var warning []*HostResult
	for _, host := range r.Hosts {
		if host != nil && host.Status == HostStatusWarning {
			warning = append(warning, host)
		}
	}
	return warning
}

// GetFailedHosts returns all hosts that failed collection.
func (r *InspectionResult) GetFailedHosts() []*HostResult {
	var failed []*HostResult
	for _, host := range r.Hosts {
		if host != nil && host.Status == HostStatusFailed {
			failed = append(failed, host)
		}
	}
	return failed
}

// HasCritical returns true if any host has critical status.
func (r *InspectionResult) HasCritical() bool {
	return r.Summary != nil && r.Summary.CriticalHosts > 0
}

// HasWarning returns true if any host has warning status.
func (r *InspectionResult) HasWarning() bool {
	return r.Summary != nil && r.Summary.WarningHosts > 0
}

// HasAlerts returns true if there are any alerts.
func (r *InspectionResult) HasAlerts() bool {
	return len(r.Alerts) > 0
}
