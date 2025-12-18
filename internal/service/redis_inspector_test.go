package service

import (
	"context"
	"encoding/json"
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
// Test Helpers
// =============================================================================

// createRedisTestConfig creates a test configuration for Redis inspection.
func createRedisTestConfig() *config.Config {
	return &config.Config{
		Datasources: config.DatasourcesConfig{
			VictoriaMetrics: config.VictoriaMetricsConfig{
				Endpoint: "http://localhost:8428",
			},
		},
		Report: config.ReportConfig{
			Timezone: "Asia/Shanghai",
		},
		HTTP: config.HTTPConfig{
			Retry: config.RetryConfig{
				MaxRetries: 0, // No retry in tests
				BaseDelay:  100 * time.Millisecond,
			},
		},
		Redis: config.RedisInspectionConfig{
			Enabled:     true,
			ClusterMode: "3m3s",
			Thresholds: config.RedisThresholds{
				ConnectionUsageWarning:  70.0,
				ConnectionUsageCritical: 90.0,
				ReplicationLagWarning:   1048576,  // 1MB
				ReplicationLagCritical:  10485760, // 10MB
			},
		},
	}
}

// createRedisTestMetrics creates test metric definitions for Redis.
func createRedisTestMetrics() []*model.RedisMetricDefinition {
	return []*model.RedisMetricDefinition{
		{Name: "redis_up", DisplayName: "连接状态", Query: "redis_up", Category: "connection"},
		{Name: "redis_cluster_enabled", DisplayName: "集群模式", Query: "redis_cluster_enabled", Category: "cluster"},
		{Name: "redis_master_link_status", DisplayName: "主从链接状态", Query: "redis_master_link_status", Category: "replication"},
		{Name: "redis_connected_clients", DisplayName: "当前连接数", Query: "redis_connected_clients", Category: "connection"},
		{Name: "redis_maxclients", DisplayName: "最大连接数", Query: "redis_maxclients", Category: "connection"},
		{Name: "redis_master_repl_offset", DisplayName: "Master 复制偏移量", Query: "redis_master_repl_offset", Category: "replication"},
		{Name: "redis_slave_repl_offset", DisplayName: "Slave 复制偏移量", Query: "redis_slave_repl_offset", Category: "replication"},
		{Name: "redis_uptime_in_seconds", DisplayName: "运行时间", Query: "redis_uptime_in_seconds", Category: "status"},
		{Name: "redis_master_port", DisplayName: "Master 端口", Query: "redis_master_port", Category: "replication"},
		{Name: "redis_connected_slaves", DisplayName: "连接的 Slave 数", Query: "redis_connected_slaves", Category: "replication"},
		{Name: "redis_version", DisplayName: "Redis 版本", Query: "", Category: "info", Status: "pending"},
		{Name: "non_root_user", DisplayName: "非 root 用户启动", Query: "", Category: "security", Status: "pending"},
	}
}

// setupRedisInspectorVMTestServer creates a mock VictoriaMetrics server for testing.
func setupRedisInspectorVMTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

// =============================================================================
// Test Case 1: Constructor Tests
// =============================================================================

func TestNewRedisInspector(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createRedisTestMetrics()

	vmServer := setupRedisInspectorVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result":     []interface{}{},
			},
		})
	})
	defer vmServer.Close()

	cfg := createRedisTestConfig()
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL

	vmClient := vm.NewClient(&cfg.Datasources.VictoriaMetrics, &cfg.HTTP.Retry, logger)
	collector := NewRedisCollector(&cfg.Redis, vmClient, metrics, logger)
	evaluator := NewRedisEvaluator(&cfg.Redis.Thresholds, metrics, logger)

	t.Run("basic construction", func(t *testing.T) {
		inspector, err := NewRedisInspector(cfg, collector, evaluator, logger)
		if err != nil {
			t.Fatalf("NewRedisInspector failed: %v", err)
		}
		if inspector == nil {
			t.Fatal("expected non-nil inspector")
		}
		if inspector.GetVersion() != "dev" {
			t.Errorf("expected default version 'dev', got %s", inspector.GetVersion())
		}
		if inspector.GetTimezone().String() != "Asia/Shanghai" {
			t.Errorf("expected timezone Asia/Shanghai, got %s", inspector.GetTimezone().String())
		}
	})

	t.Run("with version option", func(t *testing.T) {
		inspector, err := NewRedisInspector(cfg, collector, evaluator, logger, WithRedisVersion("v1.0.0"))
		if err != nil {
			t.Fatalf("NewRedisInspector failed: %v", err)
		}
		if inspector.GetVersion() != "v1.0.0" {
			t.Errorf("expected version v1.0.0, got %s", inspector.GetVersion())
		}
	})

	t.Run("invalid timezone", func(t *testing.T) {
		badCfg := createRedisTestConfig()
		badCfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL
		badCfg.Report.Timezone = "Invalid/Timezone"

		_, err := NewRedisInspector(badCfg, collector, evaluator, logger)
		if err == nil {
			t.Error("expected error for invalid timezone")
		}
	})
}

