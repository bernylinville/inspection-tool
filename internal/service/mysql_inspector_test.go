package service

import (
	"context"
	"encoding/json"
	"net/http"
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

// createMySQLTestConfig creates a test configuration for MySQL inspection.
func createMySQLTestConfig() *config.Config {
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
				MaxRetries: 3,
				BaseDelay:  100 * time.Millisecond,
			},
		},
		MySQL: config.MySQLInspectionConfig{
			Enabled:     true,
			ClusterMode: "mgr",
			Thresholds: config.MySQLThresholds{
				ConnectionUsageWarning:  70.0,
				ConnectionUsageCritical: 90.0,
				MGRMemberCountExpected:  3,
			},
		},
	}
}

// createMySQLTestMetrics creates test metric definitions for MySQL.
func createMySQLTestMetrics() []*model.MySQLMetricDefinition {
	return []*model.MySQLMetricDefinition{
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
		{
			Name:        "current_connections",
			DisplayName: "当前连接数",
			Query:       "mysql_global_status_threads_connected",
			Category:    "connection",
		},
		{
			Name:        "mgr_member_count",
			DisplayName: "MGR 成员数",
			Query:       "mysql_innodb_cluster_mgr_member_count",
			Category:    "mgr",
			ClusterMode: "mgr",
		},
		{
			Name:        "mgr_state_online",
			DisplayName: "MGR 在线状态",
			Query:       "mysql_innodb_cluster_mgr_state_online",
			Category:    "mgr",
			ClusterMode: "mgr",
		},
	}
}

// =============================================================================
// 测试用例 1-3：构造函数测试
// =============================================================================

func TestNewMySQLInspector(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createMySQLTestMetrics()

	vmServer := setupMySQLVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result":     []interface{}{},
			},
		})
	})
	defer vmServer.Close()

	cfg := createMySQLTestConfig()
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL

	vmClient := vm.NewClient(&cfg.Datasources.VictoriaMetrics, &cfg.HTTP.Retry, logger)
	collector := NewMySQLCollector(&cfg.MySQL, vmClient, metrics, logger)
	evaluator := NewMySQLEvaluator(&cfg.MySQL.Thresholds, metrics, logger)

	t.Run("basic construction", func(t *testing.T) {
		inspector, err := NewMySQLInspector(cfg, collector, evaluator, logger)
		if err != nil {
			t.Fatalf("NewMySQLInspector failed: %v", err)
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
		inspector, err := NewMySQLInspector(cfg, collector, evaluator, logger, WithMySQLVersion("v1.0.0"))
		if err != nil {
			t.Fatalf("NewMySQLInspector failed: %v", err)
		}
		if inspector.GetVersion() != "v1.0.0" {
			t.Errorf("expected version v1.0.0, got %s", inspector.GetVersion())
		}
	})

	t.Run("invalid timezone", func(t *testing.T) {
		badCfg := createMySQLTestConfig()
		badCfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL
		badCfg.Report.Timezone = "Invalid/Timezone"

		_, err := NewMySQLInspector(badCfg, collector, evaluator, logger)
		if err == nil {
			t.Error("expected error for invalid timezone")
		}
	})
}

// =============================================================================
// 测试用例 4：正常巡检流程
// =============================================================================

