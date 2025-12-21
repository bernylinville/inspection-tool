// Package config provides configuration management for the inspection tool.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Load reads configuration from the specified YAML file and environment variables.
// Environment variables take precedence over file values.
// Environment variable format: INSPECT_<SECTION>_<KEY> (e.g., INSPECT_DATASOURCES_N9E_TOKEN)
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults first
	setDefaults(v)

	// Configure environment variable binding
	v.SetEnvPrefix("INSPECT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Check if config file exists
	if configPath == "" {
		return nil, fmt.Errorf("config file path is required")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	// Set config file
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal into Config struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := Validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// setDefaults sets default values for all configuration options.
func setDefaults(v *viper.Viper) {
	// Datasources defaults
	v.SetDefault("datasources.n9e.timeout", 30*time.Second)
	v.SetDefault("datasources.victoriametrics.timeout", 30*time.Second)

	// Inspection defaults
	v.SetDefault("inspection.concurrency", 20)
	v.SetDefault("inspection.host_timeout", 10*time.Second)

	// Thresholds defaults - based on PRD
	v.SetDefault("thresholds.cpu_usage.warning", 70.0)
	v.SetDefault("thresholds.cpu_usage.critical", 90.0)
	v.SetDefault("thresholds.memory_usage.warning", 70.0)
	v.SetDefault("thresholds.memory_usage.critical", 90.0)
	v.SetDefault("thresholds.disk_usage.warning", 70.0)
	v.SetDefault("thresholds.disk_usage.critical", 90.0)
	v.SetDefault("thresholds.zombie_processes.warning", 1.0)
	v.SetDefault("thresholds.zombie_processes.critical", 10.0)
	v.SetDefault("thresholds.load_per_core.warning", 0.7)
	v.SetDefault("thresholds.load_per_core.critical", 1.0)

	// Report defaults
	v.SetDefault("report.output_dir", "./reports")
	v.SetDefault("report.formats", []string{"excel", "html"})
	v.SetDefault("report.filename_template", "inspection_report_{{.Date}}")
	v.SetDefault("report.timezone", "Asia/Shanghai")

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")

	// HTTP retry defaults
	v.SetDefault("http.retry.max_retries", 3)
	v.SetDefault("http.retry.base_delay", 1*time.Second)

	// MySQL inspection defaults
	v.SetDefault("mysql.enabled", false)
	v.SetDefault("mysql.thresholds.connection_usage_warning", 70.0)
	v.SetDefault("mysql.thresholds.connection_usage_critical", 90.0)
	v.SetDefault("mysql.thresholds.mgr_member_count_expected", 3)

	// Redis inspection defaults
	v.SetDefault("redis.enabled", false)
	v.SetDefault("redis.cluster_mode", "3m3s")
	v.SetDefault("redis.thresholds.connection_usage_warning", 70.0)
	v.SetDefault("redis.thresholds.connection_usage_critical", 90.0)
	v.SetDefault("redis.thresholds.replication_lag_warning", 1048576)   // 1MB
	v.SetDefault("redis.thresholds.replication_lag_critical", 10485760) // 10MB

	// Nginx inspection defaults
	v.SetDefault("nginx.enabled", false)
	v.SetDefault("nginx.thresholds.connection_usage_warning", 70.0)
	v.SetDefault("nginx.thresholds.connection_usage_critical", 90.0)
	v.SetDefault("nginx.thresholds.last_error_warning_minutes", 60)
	v.SetDefault("nginx.thresholds.last_error_critical_minutes", 10)
}