// =============================================================================
// Test Case 2: Nil Parameter Validation
// =============================================================================

func TestNewRedisInspector_NilParameters(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createRedisTestMetrics()

	vmServer := setupRedisInspectorVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result":     []interface{}{},
			},
		})
	})
	defer vmServer.Close()

	cfg := createRedisTestConfig()
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL

	vmClient := vm.NewClient(&cfg.Datasources.VictoriaMetrics, &cfg.HTTP.Retry, logger)
	collector := NewRedisCollector(&cfg.Redis, vmClient, metrics, logger)
	evaluator := NewRedisEvaluator(&cfg.Redis.Thresholds, metrics, logger)

	t.Run("nil config", func(t *testing.T) {
		_, err := NewRedisInspector(nil, collector, evaluator, logger)
		if err == nil {
			t.Error("expected error for nil config")
		}
	})

	t.Run("nil collector", func(t *testing.T) {
		_, err := NewRedisInspector(cfg, nil, evaluator, logger)
		if err == nil {
			t.Error("expected error for nil collector")
		}
	})

	t.Run("nil evaluator", func(t *testing.T) {
		_, err := NewRedisInspector(cfg, collector, nil, logger)
		if err == nil {
			t.Error("expected error for nil evaluator")
		}
	})
}

// =============================================================================
// Test Case 3: Successful Inspection Flow
// =============================================================================

