// Package config provides configuration management for the inspection tool.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"inspection-tool/internal/model"
)

// LoadMetrics reads metric definitions from the specified YAML file.
// It returns a slice of MetricDefinition pointers for use with Collector and Evaluator.
func LoadMetrics(metricsPath string) ([]*model.MetricDefinition, error) {
	if metricsPath == "" {
		return nil, fmt.Errorf("metrics file path is required")
	}

	// Check if file exists
	if _, err := os.Stat(metricsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("metrics file not found: %s", metricsPath)
	}

	// Read file content
	data, err := os.ReadFile(metricsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metrics file: %w", err)
	}

	// Parse YAML
	var cfg model.MetricsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse metrics file: %w", err)
	}

	// Validate metrics
	if len(cfg.Metrics) == 0 {
		return nil, fmt.Errorf("no metrics defined in file: %s", metricsPath)
	}

	// Validate each metric definition
	for i, m := range cfg.Metrics {
		if m.Name == "" {
			return nil, fmt.Errorf("metric at index %d has no name", i)
		}
		if m.DisplayName == "" {
			return nil, fmt.Errorf("metric %q has no display_name", m.Name)
		}
	}

	return cfg.Metrics, nil
}

// CountActiveMetrics returns the count of active (non-pending) metrics.
func CountActiveMetrics(metrics []*model.MetricDefinition) int {
	count := 0
	for _, m := range metrics {
		if !m.IsPending() {
			count++
		}
	}
	return count
}

// LoadMySQLMetrics reads MySQL metric definitions from the specified YAML file.
// It returns a slice of MySQLMetricDefinition pointers for use with MySQLCollector and MySQLEvaluator.
func LoadMySQLMetrics(metricsPath string) ([]*model.MySQLMetricDefinition, error) {
	if metricsPath == "" {
		return nil, fmt.Errorf("MySQL metrics file path is required")
	}

	// Check if file exists
	if _, err := os.Stat(metricsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("MySQL metrics file not found: %s", metricsPath)
	}

	// Read file content
	data, err := os.ReadFile(metricsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read MySQL metrics file: %w", err)
	}

	// Parse YAML
	var cfg model.MySQLMetricsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse MySQL metrics file: %w", err)
	}

	// Validate metrics
	if len(cfg.Metrics) == 0 {
		return nil, fmt.Errorf("no MySQL metrics defined in file: %s", metricsPath)
	}

	// Validate each metric definition
	for i, m := range cfg.Metrics {
		if m.Name == "" {
			return nil, fmt.Errorf("MySQL metric at index %d has no name", i)
		}
		if m.DisplayName == "" {
			return nil, fmt.Errorf("MySQL metric %q has no display_name", m.Name)
		}
	}

	return cfg.Metrics, nil
}

// CountActiveMySQLMetrics returns the count of active (non-pending) MySQL metrics.
func CountActiveMySQLMetrics(metrics []*model.MySQLMetricDefinition) int {
	count := 0
	for _, m := range metrics {
		if !m.IsPending() {
			count++
		}
	}
	return count
}

// LoadRedisMetrics reads Redis metric definitions from the specified YAML file.
// It returns a slice of RedisMetricDefinition pointers for use with RedisCollector and RedisEvaluator.
func LoadRedisMetrics(metricsPath string) ([]*model.RedisMetricDefinition, error) {
	if metricsPath == "" {
		return nil, fmt.Errorf("Redis metrics file path is required")
	}

	// Check if file exists
	if _, err := os.Stat(metricsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("Redis metrics file not found: %s", metricsPath)
	}

	// Read file content
	data, err := os.ReadFile(metricsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Redis metrics file: %w", err)
	}

	// Parse YAML
	var cfg model.RedisMetricsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse Redis metrics file: %w", err)
	}

	// Validate metrics
	if len(cfg.Metrics) == 0 {
		return nil, fmt.Errorf("no Redis metrics defined in file: %s", metricsPath)
	}

	// Validate each metric definition
	for i, m := range cfg.Metrics {
		if m.Name == "" {
			return nil, fmt.Errorf("Redis metric at index %d has no name", i)
		}
		if m.DisplayName == "" {
			return nil, fmt.Errorf("Redis metric %q has no display_name", m.Name)
		}
	}

	return cfg.Metrics, nil
}

// CountActiveRedisMetrics returns the count of active (non-pending) Redis metrics.
func CountActiveRedisMetrics(metrics []*model.RedisMetricDefinition) int {
	count := 0
	for _, m := range metrics {
		if !m.IsPending() {
			count++
		}
	}
	return count
}

// LoadNginxMetrics reads Nginx metric definitions from the specified YAML file.
// It returns a slice of NginxMetricDefinition pointers for use with NginxCollector and NginxEvaluator.
func LoadNginxMetrics(metricsPath string) ([]*model.NginxMetricDefinition, error) {
	if metricsPath == "" {
		return nil, fmt.Errorf("Nginx metrics file path is required")
	}

	// Check if file exists
	if _, err := os.Stat(metricsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("Nginx metrics file not found: %s", metricsPath)
	}

	// Read file content
	data, err := os.ReadFile(metricsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Nginx metrics file: %w", err)
	}

	// Parse YAML
	var cfg model.NginxMetricsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse Nginx metrics file: %w", err)
	}

	// Validate metrics
	if len(cfg.Metrics) == 0 {
		return nil, fmt.Errorf("no Nginx metrics defined in file: %s", metricsPath)
	}

	// Validate each metric definition
	for i, m := range cfg.Metrics {
		if m.Name == "" {
			return nil, fmt.Errorf("Nginx metric at index %d has no name", i)
		}
		if m.DisplayName == "" {
			return nil, fmt.Errorf("Nginx metric %q has no display_name", m.Name)
		}
	}

	return cfg.Metrics, nil
}

// CountActiveNginxMetrics returns the count of active (non-pending) Nginx metrics.
func CountActiveNginxMetrics(metrics []*model.NginxMetricDefinition) int {
	count := 0
	for _, m := range metrics {
		if !m.IsPending() {
			count++
		}
	}
	return count
}

// LoadTomcatMetrics reads Tomcat metric definitions from the specified YAML file.
// It returns a slice of TomcatMetricDefinition pointers for use with TomcatCollector and TomcatEvaluator.
func LoadTomcatMetrics(metricsPath string) ([]*model.TomcatMetricDefinition, error) {
	if metricsPath == "" {
		return nil, fmt.Errorf("Tomcat metrics file path is required")
	}

	// Check if file exists
	if _, err := os.Stat(metricsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("Tomcat metrics file not found: %s", metricsPath)
	}

	// Read file content
	data, err := os.ReadFile(metricsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Tomcat metrics file: %w", err)
	}

	// Parse YAML
	var cfg model.TomcatMetricsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse Tomcat metrics file: %w", err)
	}

	// Validate metrics
	if len(cfg.Metrics) == 0 {
		return nil, fmt.Errorf("no Tomcat metrics defined in file: %s", metricsPath)
	}

	// Validate each metric definition
	for i, m := range cfg.Metrics {
		if m.Name == "" {
			return nil, fmt.Errorf("Tomcat metric at index %d has no name", i)
		}
		if m.DisplayName == "" {
			return nil, fmt.Errorf("Tomcat metric %q has no display_name", m.Name)
		}
	}

	return cfg.Metrics, nil
}

// CountActiveTomcatMetrics returns the count of active (non-pending) Tomcat metrics.
func CountActiveTomcatMetrics(metrics []*model.TomcatMetricDefinition) int {
	count := 0
	for _, m := range metrics {
		if !m.IsPending() {
			count++
		}
	}
	return count
}
