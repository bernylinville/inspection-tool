package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"inspection-tool/internal/config"
	"inspection-tool/internal/model"
)

// =============================================================================
// Inspector Tests
// =============================================================================

// TestNewInspector tests the Inspector constructor.
func TestNewInspector(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createTestMetrics()

	// Create mock servers
	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"dat": []interface{}{},
			"err": "",
		})
	})
	defer n9eServer.Close()

	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result":     []interface{}{},
			},
		})
	})
	defer vmServer.Close()

	cfg := createTestConfig()
	cfg.Datasources.N9E.Endpoint = n9eServer.URL
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL
	cfg.Report.Timezone = "Asia/Shanghai"
	cfg.Thresholds = config.ThresholdsConfig{
		CPUUsage:        config.ThresholdPair{Warning: 70, Critical: 90},
		MemoryUsage:     config.ThresholdPair{Warning: 70, Critical: 90},
		DiskUsage:       config.ThresholdPair{Warning: 70, Critical: 90},
		ZombieProcesses: config.ThresholdPair{Warning: 1, Critical: 10},
		LoadPerCore:     config.ThresholdPair{Warning: 0.7, Critical: 1.0},
	}

	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	collector := NewCollector(cfg, n9eClient, vmClient, metrics, logger)
	evaluator := NewEvaluator(&cfg.Thresholds, metrics, logger)

	t.Run("basic construction", func(t *testing.T) {
		inspector, err := NewInspector(cfg, collector, evaluator, logger)
		if err != nil {
			t.Fatalf("NewInspector failed: %v", err)
		}
		if inspector == nil {
			t.Fatal("expected non-nil inspector")
		}
		if inspector.version != "dev" {
			t.Errorf("expected default version 'dev', got %s", inspector.version)
		}
		if inspector.timezone.String() != "Asia/Shanghai" {
			t.Errorf("expected timezone Asia/Shanghai, got %s", inspector.timezone.String())
		}
	})

	t.Run("with version option", func(t *testing.T) {
		inspector, err := NewInspector(cfg, collector, evaluator, logger, WithVersion("v1.2.3"))
		if err != nil {
			t.Fatalf("NewInspector failed: %v", err)
		}
		if inspector.version != "v1.2.3" {
			t.Errorf("expected version v1.2.3, got %s", inspector.version)
		}
	})

	t.Run("invalid timezone", func(t *testing.T) {
		badCfg := createTestConfig()
		badCfg.Datasources.N9E.Endpoint = n9eServer.URL
		badCfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL
		badCfg.Report.Timezone = "Invalid/Timezone"

		_, err := NewInspector(badCfg, collector, evaluator, logger)
		if err == nil {
			t.Error("expected error for invalid timezone")
		}
	})
}

