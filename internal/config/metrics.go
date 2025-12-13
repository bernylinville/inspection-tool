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
