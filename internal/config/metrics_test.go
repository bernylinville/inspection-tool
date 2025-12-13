package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMetrics_Success(t *testing.T) {
	// Create a temporary metrics file
	content := `
metrics:
  - name: cpu_usage
    display_name: "CPU 利用率"
    query: 'cpu_usage_active{cpu="cpu-total"}'
    unit: "%"
    category: cpu
    format: percent
  - name: memory_usage
    display_name: "内存利用率"
    query: '100 - mem_available_percent'
    unit: "%"
    category: memory
`
	tmpDir := t.TempDir()
	metricsPath := filepath.Join(tmpDir, "metrics.yaml")
	if err := os.WriteFile(metricsPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	metrics, err := LoadMetrics(metricsPath)
	if err != nil {
		t.Fatalf("LoadMetrics() error = %v", err)
	}

	if len(metrics) != 2 {
		t.Errorf("expected 2 metrics, got %d", len(metrics))
	}

	// Verify first metric
	if metrics[0].Name != "cpu_usage" {
		t.Errorf("expected name 'cpu_usage', got %q", metrics[0].Name)
	}
	if metrics[0].DisplayName != "CPU 利用率" {
		t.Errorf("expected display_name 'CPU 利用率', got %q", metrics[0].DisplayName)
	}
}

func TestLoadMetrics_FileNotFound(t *testing.T) {
	_, err := LoadMetrics("/nonexistent/path/metrics.yaml")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestLoadMetrics_EmptyPath(t *testing.T) {
	_, err := LoadMetrics("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestLoadMetrics_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	metricsPath := filepath.Join(tmpDir, "invalid.yaml")
	content := `metrics: [invalid: yaml: content`
	if err := os.WriteFile(metricsPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := LoadMetrics(metricsPath)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadMetrics_EmptyMetrics(t *testing.T) {
	tmpDir := t.TempDir()
	metricsPath := filepath.Join(tmpDir, "empty.yaml")
	content := `metrics: []`
	if err := os.WriteFile(metricsPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := LoadMetrics(metricsPath)
	if err == nil {
		t.Fatal("expected error for empty metrics list")
	}
}

func TestLoadMetrics_MissingName(t *testing.T) {
	tmpDir := t.TempDir()
	metricsPath := filepath.Join(tmpDir, "no_name.yaml")
	content := `
metrics:
  - display_name: "CPU 利用率"
    query: 'cpu_usage'
`
	if err := os.WriteFile(metricsPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := LoadMetrics(metricsPath)
	if err == nil {
		t.Fatal("expected error for missing metric name")
	}
}

func TestLoadMetrics_MissingDisplayName(t *testing.T) {
	tmpDir := t.TempDir()
	metricsPath := filepath.Join(tmpDir, "no_display.yaml")
	content := `
metrics:
  - name: cpu_usage
    query: 'cpu_usage'
`
	if err := os.WriteFile(metricsPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := LoadMetrics(metricsPath)
	if err == nil {
		t.Fatal("expected error for missing display_name")
	}
}

func TestLoadMetrics_WithPendingMetrics(t *testing.T) {
	content := `
metrics:
  - name: cpu_usage
    display_name: "CPU 利用率"
    query: 'cpu_usage_active{cpu="cpu-total"}'
    category: cpu
  - name: ntp_check
    display_name: "NTP 检查"
    query: ""
    status: pending
`
	tmpDir := t.TempDir()
	metricsPath := filepath.Join(tmpDir, "with_pending.yaml")
	if err := os.WriteFile(metricsPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	metrics, err := LoadMetrics(metricsPath)
	if err != nil {
		t.Fatalf("LoadMetrics() error = %v", err)
	}

	if len(metrics) != 2 {
		t.Errorf("expected 2 metrics, got %d", len(metrics))
	}

	// Check pending status
	if !metrics[1].IsPending() {
		t.Error("expected ntp_check to be pending")
	}
}

func TestLoadMetrics_RealFile(t *testing.T) {
	// Test with the actual metrics.yaml file if it exists
	metricsPath := "../../configs/metrics.yaml"
	if _, err := os.Stat(metricsPath); os.IsNotExist(err) {
		t.Skip("configs/metrics.yaml not found, skipping real file test")
	}

	metrics, err := LoadMetrics(metricsPath)
	if err != nil {
		t.Fatalf("LoadMetrics() error = %v", err)
	}

	if len(metrics) == 0 {
		t.Error("expected at least one metric from real file")
	}
}

func TestCountActiveMetrics(t *testing.T) {
	content := `
metrics:
  - name: cpu_usage
    display_name: "CPU 利用率"
    query: 'cpu_usage_active{cpu="cpu-total"}'
    category: cpu
  - name: memory_usage
    display_name: "内存利用率"
    query: '100 - mem_available_percent'
    category: memory
  - name: ntp_check
    display_name: "NTP 检查"
    query: ""
    status: pending
  - name: public_network
    display_name: "公网访问检查"
    query: ""
    status: pending
`
	tmpDir := t.TempDir()
	metricsPath := filepath.Join(tmpDir, "mixed.yaml")
	if err := os.WriteFile(metricsPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	metrics, err := LoadMetrics(metricsPath)
	if err != nil {
		t.Fatalf("LoadMetrics() error = %v", err)
	}

	activeCount := CountActiveMetrics(metrics)
	if activeCount != 2 {
		t.Errorf("expected 2 active metrics, got %d", activeCount)
	}
}

func TestCountActiveMetrics_AllActive(t *testing.T) {
	content := `
metrics:
  - name: cpu_usage
    display_name: "CPU 利用率"
    query: 'cpu_usage_active'
  - name: memory_usage
    display_name: "内存利用率"
    query: 'memory_used'
`
	tmpDir := t.TempDir()
	metricsPath := filepath.Join(tmpDir, "all_active.yaml")
	if err := os.WriteFile(metricsPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	metrics, err := LoadMetrics(metricsPath)
	if err != nil {
		t.Fatalf("LoadMetrics() error = %v", err)
	}

	activeCount := CountActiveMetrics(metrics)
	if activeCount != 2 {
		t.Errorf("expected 2 active metrics, got %d", activeCount)
	}
}

func TestCountActiveMetrics_AllPending(t *testing.T) {
	content := `
metrics:
  - name: ntp_check
    display_name: "NTP 检查"
    query: ""
    status: pending
  - name: public_network
    display_name: "公网检查"
    status: pending
`
	tmpDir := t.TempDir()
	metricsPath := filepath.Join(tmpDir, "all_pending.yaml")
	if err := os.WriteFile(metricsPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	metrics, err := LoadMetrics(metricsPath)
	if err != nil {
		t.Fatalf("LoadMetrics() error = %v", err)
	}

	activeCount := CountActiveMetrics(metrics)
	if activeCount != 0 {
		t.Errorf("expected 0 active metrics, got %d", activeCount)
	}
}