// TestInspector_Run_Success tests a successful inspection run.
func TestInspector_Run_Success(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createTestMetrics()

	// Create N9E mock server
	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"dat": [
				{"ident": "test-host-1", "extend_info": "{\"cpu\":{\"cpu_cores\":\"4\",\"model_name\":\"Intel Xeon\"},\"memory\":{\"total\":\"16496934912\"},\"network\":{\"ipaddress\":\"192.168.1.100\"},\"platform\":{\"hostname\":\"test-host-1\",\"os\":\"GNU/Linux\",\"kernel_release\":\"5.14.0\"},\"filesystem\":[]}"}
			],
			"err": ""
		}`))
	})
	defer n9eServer.Close()

	// Create VM mock server
	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		var value string

		switch query {
		case `cpu_usage_active{cpu="cpu-total"}`:
			value = "45.5"
		case "100 - mem_available_percent":
			value = "60.0"
		default:
			value = "0"
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result": []map[string]interface{}{
					{
						"metric": map[string]string{"ident": "test-host-1"},
						"value":  []interface{}{float64(time.Now().Unix()), value},
					},
				},
			},
		})
	})
	defer vmServer.Close()

	cfg := createTestConfig()
	cfg.Datasources.N9E.Endpoint = n9eServer.URL
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL
	cfg.Report.Timezone = "Asia/Shanghai"
	cfg.Thresholds = config.ThresholdsConfig{
		CPUUsage:        config.ThresholdPair{Warning: 70, Critical: 90},
		MemoryUsage:     config.ThresholdPair{Warning: 70, Critical: 90},
		DiskUsage:       config.ThresholdPair{Warning: 70, Critical: 90},
		ZombieProcesses: config.ThresholdPair{Warning: 1, Critical: 10},
		LoadPerCore:     config.ThresholdPair{Warning: 0.7, Critical: 1.0},
	}

	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	collector := NewCollector(cfg, n9eClient, vmClient, metrics, logger)
	evaluator := NewEvaluator(&cfg.Thresholds, metrics, logger)

	inspector, err := NewInspector(cfg, collector, evaluator, logger, WithVersion("v1.0.0"))
	if err != nil {
		t.Fatalf("NewInspector failed: %v", err)
	}

	result, err := inspector.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify result
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Version != "v1.0.0" {
		t.Errorf("expected version v1.0.0, got %s", result.Version)
	}
	if result.Summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if result.Summary.TotalHosts != 1 {
		t.Errorf("expected 1 total host, got %d", result.Summary.TotalHosts)
	}
	if result.Summary.NormalHosts != 1 {
		t.Errorf("expected 1 normal host, got %d", result.Summary.NormalHosts)
	}
	if len(result.Hosts) != 1 {
		t.Fatalf("expected 1 host result, got %d", len(result.Hosts))
	}

	// Verify host result
	host := result.Hosts[0]
	if host.Hostname != "test-host-1" {
		t.Errorf("expected hostname test-host-1, got %s", host.Hostname)
	}
	if host.Status != model.HostStatusNormal {
		t.Errorf("expected status normal, got %s", host.Status)
	}

	// Verify timezone
	if result.InspectionTime.Location().String() != "Asia/Shanghai" {
		t.Errorf("expected timezone Asia/Shanghai, got %s", result.InspectionTime.Location().String())
	}

	// Verify duration is positive
	if result.Duration <= 0 {
		t.Errorf("expected positive duration, got %v", result.Duration)
	}
}

// TestInspector_Run_NoHosts tests inspection with no hosts.
func TestInspector_Run_NoHosts(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createTestMetrics()

	// Create N9E mock server returning empty list
	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"dat": [], "err": ""}`))
	})
	defer n9eServer.Close()

	// Create VM mock server
	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data":   map[string]interface{}{"resultType": "vector", "result": []interface{}{}},
		})
	})
	defer vmServer.Close()

	cfg := createTestConfig()
	cfg.Datasources.N9E.Endpoint = n9eServer.URL
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL
	cfg.Report.Timezone = "Asia/Shanghai"

	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	collector := NewCollector(cfg, n9eClient, vmClient, metrics, logger)
	evaluator := NewEvaluator(&cfg.Thresholds, metrics, logger)

	inspector, err := NewInspector(cfg, collector, evaluator, logger)
	if err != nil {
		t.Fatalf("NewInspector failed: %v", err)
	}

	result, err := inspector.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if result.Summary.TotalHosts != 0 {
		t.Errorf("expected 0 total hosts, got %d", result.Summary.TotalHosts)
	}
	if len(result.Hosts) != 0 {
		t.Errorf("expected 0 hosts, got %d", len(result.Hosts))
	}
}

