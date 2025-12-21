package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"inspection-tool/internal/client/vm"
	"inspection-tool/internal/config"
	"inspection-tool/internal/model"
)

// =============================================================================
// Test Helper Functions
// =============================================================================

// setupNginxVMTestServer creates a mock VictoriaMetrics server for Nginx tests.
func setupNginxVMTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

// createTestNginxCollector creates a NginxCollector for testing.
func createTestNginxCollector(vmURL string, cfg *config.NginxInspectionConfig, metrics []*model.NginxMetricDefinition) *NginxCollector {
	vmConfig := &config.VictoriaMetricsConfig{Endpoint: vmURL}
	retryConfig := &config.RetryConfig{
		MaxRetries: 0, // No retry in tests
	}
	vmClient := vm.NewClient(vmConfig, retryConfig, zerolog.Nop())

	return NewNginxCollector(cfg, vmClient, nil, metrics, zerolog.Nop())
}

// createTestNginxMetricDefs creates test Nginx metric definitions.
func createTestNginxMetricDefs() []*model.NginxMetricDefinition {
	return []*model.NginxMetricDefinition{
		{
			Name:        "nginx_up",
			DisplayName: "连接状态",
			Query:       "nginx_up",
			Category:    "connection",
		},
		{
			Name:        "nginx_active",
			DisplayName: "当前活跃连接数",
			Query:       "nginx_active",
			Category:    "connection",
		},
		{
			Name:         "nginx_info",
			DisplayName:  "Nginx 信息",
			Query:        "nginx_info",
			Category:     "info",
			LabelExtract: []string{"port", "app_type", "install_path", "version"},
		},
		{
			Name:        "nginx_worker_processes",
			DisplayName: "工作进程数",
			Query:       "nginx_worker_processes",
			Category:    "config",
		},
		{
			Name:        "nginx_worker_connections",
			DisplayName: "单进程最大连接数",
			Query:       "nginx_worker_connections",
			Category:    "config",
		},
		{
			Name:        "nginx_non_root_user",
			DisplayName: "非 root 用户启动",
			Query:       "nginx_non_root_user",
			Category:    "security",
		},
		{
			Name:         "nginx_last_error_timestamp",
			DisplayName:  "最后错误日志时间",
			Query:        "nginx_last_error_timestamp",
			Category:     "log",
			LabelExtract: []string{"error_log_path"},
			Format:       "timestamp",
		},
	}
}

// writeNginxInfoResponse writes mock nginx_info query response.
func writeNginxInfoResponse(w http.ResponseWriter, hostnames []string) {
	w.Header().Set("Content-Type", "application/json")
	timestamp := time.Now().Unix()

	results := ""
	for i, hostname := range hostnames {
		if i > 0 {
			results += ","
		}
		results += fmt.Sprintf(`{
			"metric": {
				"__name__": "nginx_info",
				"agent_hostname": "%s",
				"port": "80",
				"app_type": "openresty",
				"install_path": "/usr/local/openresty",
				"version": "1.21.4.1"
			},
			"value": [%d, "1"]
		}`, hostname, timestamp)
	}

	jsonResp := fmt.Sprintf(`{
		"status": "success",
		"data": {
			"resultType": "vector",
			"result": [%s]
		}
	}`, results)
	w.Write([]byte(jsonResp))
}

// writeNginxUpResponse writes mock nginx_up query response.
func writeNginxUpResponse(w http.ResponseWriter, hostnames []string, container string) {
	w.Header().Set("Content-Type", "application/json")
	timestamp := time.Now().Unix()

	results := ""
	for i, hostname := range hostnames {
		if i > 0 {
			results += ","
		}
		containerLabel := ""
		if container != "" {
			containerLabel = fmt.Sprintf(`, "container": "%s"`, container)
		}
		results += fmt.Sprintf(`{
			"metric": {
				"__name__": "nginx_up",
				"agent_hostname": "%s"%s
			},
			"value": [%d, "1"]
		}`, hostname, containerLabel, timestamp)
	}

	jsonResp := fmt.Sprintf(`{
		"status": "success",
		"data": {
			"resultType": "vector",
			"result": [%s]
		}
	}`, results)
	w.Write([]byte(jsonResp))
}

// =============================================================================
// DiscoverInstances Tests
// =============================================================================

