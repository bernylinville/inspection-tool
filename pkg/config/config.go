// Package config provides shared configuration types for the inspection tool.
package config

import "time"

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

// RetryConfig defines retry behavior for HTTP requests.
type RetryConfig struct {
	MaxRetries int           `mapstructure:"max_retries" validate:"gte=0,lte=10"`
	BaseDelay  time.Duration `mapstructure:"base_delay"`
}
