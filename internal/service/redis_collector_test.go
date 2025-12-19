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

// setupRedisVMTestServer creates a mock VictoriaMetrics server for testing.
func setupRedisVMTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

// createTestRedisCollector creates a RedisCollector for testing.
func createTestRedisCollector(vmURL string, cfg *config.RedisInspectionConfig) *RedisCollector {
	vmConfig := &config.VictoriaMetricsConfig{Endpoint: vmURL}
	retryConfig := &config.RetryConfig{
		MaxRetries: 0, // No retry in tests
	}
	vmClient := vm.NewClient(vmConfig, retryConfig, zerolog.Nop())

	return NewRedisCollector(cfg, vmClient, nil, zerolog.Nop())
}

// writeRedisVMJSONResponse writes a VictoriaMetrics JSON response for Redis instances.
func writeRedisVMJSONResponse(w http.ResponseWriter, instances []struct {
	Address string
	Role    string
}) {
	w.Header().Set("Content-Type", "application/json")
	timestamp := time.Now().Unix()

	results := ""
	for i, inst := range instances {
		if i > 0 {
			results += ","
		}
		results += fmt.Sprintf(`{"metric": {"__name__": "redis_up", "address": "%s", "replica_role": "%s"}, "value": [%d, "1"]}`,
			inst.Address, inst.Role, timestamp)
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

// TestRedisDiscoverInstances_Success tests successful discovery of multiple Redis instances.
func TestRedisDiscoverInstances_Success(t *testing.T) {
	server := setupRedisVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Verify request path
		if r.URL.Path != "/api/v1/query" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Verify query parameter
		query := r.URL.Query().Get("query")
		if query != "redis_up" {
			t.Errorf("expected redis_up query, got: %s", query)
		}

		// Return 6 Redis instances (3 master + 3 slave)
		writeRedisVMJSONResponse(w, []struct {
			Address string
			Role    string
		}{
			{"192.18.102.2:7000", "slave"},
			{"192.18.102.2:7001", "slave"},
			{"192.18.102.3:7000", "slave"},
			{"192.18.102.3:7001", "master"},
			{"192.18.102.4:7000", "master"},
			{"192.18.102.4:7001", "master"},
		})
	})
	defer server.Close()

	cfg := &config.RedisInspectionConfig{
		Enabled:     true,
		ClusterMode: "3m3s",
	}
	collector := createTestRedisCollector(server.URL, cfg)

	// Execute discovery
	instances, err := collector.DiscoverInstances(context.Background())

	// Verify results
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(instances) != 6 {
		t.Errorf("expected 6 instances, got: %d", len(instances))
	}

	// Count master and slave roles
	masterCount := 0
	slaveCount := 0
	for _, inst := range instances {
		if inst.Role == model.RedisRoleMaster {
			masterCount++
		} else if inst.Role == model.RedisRoleSlave {
			slaveCount++
		}
	}

	if masterCount != 3 {
		t.Errorf("expected 3 masters, got: %d", masterCount)
	}
	if slaveCount != 3 {
		t.Errorf("expected 3 slaves, got: %d", slaveCount)
	}
}

// TestRedisDiscoverInstances_RoleExtraction tests correct role extraction from replica_role label.
func TestRedisDiscoverInstances_RoleExtraction(t *testing.T) {
	server := setupRedisVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeRedisVMJSONResponse(w, []struct {
			Address string
			Role    string
		}{
			{"192.18.102.2:7000", "master"},
			{"192.18.102.2:7001", "slave"},
		})
	})
	defer server.Close()

	cfg := &config.RedisInspectionConfig{
		Enabled:     true,
		ClusterMode: "3m3s",
	}
	collector := createTestRedisCollector(server.URL, cfg)

	instances, err := collector.DiscoverInstances(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Find instances by address and verify roles
	for _, inst := range instances {
		switch inst.Address {
		case "192.18.102.2:7000":
			if inst.Role != model.RedisRoleMaster {
				t.Errorf("expected master for %s, got: %s", inst.Address, inst.Role)
			}
		case "192.18.102.2:7001":
			if inst.Role != model.RedisRoleSlave {
				t.Errorf("expected slave for %s, got: %s", inst.Address, inst.Role)
			}
		}
	}
}

// TestRedisDiscoverInstances_AddressParsing tests correct IP and Port parsing.
func TestRedisDiscoverInstances_AddressParsing(t *testing.T) {
	server := setupRedisVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeRedisVMJSONResponse(w, []struct {
			Address string
			Role    string
		}{
			{"192.18.102.2:7000", "master"},
			{"10.0.0.100:6379", "slave"},
		})
	})
	defer server.Close()

	cfg := &config.RedisInspectionConfig{
		Enabled: true,
	}
	collector := createTestRedisCollector(server.URL, cfg)

	instances, err := collector.DiscoverInstances(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify parsing for each instance
	for _, inst := range instances {
		switch inst.Address {
		case "192.18.102.2:7000":
			if inst.IP != "192.18.102.2" {
				t.Errorf("expected IP 192.18.102.2, got: %s", inst.IP)
			}
			if inst.Port != 7000 {
				t.Errorf("expected port 7000, got: %d", inst.Port)
			}
		case "10.0.0.100:6379":
			if inst.IP != "10.0.0.100" {
				t.Errorf("expected IP 10.0.0.100, got: %s", inst.IP)
			}
			if inst.Port != 6379 {
				t.Errorf("expected port 6379, got: %d", inst.Port)
			}
		}

		// Verify ApplicationType is always "Redis"
		if inst.ApplicationType != "Redis" {
			t.Errorf("expected ApplicationType 'Redis', got: %s", inst.ApplicationType)
		}
	}
}

