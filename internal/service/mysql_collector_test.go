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

// setupVMTestServer 创建 Mock VictoriaMetrics 服务器
func setupMySQLVMTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

// createTestMySQLCollector 创建测试用 MySQLCollector
func createTestMySQLCollector(vmURL string, cfg *config.MySQLInspectionConfig) *MySQLCollector {
	vmConfig := &config.VictoriaMetricsConfig{Endpoint: vmURL}
	retryConfig := &config.RetryConfig{
		MaxRetries: 0, // 测试时不重试
	}
	vmClient := vm.NewClient(vmConfig, retryConfig, zerolog.Nop())

	return NewMySQLCollector(cfg, vmClient, nil, zerolog.Nop())
}

// writeVMJSONResponse 写入 VictoriaMetrics JSON 响应
func writeVMJSONResponse(w http.ResponseWriter, addresses []string) {
	w.Header().Set("Content-Type", "application/json")
	timestamp := time.Now().Unix()

	// 构建结果数组
	results := ""
	for i, addr := range addresses {
		if i > 0 {
			results += ","
		}
		results += fmt.Sprintf(`{"metric": {"__name__": "mysql_up", "address": "%s"}, "value": [%d, "1"]}`, addr, timestamp)
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

// mockMySQLUpResponse 创建 mock mysql_up 查询响应（已弃用，使用 writeVMJSONResponse）
func mockMySQLUpResponse(addresses []string) *vm.QueryResponse {
	samples := make([]vm.Sample, 0, len(addresses))
	timestamp := float64(time.Now().Unix())

	for _, addr := range addresses {
		samples = append(samples, vm.Sample{
			Metric: vm.Metric{
				"__name__": "mysql_up",
				"address":  addr,
			},
			Value: vm.SampleValue{timestamp, "1"},
		})
	}

	return &vm.QueryResponse{
		Status: "success",
		Data: vm.QueryData{
			ResultType: "vector",
			Result:     samples,
		},
	}
}

// TestDiscoverInstances_Success 测试正常发现多个实例
func TestDiscoverInstances_Success(t *testing.T) {
	server := setupMySQLVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// 验证请求路径
		if !contains(r.URL.Path, "/api/v1/query") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// 验证查询参数
		query := r.URL.Query().Get("query")
		if !contains(query, "mysql_up") {
			t.Errorf("expected mysql_up query, got: %s", query)
		}

		// 返回 JSON 响应（手动构造以确保格式正确）
		w.Header().Set("Content-Type", "application/json")
		timestamp := time.Now().Unix()
		jsonResp := fmt.Sprintf(`{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": [
					{"metric": {"__name__": "mysql_up", "address": "172.18.182.91:3306"}, "value": [%d, "1"]},
					{"metric": {"__name__": "mysql_up", "address": "172.18.182.92:3306"}, "value": [%d, "1"]},
					{"metric": {"__name__": "mysql_up", "address": "172.18.182.93:3306"}, "value": [%d, "1"]}
				]
			}
		}`, timestamp, timestamp, timestamp)
		w.Write([]byte(jsonResp))
	})
	defer server.Close()

	cfg := &config.MySQLInspectionConfig{
		Enabled:     true,
		ClusterMode: "mgr",
	}
	collector := createTestMySQLCollector(server.URL, cfg)

	// 执行发现
	instances, err := collector.DiscoverInstances(context.Background())

	// 验证结果
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(instances) != 3 {
		t.Errorf("expected 3 instances, got: %d", len(instances))
	}

	// 验证第一个实例
	if instances[0].Address != "172.18.182.91:3306" {
		t.Errorf("expected address 172.18.182.91:3306, got: %s", instances[0].Address)
	}
	if instances[0].IP != "172.18.182.91" {
		t.Errorf("expected IP 172.18.182.91, got: %s", instances[0].IP)
	}
	if instances[0].Port != 3306 {
		t.Errorf("expected port 3306, got: %d", instances[0].Port)
	}
	if instances[0].ClusterMode != model.ClusterModeMGR {
		t.Errorf("expected MGR mode, got: %s", instances[0].ClusterMode)
	}
	if instances[0].DatabaseType != "MySQL" {
		t.Errorf("expected MySQL type, got: %s", instances[0].DatabaseType)
	}
}

