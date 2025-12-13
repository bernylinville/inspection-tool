package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"inspection-tool/internal/client/n9e"
	"inspection-tool/internal/client/vm"
	"inspection-tool/internal/config"
	"inspection-tool/internal/model"
)

// Sample extend_info for testing
const testExtendInfo = `{
	"cpu": {"cpu_cores": "4", "model_name": "Intel Xeon"},
	"memory": {"total": "16496934912"},
	"network": {"ipaddress": "192.168.1.100"},
	"platform": {"hostname": "test-host-1", "os": "GNU/Linux", "kernel_release": "5.14.0"},
	"filesystem": [{"kb_size": "103084600", "mounted_on": "/", "name": "/dev/sda1"}]
}`

const testExtendInfo2 = `{
	"cpu": {"cpu_cores": "8", "model_name": "AMD EPYC"},
	"memory": {"total": "32993869824"},
	"network": {"ipaddress": "192.168.1.101"},
	"platform": {"hostname": "test-host-2", "os": "GNU/Linux", "kernel_release": "5.15.0"},
	"filesystem": [{"kb_size": "206169200", "mounted_on": "/", "name": "/dev/sda1"}]
}`

// setupN9ETestServer creates a test N9E server.
func setupN9ETestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

// setupVMTestServer creates a test VictoriaMetrics server.
func setupVMTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

// createN9EClient creates a test N9E client.
func createN9EClient(serverURL string) *n9e.Client {
	cfg := &config.N9EConfig{
		Endpoint: serverURL,
		Token:    "test-token",
		Timeout:  5 * time.Second,
	}
	retryCfg := &config.RetryConfig{
		MaxRetries: 1,
		BaseDelay:  10 * time.Millisecond,
	}
	logger := zerolog.Nop()
	return n9e.NewClient(cfg, retryCfg, logger)
}

// createVMClient creates a test VM client.
func createVMClient(serverURL string) *vm.Client {
	cfg := &config.VictoriaMetricsConfig{
		Endpoint: serverURL,
		Timeout:  5 * time.Second,
	}
	retryCfg := &config.RetryConfig{
		MaxRetries: 1,
		BaseDelay:  10 * time.Millisecond,
	}
	logger := zerolog.Nop()
	return vm.NewClient(cfg, retryCfg, logger)
}

// createTestConfig creates a test configuration.
func createTestConfig() *config.Config {
	return &config.Config{
		Datasources: config.DatasourcesConfig{
			N9E: config.N9EConfig{
				Endpoint: "http://localhost:17000",
				Token:    "test-token",
				Timeout:  30 * time.Second,
			},
			VictoriaMetrics: config.VictoriaMetricsConfig{
				Endpoint: "http://localhost:8428",
				Timeout:  30 * time.Second,
			},
		},
		Inspection: config.InspectionConfig{
			Concurrency: 10,
			HostTimeout: 30 * time.Second,
		},
	}
}

// createTestMetrics creates test metric definitions.
func createTestMetrics() []*model.MetricDefinition {
	return []*model.MetricDefinition{
		{
			Name:        "cpu_usage",
			DisplayName: "CPU 利用率",
			Query:       `cpu_usage_active{cpu="cpu-total"}`,
			Unit:        "%",
			Category:    model.MetricCategoryCPU,
			Format:      model.MetricFormatPercent,
		},
		{
			Name:        "memory_usage",
			DisplayName: "内存利用率",
			Query:       "100 - mem_available_percent",
			Unit:        "%",
			Category:    model.MetricCategoryMemory,
			Format:      model.MetricFormatPercent,
		},
		{
			Name:          "disk_usage",
			DisplayName:   "磁盘利用率",
			Query:         "disk_used_percent",
			Unit:          "%",
			Category:      model.MetricCategoryDisk,
			Format:        model.MetricFormatPercent,
			Aggregate:     model.AggregateMax,
			ExpandByLabel: "path",
		},
		{
			Name:        "ntp_check",
			DisplayName: "NTP 检查",
			Query:       "",
			Status:      "pending",
			Category:    model.MetricCategorySystem,
		},
	}
}

// =============================================================================
// NewCollector Tests
// =============================================================================

