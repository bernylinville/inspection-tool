// Package config provides configuration management for the inspection tool.
package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_Success(t *testing.T) {
	// Create a temporary config file
	content := `
datasources:
  n9e:
    endpoint: "http://localhost:17000"
    token: "test-token"
  victoriametrics:
    endpoint: "http://localhost:8428"
`
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	// Load config
	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify required values
	if cfg.Datasources.N9E.Endpoint != "http://localhost:17000" {
		t.Errorf("N9E endpoint = %v, want http://localhost:17000", cfg.Datasources.N9E.Endpoint)
	}
	if cfg.Datasources.N9E.Token != "test-token" {
		t.Errorf("N9E token = %v, want test-token", cfg.Datasources.N9E.Token)
	}

	// Verify defaults
	if cfg.Inspection.Concurrency != 20 {
		t.Errorf("Concurrency = %v, want 20", cfg.Inspection.Concurrency)
	}
	if cfg.HTTP.Retry.MaxRetries != 3 {
		t.Errorf("MaxRetries = %v, want 3", cfg.HTTP.Retry.MaxRetries)
	}
	if cfg.Report.Timezone != "Asia/Shanghai" {
		t.Errorf("Timezone = %v, want Asia/Shanghai", cfg.Report.Timezone)
	}
	if cfg.Inspection.HostTimeout != 10*time.Second {
		t.Errorf("HostTimeout = %v, want 10s", cfg.Inspection.HostTimeout)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Load() should return error for nonexistent file")
	}
}

func TestLoad_EmptyPath(t *testing.T) {
	_, err := Load("")
	if err == nil {
		t.Error("Load() should return error for empty path")
	}
}

func TestLoad_EnvironmentOverride(t *testing.T) {
	// Create a temporary config file
	content := `
datasources:
  n9e:
    endpoint: "http://localhost:17000"
    token: "file-token"
  victoriametrics:
    endpoint: "http://localhost:8428"
`
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	// Set environment variable
	os.Setenv("INSPECT_DATASOURCES_N9E_TOKEN", "env-token")
	defer os.Unsetenv("INSPECT_DATASOURCES_N9E_TOKEN")

	// Load config
	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Environment variable should override file value
	if cfg.Datasources.N9E.Token != "env-token" {
		t.Errorf("N9E token = %v, want env-token (env override)", cfg.Datasources.N9E.Token)
	}
}