func TestRedisInspector_Inspect_Success(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createRedisTestMetrics()

	vmServer := setupRedisInspectorVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		query := r.URL.Query().Get("query")

		// Define 6 instances (3 master + 3 slave)
		addresses := []string{
			"192.18.102.3:7001", // master
			"192.18.102.4:7001", // master
			"192.18.102.4:7000", // master
			"192.18.102.2:7000", // slave
			"192.18.102.2:7001", // slave
			"192.18.102.3:7000", // slave
		}
		roles := []string{"master", "master", "master", "slave", "slave", "slave"}

		var result []map[string]interface{}

		switch {
		case query == "redis_up":
			for i, addr := range addresses {
				result = append(result, map[string]interface{}{
					"metric": map[string]string{"address": addr, "replica_role": roles[i]},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				})
			}
		case query == "redis_connected_clients":
			for _, addr := range addresses {
				result = append(result, map[string]interface{}{
					"metric": map[string]string{"address": addr},
					"value":  []interface{}{float64(time.Now().Unix()), "100"},
				})
			}
		case query == "redis_maxclients":
			for _, addr := range addresses {
				result = append(result, map[string]interface{}{
					"metric": map[string]string{"address": addr},
					"value":  []interface{}{float64(time.Now().Unix()), "10000"},
				})
			}
		case query == "redis_cluster_enabled":
			for _, addr := range addresses {
				result = append(result, map[string]interface{}{
					"metric": map[string]string{"address": addr},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				})
			}
		case query == "redis_master_link_status":
			// Only slaves have this metric
			for i, addr := range addresses {
				if roles[i] == "slave" {
					result = append(result, map[string]interface{}{
						"metric": map[string]string{"address": addr},
						"value":  []interface{}{float64(time.Now().Unix()), "1"},
					})
				}
			}
		case query == "redis_connected_slaves":
			// Only masters have this metric
			for i, addr := range addresses {
				if roles[i] == "master" {
					result = append(result, map[string]interface{}{
						"metric": map[string]string{"address": addr},
						"value":  []interface{}{float64(time.Now().Unix()), "1"},
					})
				}
			}
		case query == "redis_master_repl_offset":
			for _, addr := range addresses {
				result = append(result, map[string]interface{}{
					"metric": map[string]string{"address": addr},
					"value":  []interface{}{float64(time.Now().Unix()), "650463763"},
				})
			}
		case query == "redis_slave_repl_offset":
			// Only slaves have this metric
			for i, addr := range addresses {
				if roles[i] == "slave" {
					result = append(result, map[string]interface{}{
						"metric": map[string]string{"address": addr},
						"value":  []interface{}{float64(time.Now().Unix()), "650463763"},
					})
				}
			}
		default:
			result = []map[string]interface{}{}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result":     result,
			},
		})
	})
	defer vmServer.Close()

	cfg := createRedisTestConfig()
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL

	vmClient := vm.NewClient(&cfg.Datasources.VictoriaMetrics, &cfg.HTTP.Retry, logger)
	collector := NewRedisCollector(&cfg.Redis, vmClient, metrics, logger)
	evaluator := NewRedisEvaluator(&cfg.Redis.Thresholds, metrics, logger)

	inspector, err := NewRedisInspector(cfg, collector, evaluator, logger, WithRedisVersion("v1.0.0"))
	if err != nil {
		t.Fatalf("NewRedisInspector failed: %v", err)
	}

	result, err := inspector.Inspect(context.Background())
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.Version != "v1.0.0" {
		t.Errorf("expected version v1.0.0, got %s", result.Version)
	}
	if result.Summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if result.Summary.TotalInstances != 6 {
		t.Errorf("expected 6 total instances, got %d", result.Summary.TotalInstances)
	}
	if result.Summary.NormalInstances != 6 {
		t.Errorf("expected 6 normal instances, got %d", result.Summary.NormalInstances)
	}
	if result.Duration <= 0 {
		t.Error("expected positive duration")
	}
}

// =============================================================================
// Test Case 4: Empty Instances
// =============================================================================

func TestRedisInspector_Inspect_NoInstances(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createRedisTestMetrics()

	vmServer := setupRedisInspectorVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result":     []interface{}{},
			},
		})
	})
	defer vmServer.Close()

	cfg := createRedisTestConfig()
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL

	vmClient := vm.NewClient(&cfg.Datasources.VictoriaMetrics, &cfg.HTTP.Retry, logger)
	collector := NewRedisCollector(&cfg.Redis, vmClient, metrics, logger)
	evaluator := NewRedisEvaluator(&cfg.Redis.Thresholds, metrics, logger)

	inspector, err := NewRedisInspector(cfg, collector, evaluator, logger)
	if err != nil {
		t.Fatalf("NewRedisInspector failed: %v", err)
	}

	result, err := inspector.Inspect(context.Background())
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}

	if result.Summary.TotalInstances != 0 {
		t.Errorf("expected 0 total instances, got %d", result.Summary.TotalInstances)
	}
	if result.Duration <= 0 {
		t.Error("expected positive duration even for empty result")
	}
}

// =============================================================================
// Test Case 5: Warning Alert - Connection Usage 75%
// =============================================================================