func TestMySQLInspector_Inspect_Success(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createMySQLTestMetrics()

	vmServer := setupMySQLVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		query := r.URL.Query().Get("query")
		var result []map[string]interface{}

		if contains(query, "mysql_up") {
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "172.18.182.91:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				},
			}
		} else if contains(query, "mysql_global_variables_max_connections") {
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "172.18.182.91:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "1000"},
				},
			}
		} else if contains(query, "mysql_global_status_threads_connected") {
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "172.18.182.91:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "100"},
				},
			}
		} else if contains(query, "mysql_innodb_cluster_mgr_member_count") {
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "172.18.182.91:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "3"},
				},
			}
		} else if contains(query, "mysql_innodb_cluster_mgr_state_online") {
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "172.18.182.91:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				},
			}
		} else {
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

	cfg := createMySQLTestConfig()
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL

	vmClient := vm.NewClient(&cfg.Datasources.VictoriaMetrics, &cfg.HTTP.Retry, logger)
	collector := NewMySQLCollector(&cfg.MySQL, vmClient, metrics, logger)
	evaluator := NewMySQLEvaluator(&cfg.MySQL.Thresholds, metrics, logger)

	inspector, err := NewMySQLInspector(cfg, collector, evaluator, logger, WithMySQLVersion("v1.0.0"))
	if err != nil {
		t.Fatalf("NewMySQLInspector failed: %v", err)
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
	if result.Summary.TotalInstances != 1 {
		t.Errorf("expected 1 total instance, got %d", result.Summary.TotalInstances)
	}
	if result.Summary.NormalInstances != 1 {
		t.Errorf("expected 1 normal instance, got %d", result.Summary.NormalInstances)
	}
	if result.Duration <= 0 {
		t.Error("expected positive duration")
	}
}

// =============================================================================
// 测试用例 5：无实例场景
// =============================================================================

func TestMySQLInspector_Inspect_NoInstances(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createMySQLTestMetrics()

	vmServer := setupMySQLVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
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

	cfg := createMySQLTestConfig()
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL

	vmClient := vm.NewClient(&cfg.Datasources.VictoriaMetrics, &cfg.HTTP.Retry, logger)
	collector := NewMySQLCollector(&cfg.MySQL, vmClient, metrics, logger)
	evaluator := NewMySQLEvaluator(&cfg.MySQL.Thresholds, metrics, logger)

	inspector, err := NewMySQLInspector(cfg, collector, evaluator, logger)
	if err != nil {
		t.Fatalf("NewMySQLInspector failed: %v", err)
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
// 测试用例 6：警告告警场景
// =============================================================================

func TestMySQLInspector_Inspect_WithWarning(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createMySQLTestMetrics()

	vmServer := setupMySQLVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		query := r.URL.Query().Get("query")
		var result []map[string]interface{}

		if contains(query, "mysql_up") {
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "172.18.182.91:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				},
			}
		} else if contains(query, "mysql_global_variables_max_connections") {
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "172.18.182.91:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "1000"},
				},
			}
		} else if contains(query, "mysql_global_status_threads_connected") {
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "172.18.182.91:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "750"}, // 75% usage
				},
			}
		} else if contains(query, "mysql_innodb_cluster_mgr_member_count") {
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "172.18.182.91:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "3"},
				},
			}
		} else if contains(query, "mysql_innodb_cluster_mgr_state_online") {
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "172.18.182.91:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				},
			}
		} else {
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

	cfg := createMySQLTestConfig()
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL

	vmClient := vm.NewClient(&cfg.Datasources.VictoriaMetrics, &cfg.HTTP.Retry, logger)
	collector := NewMySQLCollector(&cfg.MySQL, vmClient, metrics, logger)
	evaluator := NewMySQLEvaluator(&cfg.MySQL.Thresholds, metrics, logger)

	inspector, err := NewMySQLInspector(cfg, collector, evaluator, logger)
	if err != nil {
		t.Fatalf("NewMySQLInspector failed: %v", err)
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
// 测试用例 7：严重告警场景
// =============================================================================

func TestMySQLInspector_Inspect_WithCritical(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createMySQLTestMetrics()

	vmServer := setupMySQLVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		query := r.URL.Query().Get("query")
		var result []map[string]interface{}

		if contains(query, "mysql_up") {
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "172.18.182.91:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				},
			}
		} else if contains(query, "mysql_global_variables_max_connections") {
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "172.18.182.91:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "1000"},
				},
			}
		} else if contains(query, "mysql_global_status_threads_connected") {
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "172.18.182.91:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "950"}, // 95% usage
				},
			}
		} else if contains(query, "mysql_innodb_cluster_mgr_member_count") {
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "172.18.182.91:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "1"}, // Only 1 member (expect 3)
				},
			}
		} else {
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

	cfg := createMySQLTestConfig()
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL

	vmClient := vm.NewClient(&cfg.Datasources.VictoriaMetrics, &cfg.HTTP.Retry, logger)
	collector := NewMySQLCollector(&cfg.MySQL, vmClient, metrics, logger)
	evaluator := NewMySQLEvaluator(&cfg.MySQL.Thresholds, metrics, logger)

	inspector, err := NewMySQLInspector(cfg, collector, evaluator, logger)
	if err != nil {
		t.Fatalf("NewMySQLInspector failed: %v", err)
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
// 测试用例 8：多实例混合状态
// =============================================================================

func TestMySQLInspector_Inspect_MultipleInstances(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createMySQLTestMetrics()

	vmServer := setupMySQLVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		query := r.URL.Query().Get("query")
		var result []map[string]interface{}

		if contains(query, "mysql_up") {
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "172.18.182.91:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				},
				{
					"metric": map[string]string{"address": "172.18.182.92:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				},
				{
					"metric": map[string]string{"address": "172.18.182.93:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				},
			}
		} else if contains(query, "mysql_global_variables_max_connections") {
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "172.18.182.91:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "1000"},
				},
				{
					"metric": map[string]string{"address": "172.18.182.92:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "1000"},
				},
				{
					"metric": map[string]string{"address": "172.18.182.93:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "1000"},
				},
			}
		} else if contains(query, "mysql_global_status_threads_connected") {
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "172.18.182.91:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "100"}, // Normal
				},
				{
					"metric": map[string]string{"address": "172.18.182.92:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "750"}, // Warning
				},
				{
					"metric": map[string]string{"address": "172.18.182.93:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "950"}, // Critical
				},
			}
		} else if contains(query, "mysql_innodb_cluster_mgr_member_count") {
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "172.18.182.91:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "3"},
				},
				{
					"metric": map[string]string{"address": "172.18.182.92:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "3"},
				},
				{
					"metric": map[string]string{"address": "172.18.182.93:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "3"},
				},
			}
		} else if contains(query, "mysql_innodb_cluster_mgr_state_online") {
			result = []map[string]interface{}{
				{
					"metric": map[string]string{"address": "172.18.182.91:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				},
				{
					"metric": map[string]string{"address": "172.18.182.92:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				},
				{
					"metric": map[string]string{"address": "172.18.182.93:3306"},
					"value":  []interface{}{float64(time.Now().Unix()), "1"},
				},
			}
		} else {
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

	cfg := createMySQLTestConfig()
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL

	vmClient := vm.NewClient(&cfg.Datasources.VictoriaMetrics, &cfg.HTTP.Retry, logger)
	collector := NewMySQLCollector(&cfg.MySQL, vmClient, metrics, logger)
	evaluator := NewMySQLEvaluator(&cfg.MySQL.Thresholds, metrics, logger)

	inspector, err := NewMySQLInspector(cfg, collector, evaluator, logger)
	if err != nil {
		t.Fatalf("NewMySQLInspector failed: %v", err)
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
// 测试用例 9：实例发现失败
// =============================================================================

func TestMySQLInspector_Inspect_DiscoveryError(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createMySQLTestMetrics()

	vmServer := setupMySQLVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "error",
			"error":  "internal server error",
		})
	})
	defer vmServer.Close()

	cfg := createMySQLTestConfig()
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL

	vmClient := vm.NewClient(&cfg.Datasources.VictoriaMetrics, &cfg.HTTP.Retry, logger)
	collector := NewMySQLCollector(&cfg.MySQL, vmClient, metrics, logger)
	evaluator := NewMySQLEvaluator(&cfg.MySQL.Thresholds, metrics, logger)

	inspector, err := NewMySQLInspector(cfg, collector, evaluator, logger)
	if err != nil {
		t.Fatalf("NewMySQLInspector failed: %v", err)
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
// 测试用例 10：上下文取消
// =============================================================================

func TestMySQLInspector_Inspect_ContextCanceled(t *testing.T) {
	logger := zerolog.Nop()
	metrics := createMySQLTestMetrics()

	vmServer := setupMySQLVMTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Simulate slow response
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result":     []interface{}{},
			},
		})
	})
	defer vmServer.Close()

	cfg := createMySQLTestConfig()
	cfg.Datasources.VictoriaMetrics.Endpoint = vmServer.URL

	vmClient := vm.NewClient(&cfg.Datasources.VictoriaMetrics, &cfg.HTTP.Retry, logger)
	collector := NewMySQLCollector(&cfg.MySQL, vmClient, metrics, logger)
	evaluator := NewMySQLEvaluator(&cfg.MySQL.Thresholds, metrics, logger)

	inspector, err := NewMySQLInspector(cfg, collector, evaluator, logger)
	if err != nil {
		t.Fatalf("NewMySQLInspector failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := inspector.Inspect(ctx)
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
	if result != nil {
		t.Error("expected nil result on context cancel")
	}
}
