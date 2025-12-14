// Package config provides configuration management for the inspection tool.
package config

import "time"

// Config is the root configuration structure for the inspection tool.
type Config struct {
	Datasources DatasourcesConfig `mapstructure:"datasources" validate:"required"`
	Inspection  InspectionConfig  `mapstructure:"inspection"`
	Thresholds  ThresholdsConfig  `mapstructure:"thresholds"`
	Report      ReportConfig      `mapstructure:"report"`
	Logging     LoggingConfig     `mapstructure:"logging"`
	HTTP        HTTPConfig        `mapstructure:"http"`
}

// DatasourcesConfig contains configurations for data sources.
type DatasourcesConfig struct {
	N9E             N9EConfig             `mapstructure:"n9e" validate:"required"`
	VictoriaMetrics VictoriaMetricsConfig `mapstructure:"victoriametrics" validate:"required"`
}

// N9EConfig contains configuration for N9E (Nightingale) API.
type N9EConfig struct {
	Endpoint string        `mapstructure:"endpoint" validate:"required,url"`
	Token    string        `mapstructure:"token" validate:"required"`
	Timeout  time.Duration `mapstructure:"timeout"`
	Query    string        `mapstructure:"query"` // Host filter query (e.g., "items=短剧项目")
}

// VictoriaMetricsConfig contains configuration for VictoriaMetrics API.
type VictoriaMetricsConfig struct {
	Endpoint string        `mapstructure:"endpoint" validate:"required,url"`
	Timeout  time.Duration `mapstructure:"timeout"`
}

// InspectionConfig contains configurations for inspection behavior.
type InspectionConfig struct {
	Concurrency int           `mapstructure:"concurrency" validate:"gte=1,lte=100"`
	HostTimeout time.Duration `mapstructure:"host_timeout"`
	HostFilter  HostFilter    `mapstructure:"host_filter"`
}

// HostFilter defines host filtering criteria.
// BusinessGroups uses OR logic; Tags uses AND logic with BusinessGroups.
type HostFilter struct {
	BusinessGroups []string          `mapstructure:"business_groups"` // OR relation
	Tags           map[string]string `mapstructure:"tags"`            // AND relation with business groups
}

// ThresholdsConfig contains threshold configurations for alerts.
type ThresholdsConfig struct {
	CPUUsage        ThresholdPair `mapstructure:"cpu_usage"`
	MemoryUsage     ThresholdPair `mapstructure:"memory_usage"`
	DiskUsage       ThresholdPair `mapstructure:"disk_usage"`
	ZombieProcesses ThresholdPair `mapstructure:"zombie_processes"`
	LoadPerCore     ThresholdPair `mapstructure:"load_per_core"`
}

// ThresholdPair defines warning and critical thresholds for a metric.
type ThresholdPair struct {
	Warning  float64 `mapstructure:"warning" validate:"gte=0"`
	Critical float64 `mapstructure:"critical" validate:"gte=0"`
}

// ReportConfig contains configurations for report generation.
type ReportConfig struct {
	OutputDir        string   `mapstructure:"output_dir"`
	Formats          []string `mapstructure:"formats" validate:"dive,oneof=excel html"`
	FilenameTemplate string   `mapstructure:"filename_template"`
	HTMLTemplate     string   `mapstructure:"html_template"`
	Timezone         string   `mapstructure:"timezone"`
}

// LoggingConfig contains configurations for logging.
type LoggingConfig struct {
	Level  string `mapstructure:"level" validate:"oneof=debug info warn error"`
	Format string `mapstructure:"format" validate:"oneof=json console"`
}

// HTTPConfig contains HTTP client configurations including retry settings.
type HTTPConfig struct {
	Retry RetryConfig `mapstructure:"retry"`
}

// RetryConfig defines retry behavior for HTTP requests.
type RetryConfig struct {
	MaxRetries int           `mapstructure:"max_retries" validate:"gte=0,lte=10"`
	BaseDelay  time.Duration `mapstructure:"base_delay"`
}
