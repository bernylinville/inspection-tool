package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inspection-tool/internal/client/vm"
	"inspection-tool/internal/config"
	"inspection-tool/internal/model"
)

// =============================================================================
// Test Helper Functions
// =============================================================================

// createTestNginxInspector creates a NginxInspector for testing.
func createTestNginxInspector(t *testing.T, cfg *config.NginxInspectionConfig, metrics []*model.NginxMetricDefinition) (*NginxInspector, *httptest.Server) {
	t.Helper()

	// Create mock VM server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("query") {
		case "nginx_info":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [{
						"metric": {
							"agent_hostname": "test-host-01",
							"port": "80",
							"app_type": "nginx",
							"version": "1.20.1",
							"install_path": "/usr/local/nginx"
						},
						"value": [1703123456, "1"]
					}]
				}
			}`))
		case "nginx_up":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [{
						"metric": {
							"agent_hostname": "test-host-01"
						},
						"value": [1703123456, "1"]
					}]
				}
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	// Create VM client
	vmConfig := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	retryConfig := &config.RetryConfig{MaxRetries: 0}
	vmClient := vm.NewClient(vmConfig, retryConfig, zerolog.Nop())

	// Create collector
	collector := NewNginxCollector(cfg, vmClient, nil, metrics, zerolog.Nop())

	// Create evaluator
	evaluator := NewNginxEvaluator(&cfg.Thresholds, metrics, time.UTC, zerolog.Nop())

	// Create full config
	fullCfg := &config.Config{
		Nginx: *cfg,
		Report: config.ReportConfig{
			Timezone: "UTC",
		},
	}

	// Create inspector
	inspector, err := NewNginxInspector(fullCfg, collector, evaluator, zerolog.Nop())
	require.NoError(t, err)

	return inspector, server
}

// =============================================================================
// NewNginxInspector Tests
// =============================================================================

func TestNewNginxInspector(t *testing.T) {
	cfg := &config.NginxInspectionConfig{
		Enabled: true,
		Thresholds: config.NginxThresholds{
			ConnectionUsageWarning:   70,
			ConnectionUsageCritical:  90,
			LastErrorWarningMinutes:  60,
			LastErrorCriticalMinutes: 10,
		},
	}

	metrics := createTestNginxMetricDefs()
	inspector, server := createTestNginxInspector(t, cfg, metrics)
	defer server.Close()

	assert.NotNil(t, inspector)
	assert.Equal(t, "dev", inspector.GetVersion())
	assert.Equal(t, time.UTC, inspector.GetTimezone())
	assert.True(t, inspector.IsEnabled())
}

func TestNewNginxInspector_WithVersion(t *testing.T) {
	cfg := &config.NginxInspectionConfig{
		Enabled: true,
	}

	metrics := createTestNginxMetricDefs()
	inspector, server := createTestNginxInspector(t, cfg, metrics)
	defer server.Close()

	// Apply version option
	versionOpt := WithNginxVersion("v1.0.0")
	versionOpt(inspector)

	assert.Equal(t, "v1.0.0", inspector.GetVersion())
}

func TestNewNginxInspector_NilConfig(t *testing.T) {
	inspector, err := NewNginxInspector(nil, nil, nil, zerolog.Nop())
	assert.Error(t, err)
	assert.Nil(t, inspector)
	assert.Contains(t, err.Error(), "config cannot be nil")
}

func TestNewNginxInspector_NilCollector(t *testing.T) {
	cfg := &config.Config{}
	inspector, err := NewNginxInspector(cfg, nil, nil, zerolog.Nop())
	assert.Error(t, err)
	assert.Nil(t, inspector)
	assert.Contains(t, err.Error(), "collector cannot be nil")
}

func TestNewNginxInspector_NilEvaluator(t *testing.T) {
	cfg := &config.Config{}
	collector := &NginxCollector{}
	inspector, err := NewNginxInspector(cfg, collector, nil, zerolog.Nop())
	assert.Error(t, err)
	assert.Nil(t, inspector)
	assert.Contains(t, err.Error(), "evaluator cannot be nil")
}

func TestNewNginxInspector_InvalidTimezone(t *testing.T) {
	cfg := &config.Config{
		Nginx: config.NginxInspectionConfig{Enabled: true},
		Report: config.ReportConfig{
			Timezone: "Invalid/Timezone",
		},
	}

	inspector, err := NewNginxInspector(cfg, &NginxCollector{}, &NginxEvaluator{}, zerolog.Nop())
	assert.Error(t, err)
	assert.Nil(t, inspector)
	assert.Contains(t, err.Error(), "failed to load timezone")
}

// =============================================================================
// IsEnabled Tests
// =============================================================================

func TestNginxInspector_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		cfg      config.NginxInspectionConfig
		expected bool
	}{
		{
			name: "enabled",
			cfg: config.NginxInspectionConfig{
				Enabled: true,
			},
			expected: true,
		},
		{
			name: "disabled",
			cfg: config.NginxInspectionConfig{
				Enabled: false,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullCfg := &config.Config{
				Nginx: tt.cfg,
			}
			inspector := &NginxInspector{config: fullCfg}
			assert.Equal(t, tt.expected, inspector.IsEnabled())
		})
	}

	// Test with nil config
	inspector := &NginxInspector{config: nil}
	assert.False(t, inspector.IsEnabled())
}

// =============================================================================
// GetConfig Tests
// =============================================================================

func TestNginxInspector_GetConfig(t *testing.T) {
	cfg := config.NginxInspectionConfig{
		Enabled: true,
	}
	fullCfg := &config.Config{
		Nginx: cfg,
	}
	inspector := &NginxInspector{config: fullCfg}
	assert.Equal(t, &cfg, inspector.GetConfig())
}

func TestNginxInspector_GetConfig_Nil(t *testing.T) {
	inspector := &NginxInspector{config: nil}
	assert.Nil(t, inspector.GetConfig())
}

// =============================================================================
// Inspect Tests
// =============================================================================

func TestNginxInspector_Inspect_Success(t *testing.T) {
	cfg := &config.NginxInspectionConfig{
		Enabled: true,
		Thresholds: config.NginxThresholds{
			ConnectionUsageWarning:   70,
			ConnectionUsageCritical:  90,
			LastErrorWarningMinutes:  60,
			LastErrorCriticalMinutes: 10,
		},
	}

	metrics := createTestNginxMetricDefs()
	inspector, server := createTestNginxInspector(t, cfg, metrics)
	defer server.Close()

	// Execute inspection
	ctx := context.Background()
	result, err := inspector.Inspect(ctx)

	// Verify result
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.Summary.TotalInstances)
	assert.Greater(t, result.Duration, time.Duration(0))
	assert.Equal(t, "dev", result.Version)
}

func TestNginxInspector_Inspect_NoInstances(t *testing.T) {
	// Create a server that returns empty results
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": []
			}
		}`))
	}))
	defer server.Close()

	cfg := config.NginxInspectionConfig{Enabled: true}
	vmConfig := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	retryConfig := &config.RetryConfig{MaxRetries: 0}
	vmClient := vm.NewClient(vmConfig, retryConfig, zerolog.Nop())

	collector := NewNginxCollector(&cfg, vmClient, nil, createTestNginxMetricDefs(), zerolog.Nop())
	evaluator := NewNginxEvaluator(&cfg.Thresholds, createTestNginxMetricDefs(), time.UTC, zerolog.Nop())

	fullCfg := &config.Config{
		Nginx: cfg,
		Report: config.ReportConfig{Timezone: "UTC"},
	}

	inspector, err := NewNginxInspector(fullCfg, collector, evaluator, zerolog.Nop())
	require.NoError(t, err)

	ctx := context.Background()
	result, err := inspector.Inspect(ctx)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.Summary.TotalInstances)
}

