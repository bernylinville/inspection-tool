// Package config provides configuration management for the inspection tool.
package config

import "time"

// Config is the root configuration structure for the inspection tool.
type Config struct {
	Datasources DatasourcesConfig     `mapstructure:"datasources" validate:"required"`
	Inspection  InspectionConfig      `mapstructure:"inspection"`
	Thresholds  ThresholdsConfig      `mapstructure:"thresholds"`
	Report      ReportConfig          `mapstructure:"report"`
	Logging     LoggingConfig         `mapstructure:"logging"`
	HTTP        HTTPConfig            `mapstructure:"http"`
	MySQL       MySQLInspectionConfig  `mapstructure:"mysql"`
	Redis       RedisInspectionConfig  `mapstructure:"redis"`
	Nginx       NginxInspectionConfig  `mapstructure:"nginx"`
	Tomcat      TomcatInspectionConfig `mapstructure:"tomcat"`
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

// =============================================================================
// MySQL Inspection Configuration
// =============================================================================

// MySQLInspectionConfig contains configurations for MySQL inspection.
type MySQLInspectionConfig struct {
	Enabled        bool            `mapstructure:"enabled"`
	ClusterMode    string          `mapstructure:"cluster_mode" validate:"omitempty,oneof=mgr dual-master master-slave"`
	InstanceFilter MySQLFilter     `mapstructure:"instance_filter"`
	Thresholds     MySQLThresholds `mapstructure:"thresholds"`
}

// MySQLFilter defines MySQL instance filtering criteria.
type MySQLFilter struct {
	AddressPatterns []string          `mapstructure:"address_patterns"` // Address matching patterns (e.g., "172.18.182.*")
	BusinessGroups  []string          `mapstructure:"business_groups"`  // Business groups (OR relation)
	Tags            map[string]string `mapstructure:"tags"`             // Tags (AND relation)
}

// MySQLThresholds contains threshold configurations for MySQL alerts.
type MySQLThresholds struct {
	ConnectionUsageWarning  float64 `mapstructure:"connection_usage_warning" validate:"gte=0,lte=100"`  // Default: 70
	ConnectionUsageCritical float64 `mapstructure:"connection_usage_critical" validate:"gte=0,lte=100"` // Default: 90
	MGRMemberCountExpected  int     `mapstructure:"mgr_member_count_expected" validate:"gte=1"`         // Default: 3
}

// =============================================================================
// Redis Inspection Configuration
// =============================================================================

// RedisInspectionConfig contains configurations for Redis inspection.
type RedisInspectionConfig struct {
	Enabled        bool            `mapstructure:"enabled"`
	ClusterMode    string          `mapstructure:"cluster_mode" validate:"omitempty,oneof=3m3s 3m6s"` // "3m3s" or "3m6s"
	InstanceFilter RedisFilter     `mapstructure:"instance_filter"`
	Thresholds     RedisThresholds `mapstructure:"thresholds"`
}

// RedisFilter defines Redis instance filtering criteria.
type RedisFilter struct {
	AddressPatterns []string          `mapstructure:"address_patterns"` // Address matching patterns (glob)
	BusinessGroups  []string          `mapstructure:"business_groups"`  // Business groups (OR relation)
	Tags            map[string]string `mapstructure:"tags"`             // Tags (AND relation)
}

// RedisThresholds contains threshold configurations for Redis alerts.
type RedisThresholds struct {
	ConnectionUsageWarning  float64 `mapstructure:"connection_usage_warning" validate:"gte=0,lte=100"`  // Default: 70
	ConnectionUsageCritical float64 `mapstructure:"connection_usage_critical" validate:"gte=0,lte=100"` // Default: 90
	ReplicationLagWarning   int64   `mapstructure:"replication_lag_warning" validate:"gte=0"`           // Default: 1MB (1048576)
	ReplicationLagCritical  int64   `mapstructure:"replication_lag_critical" validate:"gte=0"`          // Default: 10MB (10485760)
}

// =============================================================================
// Nginx Inspection Configuration
// =============================================================================

// NginxInspectionConfig contains configurations for Nginx inspection.
type NginxInspectionConfig struct {
	Enabled        bool            `mapstructure:"enabled"`
	InstanceFilter NginxFilter     `mapstructure:"instance_filter"`
	Thresholds     NginxThresholds `mapstructure:"thresholds"`
}

// NginxFilter defines Nginx instance filtering criteria.
type NginxFilter struct {
	HostnamePatterns []string          `mapstructure:"hostname_patterns"` // Hostname patterns (glob, e.g., "GX-NM-*")
	BusinessGroups   []string          `mapstructure:"business_groups"`   // Business groups (OR relation)
	Tags             map[string]string `mapstructure:"tags"`              // Tags (AND relation)
}

// NginxThresholds contains threshold configurations for Nginx alerts.
type NginxThresholds struct {
	ConnectionUsageWarning   float64 `mapstructure:"connection_usage_warning" validate:"gte=0,lte=100"`  // Default: 70
	ConnectionUsageCritical  float64 `mapstructure:"connection_usage_critical" validate:"gte=0,lte=100"` // Default: 90
	LastErrorWarningMinutes  int     `mapstructure:"last_error_warning_minutes" validate:"gte=0"`        // Default: 60
	LastErrorCriticalMinutes int     `mapstructure:"last_error_critical_minutes" validate:"gte=0"`       // Default: 10
}

// =============================================================================
// Tomcat Inspection Configuration
// =============================================================================

// TomcatInspectionConfig contains configurations for Tomcat inspection.
type TomcatInspectionConfig struct {
	Enabled        bool             `mapstructure:"enabled"`
	InstanceFilter TomcatFilter     `mapstructure:"instance_filter"`
	Thresholds     TomcatThresholds `mapstructure:"thresholds"`
}

// TomcatFilter defines Tomcat instance filtering criteria.
// UNIQUE: Supports both HostnamePatterns AND ContainerPatterns (dual deployment).
type TomcatFilter struct {
	HostnamePatterns  []string          `mapstructure:"hostname_patterns"`  // Hostname patterns (glob, e.g., "GX-MFUI-*")
	ContainerPatterns []string          `mapstructure:"container_patterns"` // Container name patterns (glob, e.g., "tomcat-18001")
	BusinessGroups    []string          `mapstructure:"business_groups"`    // Business groups (OR relation)
	Tags              map[string]string `mapstructure:"tags"`               // Tags (AND relation)
}

// TomcatThresholds contains threshold configurations for Tomcat alerts.
type TomcatThresholds struct {
	// LastErrorWarningMinutes defines the warning threshold for recent error logs.
	// Time since last error in error.log (in minutes).
	// Note: Shorter time is MORE severe, so warning > critical.
	// Default: 60 minutes.
	LastErrorWarningMinutes int `mapstructure:"last_error_warning_minutes" validate:"gte=0"`
	// LastErrorCriticalMinutes defines the critical threshold for recent error logs.
	// Time since last error in error.log (in minutes).
	// Default: 10 minutes.
	LastErrorCriticalMinutes int `mapstructure:"last_error_critical_minutes" validate:"gte=0"`
}