// TestInspector_Run_WithWarning tests inspection with warning alerts.
func TestInspector_Run_WithWarning(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createTestMetrics()

	// Create N9E mock server
	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"dat": [
				{"ident": "warning-host", "extend_info": "{\"cpu\":{\"cpu_cores\":\"4\"},\"memory\":{\"total\":\"16496934912\"},\"network\":{\"ipaddress\":\"192.168.1.101\"},\"platform\":{\"hostname\":\"warning-host\",\"os\":\"Linux\",\"kernel_release\":\"5.14.0\"},\"filesystem\":[]}"}
			],
			"err": ""
		}`))
	})
	defer n9eServer.Close()

	// Create VM mock server with CPU at warning level (75%)
	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		var value string

		switch query {
		case `cpu_usage_active{cpu="cpu-total"}`:
			value = "75.0" // Warning level
		case "100 - mem_available_percent":
			value = "50.0" // Normal
		default:
			value = "0"
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result": []map[string]interface{}{
					{"metric": map[string]string{"ident": "warning-host"}, "value": []interface{}{float64(time.Now().Unix()), value}},
				},
			},
		})
	})
	defer vmServer.Close()

	cfg := createTestConfig()
	cfg.Datasources.N9E.Endpoint = n9eServer.URL
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL
	cfg.Report.Timezone = "Asia/Shanghai"
	cfg.Thresholds = config.ThresholdsConfig{
		CPUUsage:        config.ThresholdPair{Warning: 70, Critical: 90},
		MemoryUsage:     config.ThresholdPair{Warning: 70, Critical: 90},
		DiskUsage:       config.ThresholdPair{Warning: 70, Critical: 90},
		ZombieProcesses: config.ThresholdPair{Warning: 1, Critical: 10},
		LoadPerCore:     config.ThresholdPair{Warning: 0.7, Critical: 1.0},
	}

	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	collector := NewCollector(cfg, n9eClient, vmClient, metrics, logger)
	evaluator := NewEvaluator(&cfg.Thresholds, metrics, logger)

	inspector, err := NewInspector(cfg, collector, evaluator, logger)
	if err != nil {
		t.Fatalf("NewInspector failed: %v", err)
	}

	result, err := inspector.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.Summary.WarningHosts != 1 {
		t.Errorf("expected 1 warning host, got %d", result.Summary.WarningHosts)
	}
	if result.Summary.CriticalHosts != 0 {
		t.Errorf("expected 0 critical hosts, got %d", result.Summary.CriticalHosts)
	}
	if result.AlertSummary.WarningCount != 1 {
		t.Errorf("expected 1 warning alert, got %d", result.AlertSummary.WarningCount)
	}
	if len(result.Hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(result.Hosts))
	}
	if result.Hosts[0].Status != model.HostStatusWarning {
		t.Errorf("expected warning status, got %s", result.Hosts[0].Status)
	}
}

// TestInspector_Run_WithCritical tests inspection with critical alerts.
func TestInspector_Run_WithCritical(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createTestMetrics()

	// Create N9E mock server
	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"dat": [
				{"ident": "critical-host", "extend_info": "{\"cpu\":{\"cpu_cores\":\"4\"},\"memory\":{\"total\":\"16496934912\"},\"network\":{\"ipaddress\":\"192.168.1.102\"},\"platform\":{\"hostname\":\"critical-host\",\"os\":\"Linux\",\"kernel_release\":\"5.14.0\"},\"filesystem\":[]}"}
			],
			"err": ""
		}`))
	})
	defer n9eServer.Close()

	// Create VM mock server with CPU at critical level (95%)
	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		var value string

		switch query {
		case `cpu_usage_active{cpu="cpu-total"}`:
			value = "95.0" // Critical level
		case "100 - mem_available_percent":
			value = "50.0" // Normal
		default:
			value = "0"
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result": []map[string]interface{}{
					{"metric": map[string]string{"ident": "critical-host"}, "value": []interface{}{float64(time.Now().Unix()), value}},
				},
			},
		})
	})
	defer vmServer.Close()

	cfg := createTestConfig()
	cfg.Datasources.N9E.Endpoint = n9eServer.URL
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL
	cfg.Report.Timezone = "Asia/Shanghai"
	cfg.Thresholds = config.ThresholdsConfig{
		CPUUsage:        config.ThresholdPair{Warning: 70, Critical: 90},
		MemoryUsage:     config.ThresholdPair{Warning: 70, Critical: 90},
		DiskUsage:       config.ThresholdPair{Warning: 70, Critical: 90},
		ZombieProcesses: config.ThresholdPair{Warning: 1, Critical: 10},
		LoadPerCore:     config.ThresholdPair{Warning: 0.7, Critical: 1.0},
	}

	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	collector := NewCollector(cfg, n9eClient, vmClient, metrics, logger)
	evaluator := NewEvaluator(&cfg.Thresholds, metrics, logger)

	inspector, err := NewInspector(cfg, collector, evaluator, logger)
	if err != nil {
		t.Fatalf("NewInspector failed: %v", err)
	}

	result, err := inspector.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.Summary.CriticalHosts != 1 {
		t.Errorf("expected 1 critical host, got %d", result.Summary.CriticalHosts)
	}
	if result.AlertSummary.CriticalCount != 1 {
		t.Errorf("expected 1 critical alert, got %d", result.AlertSummary.CriticalCount)
	}
	if len(result.Hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(result.Hosts))
	}
	if result.Hosts[0].Status != model.HostStatusCritical {
		t.Errorf("expected critical status, got %s", result.Hosts[0].Status)
	}
}