// TestDiscoverInstances_WithAddressPatternFilter 测试地址模式过滤
func TestDiscoverInstances_WithAddressPatternFilter(t *testing.T) {
	server := setupMySQLVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeVMJSONResponse(w, []string{
			"172.18.182.91:3306",
			"172.18.182.92:3306",
			"192.168.1.100:3306",
			"10.0.0.50:3306",
		})
	})
	defer server.Close()

	cfg := &config.MySQLInspectionConfig{
		Enabled:     true,
		ClusterMode: "mgr",
		InstanceFilter: config.MySQLFilter{
			AddressPatterns: []string{"172.18.182.*"},
		},
	}
	collector := createTestMySQLCollector(server.URL, cfg)

	instances, err := collector.DiscoverInstances(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// 应该只返回匹配 172.18.182.* 的实例
	if len(instances) != 2 {
		t.Errorf("expected 2 instances (filtered), got: %d", len(instances))
	}

	// 验证所有实例都匹配过滤器
	for _, inst := range instances {
		if !contains(inst.IP, "172.18.182") {
			t.Errorf("instance %s should be filtered out", inst.Address)
		}
	}
}

// TestDiscoverInstances_WithBusinessGroupFilter 测试业务组过滤
func TestDiscoverInstances_WithBusinessGroupFilter(t *testing.T) {
	server := setupMySQLVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// 验证业务组注入到查询中
		query := r.URL.Query().Get("query")
		if !contains(query, "busigroup") {
			t.Errorf("expected busigroup filter in query, got: %s", query)
		}

		writeVMJSONResponse(w, []string{"172.18.182.91:3306"})
	})
	defer server.Close()

	cfg := &config.MySQLInspectionConfig{
		Enabled:     true,
		ClusterMode: "mgr",
		InstanceFilter: config.MySQLFilter{
			BusinessGroups: []string{"MySQL-Production"},
		},
	}
	collector := createTestMySQLCollector(server.URL, cfg)

	instances, err := collector.DiscoverInstances(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(instances) != 1 {
		t.Errorf("expected 1 instance, got: %d", len(instances))
	}
}

// TestDiscoverInstances_EmptyResults 测试无结果场景
func TestDiscoverInstances_EmptyResults(t *testing.T) {
	server := setupMySQLVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeVMJSONResponse(w, []string{}) // 空结果
	})
	defer server.Close()

	cfg := &config.MySQLInspectionConfig{
		Enabled:     true,
		ClusterMode: "mgr",
	}
	collector := createTestMySQLCollector(server.URL, cfg)

	instances, err := collector.DiscoverInstances(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(instances) != 0 {
		t.Errorf("expected 0 instances, got: %d", len(instances))
	}
}