func TestRedisInspector_Inspect_WithWarning(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createRedisTestMetrics()

	vmServer := setupRedisInspectorVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		query := r.URL.Query().Get("query")
		var result []map[string]interface{}

		switch {
		case query == "redis_up":
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "192.18.102.3:7001", "replica_role": "master"},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				},
			}
		case query == "redis_connected_clients":
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "192.18.102.3:7001"},
					"value":  []interface{}{float64(time.Now().Unix()), "7500"}, // 75% of 10000
				},
			}
		case query == "redis_maxclients":
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "192.18.102.3:7001"},
					"value":  []interface{}{float64(time.Now().Unix()), "10000"},
				},
			}
		case query == "redis_connected_slaves":
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "192.18.102.3:7001"},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				},
			}
		default:
			result = []map[string]interface{}{}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result":     result,
			},
		})
	})
	defer vmServer.Close()

	cfg := createRedisTestConfig()
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL

	vmClient := vm.NewClient(&cfg.Datasources.VictoriaMetrics, &cfg.HTTP.Retry, logger)
	collector := NewRedisCollector(&cfg.Redis, vmClient, metrics, logger)
	evaluator := NewRedisEvaluator(&cfg.Redis.Thresholds, metrics, logger)

	inspector, err := NewRedisInspector(cfg, collector, evaluator, logger)
	if err != nil {
		t.Fatalf("NewRedisInspector failed: %v", err)
	}

	result, err := inspector.Inspect(context.Background())
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}

	if result.Summary.WarningInstances != 1 {
		t.Errorf("expected 1 warning instance, got %d", result.Summary.WarningInstances)
	}
	if result.AlertSummary.WarningCount < 1 {
		t.Errorf("expected at least 1 warning alert, got %d", result.AlertSummary.WarningCount)
	}
}

// =============================================================================
// Test Case 6: Critical Alert - Connection Usage 95%
// =============================================================================

func TestRedisInspector_Inspect_WithCritical(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createRedisTestMetrics()

	vmServer := setupRedisInspectorVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		query := r.URL.Query().Get("query")
		var result []map[string]interface{}

		switch {
		case query == "redis_up":
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "192.18.102.3:7001", "replica_role": "master"},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				},
			}
		case query == "redis_connected_clients":
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "192.18.102.3:7001"},
					"value":  []interface{}{float64(time.Now().Unix()), "9500"}, // 95% of 10000
				},
			}
		case query == "redis_maxclients":
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "192.18.102.3:7001"},
					"value":  []interface{}{float64(time.Now().Unix()), "10000"},
				},
			}
		case query == "redis_connected_slaves":
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "192.18.102.3:7001"},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				},
			}
		default:
			result = []map[string]interface{}{}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result":     result,
			},
		})
	})
	defer vmServer.Close()

	cfg := createRedisTestConfig()
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL

	vmClient := vm.NewClient(&cfg.Datasources.VictoriaMetrics, &cfg.HTTP.Retry, logger)
	collector := NewRedisCollector(&cfg.Redis, vmClient, metrics, logger)
	evaluator := NewRedisEvaluator(&cfg.Redis.Thresholds, metrics, logger)

	inspector, err := NewRedisInspector(cfg, collector, evaluator, logger)
	if err != nil {
		t.Fatalf("NewRedisInspector failed: %v", err)
	}

	result, err := inspector.Inspect(context.Background())
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}

	if result.Summary.CriticalInstances != 1 {
		t.Errorf("expected 1 critical instance, got %d", result.Summary.CriticalInstances)
	}
	if result.AlertSummary.CriticalCount < 1 {
		t.Errorf("expected at least 1 critical alert, got %d", result.AlertSummary.CriticalCount)
	}
}

// =============================================================================
// Test Case 7: Multiple Instances with Mixed Status
// =============================================================================