// TestNginxCollector_DiscoverInstances_Success tests normal instance discovery.
func TestNginxCollector_DiscoverInstances_Success(t *testing.T) {
	server := setupNginxVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")

		if contains(query, "nginx_info") {
			writeNginxInfoResponse(w, []string{
				"GX-NM-MNS-NGX-01",
				"GX-NM-MNS-NGX-02",
			})
		} else if contains(query, "nginx_up") {
			writeNginxUpResponse(w, []string{
				"GX-NM-MNS-NGX-01",
				"GX-NM-MNS-NGX-02",
			}, "")
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))
		}
	})
	defer server.Close()

	cfg := &config.NginxInspectionConfig{
		Enabled: true,
	}
	metrics := createTestNginxMetricDefs()
	collector := createTestNginxCollector(server.URL, cfg, metrics)

	instances, err := collector.DiscoverInstances(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(instances) != 2 {
		t.Errorf("expected 2 instances, got: %d", len(instances))
	}

	// Verify first instance
	if instances[0].Hostname != "GX-NM-MNS-NGX-01" {
		t.Errorf("expected hostname GX-NM-MNS-NGX-01, got: %s", instances[0].Hostname)
	}
	if instances[0].Port != 80 {
		t.Errorf("expected port 80, got: %d", instances[0].Port)
	}
	if instances[0].ApplicationType != "openresty" {
		t.Errorf("expected application type openresty, got: %s", instances[0].ApplicationType)
	}
	if instances[0].Version != "1.21.4.1" {
		t.Errorf("expected version 1.21.4.1, got: %s", instances[0].Version)
	}
}

// TestNginxCollector_DiscoverInstances_WithFilter tests hostname pattern filtering.
func TestNginxCollector_DiscoverInstances_WithFilter(t *testing.T) {
	server := setupNginxVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")

		if contains(query, "nginx_info") {
			writeNginxInfoResponse(w, []string{
				"GX-NM-MNS-NGX-01",
				"GX-NM-MNS-NGX-02",
				"SH-PD-WEB-NGX-01",
				"BJ-DC-API-NGX-01",
			})
		} else if contains(query, "nginx_up") {
			writeNginxUpResponse(w, []string{
				"GX-NM-MNS-NGX-01",
				"GX-NM-MNS-NGX-02",
				"SH-PD-WEB-NGX-01",
				"BJ-DC-API-NGX-01",
			}, "")
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))
		}
	})
	defer server.Close()

	cfg := &config.NginxInspectionConfig{
		Enabled: true,
		InstanceFilter: config.NginxFilter{
			HostnamePatterns: []string{"GX-NM-*"},
		},
	}
	metrics := createTestNginxMetricDefs()
	collector := createTestNginxCollector(server.URL, cfg, metrics)

	instances, err := collector.DiscoverInstances(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Should only return instances matching GX-NM-* pattern
	if len(instances) != 2 {
		t.Errorf("expected 2 instances (filtered), got: %d", len(instances))
	}

	// Verify all instances match the pattern
	for _, inst := range instances {
		if !contains(inst.Hostname, "GX-NM") {
			t.Errorf("instance %s should be filtered out", inst.Hostname)
		}
	}
}

// TestNginxCollector_DiscoverInstances_ContainerDeployment tests container deployment discovery.
func TestNginxCollector_DiscoverInstances_ContainerDeployment(t *testing.T) {
	server := setupNginxVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")

		if contains(query, "nginx_info") {
			writeNginxInfoResponse(w, []string{"GX-NM-MNS-K8S-01"})
		} else if contains(query, "nginx_up") {
			writeNginxUpResponse(w, []string{"GX-NM-MNS-K8S-01"}, "nginx-gateway-abc123")
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))
		}
	})
	defer server.Close()

	cfg := &config.NginxInspectionConfig{
		Enabled: true,
	}
	metrics := createTestNginxMetricDefs()
	collector := createTestNginxCollector(server.URL, cfg, metrics)

	instances, err := collector.DiscoverInstances(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(instances) != 1 {
		t.Errorf("expected 1 instance, got: %d", len(instances))
	}

	// Verify container info
	if instances[0].Container != "nginx-gateway-abc123" {
		t.Errorf("expected container nginx-gateway-abc123, got: %s", instances[0].Container)
	}
	if instances[0].Identifier != "GX-NM-MNS-K8S-01:nginx-gateway-abc123" {
		t.Errorf("expected identifier GX-NM-MNS-K8S-01:nginx-gateway-abc123, got: %s", instances[0].Identifier)
	}
}