// TestDiscoverInstances_QueryError 测试 VM 查询错误
func TestDiscoverInstances_QueryError(t *testing.T) {
	server := setupMySQLVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// 返回 HTTP 500 错误
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"error": "internal server error"}`))
	})
	defer server.Close()

	cfg := &config.MySQLInspectionConfig{
		Enabled:     true,
		ClusterMode: "mgr",
	}
	collector := createTestMySQLCollector(server.URL, cfg)

	instances, err := collector.DiscoverInstances(context.Background())

	// 应该返回错误
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if instances != nil {
		t.Errorf("expected nil instances on error, got: %v", instances)
	}
	if !contains(err.Error(), "failed to query mysql_up") {
		t.Errorf("expected 'failed to query mysql_up' in error, got: %v", err)
	}
}

// TestDiscoverInstances_MissingAddressLabel 测试缺失地址标签
func TestDiscoverInstances_MissingAddressLabel(t *testing.T) {
	server := setupMySQLVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		timestamp := time.Now().Unix()
		// 第一个没有地址标签，第二个有
		jsonResp := fmt.Sprintf(`{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": [
					{"metric": {"__name__": "mysql_up"}, "value": [%d, "1"]},
					{"metric": {"__name__": "mysql_up", "address": "172.18.182.91:3306"}, "value": [%d, "1"]}
				]
			}
		}`, timestamp, timestamp)
		w.Write([]byte(jsonResp))
	})
	defer server.Close()

	cfg := &config.MySQLInspectionConfig{
		Enabled:     true,
		ClusterMode: "mgr",
	}
	collector := createTestMySQLCollector(server.URL, cfg)

	instances, err := collector.DiscoverInstances(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// 应该只返回有地址的实例，缺失地址的被跳过
	if len(instances) != 1 {
		t.Errorf("expected 1 instance (missing label skipped), got: %d", len(instances))
	}
}

// TestDiscoverInstances_DuplicateAddresses 测试地址去重
func TestDiscoverInstances_DuplicateAddresses(t *testing.T) {
	server := setupMySQLVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		timestamp := time.Now().Unix()
		// 两个重复地址，一个唯一地址
		jsonResp := fmt.Sprintf(`{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": [
					{"metric": {"address": "172.18.182.91:3306"}, "value": [%d, "1"]},
					{"metric": {"address": "172.18.182.91:3306"}, "value": [%d, "1"]},
					{"metric": {"address": "172.18.182.92:3306"}, "value": [%d, "1"]}
				]
			}
		}`, timestamp, timestamp, timestamp)
		w.Write([]byte(jsonResp))
	})
	defer server.Close()

	cfg := &config.MySQLInspectionConfig{
		Enabled:     true,
		ClusterMode: "mgr",
	}
	collector := createTestMySQLCollector(server.URL, cfg)

	instances, err := collector.DiscoverInstances(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// 应该去重，只返回 2 个唯一实例
	if len(instances) != 2 {
		t.Errorf("expected 2 unique instances, got: %d", len(instances))
	}
}

// TestDiscoverInstances_InvalidAddress 测试地址解析失败
func TestDiscoverInstances_InvalidAddress(t *testing.T) {
	server := setupMySQLVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		timestamp := time.Now().Unix()
		// 一个无效地址，一个有效地址
		jsonResp := fmt.Sprintf(`{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": [
					{"metric": {"address": "invalid-address"}, "value": [%d, "1"]},
					{"metric": {"address": "172.18.182.91:3306"}, "value": [%d, "1"]}
				]
			}
		}`, timestamp, timestamp)
		w.Write([]byte(jsonResp))
	})
	defer server.Close()

	cfg := &config.MySQLInspectionConfig{
		Enabled:     true,
		ClusterMode: "mgr",
	}
	collector := createTestMySQLCollector(server.URL, cfg)

	instances, err := collector.DiscoverInstances(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// 应该跳过无效地址，只返回有效的实例
	if len(instances) != 1 {
		t.Errorf("expected 1 valid instance, got: %d", len(instances))
	}
	if instances[0].Address != "172.18.182.91:3306" {
		t.Errorf("expected valid address, got: %s", instances[0].Address)
	}
}

// TestMatchAddressPattern_Wildcard 测试通配符匹配
func TestMatchAddressPattern_Wildcard(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		pattern  string
		expected bool
	}{
		{"exact match", "172.18.182.91:3306", "172.18.182.91:3306", true},
		{"wildcard IP", "172.18.182.91:3306", "172.18.182.*", true},
		{"wildcard IP segment", "172.18.182.91:3306", "172.18.*.*", true},
		{"wildcard port", "172.18.182.91:3306", "172.18.182.91:*", true},
		{"wildcard all", "172.18.182.91:3306", "*", true},
		{"no match IP", "172.18.182.91:3306", "192.168.*", false},
		{"no match port", "172.18.182.91:3306", "172.18.182.91:3307", false},
		{"partial no match", "172.18.182.91:3306", "172.18.182.9", false},
		{"empty pattern", "", "", true},
		{"wildcard with colon", "172.18.182.91:3306", "*:3306", true},
		{"wildcard end", "172.18.182.91:3306", "172.18.182.*:3306", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchAddressPattern(tt.address, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchAddressPattern(%q, %q) = %v, want %v",
					tt.address, tt.pattern, result, tt.expected)
			}
		})
	}
}

// TestMatchAddressPattern_EdgeCases 测试边界情况
func TestMatchAddressPattern_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		pattern  string
		expected bool
	}{
		{"empty address and pattern", "", "", true},
		{"empty address", "", "pattern", false},
		{"empty pattern no wildcard", "address", "", false},
		{"special regex chars", "172.18.182.91:3306", "172.18.182.91:3306", true},
		{"invalid regex in pattern", "172.18.182.91:3306", "[invalid(", false},
		{"multiple wildcards", "172.18.182.91:3306", "*.*.*.*:*", true},
		{"wildcard only", "anything", "*", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchAddressPattern(tt.address, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchAddressPattern(%q, %q) = %v, want %v",
					tt.address, tt.pattern, result, tt.expected)
			}
		})
	}
}

// TestExtractAddress_Priority 测试标签优先级
func TestExtractAddress_Priority(t *testing.T) {
	cfg := &config.MySQLInspectionConfig{
		Enabled:     true,
		ClusterMode: "mgr",
	}
	collector := createTestMySQLCollector("http://dummy", cfg)

	tests := []struct {
		name     string
		labels   map[string]string
		expected string
	}{
		{
			name:     "address label present",
			labels:   map[string]string{"address": "172.18.182.91:3306"},
			expected: "172.18.182.91:3306",
		},
		{
			name:     "instance label fallback",
			labels:   map[string]string{"instance": "172.18.182.92:3306"},
			expected: "172.18.182.92:3306",
		},
		{
			name:     "server label fallback",
			labels:   map[string]string{"server": "172.18.182.93:3306"},
			expected: "172.18.182.93:3306",
		},
		{
			name: "address has priority over instance",
			labels: map[string]string{
				"address":  "172.18.182.91:3306",
				"instance": "172.18.182.92:3306",
			},
			expected: "172.18.182.91:3306",
		},
		{
			name: "address has priority over server",
			labels: map[string]string{
				"address": "172.18.182.91:3306",
				"server":  "172.18.182.93:3306",
			},
			expected: "172.18.182.91:3306",
		},
		{
			name: "instance has priority over server",
			labels: map[string]string{
				"instance": "172.18.182.92:3306",
				"server":   "172.18.182.93:3306",
			},
			expected: "172.18.182.92:3306",
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
		{
			name: "empty address value",
			labels: map[string]string{
				"address": "",
				"instance": "172.18.182.92:3306",
			},
			expected: "172.18.182.92:3306",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.extractAddress(tt.labels)
			if result != tt.expected {
				t.Errorf("extractAddress(%v) = %q, want %q", tt.labels, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// CollectMetrics Tests
// =============================================================================

// TestCollectMetrics_Success tests normal metric collection
func TestCollectMetrics_Success(t *testing.T) {
	server := setupMySQLVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")

		w.Header().Set("Content-Type", "application/json")
		timestamp := time.Now().Unix()

		// Return different responses based on query
		var jsonResp string
		if contains(query, "mysql_up") {
			jsonResp = fmt.Sprintf(`{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [
						{"metric": {"address": "172.18.182.91:3306"}, "value": [%d, "1"]}
					]
				}
			}`, timestamp)
		} else if contains(query, "max_connections") {
			jsonResp = fmt.Sprintf(`{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [
						{"metric": {"address": "172.18.182.91:3306"}, "value": [%d, "1000"]}
					]
				}
			}`, timestamp)
		} else {
			jsonResp = `{"status": "success", "data": {"resultType": "vector", "result": []}}`
		}

		w.Write([]byte(jsonResp))
	})
	defer server.Close()

	cfg := &config.MySQLInspectionConfig{
		ClusterMode: "mgr",
	}

	collector := createTestMySQLCollector(server.URL, cfg)

	// Create test instances
	instances := []*model.MySQLInstance{
		model.NewMySQLInstanceWithClusterMode("172.18.182.91:3306", model.ClusterModeMGR),
	}

	// Create test metrics
	metrics := []*model.MySQLMetricDefinition{
		{
			Name:        "mysql_up",
			DisplayName: "连接状态",
			Query:       "mysql_up",
			Category:    "connection",
		},
		{
			Name:        "max_connections",
			DisplayName: "最大连接数",
			Query:       "mysql_global_variables_max_connections",
			Category:    "connection",
		},
	}

	// Collect metrics
	ctx := context.Background()
	results, err := collector.CollectMetrics(ctx, instances, metrics)

	// Verify
	if err != nil {
		t.Fatalf("CollectMetrics failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	result, ok := results["172.18.182.91:3306"]
	if !ok {
		t.Fatal("result for 172.18.182.91:3306 not found")
	}

	// Check metrics collected
	if result.Metrics == nil {
		t.Fatal("Metrics map is nil")
	}

	if len(result.Metrics) != 2 {
		t.Errorf("expected 2 metrics, got %d", len(result.Metrics))
	}

	// Verify mysql_up metric
	upMetric := result.GetMetric("mysql_up")
	if upMetric == nil {
		t.Error("mysql_up metric not found")
	} else if upMetric.RawValue != 1 {
		t.Errorf("mysql_up RawValue = %f, want 1", upMetric.RawValue)
	}

	// Verify max_connections metric
	maxConnMetric := result.GetMetric("max_connections")
	if maxConnMetric == nil {
		t.Error("max_connections metric not found")
	} else if maxConnMetric.RawValue != 1000 {
		t.Errorf("max_connections RawValue = %f, want 1000", maxConnMetric.RawValue)
	}
}

// TestCollectMetrics_PendingMetrics tests pending metrics are set to N/A
func TestCollectMetrics_PendingMetrics(t *testing.T) {
	server := setupMySQLVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))
	})
	defer server.Close()

	cfg := &config.MySQLInspectionConfig{
		ClusterMode: "mgr",
	}

	collector := createTestMySQLCollector(server.URL, cfg)

	instances := []*model.MySQLInstance{
		model.NewMySQLInstanceWithClusterMode("172.18.182.91:3306", model.ClusterModeMGR),
	}

	// Create pending metrics
	metrics := []*model.MySQLMetricDefinition{
		{
			Name:        "non_root_user",
			DisplayName: "非 root 用户启动",
			Query:       "", // No query = pending
			Category:    "security",
			Status:      "pending",
		},
		{
			Name:        "slave_running",
			DisplayName: "Slave 是否启动",
			Query:       "",
			Category:    "replication",
			Status:      "pending",
		},
	}

	ctx := context.Background()
	results, err := collector.CollectMetrics(ctx, instances, metrics)

	if err != nil {
		t.Fatalf("CollectMetrics failed: %v", err)
	}

	result := results["172.18.182.91:3306"]
	if result == nil {
		t.Fatal("result not found")
	}

	// Check pending metrics are set to N/A
	for _, metric := range metrics {
		mv := result.GetMetric(metric.Name)
		if mv == nil {
			t.Errorf("pending metric %s not found", metric.Name)
			continue
		}

		if !mv.IsNA {
			t.Errorf("metric %s IsNA = false, want true", metric.Name)
		}

		if mv.FormattedValue != "N/A" {
			t.Errorf("metric %s FormattedValue = %q, want N/A", metric.Name, mv.FormattedValue)
		}
	}
}

// =============================================================================
// Label Extraction Tests
// =============================================================================

// TestCollectMetrics_VersionLabelExtract tests version label extraction
func TestCollectMetrics_VersionLabelExtract(t *testing.T) {
	server := setupMySQLVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")

		w.Header().Set("Content-Type", "application/json")
		timestamp := time.Now().Unix()

		var jsonResp string
		if contains(query, "mysql_version_info") {
			// Return version label
			jsonResp = fmt.Sprintf(`{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [
						{
							"metric": {
								"__name__": "mysql_version_info",
								"address": "172.18.182.91:3306",
								"version": "8.0.39"
							},
							"value": [%d, "1"]
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

	cfg := &config.MySQLInspectionConfig{
		ClusterMode: "mgr",
	}

	collector := createTestMySQLCollector(server.URL, cfg)

	instances := []*model.MySQLInstance{
		model.NewMySQLInstanceWithClusterMode("172.18.182.91:3306", model.ClusterModeMGR),
	}

	// Metric with label extraction
	metrics := []*model.MySQLMetricDefinition{
		{
			Name:         "mysql_version",
			DisplayName:  "数据库版本",
			Query:        "mysql_version_info",
			Category:     "info",
			LabelExtract: "version",
		},
	}

	ctx := context.Background()
	results, err := collector.CollectMetrics(ctx, instances, metrics)

	if err != nil {
		t.Fatalf("CollectMetrics failed: %v", err)
	}

	result, ok := results["172.18.182.91:3306"]
	if !ok {
		t.Fatal("result for 172.18.182.91:3306 not found")
	}

	// Verify version label extracted
	versionMetric := result.GetMetric("mysql_version")
	if versionMetric == nil {
		t.Fatal("mysql_version metric not found")
	}

	if versionMetric.StringValue != "8.0.39" {
		t.Errorf("StringValue = %q, want \"8.0.39\"", versionMetric.StringValue)
	}

	if versionMetric.Name != "mysql_version" {
		t.Errorf("Name = %q, want \"mysql_version\"", versionMetric.Name)
	}

	// Check labels contain version
	if versionMetric.Labels["version"] != "8.0.39" {
		t.Errorf("Labels[version] = %q, want \"8.0.39\"", versionMetric.Labels["version"])
	}
}

// TestCollectMetrics_ServerIDLabelExtract tests Server ID label extraction
func TestCollectMetrics_ServerIDLabelExtract(t *testing.T) {
	server := setupMySQLVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")

		w.Header().Set("Content-Type", "application/json")
		timestamp := time.Now().Unix()

		var jsonResp string
		if contains(query, "mgr_role_primary") {
			// Return member_id label
			jsonResp = fmt.Sprintf(`{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [
						{
							"metric": {
								"__name__": "mysql_innodb_cluster_mgr_role_primary",
								"address": "172.18.182.91:3306",
								"member_id": "91abc-def1-2345-6789"
							},
							"value": [%d, "1"]
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

	cfg := &config.MySQLInspectionConfig{
		ClusterMode: "mgr",
	}

	collector := createTestMySQLCollector(server.URL, cfg)

	instances := []*model.MySQLInstance{
		model.NewMySQLInstanceWithClusterMode("172.18.182.91:3306", model.ClusterModeMGR),
	}

	// Metric with member_id label extraction
	metrics := []*model.MySQLMetricDefinition{
		{
			Name:         "server_id",
			DisplayName:  "Server ID",
			Query:        "mysql_innodb_cluster_mgr_role_primary",
			Category:     "info",
			ClusterMode:  "mgr",
			LabelExtract: "member_id",
		},
	}

	ctx := context.Background()
	results, err := collector.CollectMetrics(ctx, instances, metrics)

	if err != nil {
		t.Fatalf("CollectMetrics failed: %v", err)
	}

	result, ok := results["172.18.182.91:3306"]
	if !ok {
		t.Fatal("result for 172.18.182.91:3306 not found")
	}

	// Verify member_id label extracted as server_id
	serverIDMetric := result.GetMetric("server_id")
	if serverIDMetric == nil {
		t.Fatal("server_id metric not found")
	}

	if serverIDMetric.StringValue != "91abc-def1-2345-6789" {
		t.Errorf("StringValue = %q, want \"91abc-def1-2345-6789\"", serverIDMetric.StringValue)
	}

	// Check labels contain member_id
	if serverIDMetric.Labels["member_id"] != "91abc-def1-2345-6789" {
		t.Errorf("Labels[member_id] = %q, want \"91abc-def1-2345-6789\"", serverIDMetric.Labels["member_id"])
	}
}

// TestCollectMetrics_MissingLabelExtract tests handling of missing label
func TestCollectMetrics_MissingLabelExtract(t *testing.T) {
	server := setupMySQLVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")

		w.Header().Set("Content-Type", "application/json")
		timestamp := time.Now().Unix()

		var jsonResp string
		if contains(query, "mysql_version_info") {
			// Return WITHOUT version label
			jsonResp = fmt.Sprintf(`{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [
						{
							"metric": {
								"__name__": "mysql_version_info",
								"address": "172.18.182.91:3306"
							},
							"value": [%d, "1"]
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

	cfg := &config.MySQLInspectionConfig{
		ClusterMode: "mgr",
	}

	collector := createTestMySQLCollector(server.URL, cfg)

	instances := []*model.MySQLInstance{
		model.NewMySQLInstanceWithClusterMode("172.18.182.91:3306", model.ClusterModeMGR),
	}

	metrics := []*model.MySQLMetricDefinition{
		{
			Name:         "mysql_version",
			DisplayName:  "数据库版本",
			Query:        "mysql_version_info",
			Category:     "info",
			LabelExtract: "version",
		},
	}

	ctx := context.Background()
	results, err := collector.CollectMetrics(ctx, instances, metrics)

	if err != nil {
		t.Fatalf("CollectMetrics failed: %v", err)
	}

	result, ok := results["172.18.182.91:3306"]
	if !ok {
		t.Fatal("result for 172.18.182.91:3306 not found")
	}

	// Metric should not exist when label is missing
	versionMetric := result.GetMetric("mysql_version")
	if versionMetric != nil {
		t.Error("Expected no metric when label extraction fails, but found one")
	}
}

// =============================================================================
// Cluster Mode Filtering Tests
// =============================================================================

// TestCollectMetrics_ClusterModeFiltering tests cluster mode metric filtering
func TestCollectMetrics_ClusterModeFiltering(t *testing.T) {
	tests := []struct {
		name                   string
		clusterMode            string
		metrics                []*model.MySQLMetricDefinition
		expectedMetricNames    []string
		notExpectedMetricNames []string
	}{
		{
			name:        "MGR mode includes MGR metrics and universal metrics",
			clusterMode: "mgr",
			metrics: []*model.MySQLMetricDefinition{
				{
					Name:        "mgr_member_count",
					DisplayName: "MGR 成员数",
					Query:       "mysql_innodb_cluster_mgr_member_count",
					Category:    "mgr",
					ClusterMode: "mgr",
				},
				{
					Name:        "max_connections",
					DisplayName: "最大连接数",
					Query:       "mysql_global_variables_max_connections",
					Category:    "connection",
					ClusterMode: "", // Universal metric
				},
				{
					Name:        "slave_running",
					DisplayName: "Slave 是否启动",
					Query:       "mysql_slave_status_slave_running",
					Category:    "replication",
					ClusterMode: "master-slave",
				},
			},
			expectedMetricNames:    []string{"mgr_member_count", "max_connections"},
			notExpectedMetricNames: []string{"slave_running"},
		},
		{
			name:        "Master-Slave mode excludes MGR metrics",
			clusterMode: "master-slave",
			metrics: []*model.MySQLMetricDefinition{
				{
					Name:        "mgr_member_count",
					DisplayName: "MGR 成员数",
					Query:       "mysql_innodb_cluster_mgr_member_count",
					Category:    "mgr",
					ClusterMode: "mgr",
				},
				{
					Name:        "slave_running",
					DisplayName: "Slave 是否启动",
					Query:       "mysql_slave_status_slave_running",
					Category:    "replication",
					ClusterMode: "master-slave",
				},
				{
					Name:        "max_connections",
					DisplayName: "最大连接数",
					Query:       "mysql_global_variables_max_connections",
					Category:    "connection",
					ClusterMode: "", // Universal metric
				},
			},
			expectedMetricNames:    []string{"slave_running", "max_connections"},
			notExpectedMetricNames: []string{"mgr_member_count"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := setupMySQLVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				query := r.URL.Query().Get("query")

				w.Header().Set("Content-Type", "application/json")
				timestamp := time.Now().Unix()

				var jsonResp string

				// Match different metrics
				if contains(query, "mgr_member_count") {
					jsonResp = fmt.Sprintf(`{
						"status": "success",
						"data": {
							"resultType": "vector",
							"result": [
								{"metric": {"address": "172.18.182.91:3306"}, "value": [%d, "3"]}
							]
						}
					}`, timestamp)
				} else if contains(query, "slave_running") {
					jsonResp = fmt.Sprintf(`{
						"status": "success",
						"data": {
							"resultType": "vector",
							"result": [
								{"metric": {"address": "172.18.182.91:3306"}, "value": [%d, "1"]}
							]
						}
					}`, timestamp)
				} else if contains(query, "max_connections") {
					jsonResp = fmt.Sprintf(`{
						"status": "success",
						"data": {
							"resultType": "vector",
							"result": [
								{"metric": {"address": "172.18.182.91:3306"}, "value": [%d, "1000"]}
							]
						}
					}`, timestamp)
				} else {
					jsonResp = `{"status": "success", "data": {"resultType": "vector", "result": []}}`
				}

				w.Write([]byte(jsonResp))
			})
			defer server.Close()

			cfg := &config.MySQLInspectionConfig{
				ClusterMode: tt.clusterMode,
			}

			collector := createTestMySQLCollector(server.URL, cfg)

			instances := []*model.MySQLInstance{
				model.NewMySQLInstanceWithClusterMode("172.18.182.91:3306", model.MySQLClusterMode(tt.clusterMode)),
			}

			ctx := context.Background()
			results, err := collector.CollectMetrics(ctx, instances, tt.metrics)

			if err != nil {
				t.Fatalf("CollectMetrics failed: %v", err)
			}

			result, ok := results["172.18.182.91:3306"]
			if !ok {
				t.Fatal("result for 172.18.182.91:3306 not found")
			}

			// Check expected metrics are present
			for _, metricName := range tt.expectedMetricNames {
				metric := result.GetMetric(metricName)
				if metric == nil {
					t.Errorf("Expected metric %q not found in results", metricName)
				}
			}

			// Check not expected metrics are absent
			for _, metricName := range tt.notExpectedMetricNames {
				metric := result.GetMetric(metricName)
				if metric != nil {
					t.Errorf("Unexpected metric %q found in results (should be filtered)", metricName)
				}
			}
		})
	}
}