func TestRedisInspector_Inspect_MultipleInstances(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createRedisTestMetrics()

	vmServer := setupRedisInspectorVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		query := r.URL.Query().Get("query")
		var result []map[string]interface{}

		addresses := []string{"192.18.102.3:7001", "192.18.102.4:7001", "192.18.102.4:7000"}
		connectionCounts := []string{"1000", "7500", "9500"} // Normal, Warning, Critical

		switch {
		case query == "redis_up":
			for _, addr := range addresses {
				result = append(result, map[string]interface{}{
					"metric": map[string]string{"address": addr, "replica_role": "master"},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				})
			}
		case query == "redis_connected_clients":
			for i, addr := range addresses {
				result = append(result, map[string]interface{}{
					"metric": map[string]string{"address": addr},
					"value":  []interface{}{float64(time.Now().Unix()), connectionCounts[i]},
				})
			}
		case query == "redis_maxclients":
			for _, addr := range addresses {
				result = append(result, map[string]interface{}{
					"metric": map[string]string{"address": addr},
					"value":  []interface{}{float64(time.Now().Unix()), "10000"},
				})
			}
		case query == "redis_connected_slaves":
			for _, addr := range addresses {
				result = append(result, map[string]interface{}{
					"metric": map[string]string{"address": addr},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				})
			}
		default:
			result = []map[string]interface{}{}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result":     result,
			},
		})
	})
	defer vmServer.Close()

	cfg := createRedisTestConfig()
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL

	vmClient := vm.NewClient(&cfg.Datasources.VictoriaMetrics, &cfg.HTTP.Retry, logger)
	collector := NewRedisCollector(&cfg.Redis, vmClient, metrics, logger)
	evaluator := NewRedisEvaluator(&cfg.Redis.Thresholds, metrics, logger)

	inspector, err := NewRedisInspector(cfg, collector, evaluator, logger)
	if err != nil {
		t.Fatalf("NewRedisInspector failed: %v", err)
	}

	result, err := inspector.Inspect(context.Background())
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}

	if result.Summary.TotalInstances != 3 {
		t.Errorf("expected 3 total instances, got %d", result.Summary.TotalInstances)
	}
	if result.Summary.NormalInstances != 1 {
		t.Errorf("expected 1 normal instance, got %d", result.Summary.NormalInstances)
	}
	if result.Summary.WarningInstances != 1 {
		t.Errorf("expected 1 warning instance, got %d", result.Summary.WarningInstances)
	}
	if result.Summary.CriticalInstances != 1 {
		t.Errorf("expected 1 critical instance, got %d", result.Summary.CriticalInstances)
	}
}

// =============================================================================
// Test Case 8: Discovery Error
// =============================================================================

func TestRedisInspector_Inspect_DiscoveryError(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createRedisTestMetrics()

	vmServer := setupRedisInspectorVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "error",
			"error":  "internal server error",
		})
	})
	defer vmServer.Close()

	cfg := createRedisTestConfig()
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL

	vmClient := vm.NewClient(&cfg.Datasources.VictoriaMetrics, &cfg.HTTP.Retry, logger)
	collector := NewRedisCollector(&cfg.Redis, vmClient, metrics, logger)
	evaluator := NewRedisEvaluator(&cfg.Redis.Thresholds, metrics, logger)

	inspector, err := NewRedisInspector(cfg, collector, evaluator, logger)
	if err != nil {
		t.Fatalf("NewRedisInspector failed: %v", err)
	}

	result, err := inspector.Inspect(context.Background())
	if err == nil {
		t.Fatal("expected error for discovery failure, got nil")
	}
	if result != nil {
		t.Error("expected nil result on discovery error")
	}
}

// =============================================================================
// Test Case 9: Context Canceled
// =============================================================================

func TestRedisInspector_Inspect_ContextCanceled(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createRedisTestMetrics()

	vmServer := setupRedisInspectorVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result":     []interface{}{},
			},
		})
	})
	defer vmServer.Close()

	cfg := createRedisTestConfig()
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL

	vmClient := vm.NewClient(&cfg.Datasources.VictoriaMetrics, &cfg.HTTP.Retry, logger)
	collector := NewRedisCollector(&cfg.Redis, vmClient, metrics, logger)
	evaluator := NewRedisEvaluator(&cfg.Redis.Thresholds, metrics, logger)

	inspector, err := NewRedisInspector(cfg, collector, evaluator, logger)
	if err != nil {
		t.Fatalf("NewRedisInspector failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := inspector.Inspect(ctx)
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
	if result != nil {
		t.Error("expected nil result on context cancel")
	}
}

// =============================================================================
// Test Case 10: Slave Master Link Status Down (Critical)
// =============================================================================

func TestRedisInspector_Inspect_SlaveMasterLinkDown(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createRedisTestMetrics()

	vmServer := setupRedisInspectorVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		query := r.URL.Query().Get("query")
		var result []map[string]interface{}

		switch {
		case query == "redis_up":
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "192.18.102.2:7000", "replica_role": "slave"},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				},
			}
		case query == "redis_connected_clients":
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "192.18.102.2:7000"},
					"value":  []interface{}{float64(time.Now().Unix()), "100"},
				},
			}
		case query == "redis_maxclients":
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "192.18.102.2:7000"},
					"value":  []interface{}{float64(time.Now().Unix()), "10000"},
				},
			}
		case query == "redis_master_link_status":
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "192.18.102.2:7000"},
					"value":  []interface{}{float64(time.Now().Unix()), "0"}, // Link down = Critical
				},
			}
		default:
			result = []map[string]interface{}{}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result":     result,
			},
		})
	})
	defer vmServer.Close()

	cfg := createRedisTestConfig()
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL

	vmClient := vm.NewClient(&cfg.Datasources.VictoriaMetrics, &cfg.HTTP.Retry, logger)
	collector := NewRedisCollector(&cfg.Redis, vmClient, metrics, logger)
	evaluator := NewRedisEvaluator(&cfg.Redis.Thresholds, metrics, logger)

	inspector, err := NewRedisInspector(cfg, collector, evaluator, logger)
	if err != nil {
		t.Fatalf("NewRedisInspector failed: %v", err)
	}

	result, err := inspector.Inspect(context.Background())
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}

	if result.Summary.CriticalInstances != 1 {
		t.Errorf("expected 1 critical instance, got %d", result.Summary.CriticalInstances)
	}
	if result.AlertSummary.CriticalCount < 1 {
		t.Errorf("expected at least 1 critical alert for master link down, got %d", result.AlertSummary.CriticalCount)
	}
}

