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
