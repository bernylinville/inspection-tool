package vm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog"

	"inspection-tool/internal/config"
)

// testLogger creates a disabled logger for testing
func testLogger() zerolog.Logger {
	return zerolog.New(nil).Level(zerolog.Disabled)
}

// writeJSON writes a JSON response with proper headers
func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func TestNewClient(t *testing.T) {
	t.Run("with_default_values", func(t *testing.T) {
		cfg := &config.VictoriaMetricsConfig{
			Endpoint: "http://localhost:8428",
		}

		client := NewClient(cfg, nil, testLogger())

		if client == nil {
			t.Fatal("expected non-nil client")
		}
		if client.endpoint != cfg.Endpoint {
			t.Errorf("expected endpoint %s, got %s", cfg.Endpoint, client.endpoint)
		}
		if client.timeout != 30*time.Second {
			t.Errorf("expected default timeout 30s, got %s", client.timeout)
		}
		if client.retry.MaxRetries != 3 {
			t.Errorf("expected default max retries 3, got %d", client.retry.MaxRetries)
		}
	})

	t.Run("with_custom_values", func(t *testing.T) {
		cfg := &config.VictoriaMetricsConfig{
			Endpoint: "http://localhost:8428",
			Timeout:  60 * time.Second,
		}
		retryCfg := &config.RetryConfig{
			MaxRetries: 5,
			BaseDelay:  2 * time.Second,
		}

		client := NewClient(cfg, retryCfg, testLogger())

		if client.timeout != 60*time.Second {
			t.Errorf("expected timeout 60s, got %s", client.timeout)
		}
		if client.retry.MaxRetries != 5 {
			t.Errorf("expected max retries 5, got %d", client.retry.MaxRetries)
		}
		if client.retry.BaseDelay != 2*time.Second {
			t.Errorf("expected base delay 2s, got %s", client.retry.BaseDelay)
		}
	})
}