func TestNewCollector(t *testing.T) {
	cfg := createTestConfig()
	logger := zerolog.Nop()
	metrics := createTestMetrics()

	// Create mock servers (not used, just for client creation)
	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {})
	defer n9eServer.Close()

	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {})
	defer vmServer.Close()

	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)

	collector := NewCollector(cfg, n9eClient, vmClient, metrics, logger)

	if collector == nil {
		t.Fatal("NewCollector returned nil")
	}
	if collector.config != cfg {
		t.Error("Config not set correctly")
	}
	if collector.n9eClient != n9eClient {
		t.Error("N9E client not set correctly")
	}
	if collector.vmClient != vmClient {
		t.Error("VM client not set correctly")
	}
	if len(collector.metrics) != len(metrics) {
		t.Errorf("Expected %d metrics, got %d", len(metrics), len(collector.metrics))
	}
}

func TestNewCollector_WithHostFilter(t *testing.T) {
	cfg := createTestConfig()
	cfg.Inspection.HostFilter = config.HostFilter{
		BusinessGroups: []string{"production", "staging"},
		Tags:           map[string]string{"env": "prod"},
	}
	logger := zerolog.Nop()
	metrics := createTestMetrics()

	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {})
	defer n9eServer.Close()

	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {})
	defer vmServer.Close()

	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)

	collector := NewCollector(cfg, n9eClient, vmClient, metrics, logger)

	if collector.hostFilter == nil {
		t.Fatal("Host filter should be set")
	}
	if len(collector.hostFilter.BusinessGroups) != 2 {
		t.Errorf("Expected 2 business groups, got %d", len(collector.hostFilter.BusinessGroups))
	}
	if collector.hostFilter.Tags["env"] != "prod" {
		t.Errorf("Expected tag env=prod, got %v", collector.hostFilter.Tags)
	}
}

// =============================================================================
// CollectHostMetas Tests
// =============================================================================

func TestCollector_CollectHostMetas_Success(t *testing.T) {
	// Setup N9E server
	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/n9e/targets" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// Note: extend_info must be a JSON string (escaped)
			_, _ = w.Write([]byte(`{
				"dat": [
					{"ident": "test-host-1", "extend_info": "{\"cpu\":{\"cpu_cores\":\"4\",\"model_name\":\"Intel Xeon\"},\"memory\":{\"total\":\"16496934912\"},\"network\":{\"ipaddress\":\"192.168.1.100\"},\"platform\":{\"hostname\":\"test-host-1\",\"os\":\"GNU/Linux\",\"kernel_release\":\"5.14.0\"},\"filesystem\":[{\"kb_size\":\"103084600\",\"mounted_on\":\"/\",\"name\":\"/dev/sda1\"}]}"},
					{"ident": "test-host-2", "extend_info": "{\"cpu\":{\"cpu_cores\":\"8\",\"model_name\":\"AMD EPYC\"},\"memory\":{\"total\":\"32993869824\"},\"network\":{\"ipaddress\":\"192.168.1.101\"},\"platform\":{\"hostname\":\"test-host-2\",\"os\":\"GNU/Linux\",\"kernel_release\":\"5.15.0\"},\"filesystem\":[{\"kb_size\":\"206169200\",\"mounted_on\":\"/\",\"name\":\"/dev/sda1\"}]}"}
				],
				"err": ""
			}`))
		}
	})
	defer n9eServer.Close()

	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {})
	defer vmServer.Close()

	cfg := createTestConfig()
	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	metrics := createTestMetrics()
	logger := zerolog.Nop()

	collector := NewCollector(cfg, n9eClient, vmClient, metrics, logger)

	ctx := context.Background()
	hosts, err := collector.CollectHostMetas(ctx)

	if err != nil {
		t.Fatalf("CollectHostMetas failed: %v", err)
	}
	if len(hosts) != 2 {
		t.Errorf("Expected 2 hosts, got %d", len(hosts))
	}

	// Verify first host
	if hosts[0].Hostname != "test-host-1" {
		t.Errorf("Expected hostname 'test-host-1', got '%s'", hosts[0].Hostname)
	}
	if hosts[0].CPUCores != 4 {
		t.Errorf("Expected 4 CPU cores, got %d", hosts[0].CPUCores)
	}
}

func TestCollector_CollectHostMetas_N9EError(t *testing.T) {
	// Setup N9E server that returns error
	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error"))
	})
	defer n9eServer.Close()

	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {})
	defer vmServer.Close()

	cfg := createTestConfig()
	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	metrics := createTestMetrics()
	logger := zerolog.Nop()

	collector := NewCollector(cfg, n9eClient, vmClient, metrics, logger)

	ctx := context.Background()
	_, err := collector.CollectHostMetas(ctx)

	if err == nil {
		t.Fatal("Expected error from N9E, got nil")
	}
}