// TestRedisDiscoverInstances_WithAddressPatternFilter tests address pattern filtering.
func TestRedisDiscoverInstances_WithAddressPatternFilter(t *testing.T) {
	server := setupRedisVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeRedisVMJSONResponse(w, []struct {
			Address string
			Role    string
		}{
			{"192.18.102.2:7000", "master"},
			{"192.18.102.3:7000", "slave"},
			{"10.0.0.100:6379", "master"},
			{"172.16.0.50:6379", "slave"},
		})
	})
	defer server.Close()

	cfg := &config.RedisInspectionConfig{
		Enabled:     true,
		ClusterMode: "3m3s",
		InstanceFilter: config.RedisFilter{
			AddressPatterns: []string{"192.18.102.*"},
		},
	}
	collector := createTestRedisCollector(server.URL, cfg)

	instances, err := collector.DiscoverInstances(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Should only return instances matching 192.18.102.*
	if len(instances) != 2 {
		t.Errorf("expected 2 instances (filtered), got: %d", len(instances))
	}

	// Verify all instances match the filter
	for _, inst := range instances {
		if inst.IP != "192.18.102.2" && inst.IP != "192.18.102.3" {
			t.Errorf("instance %s should be filtered out", inst.Address)
		}
	}
}

// TestRedisDiscoverInstances_EmptyResults tests handling of empty results.
func TestRedisDiscoverInstances_EmptyResults(t *testing.T) {
	server := setupRedisVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		jsonResp := `{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": []
			}
		}`
		w.Write([]byte(jsonResp))
	})
	defer server.Close()

	cfg := &config.RedisInspectionConfig{
		Enabled: true,
	}
	collector := createTestRedisCollector(server.URL, cfg)

	instances, err := collector.DiscoverInstances(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(instances) != 0 {
		t.Errorf("expected 0 instances, got: %d", len(instances))
	}
}

// TestRedisDiscoverInstances_QueryError tests handling of query errors.
func TestRedisDiscoverInstances_QueryError(t *testing.T) {
	server := setupRedisVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		jsonResp := `{
			"status": "error",
			"errorType": "internal",
			"error": "query execution failed"
		}`
		w.Write([]byte(jsonResp))
	})
	defer server.Close()

	cfg := &config.RedisInspectionConfig{
		Enabled: true,
	}
	collector := createTestRedisCollector(server.URL, cfg)

	_, err := collector.DiscoverInstances(context.Background())

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestRedisDiscoverInstances_DuplicateAddresses tests deduplication of addresses.
func TestRedisDiscoverInstances_DuplicateAddresses(t *testing.T) {
	server := setupRedisVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Return duplicate addresses
		writeRedisVMJSONResponse(w, []struct {
			Address string
			Role    string
		}{
			{"192.18.102.2:7000", "master"},
			{"192.18.102.2:7000", "master"}, // Duplicate
			{"192.18.102.3:7000", "slave"},
			{"192.18.102.3:7000", "slave"}, // Duplicate
		})
	})
	defer server.Close()

	cfg := &config.RedisInspectionConfig{
		Enabled: true,
	}
	collector := createTestRedisCollector(server.URL, cfg)

	instances, err := collector.DiscoverInstances(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Should only return 2 unique instances
	if len(instances) != 2 {
		t.Errorf("expected 2 instances (deduplicated), got: %d", len(instances))
	}
}

// TestRedisDiscoverInstances_MissingAddressLabel tests handling of missing address label.
func TestRedisDiscoverInstances_MissingAddressLabel(t *testing.T) {
	server := setupRedisVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		timestamp := time.Now().Unix()
		// Some results missing address label
		jsonResp := fmt.Sprintf(`{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": [
					{"metric": {"__name__": "redis_up", "address": "192.18.102.2:7000", "replica_role": "master"}, "value": [%d, "1"]},
					{"metric": {"__name__": "redis_up", "replica_role": "slave"}, "value": [%d, "1"]},
					{"metric": {"__name__": "redis_up", "address": "192.18.102.3:7000", "replica_role": "slave"}, "value": [%d, "1"]}
				]
			}
		}`, timestamp, timestamp, timestamp)
		w.Write([]byte(jsonResp))
	})
	defer server.Close()

	cfg := &config.RedisInspectionConfig{
		Enabled: true,
	}
	collector := createTestRedisCollector(server.URL, cfg)

	instances, err := collector.DiscoverInstances(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Should only return 2 instances (one missing address is skipped)
	if len(instances) != 2 {
		t.Errorf("expected 2 instances (skipped one without address), got: %d", len(instances))
	}
}

// TestRedisDiscoverInstances_UnknownRole tests handling of unknown role values.
func TestRedisDiscoverInstances_UnknownRole(t *testing.T) {
	server := setupRedisVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		timestamp := time.Now().Unix()
		jsonResp := fmt.Sprintf(`{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": [
					{"metric": {"__name__": "redis_up", "address": "192.18.102.2:7000", "replica_role": "unknown_role"}, "value": [%d, "1"]},
					{"metric": {"__name__": "redis_up", "address": "192.18.102.3:7000"}, "value": [%d, "1"]}
				]
			}
		}`, timestamp, timestamp)
		w.Write([]byte(jsonResp))
	})
	defer server.Close()

	cfg := &config.RedisInspectionConfig{
		Enabled: true,
	}
	collector := createTestRedisCollector(server.URL, cfg)

	instances, err := collector.DiscoverInstances(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Both instances should have unknown role
	for _, inst := range instances {
		if inst.Role != model.RedisRoleUnknown {
			t.Errorf("expected unknown role for %s, got: %s", inst.Address, inst.Role)
		}
	}
}

