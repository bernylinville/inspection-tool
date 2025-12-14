// Package config provides configuration management for the inspection tool.
package config

import (
	"strings"
	"testing"
	"time"
)

// newValidConfig creates a valid configuration for testing.
func newValidConfig() *Config {
	return &Config{
		Datasources: DatasourcesConfig{
			N9E: N9EConfig{
				Endpoint: "http://localhost:17000",
				Token:    "test-token",
				Timeout:  30 * time.Second,
			},
			VictoriaMetrics: VictoriaMetricsConfig{
				Endpoint: "http://localhost:8428",
				Timeout:  30 * time.Second,
			},
		},
		Inspection: InspectionConfig{
			Concurrency: 20,
			HostTimeout: 10 * time.Second,
		},
		Thresholds: ThresholdsConfig{
			CPUUsage:        ThresholdPair{Warning: 70, Critical: 90},
			MemoryUsage:     ThresholdPair{Warning: 70, Critical: 90},
			DiskUsage:       ThresholdPair{Warning: 70, Critical: 90},
			ZombieProcesses: ThresholdPair{Warning: 1, Critical: 10},
			LoadPerCore:     ThresholdPair{Warning: 0.7, Critical: 1.0},
		},
		Report: ReportConfig{
			OutputDir:        "./reports",
			Formats:          []string{"excel", "html"},
			FilenameTemplate: "inspection_report_{{.Date}}",
			Timezone:         "Asia/Shanghai",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		HTTP: HTTPConfig{
			Retry: RetryConfig{
				MaxRetries: 3,
				BaseDelay:  1 * time.Second,
			},
		},
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := newValidConfig()

	err := Validate(cfg)
	if err != nil {
		t.Errorf("Validate() error = %v, want nil for valid config", err)
	}
}

func TestValidate_MissingN9EEndpoint(t *testing.T) {
	cfg := newValidConfig()
	cfg.Datasources.N9E.Endpoint = ""

	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() should return error for missing N9E endpoint")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "datasources.n9e.endpoint") {
		t.Errorf("error should mention field 'datasources.n9e.endpoint', got: %s", errStr)
	}
	if !strings.Contains(errStr, "required") {
		t.Errorf("error should mention 'required', got: %s", errStr)
	}
}

func TestValidate_MissingN9EToken(t *testing.T) {
	cfg := newValidConfig()
	cfg.Datasources.N9E.Token = ""

	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() should return error for missing N9E token")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "datasources.n9e.token") {
		t.Errorf("error should mention field 'datasources.n9e.token', got: %s", errStr)
	}
}

func TestValidate_MissingVMEndpoint(t *testing.T) {
	cfg := newValidConfig()
	cfg.Datasources.VictoriaMetrics.Endpoint = ""

	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() should return error for missing VM endpoint")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "datasources.victoriametrics.endpoint") {
		t.Errorf("error should mention field 'datasources.victoriametrics.endpoint', got: %s", errStr)
	}
}

func TestValidate_InvalidURLFormat(t *testing.T) {
	cfg := newValidConfig()
	cfg.Datasources.N9E.Endpoint = "not-a-valid-url"

	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() should return error for invalid URL format")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "datasources.n9e.endpoint") {
		t.Errorf("error should mention field 'datasources.n9e.endpoint', got: %s", errStr)
	}
	if !strings.Contains(errStr, "URL") {
		t.Errorf("error should mention 'URL', got: %s", errStr)
	}
}

func TestValidate_ConcurrencyTooLow(t *testing.T) {
	cfg := newValidConfig()
	cfg.Inspection.Concurrency = 0

	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() should return error for concurrency = 0")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "inspection.concurrency") {
		t.Errorf("error should mention field 'inspection.concurrency', got: %s", errStr)
	}
}

func TestValidate_ConcurrencyTooHigh(t *testing.T) {
	cfg := newValidConfig()
	cfg.Inspection.Concurrency = 101

	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() should return error for concurrency = 101")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "inspection.concurrency") {
		t.Errorf("error should mention field 'inspection.concurrency', got: %s", errStr)
	}
}

func TestValidate_NegativeThreshold(t *testing.T) {
	cfg := newValidConfig()
	cfg.Thresholds.CPUUsage.Warning = -1

	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() should return error for negative threshold")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "thresholds.cpuusage.warning") {
		t.Errorf("error should mention threshold field, got: %s", errStr)
	}
}

func TestValidate_InvalidReportFormat(t *testing.T) {
	cfg := newValidConfig()
	cfg.Report.Formats = []string{"excel", "pdf"} // pdf is not valid

	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() should return error for invalid report format")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "report.formats") {
		t.Errorf("error should mention field 'report.formats', got: %s", errStr)
	}
}

func TestValidate_InvalidLogLevel(t *testing.T) {
	cfg := newValidConfig()
	cfg.Logging.Level = "verbose" // not valid

	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() should return error for invalid log level")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "logging.level") {
		t.Errorf("error should mention field 'logging.level', got: %s", errStr)
	}
}

