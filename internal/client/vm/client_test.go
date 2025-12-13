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