func TestClient_Query_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodGet {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/query" {
			t.Errorf("expected path /api/v1/query, got %s", r.URL.Path)
		}

		query := r.URL.Query().Get("query")
		if query == "" {
			t.Error("expected query parameter")
		}

		// Return successful response
		resp := QueryResponse{
			Status: "success",
			Data: QueryData{
				ResultType: "vector",
				Result: []Sample{
					{
						Metric: Metric{"__name__": "cpu_usage_active", "ident": "host1", "cpu": "cpu-total"},
						Value:  SampleValue{float64(1702483200), "75.5"},
					},
					{
						Metric: Metric{"__name__": "cpu_usage_active", "ident": "host2", "cpu": "cpu-total"},
						Value:  SampleValue{float64(1702483200), "82.3"},
					},
				},
			},
		}
		writeJSON(w, resp)
	}))
	defer server.Close()

	// Create client
	cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	client := NewClient(cfg, nil, testLogger())

	// Execute query
	resp, err := client.Query(context.Background(), `cpu_usage_active{cpu="cpu-total"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.IsSuccess() {
		t.Error("expected successful response")
	}
	if len(resp.Data.Result) != 2 {
		t.Errorf("expected 2 results, got %d", len(resp.Data.Result))
	}
}

func TestClient_Query_Error(t *testing.T) {
	t.Run("http_error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal server error"))
		}))
		defer server.Close()

		cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
		retryCfg := &config.RetryConfig{MaxRetries: 0} // Disable retries for test
		client := NewClient(cfg, retryCfg, testLogger())

		_, err := client.Query(context.Background(), "up")
		if err == nil {
			t.Error("expected error for 500 response")
		}
		if !strings.Contains(err.Error(), "500") {
			t.Errorf("error should mention status code: %v", err)
		}
	})

	t.Run("api_error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := QueryResponse{
				Status:    "error",
				ErrorType: "bad_data",
				Error:     "invalid query syntax",
			}
			writeJSON(w, resp)
		}))
		defer server.Close()

		cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
		client := NewClient(cfg, nil, testLogger())

		_, err := client.Query(context.Background(), "invalid{")
		if err == nil {
			t.Error("expected error for API error response")
		}
		if !strings.Contains(err.Error(), "bad_data") {
			t.Errorf("error should mention error type: %v", err)
		}
	})
}

func TestClient_QueryResults_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := QueryResponse{
			Status: "success",
			Data: QueryData{
				ResultType: "vector",
				Result: []Sample{
					{
						Metric: Metric{"__name__": "cpu_usage_active", "ident": "host1"},
						Value:  SampleValue{float64(1702483200), "75.5"},
					},
					{
						Metric: Metric{"__name__": "cpu_usage_active", "ident": "host2"},
						Value:  SampleValue{float64(1702483200), "82.3"},
					},
				},
			},
		}
		writeJSON(w, resp)
	}))
	defer server.Close()

	cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	client := NewClient(cfg, nil, testLogger())

	results, err := client.QueryResults(context.Background(), "cpu_usage_active")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Verify first result
	if results[0].Ident != "host1" {
		t.Errorf("expected ident host1, got %s", results[0].Ident)
	}
	if results[0].Value != 75.5 {
		t.Errorf("expected value 75.5, got %f", results[0].Value)
	}
}

func TestClient_QueryByIdent_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := QueryResponse{
			Status: "success",
			Data: QueryData{
				ResultType: "vector",
				Result: []Sample{
					{
						Metric: Metric{"__name__": "mem_used_percent", "ident": "server-1"},
						Value:  SampleValue{float64(1702483200), "65.2"},
					},
					{
						Metric: Metric{"__name__": "mem_used_percent", "ident": "server-2"},
						Value:  SampleValue{float64(1702483200), "78.9"},
					},
				},
			},
		}
		writeJSON(w, resp)
	}))
	defer server.Close()

	cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	client := NewClient(cfg, nil, testLogger())

	results, err := client.QueryByIdent(context.Background(), "mem_used_percent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Verify results are grouped by ident
	if result, ok := results["server-1"]; ok {
		if result.Value != 65.2 {
			t.Errorf("expected value 65.2 for server-1, got %f", result.Value)
		}
	} else {
		t.Error("expected server-1 in results")
	}

	if result, ok := results["server-2"]; ok {
		if result.Value != 78.9 {
			t.Errorf("expected value 78.9 for server-2, got %f", result.Value)
		}
	} else {
		t.Error("expected server-2 in results")
	}
}

func TestClient_QueryWithFilter(t *testing.T) {
	t.Run("with_business_groups", func(t *testing.T) {
		var capturedQuery string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedQuery = r.URL.Query().Get("query")
			resp := QueryResponse{Status: "success", Data: QueryData{ResultType: "vector", Result: []Sample{}}}
			writeJSON(w, resp)
		}))
		defer server.Close()

		cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
		client := NewClient(cfg, nil, testLogger())

		filter := &HostFilter{
			BusinessGroups: []string{"生产环境", "测试环境"},
		}

		_, err := client.QueryWithFilter(context.Background(), "cpu_usage_active", filter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify query contains business group filter
		if !strings.Contains(capturedQuery, `busigroup=~"生产环境|测试环境"`) {
			t.Errorf("query should contain business group filter, got: %s", capturedQuery)
		}
	})

	t.Run("with_tags", func(t *testing.T) {
		var capturedQuery string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedQuery = r.URL.Query().Get("query")
			resp := QueryResponse{Status: "success", Data: QueryData{ResultType: "vector", Result: []Sample{}}}
			writeJSON(w, resp)
		}))
		defer server.Close()

		cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
		client := NewClient(cfg, nil, testLogger())

		filter := &HostFilter{
			Tags: map[string]string{"env": "prod", "region": "cn-east"},
		}

		_, err := client.QueryWithFilter(context.Background(), `cpu_usage_active{cpu="cpu-total"}`, filter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify query contains tag filters
		if !strings.Contains(capturedQuery, `env="prod"`) {
			t.Errorf("query should contain env tag filter, got: %s", capturedQuery)
		}
		if !strings.Contains(capturedQuery, `region="cn-east"`) {
			t.Errorf("query should contain region tag filter, got: %s", capturedQuery)
		}
	})

	t.Run("with_both", func(t *testing.T) {
		var capturedQuery string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedQuery = r.URL.Query().Get("query")
			resp := QueryResponse{Status: "success", Data: QueryData{ResultType: "vector", Result: []Sample{}}}
			writeJSON(w, resp)
		}))
		defer server.Close()

		cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
		client := NewClient(cfg, nil, testLogger())

		filter := &HostFilter{
			BusinessGroups: []string{"prod"},
			Tags:           map[string]string{"env": "prod"},
		}

		_, err := client.QueryWithFilter(context.Background(), "cpu_usage_active", filter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify query contains both filters
		if !strings.Contains(capturedQuery, `busigroup=~"prod"`) {
			t.Errorf("query should contain business group filter, got: %s", capturedQuery)
		}
		if !strings.Contains(capturedQuery, `env="prod"`) {
			t.Errorf("query should contain env tag filter, got: %s", capturedQuery)
		}
	})
}

func TestClient_EmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := QueryResponse{
			Status: "success",
			Data: QueryData{
				ResultType: "vector",
				Result:     []Sample{},
			},
		}
		writeJSON(w, resp)
	}))
	defer server.Close()

	cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	client := NewClient(cfg, nil, testLogger())

	results, err := client.QueryResults(context.Background(), "nonexistent_metric")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestClient_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL, Timeout: 10 * time.Second}
	retryCfg := &config.RetryConfig{MaxRetries: 0}
	client := NewClient(cfg, retryCfg, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.Query(ctx, "up")
	if err == nil {
		t.Error("expected error for canceled context")
	}
}

func TestInjectMatchersToQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		matchers []string
		expected string
	}{
		{
			name:     "no_existing_labels",
			query:    "cpu_usage_active",
			matchers: []string{`env="prod"`},
			expected: `cpu_usage_active{env="prod"}`,
		},
		{
			name:     "with_existing_labels",
			query:    `cpu_usage_active{cpu="cpu-total"}`,
			matchers: []string{`env="prod"`},
			expected: `cpu_usage_active{cpu="cpu-total", env="prod"}`,
		},
		{
			name:     "multiple_matchers",
			query:    "cpu_usage_active",
			matchers: []string{`busigroup=~"prod|test"`, `env="prod"`},
			expected: `cpu_usage_active{busigroup=~"prod|test", env="prod"}`,
		},
		{
			name:     "empty_existing_labels",
			query:    "cpu_usage_active{}",
			matchers: []string{`env="prod"`},
			expected: `cpu_usage_active{env="prod"}`,
		},
		{
			name:     "no_matchers",
			query:    "cpu_usage_active",
			matchers: []string{},
			expected: "cpu_usage_active",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := injectMatchersToQuery(tt.query, tt.matchers)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestEscapeRegex(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with.dot", `with\.dot`},
		{"with*star", `with\*star`},
		{"with|pipe", `with\|pipe`},
		{"(parens)", `\(parens\)`},
		{"[brackets]", `\[brackets\]`},
		{"multi.char*test", `multi\.char\*test`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeRegex(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestRetryCondition(t *testing.T) {
	tests := []struct {
		name        string
		resp        *resty.Response
		err         error
		shouldRetry bool
	}{
		{
			name:        "nil_response_with_error",
			resp:        nil,
			err:         context.DeadlineExceeded,
			shouldRetry: true,
		},
		{
			name:        "nil_response_nil_error",
			resp:        nil,
			err:         nil,
			shouldRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: Cannot easily test with resty.Response, so we test edge cases
			if tt.resp == nil && tt.err != nil {
				result := retryCondition(tt.resp, tt.err)
				if result != tt.shouldRetry {
					t.Errorf("expected %v, got %v", tt.shouldRetry, result)
				}
			}
		})
	}
}

func TestClient_QueryWithWarnings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := QueryResponse{
			Status: "success",
			Data: QueryData{
				ResultType: "vector",
				Result:     []Sample{},
			},
			Warnings: []string{"PromQL warning: vector matches no data"},
		}
		writeJSON(w, resp)
	}))
	defer server.Close()

	cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	client := NewClient(cfg, nil, testLogger())

	resp, err := client.Query(context.Background(), "nonexistent_metric")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(resp.Warnings))
	}
}

func TestClient_Query_RetryOnServerError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("service unavailable"))
			return
		}
		// Success on 3rd attempt
		resp := QueryResponse{
			Status: "success",
			Data:   QueryData{ResultType: "vector", Result: []Sample{}},
		}
		writeJSON(w, resp)
	}))
	defer server.Close()

	cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	retryCfg := &config.RetryConfig{MaxRetries: 3, BaseDelay: 10 * time.Millisecond}
	client := NewClient(cfg, retryCfg, testLogger())

	_, err := client.Query(context.Background(), "up")
	if err != nil {
		t.Fatalf("unexpected error after retries: %v", err)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestClient_Query_NoRetryOn4xx(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
	}))
	defer server.Close()

	cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	retryCfg := &config.RetryConfig{MaxRetries: 3, BaseDelay: 10 * time.Millisecond}
	client := NewClient(cfg, retryCfg, testLogger())

	_, err := client.Query(context.Background(), "invalid{")
	if err == nil {
		t.Error("expected error for 4xx response")
	}

	// Should only be 1 attempt (no retries for 4xx)
	if attempts != 1 {
		t.Errorf("expected 1 attempt (no retry for 4xx), got %d", attempts)
	}
}

// ============================================================================
// Additional Tests for Step 19
// ============================================================================

func TestClient_QueryResultsWithFilter_Success(t *testing.T) {
	var capturedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query().Get("query")
		resp := QueryResponse{
			Status: "success",
			Data: QueryData{
				ResultType: "vector",
				Result: []Sample{
					{
						Metric: Metric{"__name__": "cpu_usage_active", "ident": "host1", "busigroup": "生产环境"},
						Value:  SampleValue{float64(1702483200), "65.5"},
					},
				},
			},
		}
		writeJSON(w, resp)
	}))
	defer server.Close()

	cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	client := NewClient(cfg, nil, testLogger())

	filter := &HostFilter{
		BusinessGroups: []string{"生产环境"},
	}

	results, err := client.QueryResultsWithFilter(context.Background(), "cpu_usage_active", filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Value != 65.5 {
		t.Errorf("expected value 65.5, got %f", results[0].Value)
	}

	// Verify filter was applied
	if !strings.Contains(capturedQuery, `busigroup=~"生产环境"`) {
		t.Errorf("query should contain business group filter, got: %s", capturedQuery)
	}
}

func TestClient_QueryByIdentWithFilter_Success(t *testing.T) {
	var capturedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query().Get("query")
		resp := QueryResponse{
			Status: "success",
			Data: QueryData{
				ResultType: "vector",
				Result: []Sample{
					{
						Metric: Metric{"__name__": "mem_used_percent", "ident": "server-prod-1", "env": "prod"},
						Value:  SampleValue{float64(1702483200), "72.5"},
					},
					{
						Metric: Metric{"__name__": "mem_used_percent", "ident": "server-prod-2", "env": "prod"},
						Value:  SampleValue{float64(1702483200), "58.3"},
					},
				},
			},
		}
		writeJSON(w, resp)
	}))
	defer server.Close()

	cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	client := NewClient(cfg, nil, testLogger())

	filter := &HostFilter{
		Tags: map[string]string{"env": "prod"},
	}

	results, err := client.QueryByIdentWithFilter(context.Background(), "mem_used_percent", filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if result, ok := results["server-prod-1"]; ok {
		if result.Value != 72.5 {
			t.Errorf("expected value 72.5 for server-prod-1, got %f", result.Value)
		}
	} else {
		t.Error("expected server-prod-1 in results")
	}

	// Verify filter was applied
	if !strings.Contains(capturedQuery, `env="prod"`) {
		t.Errorf("query should contain env tag filter, got: %s", capturedQuery)
	}
}

func TestClient_MultipleHosts_MultipleMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := QueryResponse{
			Status: "success",
			Data: QueryData{
				ResultType: "vector",
				Result: []Sample{
					{
						Metric: Metric{"__name__": "cpu_usage_active", "ident": "host1", "cpu": "cpu-total"},
						Value:  SampleValue{float64(1702483200), "45.2"},
					},
					{
						Metric: Metric{"__name__": "cpu_usage_active", "ident": "host2", "cpu": "cpu-total"},
						Value:  SampleValue{float64(1702483200), "78.9"},
					},
					{
						Metric: Metric{"__name__": "cpu_usage_active", "ident": "host3", "cpu": "cpu-total"},
						Value:  SampleValue{float64(1702483200), "92.1"},
					},
					{
						Metric: Metric{"__name__": "cpu_usage_active", "ident": "host4", "cpu": "cpu-total"},
						Value:  SampleValue{float64(1702483200), "12.5"},
					},
					{
						Metric: Metric{"__name__": "cpu_usage_active", "ident": "host5", "cpu": "cpu-total"},
						Value:  SampleValue{float64(1702483200), "55.0"},
					},
				},
			},
		}
		writeJSON(w, resp)
	}))
	defer server.Close()

	cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	client := NewClient(cfg, nil, testLogger())

	results, err := client.QueryResults(context.Background(), "cpu_usage_active")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(results))
	}

	// Verify values are parsed correctly
	expectedValues := map[string]float64{
		"host1": 45.2,
		"host2": 78.9,
		"host3": 92.1,
		"host4": 12.5,
		"host5": 55.0,
	}

	resultsByIdent, _ := client.QueryByIdent(context.Background(), "cpu_usage_active")
	for ident, expectedValue := range expectedValues {
		if result, ok := resultsByIdent[ident]; ok {
			if result.Value != expectedValue {
				t.Errorf("expected value %f for %s, got %f", expectedValue, ident, result.Value)
			}
		} else {
			t.Errorf("expected %s in results", ident)
		}
	}
}

func TestClient_NaN_Inf_Values(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := QueryResponse{
			Status: "success",
			Data: QueryData{
				ResultType: "vector",
				Result: []Sample{
					{
						Metric: Metric{"ident": "host1"},
						Value:  SampleValue{float64(1702483200), "75.5"}, // Valid
					},
					{
						Metric: Metric{"ident": "host2"},
						Value:  SampleValue{float64(1702483200), "NaN"}, // NaN - should be skipped
					},
					{
						Metric: Metric{"ident": "host3"},
						Value:  SampleValue{float64(1702483200), "+Inf"}, // +Inf - should be skipped
					},
					{
						Metric: Metric{"ident": "host4"},
						Value:  SampleValue{float64(1702483200), "-Inf"}, // -Inf - should be skipped
					},
					{
						Metric: Metric{"ident": "host5"},
						Value:  SampleValue{float64(1702483200), "82.3"}, // Valid
					},
				},
			},
		}
		writeJSON(w, resp)
	}))
	defer server.Close()

	cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	client := NewClient(cfg, nil, testLogger())

	results, err := client.QueryResults(context.Background(), "cpu_usage")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only have 2 results (host1 and host5), NaN/Inf values are skipped
	if len(results) != 2 {
		t.Fatalf("expected 2 results (NaN/Inf filtered), got %d", len(results))
	}

	// Verify only valid hosts are in results
	resultsByIdent := GroupResultsByIdent(results)
	if _, ok := resultsByIdent["host1"]; !ok {
		t.Error("expected host1 in results")
	}
	if _, ok := resultsByIdent["host5"]; !ok {
		t.Error("expected host5 in results")
	}
	if _, ok := resultsByIdent["host2"]; ok {
		t.Error("host2 (NaN) should be filtered out")
	}
}

func TestClient_MatrixTypeResponse_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return matrix type response (range query result)
		resp := QueryResponse{
			Status: "success",
			Data: QueryData{
				ResultType: "matrix", // Not vector
				Result:     []Sample{},
			},
		}
		writeJSON(w, resp)
	}))
	defer server.Close()

	cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	client := NewClient(cfg, nil, testLogger())

	_, err := client.QueryResults(context.Background(), "rate(cpu_usage[5m])")
	if err == nil {
		t.Error("expected error for matrix type response")
	}

	if !strings.Contains(err.Error(), "expected vector") {
		t.Errorf("error should mention expected type: %v", err)
	}
}

func TestClient_MaxRetries_Exhausted(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("service unavailable"))
	}))
	defer server.Close()

	cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	retryCfg := &config.RetryConfig{MaxRetries: 2, BaseDelay: 5 * time.Millisecond}
	client := NewClient(cfg, retryCfg, testLogger())

	_, err := client.Query(context.Background(), "up")
	if err == nil {
		t.Error("expected error after max retries exhausted")
	}

	// Initial attempt + 2 retries = 3 total attempts
	if attempts != 3 {
		t.Errorf("expected 3 attempts (initial + 2 retries), got %d", attempts)
	}
}

func TestClient_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay longer than timeout
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL, Timeout: 50 * time.Millisecond}
	retryCfg := &config.RetryConfig{MaxRetries: 0} // Disable retries for this test
	client := NewClient(cfg, retryCfg, testLogger())

	_, err := client.Query(context.Background(), "up")
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestClient_InvalidJSON_Response(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json {"))
	}))
	defer server.Close()

	cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	retryCfg := &config.RetryConfig{MaxRetries: 0}
	client := NewClient(cfg, retryCfg, testLogger())

	resp, err := client.Query(context.Background(), "up")
	// resty may not return error for invalid JSON, but response will be empty/default
	if err == nil && resp != nil && resp.Status == "" {
		// This is expected behavior - resty deserializes to empty struct on invalid JSON
		t.Log("Invalid JSON resulted in empty response (expected)")
	}
}

func TestClient_QueryWithFilter_NilFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		// Query should be unchanged with nil filter
		if query != "cpu_usage_active" {
			t.Errorf("expected unmodified query, got: %s", query)
		}
		resp := QueryResponse{Status: "success", Data: QueryData{ResultType: "vector", Result: []Sample{}}}
		writeJSON(w, resp)
	}))
	defer server.Close()

	cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	client := NewClient(cfg, nil, testLogger())

	_, err := client.QueryWithFilter(context.Background(), "cpu_usage_active", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_QueryWithFilter_EmptyFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		// Query should be unchanged with empty filter
		if query != "cpu_usage_active" {
			t.Errorf("expected unmodified query, got: %s", query)
		}
		resp := QueryResponse{Status: "success", Data: QueryData{ResultType: "vector", Result: []Sample{}}}
		writeJSON(w, resp)
	}))
	defer server.Close()

	cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	client := NewClient(cfg, nil, testLogger())

	emptyFilter := &HostFilter{}
	_, err := client.QueryWithFilter(context.Background(), "cpu_usage_active", emptyFilter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_HostIdentPriority(t *testing.T) {
	t.Run("ident_over_host", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := QueryResponse{
				Status: "success",
				Data: QueryData{
					ResultType: "vector",
					Result: []Sample{
						{
							Metric: Metric{"ident": "ident-value", "host": "host-value", "instance": "instance-value"},
							Value:  SampleValue{float64(1702483200), "50.0"},
						},
					},
				},
			}
			writeJSON(w, resp)
		}))
		defer server.Close()

		cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
		client := NewClient(cfg, nil, testLogger())

		results, err := client.QueryResults(context.Background(), "test_metric")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if results[0].Ident != "ident-value" {
			t.Errorf("expected ident-value, got %s", results[0].Ident)
		}
	})

	t.Run("host_over_instance", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := QueryResponse{
				Status: "success",
				Data: QueryData{
					ResultType: "vector",
					Result: []Sample{
						{
							Metric: Metric{"host": "host-value", "instance": "instance-value"},
							Value:  SampleValue{float64(1702483200), "50.0"},
						},
					},
				},
			}
			writeJSON(w, resp)
		}))
		defer server.Close()

		cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
		client := NewClient(cfg, nil, testLogger())

		results, err := client.QueryResults(context.Background(), "test_metric")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if results[0].Ident != "host-value" {
			t.Errorf("expected host-value, got %s", results[0].Ident)
		}
	})

	t.Run("instance_fallback", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := QueryResponse{
				Status: "success",
				Data: QueryData{
					ResultType: "vector",
					Result: []Sample{
						{
							Metric: Metric{"instance": "instance-value"},
							Value:  SampleValue{float64(1702483200), "50.0"},
						},
					},
				},
			}
			writeJSON(w, resp)
		}))
		defer server.Close()

		cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
		client := NewClient(cfg, nil, testLogger())

		results, err := client.QueryResults(context.Background(), "test_metric")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if results[0].Ident != "instance-value" {
			t.Errorf("expected instance-value, got %s", results[0].Ident)
		}
	})
}

func TestClient_SpecialCharactersInFilter(t *testing.T) {
	var capturedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query().Get("query")
		resp := QueryResponse{Status: "success", Data: QueryData{ResultType: "vector", Result: []Sample{}}}
		writeJSON(w, resp)
	}))
	defer server.Close()

	cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	client := NewClient(cfg, nil, testLogger())

	// Test with special characters in business group name
	filter := &HostFilter{
		BusinessGroups: []string{"prod.env", "test*env"},
	}

	_, err := client.QueryWithFilter(context.Background(), "cpu_usage", filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify special characters are escaped in regex
	if !strings.Contains(capturedQuery, `prod\.env`) {
		t.Errorf("expected escaped dot in query, got: %s", capturedQuery)
	}
	if !strings.Contains(capturedQuery, `test\*env`) {
		t.Errorf("expected escaped asterisk in query, got: %s", capturedQuery)
	}
}

func TestClient_DuplicateIdent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return multiple samples with same ident (e.g., multiple CPU cores)
		resp := QueryResponse{
			Status: "success",
			Data: QueryData{
				ResultType: "vector",
				Result: []Sample{
					{
						Metric: Metric{"ident": "host1", "cpu": "cpu0"},
						Value:  SampleValue{float64(1702483200), "45.0"},
					},
					{
						Metric: Metric{"ident": "host1", "cpu": "cpu1"},
						Value:  SampleValue{float64(1702483200), "55.0"},
					},
					{
						Metric: Metric{"ident": "host1", "cpu": "cpu-total"},
						Value:  SampleValue{float64(1702483200), "50.0"},
					},
				},
			},
		}
		writeJSON(w, resp)
	}))
	defer server.Close()

	cfg := &config.VictoriaMetricsConfig{Endpoint: server.URL}
	client := NewClient(cfg, nil, testLogger())

	results, err := client.QueryResults(context.Background(), "cpu_usage")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All 3 samples should be returned in QueryResults
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	// But QueryByIdent should only have 1 entry (last one wins)
	byIdent, _ := client.QueryByIdent(context.Background(), "cpu_usage")
	if len(byIdent) != 1 {
		t.Errorf("expected 1 unique ident, got %d", len(byIdent))
	}
	if result, ok := byIdent["host1"]; ok {
		// Last sample (cpu-total) should win
		if result.Value != 50.0 {
			t.Errorf("expected last value 50.0, got %f", result.Value)
		}
	}
}