func TestValidate_InvalidLogFormat(t *testing.T) {
	cfg := newValidConfig()
	cfg.Logging.Format = "text" // not valid, should be json or console

	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() should return error for invalid log format")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "logging.format") {
		t.Errorf("error should mention field 'logging.format', got: %s", errStr)
	}
}

func TestValidate_ThresholdWarningGreaterThanCritical(t *testing.T) {
	cfg := newValidConfig()
	cfg.Thresholds.CPUUsage.Warning = 90
	cfg.Thresholds.CPUUsage.Critical = 70 // warning > critical

	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() should return error when warning >= critical")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "thresholds.cpu_usage") {
		t.Errorf("error should mention field 'thresholds.cpu_usage', got: %s", errStr)
	}
	if !strings.Contains(errStr, "warning") && !strings.Contains(errStr, "critical") {
		t.Errorf("error should mention 'warning' and 'critical', got: %s", errStr)
	}
}

func TestValidate_ThresholdWarningEqualsCritical(t *testing.T) {
	cfg := newValidConfig()
	cfg.Thresholds.MemoryUsage.Warning = 80
	cfg.Thresholds.MemoryUsage.Critical = 80 // warning == critical

	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() should return error when warning == critical")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "thresholds.memory_usage") {
		t.Errorf("error should mention field 'thresholds.memory_usage', got: %s", errStr)
	}
}

func TestValidate_InvalidTimezone(t *testing.T) {
	cfg := newValidConfig()
	cfg.Report.Timezone = "Invalid/Timezone"

	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() should return error for invalid timezone")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "report.timezone") {
		t.Errorf("error should mention field 'report.timezone', got: %s", errStr)
	}
	if !strings.Contains(errStr, "timezone") {
		t.Errorf("error should mention 'timezone', got: %s", errStr)
	}
}

func TestValidate_EmptyTimezone(t *testing.T) {
	cfg := newValidConfig()
	cfg.Report.Timezone = "" // Empty is allowed (will use default)

	err := Validate(cfg)
	if err != nil {
		t.Errorf("Validate() should allow empty timezone, got error: %v", err)
	}
}

func TestValidate_ValidTimezones(t *testing.T) {
	validTimezones := []string{
		"Asia/Shanghai",
		"UTC",
		"America/New_York",
		"Europe/London",
	}

	for _, tz := range validTimezones {
		cfg := newValidConfig()
		cfg.Report.Timezone = tz

		err := Validate(cfg)
		if err != nil {
			t.Errorf("Validate() should allow timezone '%s', got error: %v", tz, err)
		}
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	cfg := newValidConfig()
	cfg.Datasources.N9E.Endpoint = ""     // Error 1
	cfg.Datasources.N9E.Token = ""        // Error 2
	cfg.Thresholds.CPUUsage.Warning = 90  // Error 3 (will be)
	cfg.Thresholds.CPUUsage.Critical = 70 // Error 3

	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate() should return error for multiple validation failures")
	}

	errStr := err.Error()
	// Should contain all three errors
	if !strings.Contains(errStr, "datasources.n9e.endpoint") {
		t.Errorf("error should mention 'datasources.n9e.endpoint', got: %s", errStr)
	}
	if !strings.Contains(errStr, "datasources.n9e.token") {
		t.Errorf("error should mention 'datasources.n9e.token', got: %s", errStr)
	}
	if !strings.Contains(errStr, "thresholds.cpu_usage") {
		t.Errorf("error should mention 'thresholds.cpu_usage', got: %s", errStr)
	}
}

func TestValidate_RetryMaxRetriesRange(t *testing.T) {
	tests := []struct {
		name       string
		maxRetries int
		wantErr    bool
	}{
		{"zero retries", 0, false},
		{"valid retries", 5, false},
		{"max retries", 10, false},
		{"too many retries", 11, true},
		{"negative retries", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newValidConfig()
			cfg.HTTP.Retry.MaxRetries = tt.maxRetries

			err := Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{
		Field:   "test.field",
		Tag:     "required",
		Value:   "",
		Message: "this field is required",
	}

	expected := "this field is required"
	if err.Error() != expected {
		t.Errorf("ValidationError.Error() = %v, want %v", err.Error(), expected)
	}
}

func TestValidationErrors_Error(t *testing.T) {
	errors := ValidationErrors{
		{Field: "field1", Message: "error1"},
		{Field: "field2", Message: "error2"},
	}

	errStr := errors.Error()
	if !strings.Contains(errStr, "config validation failed") {
		t.Errorf("ValidationErrors.Error() should contain header, got: %s", errStr)
	}
	if !strings.Contains(errStr, "field1") || !strings.Contains(errStr, "error1") {
		t.Errorf("ValidationErrors.Error() should contain first error, got: %s", errStr)
	}
	if !strings.Contains(errStr, "field2") || !strings.Contains(errStr, "error2") {
		t.Errorf("ValidationErrors.Error() should contain second error, got: %s", errStr)
	}
}

func TestValidationErrors_Empty(t *testing.T) {
	errors := ValidationErrors{}
	if errors.Error() != "" {
		t.Errorf("Empty ValidationErrors.Error() should return empty string, got: %s", errors.Error())
	}
}