func TestNginxInspector_Inspect_DiscoveryError(t *testing.T) {
	// Create a server that returns error for nginx_info
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := config.NginxInspectionConfig{Enabled: true}
	vmConfig := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	retryConfig := &config.RetryConfig{MaxRetries: 0}
	vmClient := vm.NewClient(vmConfig, retryConfig, zerolog.Nop())

	collector := NewNginxCollector(&cfg, vmClient, nil, createTestNginxMetricDefs(), zerolog.Nop())
	evaluator := NewNginxEvaluator(&cfg.Thresholds, createTestNginxMetricDefs(), time.UTC, zerolog.Nop())

	fullCfg := &config.Config{
		Nginx: cfg,
		Report: config.ReportConfig{Timezone: "UTC"},
	}

	inspector, err := NewNginxInspector(fullCfg, collector, evaluator, zerolog.Nop())
	require.NoError(t, err)

	ctx := context.Background()
	result, err := inspector.Inspect(ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "instance discovery failed")
}

func TestNginxInspector_Inspect_NoMetricsDefined(t *testing.T) {
	// This test ensures inspector properly validates metrics before discovery
	// We'll use a real collector but with empty metrics to hit the validation check

	// Create a mock server that returns valid nginx_info
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("query") == "nginx_info" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [{
						"metric": {
							"agent_hostname": "test-host-01",
							"port": "80"
						},
						"value": [1703123456, "1"]
					}]
				}
			}`))
		}
	}))
	defer server.Close()

	cfg := config.NginxInspectionConfig{Enabled: true}
	vmConfig := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	retryConfig := &config.RetryConfig{MaxRetries: 0}
	vmClient := vm.NewClient(vmConfig, retryConfig, zerolog.Nop())

	// Create collector with no metrics
	collector := NewNginxCollector(&cfg, vmClient, nil, []*model.NginxMetricDefinition{}, zerolog.Nop())
	evaluator := NewNginxEvaluator(&cfg.Thresholds, []*model.NginxMetricDefinition{}, time.UTC, zerolog.Nop())

	fullCfg := &config.Config{
		Nginx: cfg,
		Report: config.ReportConfig{Timezone: "UTC"},
	}

	inspector, err := NewNginxInspector(fullCfg, collector, evaluator, zerolog.Nop())
	require.NoError(t, err)

	ctx := context.Background()
	result, err := inspector.Inspect(ctx)

	// Should fail because metrics list is empty after discovery
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no Nginx metrics defined")
}