// TestRedisInstanceFilter_IsEmpty tests IsEmpty method.
func TestRedisInstanceFilter_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		filter   *RedisInstanceFilter
		expected bool
	}{
		{
			name:     "nil filter",
			filter:   nil,
			expected: true,
		},
		{
			name:     "empty filter",
			filter:   &RedisInstanceFilter{},
			expected: true,
		},
		{
			name: "with address patterns",
			filter: &RedisInstanceFilter{
				AddressPatterns: []string{"192.18.102.*"},
			},
			expected: false,
		},
		{
			name: "with business groups",
			filter: &RedisInstanceFilter{
				BusinessGroups: []string{"prod"},
			},
			expected: false,
		},
		{
			name: "with tags",
			filter: &RedisInstanceFilter{
				Tags: map[string]string{"env": "prod"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.IsEmpty()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestRedisInstanceFilter_ToVMHostFilter tests ToVMHostFilter method.
func TestRedisInstanceFilter_ToVMHostFilter(t *testing.T) {
	tests := []struct {
		name           string
		filter         *RedisInstanceFilter
		expectNil      bool
		expectBusiGrps []string
		expectTags     map[string]string
	}{
		{
			name:      "nil filter",
			filter:    nil,
			expectNil: true,
		},
		{
			name:      "empty filter",
			filter:    &RedisInstanceFilter{},
			expectNil: true,
		},
		{
			name: "only address patterns (should return nil)",
			filter: &RedisInstanceFilter{
				AddressPatterns: []string{"192.18.102.*"},
			},
			expectNil: true,
		},
		{
			name: "with business groups",
			filter: &RedisInstanceFilter{
				BusinessGroups: []string{"prod", "test"},
			},
			expectNil:      false,
			expectBusiGrps: []string{"prod", "test"},
		},
		{
			name: "with tags",
			filter: &RedisInstanceFilter{
				Tags: map[string]string{"env": "prod"},
			},
			expectNil:  false,
			expectTags: map[string]string{"env": "prod"},
		},
		{
			name: "with both business groups and tags",
			filter: &RedisInstanceFilter{
				BusinessGroups: []string{"prod"},
				Tags:           map[string]string{"region": "cn"},
			},
			expectNil:      false,
			expectBusiGrps: []string{"prod"},
			expectTags:     map[string]string{"region": "cn"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.ToVMHostFilter()

			if tt.expectNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if len(tt.expectBusiGrps) > 0 {
				if len(result.BusinessGroups) != len(tt.expectBusiGrps) {
					t.Errorf("expected %d business groups, got %d", len(tt.expectBusiGrps), len(result.BusinessGroups))
				}
			}

			if len(tt.expectTags) > 0 {
				if len(result.Tags) != len(tt.expectTags) {
					t.Errorf("expected %d tags, got %d", len(tt.expectTags), len(result.Tags))
				}
			}
		})
	}
}

// TestRedisCollector_extractRole tests extractRole method.
func TestRedisCollector_extractRole(t *testing.T) {
	collector := &RedisCollector{
		logger: zerolog.Nop(),
	}

	tests := []struct {
		name     string
		labels   map[string]string
		expected model.RedisRole
	}{
		{
			name:     "master role",
			labels:   map[string]string{"replica_role": "master"},
			expected: model.RedisRoleMaster,
		},
		{
			name:     "slave role",
			labels:   map[string]string{"replica_role": "slave"},
			expected: model.RedisRoleSlave,
		},
		{
			name:     "unknown role value",
			labels:   map[string]string{"replica_role": "sentinel"},
			expected: model.RedisRoleUnknown,
		},
		{
			name:     "missing role label",
			labels:   map[string]string{},
			expected: model.RedisRoleUnknown,
		},
		{
			name:     "empty role value",
			labels:   map[string]string{"replica_role": ""},
			expected: model.RedisRoleUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.extractRole(tt.labels)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestRedisCollector_extractAddress tests extractAddress method.
func TestRedisCollector_extractAddress(t *testing.T) {
	collector := &RedisCollector{
		logger: zerolog.Nop(),
	}

	tests := []struct {
		name     string
		labels   map[string]string
		expected string
	}{
		{
			name:     "address label",
			labels:   map[string]string{"address": "192.18.102.2:7000"},
			expected: "192.18.102.2:7000",
		},
		{
			name:     "instance label fallback",
			labels:   map[string]string{"instance": "192.18.102.2:7000"},
			expected: "192.18.102.2:7000",
		},
		{
			name:     "server label fallback",
			labels:   map[string]string{"server": "192.18.102.2:7000"},
			expected: "192.18.102.2:7000",
		},
		{
			name:     "address takes priority",
			labels:   map[string]string{"address": "192.18.102.2:7000", "instance": "other:6379"},
			expected: "192.18.102.2:7000",
		},
		{
			name:     "no address labels",
			labels:   map[string]string{"other": "value"},
			expected: "",
		},
		{
			name:     "empty labels",
			labels:   map[string]string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.extractAddress(tt.labels)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestRedisCollector_matchesAddressPatterns tests matchesAddressPatterns method.
func TestRedisCollector_matchesAddressPatterns(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		address  string
		expected bool
	}{
		{
			name:     "no patterns (all match)",
			patterns: nil,
			address:  "192.18.102.2:7000",
			expected: true,
		},
		{
			name:     "empty patterns (all match)",
			patterns: []string{},
			address:  "192.18.102.2:7000",
			expected: true,
		},
		{
			name:     "exact match",
			patterns: []string{"192.18.102.2:7000"},
			address:  "192.18.102.2:7000",
			expected: true,
		},
		{
			name:     "wildcard IP match",
			patterns: []string{"192.18.102.*"},
			address:  "192.18.102.2:7000",
			expected: true,
		},
		{
			name:     "no match",
			patterns: []string{"10.0.0.*"},
			address:  "192.18.102.2:7000",
			expected: false,
		},
		{
			name:     "multiple patterns (OR)",
			patterns: []string{"10.0.0.*", "192.18.102.*"},
			address:  "192.18.102.2:7000",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filter *RedisInstanceFilter
			if len(tt.patterns) > 0 {
				filter = &RedisInstanceFilter{
					AddressPatterns: tt.patterns,
				}
			}

			collector := &RedisCollector{
				instanceFilter: filter,
				logger:         zerolog.Nop(),
			}

			result := collector.matchesAddressPatterns(tt.address)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestRedisCollector_Getters tests the getter methods of RedisCollector.
func TestRedisCollector_Getters(t *testing.T) {
	// Setup mock server
	server := setupRedisVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
	})
	defer server.Close()

	// Create test configuration
	cfg := &config.RedisInspectionConfig{
		Enabled:     true,
		ClusterMode: "3m3s",
		InstanceFilter: config.RedisFilter{
			AddressPatterns: []string{"192.18.102.*"},
			BusinessGroups:  []string{"prod-redis"},
			Tags:            map[string]string{"env": "production"},
		},
	}

	// Create test metrics
	metrics := []*model.RedisMetricDefinition{
		{Name: "redis_up", DisplayName: "Connection Status", Query: "redis_up"},
		{Name: "redis_maxclients", DisplayName: "Max Clients", Query: "redis_maxclients"},
	}

	// Create collector
	vmConfig := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	retryConfig := &config.RetryConfig{MaxRetries: 0}
	vmClient := vm.NewClient(vmConfig, retryConfig, zerolog.Nop())
	collector := NewRedisCollector(cfg, vmClient, metrics, zerolog.Nop())

	t.Run("GetConfig returns correct config", func(t *testing.T) {
		result := collector.GetConfig()
		if result == nil {
			t.Fatal("GetConfig returned nil")
		}
		if result.Enabled != cfg.Enabled {
			t.Errorf("expected Enabled=%v, got %v", cfg.Enabled, result.Enabled)
		}
		if result.ClusterMode != cfg.ClusterMode {
			t.Errorf("expected ClusterMode=%s, got %s", cfg.ClusterMode, result.ClusterMode)
		}
	})

	t.Run("GetMetrics returns correct metrics", func(t *testing.T) {
		result := collector.GetMetrics()
		if result == nil {
			t.Fatal("GetMetrics returned nil")
		}
		if len(result) != 2 {
			t.Errorf("expected 2 metrics, got %d", len(result))
		}
		if result[0].Name != "redis_up" {
			t.Errorf("expected first metric name redis_up, got %s", result[0].Name)
		}
	})

	t.Run("GetInstanceFilter returns correct filter", func(t *testing.T) {
		result := collector.GetInstanceFilter()
		if result == nil {
			t.Fatal("GetInstanceFilter returned nil")
		}
		if len(result.AddressPatterns) != 1 {
			t.Errorf("expected 1 address pattern, got %d", len(result.AddressPatterns))
		}
		if result.AddressPatterns[0] != "192.18.102.*" {
			t.Errorf("expected pattern 192.18.102.*, got %s", result.AddressPatterns[0])
		}
	})
}

// =============================================================================
// CollectMetrics 相关测试
// =============================================================================

// writeRedisMetricResponse writes a VictoriaMetrics JSON response for a specific Redis metric.
func writeRedisMetricResponse(w http.ResponseWriter, metricName string, instances []struct {
	Address string
	Value   float64
	Labels  map[string]string
}) {
	w.Header().Set("Content-Type", "application/json")
	timestamp := time.Now().Unix()

	results := ""
	for i, inst := range instances {
		if i > 0 {
			results += ","
		}
		// Build labels JSON
		labels := fmt.Sprintf(`"__name__": "%s", "address": "%s"`, metricName, inst.Address)
		for k, v := range inst.Labels {
			labels += fmt.Sprintf(`, "%s": "%s"`, k, v)
		}
		results += fmt.Sprintf(`{"metric": {%s}, "value": [%d, "%g"]}`,
			labels, timestamp, inst.Value)
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

// createRedisMetricsForTest creates test metric definitions for Redis.
func createRedisMetricsForTest() []*model.RedisMetricDefinition {
	return []*model.RedisMetricDefinition{
		{Name: "redis_up", Query: "redis_up", Category: "connection"},
		{Name: "redis_cluster_enabled", Query: "redis_cluster_enabled", Category: "cluster"},
		{Name: "redis_master_link_status", Query: "redis_master_link_status", Category: "replication"},
		{Name: "redis_connected_clients", Query: "redis_connected_clients", Category: "connection"},
		{Name: "redis_maxclients", Query: "redis_maxclients", Category: "connection"},
		{Name: "redis_master_repl_offset", Query: "redis_master_repl_offset", Category: "replication"},
		{Name: "redis_slave_repl_offset", Query: "redis_slave_repl_offset", Category: "replication"},
		{Name: "redis_uptime_in_seconds", Query: "redis_uptime_in_seconds", Category: "status"},
		{Name: "redis_master_port", Query: "redis_master_port", Category: "replication"},
		{Name: "redis_connected_slaves", Query: "redis_connected_slaves", Category: "replication"},
		// Pending metrics
		{Name: "redis_version", Query: "", Category: "info", Status: "pending"},
		{Name: "non_root_user", Query: "", Category: "security", Status: "pending"},
	}
}

// TestRedisCollectMetrics_Success tests successful metric collection for Redis instances.
func TestRedisCollectMetrics_Success(t *testing.T) {
	// Create mock server that handles all metric queries
	server := setupRedisVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")

		// Define test instances: 3 masters + 3 slaves
		masters := []struct {
			Address string
			Value   float64
			Labels  map[string]string
		}{
			{"192.18.102.3:7001", 1, map[string]string{"replica_role": "master"}},
			{"192.18.102.4:7001", 1, map[string]string{"replica_role": "master"}},
			{"192.18.102.4:7000", 1, map[string]string{"replica_role": "master"}},
		}
		slaves := []struct {
			Address string
			Value   float64
			Labels  map[string]string
		}{
			{"192.18.102.2:7000", 1, map[string]string{"replica_role": "slave"}},
			{"192.18.102.2:7001", 1, map[string]string{"replica_role": "slave"}},
			{"192.18.102.3:7000", 1, map[string]string{"replica_role": "slave"}},
		}
		all := append(masters, slaves...)

		switch query {
		case "redis_up":
			writeRedisMetricResponse(w, "redis_up", all)
		case "redis_cluster_enabled":
			writeRedisMetricResponse(w, "redis_cluster_enabled", all)
		case "redis_connected_clients":
			// Different client counts
			instances := []struct {
				Address string
				Value   float64
				Labels  map[string]string
			}{
				{"192.18.102.3:7001", 100, nil},
				{"192.18.102.4:7001", 150, nil},
				{"192.18.102.4:7000", 120, nil},
				{"192.18.102.2:7000", 50, nil},
				{"192.18.102.2:7001", 60, nil},
				{"192.18.102.3:7000", 55, nil},
			}
			writeRedisMetricResponse(w, "redis_connected_clients", instances)
		case "redis_maxclients":
			instances := []struct {
				Address string
				Value   float64
				Labels  map[string]string
			}{
				{"192.18.102.3:7001", 10000, nil},
				{"192.18.102.4:7001", 10000, nil},
				{"192.18.102.4:7000", 10000, nil},
				{"192.18.102.2:7000", 10000, nil},
				{"192.18.102.2:7001", 10000, nil},
				{"192.18.102.3:7000", 10000, nil},
			}
			writeRedisMetricResponse(w, "redis_maxclients", instances)
		case "redis_connected_slaves":
			// Only masters have connected slaves
			instances := []struct {
				Address string
				Value   float64
				Labels  map[string]string
			}{
				{"192.18.102.3:7001", 1, nil},
				{"192.18.102.4:7001", 1, nil},
				{"192.18.102.4:7000", 1, nil},
				{"192.18.102.2:7000", 0, nil},
				{"192.18.102.2:7001", 0, nil},
				{"192.18.102.3:7000", 0, nil},
			}
			writeRedisMetricResponse(w, "redis_connected_slaves", instances)
		case "redis_master_repl_offset":
			// Slaves have master offset
			instances := []struct {
				Address string
				Value   float64
				Labels  map[string]string
			}{
				{"192.18.102.2:7000", 650463763, nil},
				{"192.18.102.2:7001", 650463763, nil},
				{"192.18.102.3:7000", 650463763, nil},
			}
			writeRedisMetricResponse(w, "redis_master_repl_offset", instances)
		case "redis_slave_repl_offset":
			// Slaves have their own offset
			instances := []struct {
				Address string
				Value   float64
				Labels  map[string]string
			}{
				{"192.18.102.2:7000", 650463763, nil}, // No lag
				{"192.18.102.2:7001", 650463000, nil}, // 763 bytes lag
				{"192.18.102.3:7000", 650460000, nil}, // 3763 bytes lag
			}
			writeRedisMetricResponse(w, "redis_slave_repl_offset", instances)
		case "redis_master_link_status":
			// Slaves have master link status
			instances := []struct {
				Address string
				Value   float64
				Labels  map[string]string
			}{
				{"192.18.102.2:7000", 1, nil},
				{"192.18.102.2:7001", 1, nil},
				{"192.18.102.3:7000", 1, nil},
			}
			writeRedisMetricResponse(w, "redis_master_link_status", instances)
		case "redis_master_port":
			// Slaves have master port
			instances := []struct {
				Address string
				Value   float64
				Labels  map[string]string
			}{
				{"192.18.102.2:7000", 7001, nil},
				{"192.18.102.2:7001", 7000, nil},
				{"192.18.102.3:7000", 7001, nil},
			}
			writeRedisMetricResponse(w, "redis_master_port", instances)
		case "redis_uptime_in_seconds":
			instances := []struct {
				Address string
				Value   float64
				Labels  map[string]string
			}{
				{"192.18.102.3:7001", 86400, nil},
				{"192.18.102.4:7001", 86400, nil},
				{"192.18.102.4:7000", 86400, nil},
				{"192.18.102.2:7000", 86400, nil},
				{"192.18.102.2:7001", 86400, nil},
				{"192.18.102.3:7000", 86400, nil},
			}
			writeRedisMetricResponse(w, "redis_uptime_in_seconds", instances)
		default:
			// Empty result for unknown metrics
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
		}
	})
	defer server.Close()

	cfg := &config.RedisInspectionConfig{
		Enabled:     true,
		ClusterMode: "3m3s",
	}

	vmConfig := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	retryConfig := &config.RetryConfig{MaxRetries: 0}
	vmClient := vm.NewClient(vmConfig, retryConfig, zerolog.Nop())

	metrics := createRedisMetricsForTest()
	collector := NewRedisCollector(cfg, vmClient, metrics, zerolog.Nop())

	// Create test instances
	instances := []*model.RedisInstance{
		model.NewRedisInstanceWithRole("192.18.102.3:7001", model.RedisRoleMaster),
		model.NewRedisInstanceWithRole("192.18.102.4:7001", model.RedisRoleMaster),
		model.NewRedisInstanceWithRole("192.18.102.4:7000", model.RedisRoleMaster),
		model.NewRedisInstanceWithRole("192.18.102.2:7000", model.RedisRoleSlave),
		model.NewRedisInstanceWithRole("192.18.102.2:7001", model.RedisRoleSlave),
		model.NewRedisInstanceWithRole("192.18.102.3:7000", model.RedisRoleSlave),
	}

	// Execute collection
	resultsMap, err := collector.CollectMetrics(context.Background(), instances, metrics)

	// Verify results
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(resultsMap) != 6 {
		t.Errorf("expected 6 results, got: %d", len(resultsMap))
	}

	// Verify a slave instance has correct values
	slaveResult := resultsMap["192.18.102.2:7001"]
	if slaveResult == nil {
		t.Fatal("expected result for slave 192.18.102.2:7001")
	}

	if slaveResult.ConnectedClients != 60 {
		t.Errorf("expected 60 connected clients, got: %d", slaveResult.ConnectedClients)
	}
	if slaveResult.MaxClients != 10000 {
		t.Errorf("expected 10000 max clients, got: %d", slaveResult.MaxClients)
	}
	if slaveResult.MasterPort != 7000 {
		t.Errorf("expected master port 7000, got: %d", slaveResult.MasterPort)
	}

	// Verify replication lag calculation
	if slaveResult.ReplicationLag != 763 {
		t.Errorf("expected replication lag 763, got: %d", slaveResult.ReplicationLag)
	}

	// Verify pending metrics are N/A
	versionMetric := slaveResult.GetMetric("redis_version")
	if versionMetric == nil || !versionMetric.IsNA {
		t.Error("expected redis_version to be N/A")
	}
}

// TestRedisCollectMetrics_PendingMetrics tests that pending metrics are set to N/A.
func TestRedisCollectMetrics_PendingMetrics(t *testing.T) {
	server := setupRedisVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Return empty results for all queries
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
	})
	defer server.Close()

	cfg := &config.RedisInspectionConfig{
		Enabled:     true,
		ClusterMode: "3m3s",
	}

	vmConfig := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	retryConfig := &config.RetryConfig{MaxRetries: 0}
	vmClient := vm.NewClient(vmConfig, retryConfig, zerolog.Nop())

	metrics := createRedisMetricsForTest()
	collector := NewRedisCollector(cfg, vmClient, metrics, zerolog.Nop())

	instances := []*model.RedisInstance{
		model.NewRedisInstanceWithRole("192.18.102.2:7000", model.RedisRoleMaster),
	}

	resultsMap, err := collector.CollectMetrics(context.Background(), instances, metrics)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	result := resultsMap["192.18.102.2:7000"]
	if result == nil {
		t.Fatal("expected result for instance")
	}

	// Check pending metrics
	pendingMetrics := []string{"redis_version", "non_root_user"}
	for _, name := range pendingMetrics {
		m := result.GetMetric(name)
		if m == nil {
			t.Errorf("expected metric %s to exist", name)
			continue
		}
		if !m.IsNA {
			t.Errorf("expected metric %s to be N/A", name)
		}
		if m.FormattedValue != "N/A" {
			t.Errorf("expected metric %s FormattedValue to be N/A, got: %s", name, m.FormattedValue)
		}
	}
}

// TestRedisCollectMetrics_EmptyInstances tests collection with empty instances.
func TestRedisCollectMetrics_EmptyInstances(t *testing.T) {
	server := setupRedisVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
	})
	defer server.Close()

	cfg := &config.RedisInspectionConfig{
		Enabled:     true,
		ClusterMode: "3m3s",
	}

	vmConfig := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	retryConfig := &config.RetryConfig{MaxRetries: 0}
	vmClient := vm.NewClient(vmConfig, retryConfig, zerolog.Nop())

	metrics := createRedisMetricsForTest()
	collector := NewRedisCollector(cfg, vmClient, metrics, zerolog.Nop())

	// Empty instances
	instances := []*model.RedisInstance{}

	resultsMap, err := collector.CollectMetrics(context.Background(), instances, metrics)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(resultsMap) != 0 {
		t.Errorf("expected 0 results for empty instances, got: %d", len(resultsMap))
	}
}

// TestRedisCollectMetrics_RoleVerification tests dual-source role verification.
func TestRedisCollectMetrics_RoleVerification(t *testing.T) {
	server := setupRedisVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")

		switch query {
		case "redis_connected_slaves":
			// Instance with unknown role but connected_slaves > 0 should become master
			instances := []struct {
				Address string
				Value   float64
				Labels  map[string]string
			}{
				{"192.18.102.2:7000", 2, nil}, // 2 slaves = master
				{"192.18.102.3:7000", 0, nil}, // 0 slaves, role stays unknown
			}
			writeRedisMetricResponse(w, "redis_connected_slaves", instances)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
		}
	})
	defer server.Close()

	cfg := &config.RedisInspectionConfig{
		Enabled:     true,
		ClusterMode: "3m3s",
	}

	vmConfig := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	retryConfig := &config.RetryConfig{MaxRetries: 0}
	vmClient := vm.NewClient(vmConfig, retryConfig, zerolog.Nop())

	metrics := createRedisMetricsForTest()
	collector := NewRedisCollector(cfg, vmClient, metrics, zerolog.Nop())

	// Create instances with unknown role
	instances := []*model.RedisInstance{
		model.NewRedisInstanceWithRole("192.18.102.2:7000", model.RedisRoleUnknown),
		model.NewRedisInstanceWithRole("192.18.102.3:7000", model.RedisRoleUnknown),
	}

	resultsMap, err := collector.CollectMetrics(context.Background(), instances, metrics)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Instance with connected_slaves > 0 should be verified as master
	result1 := resultsMap["192.18.102.2:7000"]
	if result1.Instance.Role != model.RedisRoleMaster {
		t.Errorf("expected role master for 192.18.102.2:7000, got: %s", result1.Instance.Role)
	}

	// Instance with connected_slaves == 0 should remain unknown
	result2 := resultsMap["192.18.102.3:7000"]
	if result2.Instance.Role != model.RedisRoleUnknown {
		t.Errorf("expected role unknown for 192.18.102.3:7000, got: %s", result2.Instance.Role)
	}
}

// TestRedisCollectMetrics_ReplicationLag tests replication lag calculation.
func TestRedisCollectMetrics_ReplicationLag(t *testing.T) {
	server := setupRedisVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")

		switch query {
		case "redis_master_repl_offset":
			instances := []struct {
				Address string
				Value   float64
				Labels  map[string]string
			}{
				{"192.18.102.2:7000", 1000000, nil}, // slave
				{"192.18.102.3:7000", 1000000, nil}, // master (should skip)
			}
			writeRedisMetricResponse(w, "redis_master_repl_offset", instances)
		case "redis_slave_repl_offset":
			instances := []struct {
				Address string
				Value   float64
				Labels  map[string]string
			}{
				{"192.18.102.2:7000", 999000, nil}, // 1000 bytes lag
			}
			writeRedisMetricResponse(w, "redis_slave_repl_offset", instances)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
		}
	})
	defer server.Close()

	cfg := &config.RedisInspectionConfig{
		Enabled:     true,
		ClusterMode: "3m3s",
	}

	vmConfig := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	retryConfig := &config.RetryConfig{MaxRetries: 0}
	vmClient := vm.NewClient(vmConfig, retryConfig, zerolog.Nop())

	metrics := createRedisMetricsForTest()
	collector := NewRedisCollector(cfg, vmClient, metrics, zerolog.Nop())

	instances := []*model.RedisInstance{
		model.NewRedisInstanceWithRole("192.18.102.2:7000", model.RedisRoleSlave),
		model.NewRedisInstanceWithRole("192.18.102.3:7000", model.RedisRoleMaster),
	}

	resultsMap, err := collector.CollectMetrics(context.Background(), instances, metrics)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Slave should have lag calculated
	slaveResult := resultsMap["192.18.102.2:7000"]
	if slaveResult.ReplicationLag != 1000 {
		t.Errorf("expected replication lag 1000, got: %d", slaveResult.ReplicationLag)
	}
	if slaveResult.MasterReplOffset != 1000000 {
		t.Errorf("expected master_repl_offset 1000000, got: %d", slaveResult.MasterReplOffset)
	}
	if slaveResult.SlaveReplOffset != 999000 {
		t.Errorf("expected slave_repl_offset 999000, got: %d", slaveResult.SlaveReplOffset)
	}

	// Master should have no lag calculated (only for slaves)
	masterResult := resultsMap["192.18.102.3:7000"]
	if masterResult.ReplicationLag != 0 {
		t.Errorf("expected no replication lag for master, got: %d", masterResult.ReplicationLag)
	}
}

// TestRedisCollector_setPendingMetrics tests setPendingMetrics helper.
func TestRedisCollector_setPendingMetrics(t *testing.T) {
	collector := &RedisCollector{
		logger: zerolog.Nop(),
	}

	// Create results map
	resultsMap := map[string]*model.RedisInspectionResult{
		"192.18.102.2:7000": model.NewRedisInspectionResult(model.NewRedisInstance("192.18.102.2:7000")),
		"192.18.102.3:7000": model.NewRedisInspectionResult(model.NewRedisInstance("192.18.102.3:7000")),
	}

	pendingMetrics := []*model.RedisMetricDefinition{
		{Name: "redis_version", Status: "pending"},
		{Name: "non_root_user", Status: "pending"},
	}

	collector.setPendingMetrics(resultsMap, pendingMetrics)

	// Verify all instances have pending metrics set to N/A
	for address, result := range resultsMap {
		for _, metric := range pendingMetrics {
			m := result.GetMetric(metric.Name)
			if m == nil {
				t.Errorf("instance %s: expected metric %s to exist", address, metric.Name)
				continue
			}
			if !m.IsNA {
				t.Errorf("instance %s: expected metric %s to be N/A", address, metric.Name)
			}
		}
	}
}

// TestRedisCollector_verifyRoles tests verifyRoles helper.
func TestRedisCollector_verifyRoles(t *testing.T) {
	collector := &RedisCollector{
		logger: zerolog.Nop(),
	}

	tests := []struct {
		name              string
		initialRole       model.RedisRole
		slavesCount       float64
		hasMasterLinkStatus bool
		expectedRole      model.RedisRole
	}{
		{
			name:              "already master - no change",
			initialRole:       model.RedisRoleMaster,
			slavesCount:       0,
			hasMasterLinkStatus: false,
			expectedRole:      model.RedisRoleMaster,
		},
		{
			name:              "already slave - no change",
			initialRole:       model.RedisRoleSlave,
			slavesCount:       0,
			hasMasterLinkStatus: false,
			expectedRole:      model.RedisRoleSlave,
		},
		{
			name:              "unknown with slaves > 0 - becomes master",
			initialRole:       model.RedisRoleUnknown,
			slavesCount:       2,
			hasMasterLinkStatus: false,
			expectedRole:      model.RedisRoleMaster,
		},
		{
			name:              "unknown with slaves == 0 and no master_link_status - stays unknown",
			initialRole:       model.RedisRoleUnknown,
			slavesCount:       0,
			hasMasterLinkStatus: false,
			expectedRole:      model.RedisRoleUnknown,
		},
		{
			name:              "unknown with slaves == 0 and has master_link_status - becomes slave",
			initialRole:       model.RedisRoleUnknown,
			slavesCount:       0,
			hasMasterLinkStatus: true,
			expectedRole:      model.RedisRoleSlave,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := model.NewRedisInstanceWithRole("192.18.102.2:7000", tt.initialRole)
			result := model.NewRedisInspectionResult(instance)
			result.SetMetric(&model.RedisMetricValue{
				Name:     "redis_connected_slaves",
				RawValue: tt.slavesCount,
			})
			if tt.hasMasterLinkStatus {
				result.SetMetric(&model.RedisMetricValue{
					Name:     "redis_master_link_status",
					RawValue: 1,
				})
			}

			resultsMap := map[string]*model.RedisInspectionResult{
				"192.18.102.2:7000": result,
			}

			collector.verifyRoles(resultsMap)

			if result.Instance.Role != tt.expectedRole {
				t.Errorf("expected role %s, got %s", tt.expectedRole, result.Instance.Role)
			}
		})
	}
}

// TestRedisCollector_calculateReplicationLag tests calculateReplicationLag helper.
func TestRedisCollector_calculateReplicationLag(t *testing.T) {
	collector := &RedisCollector{
		logger: zerolog.Nop(),
	}

	tests := []struct {
		name            string
		role            model.RedisRole
		masterOffset    float64
		slaveOffset     float64
		hasMasterOffset bool
		hasSlaveOffset  bool
		expectedLag     int64
	}{
		{
			name:            "slave with lag",
			role:            model.RedisRoleSlave,
			masterOffset:    1000000,
			slaveOffset:     999000,
			hasMasterOffset: true,
			hasSlaveOffset:  true,
			expectedLag:     1000,
		},
		{
			name:            "slave with no lag",
			role:            model.RedisRoleSlave,
			masterOffset:    1000000,
			slaveOffset:     1000000,
			hasMasterOffset: true,
			hasSlaveOffset:  true,
			expectedLag:     0,
		},
		{
			name:            "master - no calculation",
			role:            model.RedisRoleMaster,
			masterOffset:    1000000,
			slaveOffset:     999000,
			hasMasterOffset: true,
			hasSlaveOffset:  true,
			expectedLag:     0,
		},
		{
			name:            "slave missing master offset",
			role:            model.RedisRoleSlave,
			masterOffset:    0,
			slaveOffset:     999000,
			hasMasterOffset: false,
			hasSlaveOffset:  true,
			expectedLag:     0,
		},
		{
			name:            "slave missing slave offset",
			role:            model.RedisRoleSlave,
			masterOffset:    1000000,
			slaveOffset:     0,
			hasMasterOffset: true,
			hasSlaveOffset:  false,
			expectedLag:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := model.NewRedisInstanceWithRole("192.18.102.2:7000", tt.role)
			result := model.NewRedisInspectionResult(instance)

			if tt.hasMasterOffset {
				result.SetMetric(&model.RedisMetricValue{
					Name:     "redis_master_repl_offset",
					RawValue: tt.masterOffset,
				})
			}
			if tt.hasSlaveOffset {
				result.SetMetric(&model.RedisMetricValue{
					Name:     "redis_slave_repl_offset",
					RawValue: tt.slaveOffset,
				})
			}

			resultsMap := map[string]*model.RedisInspectionResult{
				"192.18.102.2:7000": result,
			}

			collector.calculateReplicationLag(resultsMap)

			if result.ReplicationLag != tt.expectedLag {
				t.Errorf("expected lag %d, got %d", tt.expectedLag, result.ReplicationLag)
			}
		})
	}
}

