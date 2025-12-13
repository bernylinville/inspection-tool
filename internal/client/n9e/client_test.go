package n9e

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog"

	"inspection-tool/internal/config"
)

// Sample extend_info for testing
const testExtendInfo = `{
	"cpu": {"cpu_cores": "4", "model_name": "Intel Xeon"},
	"memory": {"total": "16496934912"},
	"network": {"ipaddress": "192.168.1.100"},
	"platform": {"hostname": "test-host", "os": "GNU/Linux", "kernel_release": "5.14.0"},
	"filesystem": [{"kb_size": "103084600", "mounted_on": "/", "name": "/dev/sda1"}]
}`

// setupTestServer creates a test server and N9E client for testing.
func setupTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	t.Helper()
	server := httptest.NewServer(handler)
	cfg := &config.N9EConfig{
		Endpoint: server.URL,
		Token:    "test-token",
		Timeout:  5 * time.Second,
	}
	retryCfg := &config.RetryConfig{
		MaxRetries: 2,
		BaseDelay:  10 * time.Millisecond,
	}
	logger := zerolog.Nop()
	client := NewClient(cfg, retryCfg, logger)
	return server, client
}

// =============================================================================
// Basic Functionality Tests
// =============================================================================

func TestNewClient(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.N9EConfig
		retryCfg *config.RetryConfig
	}{
		{
			name: "with all config",
			cfg: &config.N9EConfig{
				Endpoint: "http://localhost:17000",
				Token:    "test-token",
				Timeout:  30 * time.Second,
			},
			retryCfg: &config.RetryConfig{
				MaxRetries: 5,
				BaseDelay:  2 * time.Second,
			},
		},
		{
			name: "with nil retry config",
			cfg: &config.N9EConfig{
				Endpoint: "http://localhost:17000",
				Token:    "test-token",
				Timeout:  30 * time.Second,
			},
			retryCfg: nil,
		},
		{
			name: "with zero timeout",
			cfg: &config.N9EConfig{
				Endpoint: "http://localhost:17000",
				Token:    "test-token",
				Timeout:  0,
			},
			retryCfg: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zerolog.Nop()
			client := NewClient(tt.cfg, tt.retryCfg, logger)

			if client == nil {
				t.Fatal("NewClient returned nil")
			}
			if client.endpoint != tt.cfg.Endpoint {
				t.Errorf("Expected endpoint '%s', got '%s'", tt.cfg.Endpoint, client.endpoint)
			}
			if client.token != tt.cfg.Token {
				t.Errorf("Expected token '%s', got '%s'", tt.cfg.Token, client.token)
			}
			if client.httpClient == nil {
				t.Error("HTTP client should not be nil")
			}

			// Verify default timeout when zero
			if tt.cfg.Timeout == 0 && client.timeout != 30*time.Second {
				t.Errorf("Expected default timeout 30s, got %v", client.timeout)
			}

			// Verify default retry config when nil
			if tt.retryCfg == nil && client.retry.MaxRetries != 3 {
				t.Errorf("Expected default max retries 3, got %d", client.retry.MaxRetries)
			}
		})
	}
}

func TestGetTargets_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Verify request path
		if r.URL.Path != "/api/n9e/targets" {
			t.Errorf("Expected path '/api/n9e/targets', got '%s'", r.URL.Path)
		}

		// Verify token header
		token := r.Header.Get("X-User-Token")
		if token != "test-token" {
			t.Errorf("Expected token 'test-token', got '%s'", token)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"dat": [
				{"ident": "host1", "extend_info": "{}"},
				{"ident": "host2", "extend_info": "{}"}
			],
			"err": ""
		}`))
	}

	server, client := setupTestServer(t, handler)
	defer server.Close()

	ctx := context.Background()
	targets, err := client.GetTargets(ctx)

	if err != nil {
		t.Fatalf("GetTargets failed: %v", err)
	}
	if len(targets) != 2 {
		t.Errorf("Expected 2 targets, got %d", len(targets))
	}
	if targets[0].Ident != "host1" {
		t.Errorf("Expected first target ident 'host1', got '%s'", targets[0].Ident)
	}
}

func TestGetTarget_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Verify request path
		expectedPath := "/api/n9e/target/test-host"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"dat": {"ident": "test-host", "extend_info": "` + escapeJSON(testExtendInfo) + `"},
			"err": ""
		}`))
	}

	server, client := setupTestServer(t, handler)
	defer server.Close()

	ctx := context.Background()
	target, err := client.GetTarget(ctx, "test-host")

	if err != nil {
		t.Fatalf("GetTarget failed: %v", err)
	}
	if target == nil {
		t.Fatal("GetTarget returned nil target")
	}
	if target.Ident != "test-host" {
		t.Errorf("Expected ident 'test-host', got '%s'", target.Ident)
	}
}