// =============================================================================
// Test Case 11: Replication Lag Warning
// =============================================================================

func TestRedisInspector_Inspect_ReplicationLagWarning(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createRedisTestMetrics()

	vmServer := setupRedisInspectorVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		query := r.URL.Query().Get("query")
		var result []map[string]interface{}

		switch {
		case query == "redis_up":
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "192.18.102.2:7000", "replica_role": "slave"},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				},
			}
		case query == "redis_connected_clients":
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "192.18.102.2:7000"},
					"value":  []interface{}{float64(time.Now().Unix()), "100"},
				},
			}
		case query == "redis_maxclients":
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "192.18.102.2:7000"},
					"value":  []interface{}{float64(time.Now().Unix()), "10000"},
				},
			}
		case query == "redis_master_link_status":
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "192.18.102.2:7000"},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				},
			}
		case query == "redis_master_repl_offset":
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "192.18.102.2:7000"},
					"value":  []interface{}{float64(time.Now().Unix()), "652000000"}, // Master offset known by slave
				},
			}
		case query == "redis_slave_repl_offset":
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "192.18.102.2:7000"},
					"value":  []interface{}{float64(time.Now().Unix()), "650000000"}, // Lag = 2MB > 1MB warning
				},
			}
		default:
			result = []map[string]interface{}{}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result":     result,
			},
		})
	})
	defer vmServer.Close()

	cfg := createRedisTestConfig()
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL

	vmClient := vm.NewClient(&cfg.Datasources.VictoriaMetrics, &cfg.HTTP.Retry, logger)
	collector := NewRedisCollector(&cfg.Redis, vmClient, metrics, logger)
	evaluator := NewRedisEvaluator(&cfg.Redis.Thresholds, metrics, logger)

	inspector, err := NewRedisInspector(cfg, collector, evaluator, logger)
	if err != nil {
		t.Fatalf("NewRedisInspector failed: %v", err)
	}

	result, err := inspector.Inspect(context.Background())
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}

	if result.Summary.WarningInstances != 1 {
		t.Errorf("expected 1 warning instance, got %d", result.Summary.WarningInstances)
	}
	if result.AlertSummary.WarningCount < 1 {
		t.Errorf("expected at least 1 warning alert for replication lag, got %d", result.AlertSummary.WarningCount)
	}
}

// =============================================================================
// Test Case 12: Full 3M3S Cluster - Normal
// =============================================================================

