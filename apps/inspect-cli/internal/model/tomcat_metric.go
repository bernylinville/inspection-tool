package model

// TomcatMetricDefinition defines a Tomcat metric to be collected.
// Maps to YAML in configs/tomcat-metrics.yaml.
type TomcatMetricDefinition struct {
	Name         string   `yaml:"name" json:"name"`
	DisplayName  string   `yaml:"display_name" json:"display_name"`
	Query        string   `yaml:"query" json:"query"`
	Category     string   `yaml:"category" json:"category"`
	LabelExtract []string `yaml:"label_extract" json:"label_extract"` // 从标签提取的字段
	Format       string   `yaml:"format" json:"format"`
	Status       string   `yaml:"status" json:"status"` // pending=待实现
	Note         string   `yaml:"note" json:"note"`
}

// IsPending 判断指标是否待实现
func (m *TomcatMetricDefinition) IsPending() bool {
	return m.Status == "pending" || m.Query == ""
}

// HasLabelExtract 判断是否需要从标签提取值
func (m *TomcatMetricDefinition) HasLabelExtract() bool {
	return len(m.LabelExtract) > 0
}

// GetDisplayName 获取指标显示名称
func (m *TomcatMetricDefinition) GetDisplayName() string {
	if m.DisplayName != "" {
		return m.DisplayName
	}
	return m.Name
}

// TomcatMetricsConfig represents the root structure of tomcat-metrics.yaml.
type TomcatMetricsConfig struct {
	Metrics []*TomcatMetricDefinition `yaml:"tomcat_metrics" json:"tomcat_metrics"`
}