// TestInspector_Run_MultipleHosts tests inspection with multiple hosts.
func TestInspector_Run_MultipleHosts(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createTestMetrics()

	// Create N9E mock server with 3 hosts
	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"dat": [
				{"ident": "host-normal", "extend_info": "{\"cpu\":{\"cpu_cores\":\"4\"},\"memory\":{\"total\":\"16496934912\"},\"network\":{\"ipaddress\":\"192.168.1.10\"},\"platform\":{\"hostname\":\"host-normal\",\"os\":\"Linux\",\"kernel_release\":\"5.14.0\"},\"filesystem\":[]}"},
				{"ident": "host-warning", "extend_info": "{\"cpu\":{\"cpu_cores\":\"4\"},\"memory\":{\"total\":\"16496934912\"},\"network\":{\"ipaddress\":\"192.168.1.11\"},\"platform\":{\"hostname\":\"host-warning\",\"os\":\"Linux\",\"kernel_release\":\"5.14.0\"},\"filesystem\":[]}"},
				{"ident": "host-critical", "extend_info": "{\"cpu\":{\"cpu_cores\":\"4\"},\"memory\":{\"total\":\"16496934912\"},\"network\":{\"ipaddress\":\"192.168.1.12\"},\"platform\":{\"hostname\":\"host-critical\",\"os\":\"Linux\",\"kernel_release\":\"5.14.0\"},\"filesystem\":[]}"}
			],
			"err": ""
		}`))
	})
	defer n9eServer.Close()

	// Create VM mock server with different values per host
	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")

		var results []map[string]interface{}
		ts := float64(time.Now().Unix())

		if query == `cpu_usage_active{cpu="cpu-total"}` {
			results = []map[string]interface{}{
				{"metric": map[string]string{"ident": "host-normal"}, "value": []interface{}{ts, "30.0"}},
				{"metric": map[string]string{"ident": "host-warning"}, "value": []interface{}{ts, "75.0"}},
				{"metric": map[string]string{"ident": "host-critical"}, "value": []interface{}{ts, "95.0"}},
			}
		} else if query == "100 - mem_available_percent" {
			results = []map[string]interface{}{
				{"metric": map[string]string{"ident": "host-normal"}, "value": []interface{}{ts, "40.0"}},
				{"metric": map[string]string{"ident": "host-warning"}, "value": []interface{}{ts, "50.0"}},
				{"metric": map[string]string{"ident": "host-critical"}, "value": []interface{}{ts, "60.0"}},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result":     results,
			},
		})
	})
	defer vmServer.Close()

	cfg := createTestConfig()
	cfg.Datasources.N9E.Endpoint = n9eServer.URL
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL
	cfg.Report.Timezone = "Asia/Shanghai"
	cfg.Thresholds = config.ThresholdsConfig{
		CPUUsage:        config.ThresholdPair{Warning: 70, Critical: 90},
		MemoryUsage:     config.ThresholdPair{Warning: 70, Critical: 90},
		DiskUsage:       config.ThresholdPair{Warning: 70, Critical: 90},
		ZombieProcesses: config.ThresholdPair{Warning: 1, Critical: 10},
		LoadPerCore:     config.ThresholdPair{Warning: 0.7, Critical: 1.0},
	}

	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	collector := NewCollector(cfg, n9eClient, vmClient, metrics, logger)
	evaluator := NewEvaluator(&cfg.Thresholds, metrics, logger)

	inspector, err := NewInspector(cfg, collector, evaluator, logger)
	if err != nil {
		t.Fatalf("NewInspector failed: %v", err)
	}

	result, err := inspector.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify summary
	if result.Summary.TotalHosts != 3 {
		t.Errorf("expected 3 total hosts, got %d", result.Summary.TotalHosts)
	}
	if result.Summary.NormalHosts != 1 {
		t.Errorf("expected 1 normal host, got %d", result.Summary.NormalHosts)
	}
	if result.Summary.WarningHosts != 1 {
		t.Errorf("expected 1 warning host, got %d", result.Summary.WarningHosts)
	}
	if result.Summary.CriticalHosts != 1 {
		t.Errorf("expected 1 critical host, got %d", result.Summary.CriticalHosts)
	}

	// Verify alert summary
	if result.AlertSummary.WarningCount != 1 {
		t.Errorf("expected 1 warning alert, got %d", result.AlertSummary.WarningCount)
	}
	if result.AlertSummary.CriticalCount != 1 {
		t.Errorf("expected 1 critical alert, got %d", result.AlertSummary.CriticalCount)
	}
}

// TestInspector_Run_CollectorError tests inspection when collector fails.
func TestInspector_Run_CollectorError(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createTestMetrics()

	// Create N9E mock server that returns error
	n9eServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"err": "server error"}`))
	}))
	defer n9eServer.Close()

	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data":   map[string]interface{}{"resultType": "vector", "result": []interface{}{}},
		})
	})
	defer vmServer.Close()

	cfg := createTestConfig()
	cfg.Datasources.N9E.Endpoint = n9eServer.URL
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL
	cfg.Report.Timezone = "Asia/Shanghai"
	cfg.HTTP.Retry.MaxRetries = 0 // Disable retries for faster test

	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	collector := NewCollector(cfg, n9eClient, vmClient, metrics, logger)
	evaluator := NewEvaluator(&cfg.Thresholds, metrics, logger)

	inspector, err := NewInspector(cfg, collector, evaluator, logger)
	if err != nil {
		t.Fatalf("NewInspector failed: %v", err)
	}

	_, err = inspector.Run(context.Background())
	if err == nil {
		t.Error("expected error when collector fails")
	}
}