func TestCollector_CollectHostMetas_EmptyList(t *testing.T) {
	// Setup N9E server that returns empty list
	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"dat": [], "err": ""}`))
	})
	defer n9eServer.Close()

	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {})
	defer vmServer.Close()

	cfg := createTestConfig()
	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	metrics := createTestMetrics()
	logger := zerolog.Nop()

	collector := NewCollector(cfg, n9eClient, vmClient, metrics, logger)

	ctx := context.Background()
	hosts, err := collector.CollectHostMetas(ctx)

	if err != nil {
		t.Fatalf("CollectHostMetas failed: %v", err)
	}
	if len(hosts) != 0 {
		t.Errorf("Expected 0 hosts, got %d", len(hosts))
	}
}

// =============================================================================
// CollectMetrics Tests
// =============================================================================

func TestCollector_CollectMetrics_Success(t *testing.T) {
	// Setup N9E server
	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {})
	defer n9eServer.Close()

	// Setup VM server
	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		var result []map[string]interface{}

		// Use contains instead of exact match due to possible URL encoding
		switch {
		case containsSubstring(query, "cpu_usage_active"):
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"ident": "test-host-1", "__name__": "cpu_usage_active"},
					"value":  []interface{}{1702483200.0, "45.5"},
				},
				{
					"metric": map[string]string{"ident": "test-host-2", "__name__": "cpu_usage_active"},
					"value":  []interface{}{1702483200.0, "78.2"},
				},
			}
		case containsSubstring(query, "mem_available_percent"):
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"ident": "test-host-1"},
					"value":  []interface{}{1702483200.0, "62.3"},
				},
				{
					"metric": map[string]string{"ident": "test-host-2"},
					"value":  []interface{}{1702483200.0, "85.1"},
				},
			}
		case containsSubstring(query, "disk_used_percent"):
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"ident": "test-host-1", "path": "/"},
					"value":  []interface{}{1702483200.0, "55.0"},
				},
				{
					"metric": map[string]string{"ident": "test-host-1", "path": "/home"},
					"value":  []interface{}{1702483200.0, "72.0"},
				},
				{
					"metric": map[string]string{"ident": "test-host-2", "path": "/"},
					"value":  []interface{}{1702483200.0, "90.5"},
				},
			}
		default:
			result = []map[string]interface{}{}
		}

		resp := map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result":     result,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer vmServer.Close()

	cfg := createTestConfig()
	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	metrics := createTestMetrics()
	logger := zerolog.Nop()

	collector := NewCollector(cfg, n9eClient, vmClient, metrics, logger)

	// Create test hosts
	hosts := []*model.HostMeta{
		{Ident: "test-host-1", Hostname: "test-host-1", IP: "192.168.1.100"},
		{Ident: "test-host-2", Hostname: "test-host-2", IP: "192.168.1.101"},
	}

	ctx := context.Background()
	hostMetrics, err := collector.CollectMetrics(ctx, hosts, metrics)

	if err != nil {
		t.Fatalf("CollectMetrics failed: %v", err)
	}
	if len(hostMetrics) != 2 {
		t.Fatalf("Expected 2 hosts, got %d", len(hostMetrics))
	}

	// Verify host 1 metrics
	hm1 := hostMetrics["test-host-1"]
	if hm1 == nil {
		t.Fatal("Host 1 metrics not found")
	}

	// Check CPU metric
	cpuMetric := hm1.GetMetric("cpu_usage")
	if cpuMetric == nil {
		t.Error("CPU metric not found for host 1")
	} else if cpuMetric.RawValue != 45.5 {
		t.Errorf("Expected CPU value 45.5, got %f", cpuMetric.RawValue)
	}

	// Check memory metric
	memMetric := hm1.GetMetric("memory_usage")
	if memMetric == nil {
		t.Error("Memory metric not found for host 1")
	} else if memMetric.RawValue != 62.3 {
		t.Errorf("Expected memory value 62.3, got %f", memMetric.RawValue)
	}

	// Check expanded disk metrics
	diskRoot := hm1.GetMetric("disk_usage:/")
	if diskRoot == nil {
		t.Error("Disk metric for / not found for host 1")
	} else if diskRoot.RawValue != 55.0 {
		t.Errorf("Expected disk / value 55.0, got %f", diskRoot.RawValue)
	}

	diskHome := hm1.GetMetric("disk_usage:/home")
	if diskHome == nil {
		t.Error("Disk metric for /home not found for host 1")
	} else if diskHome.RawValue != 72.0 {
		t.Errorf("Expected disk /home value 72.0, got %f", diskHome.RawValue)
	}

	// Check aggregated disk max
	diskMax := hm1.GetMetric("disk_usage_max")
	if diskMax == nil {
		t.Error("Disk max metric not found for host 1")
	} else if diskMax.RawValue != 72.0 {
		t.Errorf("Expected disk max value 72.0, got %f", diskMax.RawValue)
	}

	// Check pending metric
	ntpMetric := hm1.GetMetric("ntp_check")
	if ntpMetric == nil {
		t.Error("NTP metric not found for host 1")
	} else if !ntpMetric.IsNA {
		t.Error("NTP metric should be N/A")
	}
}

func TestCollector_CollectMetrics_PendingMetrics(t *testing.T) {
	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {})
	defer n9eServer.Close()

	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result":     []map[string]interface{}{},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer vmServer.Close()

	cfg := createTestConfig()
	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	logger := zerolog.Nop()

	// Only pending metrics
	pendingMetrics := []*model.MetricDefinition{
		{Name: "ntp_check", Status: "pending"},
		{Name: "password_expiry", Query: ""},
	}

	collector := NewCollector(cfg, n9eClient, vmClient, pendingMetrics, logger)

	hosts := []*model.HostMeta{
		{Ident: "test-host-1", Hostname: "test-host-1"},
	}

	ctx := context.Background()
	hostMetrics, err := collector.CollectMetrics(ctx, hosts, pendingMetrics)

	if err != nil {
		t.Fatalf("CollectMetrics failed: %v", err)
	}

	hm := hostMetrics["test-host-1"]
	if hm == nil {
		t.Fatal("Host metrics not found")
	}

	// Both pending metrics should be N/A
	for _, metric := range pendingMetrics {
		mv := hm.GetMetric(metric.Name)
		if mv == nil {
			t.Errorf("Metric %s not found", metric.Name)
			continue
		}
		if !mv.IsNA {
			t.Errorf("Metric %s should be N/A", metric.Name)
		}
		if mv.Status != model.MetricStatusPending {
			t.Errorf("Metric %s should have pending status", metric.Name)
		}
	}
}

func TestCollector_CollectMetrics_VMQueryError(t *testing.T) {
	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {})
	defer n9eServer.Close()

	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	})
	defer vmServer.Close()

	cfg := createTestConfig()
	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	logger := zerolog.Nop()

	metrics := []*model.MetricDefinition{
		{Name: "cpu_usage", Query: "cpu_usage_active"},
	}

	collector := NewCollector(cfg, n9eClient, vmClient, metrics, logger)

	hosts := []*model.HostMeta{
		{Ident: "test-host-1", Hostname: "test-host-1"},
	}

	ctx := context.Background()
	hostMetrics, err := collector.CollectMetrics(ctx, hosts, metrics)

	// Should not return error, just log warning and continue
	if err != nil {
		t.Fatalf("CollectMetrics should not fail on single metric error: %v", err)
	}

	// Host should exist but metric should not be set (failed to collect)
	hm := hostMetrics["test-host-1"]
	if hm == nil {
		t.Fatal("Host metrics not found")
	}

	cpuMetric := hm.GetMetric("cpu_usage")
	if cpuMetric != nil {
		t.Error("CPU metric should not be set on error")
	}
}

// =============================================================================
// CollectAll Tests
// =============================================================================

func TestCollector_CollectAll_Success(t *testing.T) {
	// Setup N9E server
	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"dat": [
				{"ident": "test-host-1", "extend_info": "{\"cpu\":{\"cpu_cores\":\"4\"},\"memory\":{\"total\":\"16496934912\"},\"network\":{\"ipaddress\":\"192.168.1.100\"},\"platform\":{\"hostname\":\"test-host-1\",\"os\":\"GNU/Linux\",\"kernel_release\":\"5.14.0\"},\"filesystem\":[]}"}
			],
			"err": ""
		}`))
	})
	defer n9eServer.Close()

	// Setup VM server
	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		var result []map[string]interface{}

		if containsSubstring(query, "cpu_usage_active") {
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"ident": "test-host-1"},
					"value":  []interface{}{1702483200.0, "50.0"},
				},
			}
		}

		resp := map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result":     result,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer vmServer.Close()

	cfg := createTestConfig()
	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	logger := zerolog.Nop()

	metrics := []*model.MetricDefinition{
		{Name: "cpu_usage", Query: `cpu_usage_active{cpu="cpu-total"}`},
	}

	collector := NewCollector(cfg, n9eClient, vmClient, metrics, logger)

	ctx := context.Background()
	result, err := collector.CollectAll(ctx)

	if err != nil {
		t.Fatalf("CollectAll failed: %v", err)
	}
	if result == nil {
		t.Fatal("CollectAll returned nil result")
	}
	if len(result.Hosts) != 1 {
		t.Errorf("Expected 1 host, got %d", len(result.Hosts))
	}
	if len(result.HostMetrics) != 1 {
		t.Errorf("Expected 1 host metrics, got %d", len(result.HostMetrics))
	}
	if result.CollectedAt.IsZero() {
		t.Error("CollectedAt should be set")
	}
}