func TestGetHostMetas_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"dat": [
				{"ident": "host1", "extend_info": "` + escapeJSON(testExtendInfo) + `"}
			],
			"err": ""
		}`))
	}

	server, client := setupTestServer(t, handler)
	defer server.Close()

	ctx := context.Background()
	hosts, err := client.GetHostMetas(ctx)

	if err != nil {
		t.Fatalf("GetHostMetas failed: %v", err)
	}
	if len(hosts) != 1 {
		t.Errorf("Expected 1 host, got %d", len(hosts))
	}
	if hosts[0].Hostname != "test-host" {
		t.Errorf("Expected hostname 'test-host', got '%s'", hosts[0].Hostname)
	}
	if hosts[0].IP != "192.168.1.100" {
		t.Errorf("Expected IP '192.168.1.100', got '%s'", hosts[0].IP)
	}
}

func TestGetHostMetaByIdent_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"dat": {"ident": "test-host", "extend_info": "` + escapeJSON(testExtendInfo) + `"},
			"err": ""
		}`))
	}

	server, client := setupTestServer(t, handler)
	defer server.Close()

	ctx := context.Background()
	host, err := client.GetHostMetaByIdent(ctx, "test-host")

	if err != nil {
		t.Fatalf("GetHostMetaByIdent failed: %v", err)
	}
	if host == nil {
		t.Fatal("GetHostMetaByIdent returned nil host")
	}
	if host.Hostname != "test-host" {
		t.Errorf("Expected hostname 'test-host', got '%s'", host.Hostname)
	}
	if host.CPUCores != 4 {
		t.Errorf("Expected 4 CPU cores, got %d", host.CPUCores)
	}
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestGetTargets_Unauthorized(t *testing.T) {
	var requestCount int32

	handler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"err": "unauthorized"}`))
	}

	server, client := setupTestServer(t, handler)
	defer server.Close()

	ctx := context.Background()
	_, err := client.GetTargets(ctx)

	if err == nil {
		t.Error("Expected error for unauthorized request")
	}

	// 4xx errors should not trigger retries
	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("Expected 1 request (no retry for 4xx), got %d", requestCount)
	}
}

func TestGetTargets_NotFound(t *testing.T) {
	var requestCount int32

	handler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"err": "not found"}`))
	}

	server, client := setupTestServer(t, handler)
	defer server.Close()

	ctx := context.Background()
	_, err := client.GetTargets(ctx)

	if err == nil {
		t.Error("Expected error for not found")
	}

	// 4xx errors should not trigger retries
	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("Expected 1 request (no retry for 4xx), got %d", requestCount)
	}
}

func TestGetTargets_APIError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"dat": [],
			"err": "internal server error: database connection failed"
		}`))
	}

	server, client := setupTestServer(t, handler)
	defer server.Close()

	ctx := context.Background()
	_, err := client.GetTargets(ctx)

	if err == nil {
		t.Error("Expected error for API error response")
	}
	if err != nil && !contains(err.Error(), "database connection failed") {
		t.Errorf("Error message should contain API error: %v", err)
	}
}

func TestGetTarget_NotFound(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"err": "target not found"}`))
	}

	server, client := setupTestServer(t, handler)
	defer server.Close()

	ctx := context.Background()
	_, err := client.GetTarget(ctx, "non-existent-host")

	if err == nil {
		t.Error("Expected error for non-existent target")
	}
}

// =============================================================================
// Retry Mechanism Tests
// =============================================================================

func TestRetryCondition(t *testing.T) {
	tests := []struct {
		name        string
		response    *resty.Response
		err         error
		shouldRetry bool
	}{
		{
			name:        "retry on error",
			response:    nil,
			err:         context.DeadlineExceeded,
			shouldRetry: true,
		},
		{
			name:        "retry on 500",
			response:    mockResponse(500),
			err:         nil,
			shouldRetry: true,
		},
		{
			name:        "retry on 502",
			response:    mockResponse(502),
			err:         nil,
			shouldRetry: true,
		},
		{
			name:        "retry on 503",
			response:    mockResponse(503),
			err:         nil,
			shouldRetry: true,
		},
		{
			name:        "no retry on 400",
			response:    mockResponse(400),
			err:         nil,
			shouldRetry: false,
		},
		{
			name:        "no retry on 401",
			response:    mockResponse(401),
			err:         nil,
			shouldRetry: false,
		},
		{
			name:        "no retry on 404",
			response:    mockResponse(404),
			err:         nil,
			shouldRetry: false,
		},
		{
			name:        "no retry on 200",
			response:    mockResponse(200),
			err:         nil,
			shouldRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := retryCondition(tt.response, tt.err)
			if result != tt.shouldRetry {
				t.Errorf("retryCondition() = %v, want %v", result, tt.shouldRetry)
			}
		})
	}
}