// TestInspector_Timezone tests timezone handling.
func TestInspector_Timezone(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createTestMetrics()

	// Create mock servers
	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"dat": [], "err": ""}`))
	})
	defer n9eServer.Close()

	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data":   map[string]interface{}{"resultType": "vector", "result": []interface{}{}},
		})
	})
	defer vmServer.Close()

	cfg := createTestConfig()
	cfg.Datasources.N9E.Endpoint = n9eServer.URL
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL
	cfg.Report.Timezone = "Asia/Shanghai"

	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	collector := NewCollector(cfg, n9eClient, vmClient, metrics, logger)
	evaluator := NewEvaluator(&cfg.Thresholds, metrics, logger)

	inspector, err := NewInspector(cfg, collector, evaluator, logger)
	if err != nil {
		t.Fatalf("NewInspector failed: %v", err)
	}

	result, err := inspector.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify timezone
	tz := result.InspectionTime.Location().String()
	if tz != "Asia/Shanghai" {
		t.Errorf("expected timezone Asia/Shanghai, got %s", tz)
	}

	// Verify GetTimezone
	if inspector.GetTimezone().String() != "Asia/Shanghai" {
		t.Errorf("GetTimezone returned wrong value: %s", inspector.GetTimezone().String())
	}
}

// TestInspector_GetVersion tests the GetVersion method.
func TestInspector_GetVersion(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createTestMetrics()

	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"dat": [], "err": ""}`))
	})
	defer n9eServer.Close()

	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "data": map[string]interface{}{"resultType": "vector", "result": []interface{}{}}})
	})
	defer vmServer.Close()

	cfg := createTestConfig()
	cfg.Datasources.N9E.Endpoint = n9eServer.URL
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL
	cfg.Report.Timezone = "Asia/Shanghai"

	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	collector := NewCollector(cfg, n9eClient, vmClient, metrics, logger)
	evaluator := NewEvaluator(&cfg.Thresholds, metrics, logger)

	inspector, err := NewInspector(cfg, collector, evaluator, logger, WithVersion("test-version"))
	if err != nil {
		t.Fatalf("NewInspector failed: %v", err)
	}

	if inspector.GetVersion() != "test-version" {
		t.Errorf("expected version test-version, got %s", inspector.GetVersion())
	}
}

// TestInspector_Run_ContextCanceled tests inspection with canceled context.
func TestInspector_Run_ContextCanceled(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createTestMetrics()

	// Create slow N9E mock server
	n9eServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"dat": [], "err": ""}`))
	}))
	defer n9eServer.Close()

	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "data": map[string]interface{}{"resultType": "vector", "result": []interface{}{}}})
	})
	defer vmServer.Close()

	cfg := createTestConfig()
	cfg.Datasources.N9E.Endpoint = n9eServer.URL
	cfg.Datasources.N9E.Timeout = 5 * time.Second
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL
	cfg.Report.Timezone = "Asia/Shanghai"

	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	collector := NewCollector(cfg, n9eClient, vmClient, metrics, logger)
	evaluator := NewEvaluator(&cfg.Thresholds, metrics, logger)

	inspector, err := NewInspector(cfg, collector, evaluator, logger)
	if err != nil {
		t.Fatalf("NewInspector failed: %v", err)
	}

	// Create a context that will be canceled immediately
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = inspector.Run(ctx)
	if err == nil {
		t.Error("expected error when context is canceled")
	}
}

// formatFloatAsString helper function for consistent float formatting in tests.
func formatFloatAsString(v float64) string {
	return fmt.Sprintf("%.2f", v)
}