func TestRedisInspector_Inspect_3M3SNormalCluster(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createRedisTestMetrics()

	vmServer := setupRedisInspectorVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		query := r.URL.Query().Get("query")

		// Real topology: 3 masters + 3 slaves
		masters := []string{"192.18.102.3:7001", "192.18.102.4:7001", "192.18.102.4:7000"}
		slaves := []string{"192.18.102.2:7000", "192.18.102.2:7001", "192.18.102.3:7000"}

		var result []map[string]interface{}

		switch {
		case query == "redis_up":
			for _, addr := range masters {
				result = append(result, map[string]interface{}{
					"metric": map[string]string{"address": addr, "replica_role": "master"},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				})
			}
			for _, addr := range slaves {
				result = append(result, map[string]interface{}{
					"metric": map[string]string{"address": addr, "replica_role": "slave"},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				})
			}
		case query == "redis_connected_clients":
			for _, addr := range append(masters, slaves...) {
				result = append(result, map[string]interface{}{
					"metric": map[string]string{"address": addr},
					"value":  []interface{}{float64(time.Now().Unix()), "100"},
				})
			}
		case query == "redis_maxclients":
			for _, addr := range append(masters, slaves...) {
				result = append(result, map[string]interface{}{
					"metric": map[string]string{"address": addr},
					"value":  []interface{}{float64(time.Now().Unix()), "10000"},
				})
			}
		case query == "redis_cluster_enabled":
			for _, addr := range append(masters, slaves...) {
				result = append(result, map[string]interface{}{
					"metric": map[string]string{"address": addr},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				})
			}
		case query == "redis_connected_slaves":
			for _, addr := range masters {
				result = append(result, map[string]interface{}{
					"metric": map[string]string{"address": addr},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				})
			}
		case query == "redis_master_link_status":
			for _, addr := range slaves {
				result = append(result, map[string]interface{}{
					"metric": map[string]string{"address": addr},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				})
			}
		case query == "redis_master_repl_offset":
			for _, addr := range append(masters, slaves...) {
				result = append(result, map[string]interface{}{
					"metric": map[string]string{"address": addr},
					"value":  []interface{}{float64(time.Now().Unix()), "650463763"},
				})
			}
		case query == "redis_slave_repl_offset":
			for _, addr := range slaves {
				result = append(result, map[string]interface{}{
					"metric": map[string]string{"address": addr},
					"value":  []interface{}{float64(time.Now().Unix()), "650463763"}, // No lag
				})
			}
		default:
			result = []map[string]interface{}{}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result":     result,
			},
		})
	})
	defer vmServer.Close()

	cfg := createRedisTestConfig()
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL

	vmClient := vm.NewClient(&cfg.Datasources.VictoriaMetrics, &cfg.HTTP.Retry, logger)
	collector := NewRedisCollector(&cfg.Redis, vmClient, metrics, logger)
	evaluator := NewRedisEvaluator(&cfg.Redis.Thresholds, metrics, logger)

	inspector, err := NewRedisInspector(cfg, collector, evaluator, logger)
	if err != nil {
		t.Fatalf("NewRedisInspector failed: %v", err)
	}

	result, err := inspector.Inspect(context.Background())
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}

	// Verify: 6 normal instances (3 master + 3 slave), no alerts
	if result.Summary.TotalInstances != 6 {
		t.Errorf("expected 6 total instances, got %d", result.Summary.TotalInstances)
	}
	if result.Summary.NormalInstances != 6 {
		t.Errorf("expected 6 normal instances, got %d", result.Summary.NormalInstances)
	}
	if result.Summary.WarningInstances != 0 {
		t.Errorf("expected 0 warning instances, got %d", result.Summary.WarningInstances)
	}
	if result.Summary.CriticalInstances != 0 {
		t.Errorf("expected 0 critical instances, got %d", result.Summary.CriticalInstances)
	}
	if result.AlertSummary.TotalAlerts != 0 {
		t.Errorf("expected 0 total alerts, got %d", result.AlertSummary.TotalAlerts)
	}

	// Count master and slave roles
	masterCount := 0
	slaveCount := 0
	for _, r := range result.Results {
		if r.Instance.Role == model.RedisRoleMaster {
			masterCount++
		} else if r.Instance.Role == model.RedisRoleSlave {
			slaveCount++
		}
	}

	if masterCount != 3 {
		t.Errorf("expected 3 masters, got %d", masterCount)
	}
	if slaveCount != 3 {
		t.Errorf("expected 3 slaves, got %d", slaveCount)
	}
}