// TestNginxCollector_DiscoverInstances_EmptyResults tests empty results scenario.
func TestNginxCollector_DiscoverInstances_EmptyResults(t *testing.T) {
	server := setupNginxVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))
	})
	defer server.Close()

	cfg := &config.NginxInspectionConfig{
		Enabled: true,
	}
	metrics := createTestNginxMetricDefs()
	collector := createTestNginxCollector(server.URL, cfg, metrics)

	instances, err := collector.DiscoverInstances(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(instances) != 0 {
		t.Errorf("expected 0 instances, got: %d", len(instances))
	}
}

// TestNginxCollector_DiscoverInstances_QueryError tests VM query error handling.
func TestNginxCollector_DiscoverInstances_QueryError(t *testing.T) {
	server := setupNginxVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"error": "internal server error"}`))
	})
	defer server.Close()

	cfg := &config.NginxInspectionConfig{
		Enabled: true,
	}
	metrics := createTestNginxMetricDefs()
	collector := createTestNginxCollector(server.URL, cfg, metrics)

	instances, err := collector.DiscoverInstances(context.Background())

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if instances != nil {
		t.Errorf("expected nil instances on error, got: %v", instances)
	}
}

// TestNginxCollector_DiscoverInstances_MissingHostnameLabel tests missing hostname label.
func TestNginxCollector_DiscoverInstances_MissingHostnameLabel(t *testing.T) {
	server := setupNginxVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")

		if contains(query, "nginx_info") {
			w.Header().Set("Content-Type", "application/json")
			timestamp := time.Now().Unix()
			jsonResp := fmt.Sprintf(`{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [
						{"metric": {"__name__": "nginx_info", "port": "80"}, "value": [%d, "1"]},
						{"metric": {"__name__": "nginx_info", "agent_hostname": "GX-NM-NGX-01", "port": "80"}, "value": [%d, "1"]}
					]
				}
			}`, timestamp, timestamp)
			w.Write([]byte(jsonResp))
		} else if contains(query, "nginx_up") {
			writeNginxUpResponse(w, []string{"GX-NM-NGX-01"}, "")
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))
		}
	})
	defer server.Close()

	cfg := &config.NginxInspectionConfig{
		Enabled: true,
	}
	metrics := createTestNginxMetricDefs()
	collector := createTestNginxCollector(server.URL, cfg, metrics)

	instances, err := collector.DiscoverInstances(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Should only return instance with hostname label
	if len(instances) != 1 {
		t.Errorf("expected 1 instance (missing label skipped), got: %d", len(instances))
	}
}

// TestNginxCollector_DiscoverInstances_DuplicateInstances tests instance deduplication.
func TestNginxCollector_DiscoverInstances_DuplicateInstances(t *testing.T) {
	server := setupNginxVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")

		if contains(query, "nginx_info") {
			w.Header().Set("Content-Type", "application/json")
			timestamp := time.Now().Unix()
			// Two duplicate hostnames with same port
			jsonResp := fmt.Sprintf(`{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [
						{"metric": {"__name__": "nginx_info", "agent_hostname": "GX-NM-NGX-01", "port": "80"}, "value": [%d, "1"]},
						{"metric": {"__name__": "nginx_info", "agent_hostname": "GX-NM-NGX-01", "port": "80"}, "value": [%d, "1"]},
						{"metric": {"__name__": "nginx_info", "agent_hostname": "GX-NM-NGX-02", "port": "80"}, "value": [%d, "1"]}
					]
				}
			}`, timestamp, timestamp, timestamp)
			w.Write([]byte(jsonResp))
		} else if contains(query, "nginx_up") {
			writeNginxUpResponse(w, []string{"GX-NM-NGX-01", "GX-NM-NGX-02"}, "")
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))
		}
	})
	defer server.Close()

	cfg := &config.NginxInspectionConfig{
		Enabled: true,
	}
	metrics := createTestNginxMetricDefs()
	collector := createTestNginxCollector(server.URL, cfg, metrics)

	instances, err := collector.DiscoverInstances(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Should deduplicate, only 2 unique instances
	if len(instances) != 2 {
		t.Errorf("expected 2 unique instances, got: %d", len(instances))
	}
}

// =============================================================================
// matchHostnamePattern Tests
// =============================================================================

// TestMatchHostnamePattern tests hostname pattern matching with wildcards.
func TestMatchHostnamePattern(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		pattern  string
		expected bool
	}{
		{"exact match", "GX-NM-MNS-NGX-01", "GX-NM-MNS-NGX-01", true},
		{"wildcard prefix", "GX-NM-MNS-NGX-01", "GX-NM-*", true},
		{"wildcard suffix", "GX-NM-MNS-NGX-01", "*-NGX-01", true},
		{"wildcard middle", "GX-NM-MNS-NGX-01", "GX-*-NGX-01", true},
		{"multiple wildcards", "GX-NM-MNS-NGX-01", "GX-*-*-01", true},
		{"wildcard all", "GX-NM-MNS-NGX-01", "*", true},
		{"no match", "GX-NM-MNS-NGX-01", "SH-*", false},
		{"no wildcard no match", "GX-NM-MNS-NGX-01", "GX-NM-MNS-NGX-02", false},
		{"partial no match", "GX-NM-MNS-NGX-01", "GX-NM-MNS-NGX", false},
		{"empty pattern", "", "", true},
		{"empty hostname", "", "pattern", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchHostnamePattern(tt.hostname, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchHostnamePattern(%q, %q) = %v, want %v",
					tt.hostname, tt.pattern, result, tt.expected)
			}
		})
	}
}

// TestMatchHostnamePattern_EdgeCases tests edge cases for hostname pattern matching.
func TestMatchHostnamePattern_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		pattern  string
		expected bool
	}{
		{"empty both", "", "", true},
		{"empty hostname", "", "pattern", false},
		{"special regex chars preserved", "host.name", "host.name", true},
		{"wildcard only", "anything", "*", true},
		{"trailing wildcard", "GX-NM-MNS-NGX-01", "GX-NM-MNS-NGX-*", true},
		{"leading wildcard", "GX-NM-MNS-NGX-01", "*-01", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchHostnamePattern(tt.hostname, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchHostnamePattern(%q, %q) = %v, want %v",
					tt.hostname, tt.pattern, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// CollectMetrics Tests
// =============================================================================

// TestNginxCollector_CollectMetrics_Success tests normal metric collection.
func TestNginxCollector_CollectMetrics_Success(t *testing.T) {
	server := setupNginxVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")

		w.Header().Set("Content-Type", "application/json")
		timestamp := time.Now().Unix()

		var jsonResp string
		if contains(query, "nginx_up") {
			jsonResp = fmt.Sprintf(`{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [
						{"metric": {"agent_hostname": "GX-NM-NGX-01"}, "value": [%d, "1"]}
					]
				}
			}`, timestamp)
		} else if contains(query, "nginx_active") {
			jsonResp = fmt.Sprintf(`{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [
						{"metric": {"agent_hostname": "GX-NM-NGX-01"}, "value": [%d, "100"]}
					]
				}
			}`, timestamp)
		} else if contains(query, "nginx_worker_processes") {
			jsonResp = fmt.Sprintf(`{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [
						{"metric": {"agent_hostname": "GX-NM-NGX-01"}, "value": [%d, "4"]}
					]
				}
			}`, timestamp)
		} else if contains(query, "nginx_worker_connections") {
			jsonResp = fmt.Sprintf(`{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [
						{"metric": {"agent_hostname": "GX-NM-NGX-01"}, "value": [%d, "1024"]}
					]
				}
			}`, timestamp)
		} else if contains(query, "nginx_non_root_user") {
			jsonResp = fmt.Sprintf(`{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [
						{"metric": {"agent_hostname": "GX-NM-NGX-01"}, "value": [%d, "1"]}
					]
				}
			}`, timestamp)
		} else {
			jsonResp = `{"status": "success", "data": {"resultType": "vector", "result": []}}`
		}

		w.Write([]byte(jsonResp))
	})
	defer server.Close()

	cfg := &config.NginxInspectionConfig{
		Enabled: true,
	}
	metrics := []*model.NginxMetricDefinition{
		{Name: "nginx_up", DisplayName: "连接状态", Query: "nginx_up", Category: "connection"},
		{Name: "nginx_active", DisplayName: "活跃连接数", Query: "nginx_active", Category: "connection"},
		{Name: "nginx_worker_processes", DisplayName: "工作进程数", Query: "nginx_worker_processes", Category: "config"},
		{Name: "nginx_worker_connections", DisplayName: "单进程最大连接数", Query: "nginx_worker_connections", Category: "config"},
		{Name: "nginx_non_root_user", DisplayName: "非 root 用户", Query: "nginx_non_root_user", Category: "security"},
	}
	collector := createTestNginxCollector(server.URL, cfg, metrics)

	instances := []*model.NginxInstance{
		model.NewNginxInstance("GX-NM-NGX-01", 80),
	}

	ctx := context.Background()
	results, err := collector.CollectMetrics(ctx, instances, metrics)

	if err != nil {
		t.Fatalf("CollectMetrics failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	result, ok := results["GX-NM-NGX-01:80"]
	if !ok {
		t.Fatal("result for GX-NM-NGX-01:80 not found")
	}

	// Verify metrics extracted to fields
	if !result.Up {
		t.Error("expected Up to be true")
	}
	if result.ActiveConnections != 100 {
		t.Errorf("expected ActiveConnections 100, got %d", result.ActiveConnections)
	}
	if result.WorkerProcesses != 4 {
		t.Errorf("expected WorkerProcesses 4, got %d", result.WorkerProcesses)
	}
	if result.WorkerConnections != 1024 {
		t.Errorf("expected WorkerConnections 1024, got %d", result.WorkerConnections)
	}
	if !result.NonRootUser {
		t.Error("expected NonRootUser to be true")
	}
}

// TestNginxCollector_CollectMetrics_PendingMetrics tests pending metrics are set to N/A.
func TestNginxCollector_CollectMetrics_PendingMetrics(t *testing.T) {
	server := setupNginxVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))
	})
	defer server.Close()

	cfg := &config.NginxInspectionConfig{
		Enabled: true,
	}

	// Create pending metrics
	metrics := []*model.NginxMetricDefinition{
		{
			Name:        "nginx_pending_metric",
			DisplayName: "待实现指标",
			Query:       "",
			Status:      "pending",
		},
	}
	collector := createTestNginxCollector(server.URL, cfg, metrics)

	instances := []*model.NginxInstance{
		model.NewNginxInstance("GX-NM-NGX-01", 80),
	}

	ctx := context.Background()
	results, err := collector.CollectMetrics(ctx, instances, metrics)

	if err != nil {
		t.Fatalf("CollectMetrics failed: %v", err)
	}

	result := results["GX-NM-NGX-01:80"]
	if result == nil {
		t.Fatal("result not found")
	}

	// Check pending metric is set to N/A
	mv := result.GetMetric("nginx_pending_metric")
	if mv == nil {
		t.Error("pending metric not found")
	} else {
		if !mv.IsNA {
			t.Error("pending metric IsNA = false, want true")
		}
		if mv.FormattedValue != "N/A" {
			t.Errorf("pending metric FormattedValue = %q, want N/A", mv.FormattedValue)
		}
	}
}

// TestNginxCollector_CollectMetrics_LabelExtract tests label extraction.
func TestNginxCollector_CollectMetrics_LabelExtract(t *testing.T) {
	server := setupNginxVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")

		w.Header().Set("Content-Type", "application/json")
		timestamp := time.Now().Unix()

		var jsonResp string
		if contains(query, "nginx_last_error_timestamp") {
			jsonResp = fmt.Sprintf(`{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [
						{
							"metric": {
								"__name__": "nginx_last_error_timestamp",
								"agent_hostname": "GX-NM-NGX-01",
								"error_log_path": "/var/log/nginx/error.log"
							},
							"value": [%d, "%d"]
						}
					]
				}
			}`, timestamp, timestamp-300) // 5 minutes ago
		} else {
			jsonResp = `{"status": "success", "data": {"resultType": "vector", "result": []}}`
		}

		w.Write([]byte(jsonResp))
	})
	defer server.Close()

	cfg := &config.NginxInspectionConfig{
		Enabled: true,
	}
	metrics := []*model.NginxMetricDefinition{
		{
			Name:         "nginx_last_error_timestamp",
			DisplayName:  "最后错误日志时间",
			Query:        "nginx_last_error_timestamp",
			Category:     "log",
			LabelExtract: []string{"error_log_path"},
			Format:       "timestamp",
		},
	}
	collector := createTestNginxCollector(server.URL, cfg, metrics)

	instances := []*model.NginxInstance{
		model.NewNginxInstance("GX-NM-NGX-01", 80),
	}

	ctx := context.Background()
	results, err := collector.CollectMetrics(ctx, instances, metrics)

	if err != nil {
		t.Fatalf("CollectMetrics failed: %v", err)
	}

	result := results["GX-NM-NGX-01:80"]
	if result == nil {
		t.Fatal("result not found")
	}

	// Verify label extraction
	mv := result.GetMetric("nginx_last_error_timestamp")
	if mv == nil {
		t.Fatal("nginx_last_error_timestamp metric not found")
	}

	if mv.StringValue != "/var/log/nginx/error.log" {
		t.Errorf("StringValue = %q, want \"/var/log/nginx/error.log\"", mv.StringValue)
	}

	// Verify error_log_path is set on instance
	if result.Instance.ErrorLogPath != "/var/log/nginx/error.log" {
		t.Errorf("Instance.ErrorLogPath = %q, want \"/var/log/nginx/error.log\"", result.Instance.ErrorLogPath)
	}
}

// =============================================================================
// CollectUpstreamStatus Tests
// =============================================================================

// TestNginxCollector_CollectUpstreamStatus_Success tests upstream status collection.
func TestNginxCollector_CollectUpstreamStatus_Success(t *testing.T) {
	server := setupNginxVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")

		w.Header().Set("Content-Type", "application/json")
		timestamp := time.Now().Unix()

		var jsonResp string
		if contains(query, "nginx_upstream_check_status_code") {
			jsonResp = fmt.Sprintf(`{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [
						{
							"metric": {
								"agent_hostname": "GX-NM-NGX-01",
								"upstream": "backend",
								"name": "172.18.182.91:8080"
							},
							"value": [%d, "1"]
						},
						{
							"metric": {
								"agent_hostname": "GX-NM-NGX-01",
								"upstream": "backend",
								"name": "172.18.182.92:8080"
							},
							"value": [%d, "0"]
						}
					]
				}
			}`, timestamp, timestamp)
		} else if contains(query, "nginx_upstream_check_rise") {
			jsonResp = fmt.Sprintf(`{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [
						{
							"metric": {
								"agent_hostname": "GX-NM-NGX-01",
								"upstream": "backend",
								"name": "172.18.182.91:8080"
							},
							"value": [%d, "5"]
						}
					]
				}
			}`, timestamp)
		} else if contains(query, "nginx_upstream_check_fall") {
			jsonResp = fmt.Sprintf(`{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [
						{
							"metric": {
								"agent_hostname": "GX-NM-NGX-01",
								"upstream": "backend",
								"name": "172.18.182.92:8080"
							},
							"value": [%d, "3"]
						}
					]
				}
			}`, timestamp)
		} else {
			jsonResp = `{"status": "success", "data": {"resultType": "vector", "result": []}}`
		}

		w.Write([]byte(jsonResp))
	})
	defer server.Close()

	cfg := &config.NginxInspectionConfig{
		Enabled: true,
	}
	metrics := createTestNginxMetricDefs()
	collector := createTestNginxCollector(server.URL, cfg, metrics)

	// Create results map with instance
	instance := model.NewNginxInstance("GX-NM-NGX-01", 80)
	resultsMap := map[string]*model.NginxInspectionResult{
		"GX-NM-NGX-01:80": model.NewNginxInspectionResult(instance),
	}

	ctx := context.Background()
	err := collector.CollectUpstreamStatus(ctx, resultsMap)

	if err != nil {
		t.Fatalf("CollectUpstreamStatus failed: %v", err)
	}

	result := resultsMap["GX-NM-NGX-01:80"]
	if len(result.UpstreamStatus) != 2 {
		t.Fatalf("expected 2 upstream statuses, got %d", len(result.UpstreamStatus))
	}

	// Verify first upstream (healthy)
	healthyFound := false
	unhealthyFound := false
	for _, us := range result.UpstreamStatus {
		if us.BackendAddress == "172.18.182.91:8080" {
			healthyFound = true
			if !us.Status {
				t.Error("expected upstream 172.18.182.91:8080 to be healthy")
			}
			if us.RiseCount != 5 {
				t.Errorf("expected rise count 5, got %d", us.RiseCount)
			}
		}
		if us.BackendAddress == "172.18.182.92:8080" {
			unhealthyFound = true
			if us.Status {
				t.Error("expected upstream 172.18.182.92:8080 to be unhealthy")
			}
			if us.FallCount != 3 {
				t.Errorf("expected fall count 3, got %d", us.FallCount)
			}
		}
	}

	if !healthyFound {
		t.Error("healthy upstream not found")
	}
	if !unhealthyFound {
		t.Error("unhealthy upstream not found")
	}
}

// TestNginxCollector_CollectUpstreamStatus_NoUpstream tests empty upstream scenario.
func TestNginxCollector_CollectUpstreamStatus_NoUpstream(t *testing.T) {
	server := setupNginxVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))
	})
	defer server.Close()

	cfg := &config.NginxInspectionConfig{
		Enabled: true,
	}
	metrics := createTestNginxMetricDefs()
	collector := createTestNginxCollector(server.URL, cfg, metrics)

	// Create results map with instance
	instance := model.NewNginxInstance("GX-NM-NGX-01", 80)
	resultsMap := map[string]*model.NginxInspectionResult{
		"GX-NM-NGX-01:80": model.NewNginxInspectionResult(instance),
	}

	ctx := context.Background()
	err := collector.CollectUpstreamStatus(ctx, resultsMap)

	if err != nil {
		t.Fatalf("CollectUpstreamStatus failed: %v", err)
	}

	result := resultsMap["GX-NM-NGX-01:80"]
	if len(result.UpstreamStatus) != 0 {
		t.Errorf("expected 0 upstream statuses, got %d", len(result.UpstreamStatus))
	}
}

// =============================================================================
// NginxInstanceFilter Tests
// =============================================================================

// TestNginxInstanceFilter_IsEmpty tests IsEmpty method.
func TestNginxInstanceFilter_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		filter   *NginxInstanceFilter
		expected bool
	}{
		{
			name:     "nil filter",
			filter:   nil,
			expected: true,
		},
		{
			name:     "empty filter",
			filter:   &NginxInstanceFilter{},
			expected: true,
		},
		{
			name: "with hostname patterns",
			filter: &NginxInstanceFilter{
				HostnamePatterns: []string{"GX-*"},
			},
			expected: false,
		},
		{
			name: "with business groups",
			filter: &NginxInstanceFilter{
				BusinessGroups: []string{"production"},
			},
			expected: false,
		},
		{
			name: "with tags",
			filter: &NginxInstanceFilter{
				Tags: map[string]string{"env": "prod"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.IsEmpty()
			if result != tt.expected {
				t.Errorf("IsEmpty() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestNginxInstanceFilter_ToVMHostFilter tests ToVMHostFilter conversion.
func TestNginxInstanceFilter_ToVMHostFilter(t *testing.T) {
	tests := []struct {
		name           string
		filter         *NginxInstanceFilter
		expectNil      bool
		expectBizGroup []string
	}{
		{
			name:      "nil filter",
			filter:    nil,
			expectNil: true,
		},
		{
			name:      "empty filter",
			filter:    &NginxInstanceFilter{},
			expectNil: true,
		},
		{
			name: "only hostname patterns - not included in VM filter",
			filter: &NginxInstanceFilter{
				HostnamePatterns: []string{"GX-*"},
			},
			expectNil: true,
		},
		{
			name: "with business groups",
			filter: &NginxInstanceFilter{
				BusinessGroups: []string{"production"},
			},
			expectNil:      false,
			expectBizGroup: []string{"production"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.ToVMHostFilter()
			if tt.expectNil {
				if result != nil {
					t.Error("expected nil VM filter")
				}
			} else {
				if result == nil {
					t.Fatal("expected non-nil VM filter")
				}
				if len(result.BusinessGroups) != len(tt.expectBizGroup) {
					t.Errorf("expected %d business groups, got %d",
						len(tt.expectBizGroup), len(result.BusinessGroups))
				}
			}
		})
	}
}