func TestGetTargets_ServerError_Retry(t *testing.T) {
	var requestCount int32

	handler := func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count < 3 {
			// First two requests fail with 500
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"err": "server error"}`))
			return
		}
		// Third request succeeds
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"dat": [{"ident": "host1", "extend_info": "{}"}], "err": ""}`))
	}

	server, client := setupTestServer(t, handler)
	defer server.Close()

	ctx := context.Background()
	targets, err := client.GetTargets(ctx)

	if err != nil {
		t.Fatalf("GetTargets failed after retries: %v", err)
	}
	if len(targets) != 1 {
		t.Errorf("Expected 1 target, got %d", len(targets))
	}

	// Should have made 3 requests (2 retries + 1 success)
	if atomic.LoadInt32(&requestCount) != 3 {
		t.Errorf("Expected 3 requests (2 retries), got %d", requestCount)
	}
}

func TestGetTargets_4xx_NoRetry(t *testing.T) {
	var requestCount int32

	handler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"err": "bad request"}`))
	}

	server, client := setupTestServer(t, handler)
	defer server.Close()

	ctx := context.Background()
	_, err := client.GetTargets(ctx)

	if err == nil {
		t.Error("Expected error for bad request")
	}

	// 4xx errors should not trigger retries - only 1 request
	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("Expected 1 request (no retry for 4xx), got %d", requestCount)
	}
}

func TestGetTargets_MaxRetries_Exceeded(t *testing.T) {
	var requestCount int32

	handler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"err": "server error"}`))
	}

	server, client := setupTestServer(t, handler)
	defer server.Close()

	ctx := context.Background()
	_, err := client.GetTargets(ctx)

	if err == nil {
		t.Error("Expected error after max retries exceeded")
	}

	// With MaxRetries=2, should have made 3 requests (1 initial + 2 retries)
	if atomic.LoadInt32(&requestCount) != 3 {
		t.Errorf("Expected 3 requests (initial + 2 retries), got %d", requestCount)
	}
}

// =============================================================================
// Boundary Condition Tests
// =============================================================================

func TestGetTargets_EmptyList(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"dat": [], "err": ""}`))
	}

	server, client := setupTestServer(t, handler)
	defer server.Close()

	ctx := context.Background()
	targets, err := client.GetTargets(ctx)

	if err != nil {
		t.Fatalf("GetTargets failed: %v", err)
	}
	if targets == nil {
		t.Error("Expected empty slice, got nil")
	}
	if len(targets) != 0 {
		t.Errorf("Expected 0 targets, got %d", len(targets))
	}
}

func TestGetHostMetas_PartialFailure(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// First target has valid extend_info, second has invalid
		_, _ = w.Write([]byte(`{
			"dat": [
				{"ident": "host1", "extend_info": "` + escapeJSON(testExtendInfo) + `"},
				{"ident": "host2", "extend_info": "invalid json"}
			],
			"err": ""
		}`))
	}

	server, client := setupTestServer(t, handler)
	defer server.Close()

	ctx := context.Background()
	hosts, err := client.GetHostMetas(ctx)

	// Should not return error, just skip failed conversions
	if err != nil {
		t.Fatalf("GetHostMetas failed: %v", err)
	}
	// Should only have 1 successful conversion
	if len(hosts) != 1 {
		t.Errorf("Expected 1 host (partial success), got %d", len(hosts))
	}
	if hosts[0].Hostname != "test-host" {
		t.Errorf("Expected hostname 'test-host', got '%s'", hosts[0].Hostname)
	}
}

func TestGetTargets_ContextCanceled(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"dat": [], "err": ""}`))
	}

	server, client := setupTestServer(t, handler)
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel context immediately
	cancel()

	_, err := client.GetTargets(ctx)

	if err == nil {
		t.Error("Expected error for canceled context")
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

// escapeJSON escapes a JSON string for embedding in another JSON string.
func escapeJSON(s string) string {
	result := ""
	for _, c := range s {
		switch c {
		case '"':
			result += `\"`
		case '\\':
			result += `\\`
		case '\n':
			result += `\n`
		case '\r':
			result += `\r`
		case '\t':
			result += `\t`
		default:
			result += string(c)
		}
	}
	return result
}

// mockResponse creates a minimal mock resty.Response for testing retryCondition.
type mockRawResponse struct {
	statusCode int
}

func (m *mockRawResponse) StatusCode() int { return m.statusCode }

func mockResponse(statusCode int) *resty.Response {
	return &resty.Response{
		RawResponse: &http.Response{StatusCode: statusCode},
	}
}

// contains is defined in types.go and reused here