func TestCollector_CollectAll_NoHosts(t *testing.T) {
	// Setup N9E server that returns empty list
	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"dat": [], "err": ""}`))
	})
	defer n9eServer.Close()

	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {})
	defer vmServer.Close()

	cfg := createTestConfig()
	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	logger := zerolog.Nop()

	collector := NewCollector(cfg, n9eClient, vmClient, createTestMetrics(), logger)

	ctx := context.Background()
	result, err := collector.CollectAll(ctx)

	if err != nil {
		t.Fatalf("CollectAll failed: %v", err)
	}
	if len(result.Hosts) != 0 {
		t.Errorf("Expected 0 hosts, got %d", len(result.Hosts))
	}
	if len(result.HostMetrics) != 0 {
		t.Errorf("Expected 0 host metrics, got %d", len(result.HostMetrics))
	}
}

func TestCollector_CollectAll_N9EFailure(t *testing.T) {
	// Setup N9E server that fails
	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer n9eServer.Close()

	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {})
	defer vmServer.Close()

	cfg := createTestConfig()
	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	logger := zerolog.Nop()

	collector := NewCollector(cfg, n9eClient, vmClient, createTestMetrics(), logger)

	ctx := context.Background()
	_, err := collector.CollectAll(ctx)

	if err == nil {
		t.Fatal("Expected error from CollectAll")
	}
}

// =============================================================================
// Host Filter Tests
// =============================================================================

func TestCollector_HostFilter_Applied(t *testing.T) {
	var capturedQuery string

	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {})
	defer n9eServer.Close()

	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query().Get("query")
		resp := map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result":     []map[string]interface{}{},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer vmServer.Close()

	cfg := createTestConfig()
	cfg.Inspection.HostFilter = config.HostFilter{
		BusinessGroups: []string{"production"},
		Tags:           map[string]string{"env": "prod"},
	}

	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	logger := zerolog.Nop()

	metrics := []*model.MetricDefinition{
		{Name: "cpu_usage", Query: "cpu_usage_active"},
	}

	collector := NewCollector(cfg, n9eClient, vmClient, metrics, logger)

	hosts := []*model.HostMeta{
		{Hostname: "test-host"},
	}

	ctx := context.Background()
	collector.CollectMetrics(ctx, hosts, metrics)

	// Verify filter was applied to query
	if capturedQuery == "" {
		t.Fatal("Query not captured")
	}
	// The filter should be injected into the query
	if !contains(capturedQuery, "busigroup") {
		t.Errorf("Expected busigroup filter in query, got: %s", capturedQuery)
	}
	if !contains(capturedQuery, "env") {
		t.Errorf("Expected env filter in query, got: %s", capturedQuery)
	}
}

func TestCollector_HostFilter_Nil(t *testing.T) {
	var capturedQuery string

	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {})
	defer n9eServer.Close()

	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query().Get("query")
		resp := map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result":     []map[string]interface{}{},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer vmServer.Close()

	cfg := createTestConfig()
	// No host filter

	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	logger := zerolog.Nop()

	metrics := []*model.MetricDefinition{
		{Name: "cpu_usage", Query: "cpu_usage_active"},
	}

	collector := NewCollector(cfg, n9eClient, vmClient, metrics, logger)

	hosts := []*model.HostMeta{
		{Hostname: "test-host"},
	}

	ctx := context.Background()
	collector.CollectMetrics(ctx, hosts, metrics)

	// Query should be unmodified
	if capturedQuery != "cpu_usage_active" {
		t.Errorf("Expected query 'cpu_usage_active', got: %s", capturedQuery)
	}
}

// =============================================================================
// Expanded Metric Tests
// =============================================================================

func TestCollector_ExpandedMetric_DiskByPath(t *testing.T) {
	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {})
	defer n9eServer.Close()

	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		result := []map[string]interface{}{
			{
				"metric": map[string]string{"ident": "host1", "path": "/"},
				"value":  []interface{}{1702483200.0, "40.0"},
			},
			{
				"metric": map[string]string{"ident": "host1", "path": "/var"},
				"value":  []interface{}{1702483200.0, "60.0"},
			},
			{
				"metric": map[string]string{"ident": "host1", "path": "/home"},
				"value":  []interface{}{1702483200.0, "80.0"},
			},
		}

		resp := map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result":     result,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer vmServer.Close()

	cfg := createTestConfig()
	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	logger := zerolog.Nop()

	metrics := []*model.MetricDefinition{
		{
			Name:          "disk_usage",
			Query:         "disk_used_percent",
			ExpandByLabel: "path",
			Aggregate:     model.AggregateMax,
		},
	}

	collector := NewCollector(cfg, n9eClient, vmClient, metrics, logger)

	hosts := []*model.HostMeta{
		{Hostname: "host1"},
	}

	ctx := context.Background()
	hostMetrics, err := collector.CollectMetrics(ctx, hosts, metrics)

	if err != nil {
		t.Fatalf("CollectMetrics failed: %v", err)
	}

	hm := hostMetrics["host1"]
	if hm == nil {
		t.Fatal("Host metrics not found")
	}

	// Check each expanded metric
	paths := []struct {
		name  string
		value float64
	}{
		{"disk_usage:/", 40.0},
		{"disk_usage:/var", 60.0},
		{"disk_usage:/home", 80.0},
	}

	for _, p := range paths {
		mv := hm.GetMetric(p.name)
		if mv == nil {
			t.Errorf("Metric %s not found", p.name)
			continue
		}
		if mv.RawValue != p.value {
			t.Errorf("Expected %s value %.1f, got %.1f", p.name, p.value, mv.RawValue)
		}
		if mv.Labels["path"] == "" {
			t.Errorf("Metric %s should have path label", p.name)
		}
	}

	// Check aggregated max value
	maxMetric := hm.GetMetric("disk_usage_max")
	if maxMetric == nil {
		t.Error("Aggregated max metric not found")
	} else if maxMetric.RawValue != 80.0 {
		t.Errorf("Expected max value 80.0, got %.1f", maxMetric.RawValue)
	}
}

// =============================================================================
// Context Cancellation Tests
// =============================================================================

func TestCollector_CollectHostMetas_ContextCanceled(t *testing.T) {
	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
	})
	defer n9eServer.Close()

	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {})
	defer vmServer.Close()

	cfg := createTestConfig()
	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	logger := zerolog.Nop()

	collector := NewCollector(cfg, n9eClient, vmClient, createTestMetrics(), logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := collector.CollectHostMetas(ctx)

	if err == nil {
		t.Fatal("Expected error from canceled context")
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// =============================================================================
// Concurrent Collection Tests
// =============================================================================

// TestCollector_CollectMetrics_Concurrent tests concurrent collection of multiple metrics.
func TestCollector_CollectMetrics_Concurrent(t *testing.T) {
	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {})
	defer n9eServer.Close()

	// Setup VM server that responds to multiple metric queries
	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		var result []map[string]interface{}

		// Simulate different metrics
		switch {
		case containsSubstring(query, "metric_1"):
			result = []map[string]interface{}{
				{"metric": map[string]string{"ident": "host1"}, "value": []interface{}{1702483200.0, "10.0"}},
				{"metric": map[string]string{"ident": "host2"}, "value": []interface{}{1702483200.0, "11.0"}},
			}
		case containsSubstring(query, "metric_2"):
			result = []map[string]interface{}{
				{"metric": map[string]string{"ident": "host1"}, "value": []interface{}{1702483200.0, "20.0"}},
				{"metric": map[string]string{"ident": "host2"}, "value": []interface{}{1702483200.0, "21.0"}},
			}
		case containsSubstring(query, "metric_3"):
			result = []map[string]interface{}{
				{"metric": map[string]string{"ident": "host1"}, "value": []interface{}{1702483200.0, "30.0"}},
				{"metric": map[string]string{"ident": "host2"}, "value": []interface{}{1702483200.0, "31.0"}},
			}
		case containsSubstring(query, "metric_4"):
			result = []map[string]interface{}{
				{"metric": map[string]string{"ident": "host1"}, "value": []interface{}{1702483200.0, "40.0"}},
				{"metric": map[string]string{"ident": "host2"}, "value": []interface{}{1702483200.0, "41.0"}},
			}
		case containsSubstring(query, "metric_5"):
			result = []map[string]interface{}{
				{"metric": map[string]string{"ident": "host1"}, "value": []interface{}{1702483200.0, "50.0"}},
				{"metric": map[string]string{"ident": "host2"}, "value": []interface{}{1702483200.0, "51.0"}},
			}
		}

		resp := map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result":     result,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer vmServer.Close()

	cfg := createTestConfig()
	cfg.Inspection.Concurrency = 5 // Allow 5 concurrent queries
	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	logger := zerolog.Nop()

	// Create 5 metrics to collect concurrently
	metrics := []*model.MetricDefinition{
		{Name: "metric_1", Query: "metric_1"},
		{Name: "metric_2", Query: "metric_2"},
		{Name: "metric_3", Query: "metric_3"},
		{Name: "metric_4", Query: "metric_4"},
		{Name: "metric_5", Query: "metric_5"},
	}

	collector := NewCollector(cfg, n9eClient, vmClient, metrics, logger)

	hosts := []*model.HostMeta{
		{Hostname: "host1"},
		{Hostname: "host2"},
	}

	ctx := context.Background()
	hostMetrics, err := collector.CollectMetrics(ctx, hosts, metrics)

	if err != nil {
		t.Fatalf("CollectMetrics failed: %v", err)
	}

	// Verify all metrics were collected for both hosts
	for _, host := range []string{"host1", "host2"} {
		hm := hostMetrics[host]
		if hm == nil {
			t.Fatalf("Host %s metrics not found", host)
		}

		for i := 1; i <= 5; i++ {
			metricName := "metric_" + string(rune('0'+i))
			mv := hm.GetMetric(metricName)
			if mv == nil {
				t.Errorf("Host %s: metric %s not found", host, metricName)
			}
		}
	}

	// Verify specific values
	if hm := hostMetrics["host1"]; hm != nil {
		if mv := hm.GetMetric("metric_1"); mv != nil && mv.RawValue != 10.0 {
			t.Errorf("Expected metric_1 value 10.0, got %f", mv.RawValue)
		}
		if mv := hm.GetMetric("metric_5"); mv != nil && mv.RawValue != 50.0 {
			t.Errorf("Expected metric_5 value 50.0, got %f", mv.RawValue)
		}
	}
}

// TestCollector_ConcurrencyLimit tests that concurrency is properly limited.
func TestCollector_ConcurrencyLimit(t *testing.T) {
	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {})
	defer n9eServer.Close()

	var (
		mu              sync.Mutex
		currentActive   int
		maxActive       int
		totalRequests   int
	)

	// Setup VM server that tracks concurrent requests
	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		currentActive++
		totalRequests++
		if currentActive > maxActive {
			maxActive = currentActive
		}
		mu.Unlock()

		// Simulate some work
		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		currentActive--
		mu.Unlock()

		resp := map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result": []map[string]interface{}{
					{"metric": map[string]string{"ident": "host1"}, "value": []interface{}{1702483200.0, "50.0"}},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer vmServer.Close()

	cfg := createTestConfig()
	cfg.Inspection.Concurrency = 2 // Limit to 2 concurrent
	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	logger := zerolog.Nop()

	// Create 6 metrics to test concurrency limit
	metrics := []*model.MetricDefinition{
		{Name: "m1", Query: "m1"},
		{Name: "m2", Query: "m2"},
		{Name: "m3", Query: "m3"},
		{Name: "m4", Query: "m4"},
		{Name: "m5", Query: "m5"},
		{Name: "m6", Query: "m6"},
	}

	collector := NewCollector(cfg, n9eClient, vmClient, metrics, logger)

	hosts := []*model.HostMeta{
		{Hostname: "host1"},
	}

	ctx := context.Background()
	_, err := collector.CollectMetrics(ctx, hosts, metrics)

	if err != nil {
		t.Fatalf("CollectMetrics failed: %v", err)
	}

	if totalRequests != 6 {
		t.Errorf("Expected 6 total requests, got %d", totalRequests)
	}

	// Max active should not exceed concurrency limit (2)
	if maxActive > 2 {
		t.Errorf("Concurrency limit exceeded: max active was %d, limit is 2", maxActive)
	}
}

// TestCollector_CollectMetrics_PartialFailure_Concurrent tests that partial failures
// in concurrent collection don't affect other metrics.
func TestCollector_CollectMetrics_PartialFailure_Concurrent(t *testing.T) {
	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {})
	defer n9eServer.Close()

	// Setup VM server that fails for some queries
	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")

		// Fail for metric_2 and metric_4
		if containsSubstring(query, "metric_2") || containsSubstring(query, "metric_4") {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
			return
		}

		var result []map[string]interface{}
		switch {
		case containsSubstring(query, "metric_1"):
			result = []map[string]interface{}{
				{"metric": map[string]string{"ident": "host1"}, "value": []interface{}{1702483200.0, "10.0"}},
			}
		case containsSubstring(query, "metric_3"):
			result = []map[string]interface{}{
				{"metric": map[string]string{"ident": "host1"}, "value": []interface{}{1702483200.0, "30.0"}},
			}
		case containsSubstring(query, "metric_5"):
			result = []map[string]interface{}{
				{"metric": map[string]string{"ident": "host1"}, "value": []interface{}{1702483200.0, "50.0"}},
			}
		}

		resp := map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result":     result,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer vmServer.Close()

	cfg := createTestConfig()
	cfg.Inspection.Concurrency = 5
	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	logger := zerolog.Nop()

	metrics := []*model.MetricDefinition{
		{Name: "metric_1", Query: "metric_1"},
		{Name: "metric_2", Query: "metric_2"}, // Will fail
		{Name: "metric_3", Query: "metric_3"},
		{Name: "metric_4", Query: "metric_4"}, // Will fail
		{Name: "metric_5", Query: "metric_5"},
	}

	collector := NewCollector(cfg, n9eClient, vmClient, metrics, logger)

	hosts := []*model.HostMeta{
		{Hostname: "host1"},
	}

	ctx := context.Background()
	hostMetrics, err := collector.CollectMetrics(ctx, hosts, metrics)

	// Should not return error even with partial failures
	if err != nil {
		t.Fatalf("CollectMetrics should not fail on partial errors: %v", err)
	}

	hm := hostMetrics["host1"]
	if hm == nil {
		t.Fatal("Host metrics not found")
	}

	// Successful metrics should be present
	for _, name := range []string{"metric_1", "metric_3", "metric_5"} {
		if mv := hm.GetMetric(name); mv == nil {
			t.Errorf("Metric %s should be present", name)
		}
	}

	// Failed metrics should NOT be present
	for _, name := range []string{"metric_2", "metric_4"} {
		if mv := hm.GetMetric(name); mv != nil {
			t.Errorf("Metric %s should not be present (query failed)", name)
		}
	}
}

// TestCollector_CollectMetrics_ContextCancel_Concurrent tests context cancellation
// during concurrent collection.
func TestCollector_CollectMetrics_ContextCancel_Concurrent(t *testing.T) {
	n9eServer := setupN9ETestServer(t, func(w http.ResponseWriter, r *http.Request) {})
	defer n9eServer.Close()

	// Setup VM server with slow responses
	vmServer := setupVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(200 * time.Millisecond)

		resp := map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result": []map[string]interface{}{
					{"metric": map[string]string{"ident": "host1"}, "value": []interface{}{1702483200.0, "50.0"}},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer vmServer.Close()

	cfg := createTestConfig()
	cfg.Inspection.Concurrency = 5
	n9eClient := createN9EClient(n9eServer.URL)
	vmClient := createVMClient(vmServer.URL)
	logger := zerolog.Nop()

	metrics := []*model.MetricDefinition{
		{Name: "slow_metric_1", Query: "slow_metric_1"},
		{Name: "slow_metric_2", Query: "slow_metric_2"},
		{Name: "slow_metric_3", Query: "slow_metric_3"},
	}

	collector := NewCollector(cfg, n9eClient, vmClient, metrics, logger)

	hosts := []*model.HostMeta{
		{Hostname: "host1"},
	}

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := collector.CollectMetrics(ctx, hosts, metrics)
	elapsed := time.Since(start)

	// The request should complete quickly due to context cancellation
	// (not waiting for full 200ms * 3 = 600ms)
	if elapsed > 300*time.Millisecond {
		t.Errorf("Expected fast cancellation, but took %v", elapsed)
	}

	// Error is expected but not returned because we return nil for single metric failures
	// The important thing is that it completes quickly
	_ = err // We don't check the error since individual failures return nil
}