// TestRedisCollector_populateResultFields tests populateResultFields helper.
func TestRedisCollector_populateResultFields(t *testing.T) {
	collector := &RedisCollector{
		logger: zerolog.Nop(),
	}

	instance := model.NewRedisInstanceWithRole("192.18.102.2:7000", model.RedisRoleSlave)
	result := model.NewRedisInspectionResult(instance)

	// Set metrics
	result.SetMetric(&model.RedisMetricValue{Name: "redis_up", RawValue: 1})
	result.SetMetric(&model.RedisMetricValue{Name: "redis_cluster_enabled", RawValue: 1})
	result.SetMetric(&model.RedisMetricValue{Name: "redis_master_link_status", RawValue: 1})
	result.SetMetric(&model.RedisMetricValue{Name: "redis_maxclients", RawValue: 10000})
	result.SetMetric(&model.RedisMetricValue{Name: "redis_connected_clients", RawValue: 150})
	result.SetMetric(&model.RedisMetricValue{Name: "redis_connected_slaves", RawValue: 0})
	result.SetMetric(&model.RedisMetricValue{Name: "redis_master_port", RawValue: 7001})
	result.SetMetric(&model.RedisMetricValue{Name: "redis_uptime_in_seconds", RawValue: 86400})

	collector.populateResultFields(result)

	// Verify fields
	if !result.ConnectionStatus {
		t.Error("expected ConnectionStatus to be true")
	}
	if !result.ClusterEnabled {
		t.Error("expected ClusterEnabled to be true")
	}
	if !result.Instance.ClusterEnabled {
		t.Error("expected Instance.ClusterEnabled to be true")
	}
	if !result.MasterLinkStatus {
		t.Error("expected MasterLinkStatus to be true")
	}
	if result.MaxClients != 10000 {
		t.Errorf("expected MaxClients 10000, got %d", result.MaxClients)
	}
	if result.ConnectedClients != 150 {
		t.Errorf("expected ConnectedClients 150, got %d", result.ConnectedClients)
	}
	if result.ConnectedSlaves != 0 {
		t.Errorf("expected ConnectedSlaves 0, got %d", result.ConnectedSlaves)
	}
	if result.MasterPort != 7001 {
		t.Errorf("expected MasterPort 7001, got %d", result.MasterPort)
	}
	if result.Uptime != 86400 {
		t.Errorf("expected Uptime 86400, got %d", result.Uptime)
	}
	if result.CollectedAt.IsZero() {
		t.Error("expected CollectedAt to be set")
	}
}
