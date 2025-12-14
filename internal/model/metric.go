// Package model provides data models for the inspection tool.
package model

// MetricStatus represents the evaluation status of a metric value.
type MetricStatus string

const (
	MetricStatusNormal   MetricStatus = "normal"   // 正常
	MetricStatusWarning  MetricStatus = "warning"  // 警告
	MetricStatusCritical MetricStatus = "critical" // 严重
	MetricStatusPending  MetricStatus = "pending"  // 待定/N/A
)

// MetricCategory represents the category of a metric.
type MetricCategory string

const (
	MetricCategoryCPU     MetricCategory = "cpu"     // CPU 相关
	MetricCategoryMemory  MetricCategory = "memory"  // 内存相关
	MetricCategoryDisk    MetricCategory = "disk"    // 磁盘相关
	MetricCategorySystem  MetricCategory = "system"  // 系统相关
	MetricCategoryProcess MetricCategory = "process" // 进程相关
)

// MetricFormat represents how to format a metric value for display.
type MetricFormat string

const (
	MetricFormatPercent  MetricFormat = "percent"  // 百分比（如 75.5%）
	MetricFormatSize     MetricFormat = "size"     // 字节大小（如 16.0 GB）
	MetricFormatDuration MetricFormat = "duration" // 时间时长（如 3天2小时）
	MetricFormatNumber   MetricFormat = "number"   // 普通数值
)

// AggregateType represents how to aggregate multiple values (e.g., across disk mounts).
type AggregateType string

const (
	AggregateMax AggregateType = "max" // 取最大值
	AggregateMin AggregateType = "min" // 取最小值
	AggregateAvg AggregateType = "avg" // 取平均值
)

// MetricDefinition defines the metadata for a metric, loaded from metrics.yaml.
type MetricDefinition struct {
	Name          string         `yaml:"name" json:"name"`                                           // 指标唯一标识
	DisplayName   string         `yaml:"display_name" json:"display_name"`                           // 中文显示名称
	Query         string         `yaml:"query" json:"query"`                                         // PromQL 查询表达式
	Unit          string         `yaml:"unit" json:"unit"`                                           // 单位（%、bytes、seconds、个）
	Category      MetricCategory `yaml:"category" json:"category"`                                   // 分类
	Format        MetricFormat   `yaml:"format,omitempty" json:"format,omitempty"`                   // 格式化类型
	Aggregate     AggregateType  `yaml:"aggregate,omitempty" json:"aggregate,omitempty"`             // 聚合方式
	ExpandByLabel string         `yaml:"expand_by_label,omitempty" json:"expand_by_label,omitempty"` // 按标签展开
	Status        string         `yaml:"status,omitempty" json:"status,omitempty"`                   // pending=待实现
	Note          string         `yaml:"note,omitempty" json:"note,omitempty"`                       // 备注说明
}

// IsPending returns true if this metric is marked as pending (not yet implemented).
func (d *MetricDefinition) IsPending() bool {
	return d.Status == "pending" || d.Query == ""
}

// HasExpandLabel returns true if this metric should be expanded by a label (e.g., disk by path).
func (d *MetricDefinition) HasExpandLabel() bool {
	return d.ExpandByLabel != ""
}

// MetricValue represents a collected metric value for a host.
type MetricValue struct {
	Name           string            `json:"name"`                // 指标名称（关联 MetricDefinition）
	RawValue       float64           `json:"raw_value"`           // 原始数值
	FormattedValue string            `json:"formatted_value"`     // 格式化后的显示值
	Status         MetricStatus      `json:"status"`              // 评估状态
	Labels         map[string]string `json:"labels,omitempty"`    // 标签（如 path 用于磁盘挂载点）
	IsNA           bool              `json:"is_na"`               // 是否为 N/A（待定项或采集失败）
	Timestamp      int64             `json:"timestamp,omitempty"` // 采集时间戳（Unix 秒）
}

// NewNAMetricValue creates a MetricValue representing "N/A" for pending metrics.
func NewNAMetricValue(name string) *MetricValue {
	return &MetricValue{
		Name:           name,
		RawValue:       0,
		FormattedValue: "N/A",
		Status:         MetricStatusPending,
		IsNA:           true,
	}
}

// NewMetricValue creates a new MetricValue with the given raw value.
func NewMetricValue(name string, rawValue float64) *MetricValue {
	return &MetricValue{
		Name:     name,
		RawValue: rawValue,
		Status:   MetricStatusNormal,
		IsNA:     false,
	}
}

// HostMetrics contains all metric values collected for a single host.
type HostMetrics struct {
	Hostname string                  `json:"hostname"` // 主机名
	Metrics  map[string]*MetricValue `json:"metrics"`  // 指标集合，key = 指标名称
}

// NewHostMetrics creates a new HostMetrics instance for the given hostname.
func NewHostMetrics(hostname string) *HostMetrics {
	return &HostMetrics{
		Hostname: hostname,
		Metrics:  make(map[string]*MetricValue),
	}
}

// SetMetric adds or updates a metric value for this host.
func (h *HostMetrics) SetMetric(value *MetricValue) {
	if h.Metrics == nil {
		h.Metrics = make(map[string]*MetricValue)
	}
	h.Metrics[value.Name] = value
}

// GetMetric retrieves a metric value by name, returns nil if not found.
func (h *HostMetrics) GetMetric(name string) *MetricValue {
	if h.Metrics == nil {
		return nil
	}
	return h.Metrics[name]
}

// MetricsConfig represents the root structure of metrics.yaml file.
type MetricsConfig struct {
	Metrics []*MetricDefinition `yaml:"metrics" json:"metrics"` // 指标定义列表
}
