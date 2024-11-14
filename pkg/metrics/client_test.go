package metrics

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	baseURL := "http://localhost:8428"
	timeout := 30
	client := NewClient(baseURL, timeout)
	if client.baseURL != baseURL {
		t.Errorf("期望 baseURL 为 %s，实际得到 %s", baseURL, client.baseURL)
	}
}

func TestGetMetrics(t *testing.T) {
	// 创建模拟的 VictoriaMetrics 服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 模拟 VictoriaMetrics 的响应
		response := `{
			"status":"success",
			"data":{
				"resultType":"matrix",
				"result":[{
					"metric":{"instance":"host1:9100"},
					"values":[[1625097600,"80"]]
				}]
			}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewClient(server.URL, 30)
	metrics, err := client.GetMetrics(QueryOptions{
		Start: "now-1h",
		End:   "now",
	})

	if err != nil {
		t.Errorf("获取指标时发生错误: %v", err)
	}

	if len(metrics) == 0 {
		t.Error("期望获取到指标数据，但是结果为空")
	}
}

func TestQueryMetric(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求参数
		if r.URL.Query().Get("query") == "" {
			t.Error("缺少 query 参数")
		}
		if r.URL.Query().Get("start") == "" {
			t.Error("缺少 start 参数")
		}
		if r.URL.Query().Get("end") == "" {
			t.Error("缺少 end 参数")
		}

		// 返回测试数据
		response := `{
			"status":"success",
			"data":{
				"resultType":"matrix",
				"result":[{
					"metric":{"instance":"host1:9100"},
					"values":[[1625097600,"80"]]
				}]
			}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewClient(server.URL, 30)
	data, err := client.queryMetric("test_query", "now-1h", "now")

	if err != nil {
		t.Errorf("查询指标时发生错误: %v", err)
	}

	if len(data) == 0 {
		t.Error("期望获取到数据，但是结果为空")
	}
}

func TestBuildLabelSelector(t *testing.T) {
	tests := []struct {
		name     string
		labels   []string
		expected string
	}{
		{
			name:     "空标签",
			labels:   []string{},
			expected: "",
		},
		{
			name:     "单个标签",
			labels:   []string{"env=prod"},
			expected: "{env=prod}",
		},
		{
			name:     "多个标签",
			labels:   []string{"env=prod", "region=cn"},
			expected: "{env=prod,region=cn}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildLabelSelector(tt.labels)
			if result != tt.expected {
				t.Errorf("buildLabelSelector() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetMetricsWithLabels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求中是否包含标签
		query := r.URL.Query().Get("query")
		if !strings.Contains(query, "{env=prod}") {
			t.Error("查询中未包含标签")
		}

		response := `{
			"status":"success",
			"data":{
				"resultType":"matrix",
				"result":[{
					"metric":{"host":"host1:9100", "env":"prod"},
					"values":[[1625097600,"80"]]
				}]
			}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewClient(server.URL, 30)
	opts := QueryOptions{
		Start:  "now-1h",
		End:    "now",
		Labels: []string{"env=prod"},
	}

	metrics, err := client.GetMetrics(opts)
	if err != nil {
		t.Errorf("获取指标时发生错误: %v", err)
	}

	if len(metrics) == 0 {
		t.Error("期望获取到指标数据，但是结果为空")
	}
}

// 添加并发查询测试

func TestConcurrentQueries(t *testing.T) {
	// 设置测试超时
	if testing.Short() {
		t.Skip("跳过并发测试")
	}

	// 创建一个计数器来跟踪并发请求
	requestCount := 0
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		currentCount := requestCount
		mu.Unlock()

		// 减少模拟延迟时间
		time.Sleep(time.Duration(10+rand.Intn(20)) * time.Millisecond)

		response := fmt.Sprintf(`{
			"status":"success",
			"data":{
				"resultType":"matrix",
				"result":[{
					"metric":{"host":"test-host-%d"},
					"values":[[1625097600,"%d"]]
				}]
			}
		}`, currentCount, currentCount*10)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewClient(server.URL, 5) // 减少超时时间

	// 记录开始时间
	start := time.Now()

	opts := QueryOptions{
		Start:       "now-1h",
		End:         "now",
		Labels:      []string{"env=prod"},
		Concurrency: 3, // 确保设置了并发数
	}

	// 修改：使用 errChan 来传递错误
	errChan := make(chan error, 1)
	metricsChan := make(chan []MetricData, 1)

	go func() {
		metrics, err := client.GetMetrics(opts)
		if err != nil {
			errChan <- err
			return
		}
		metricsChan <- metrics
	}()

	// 等待测试完成或超时
	select {
	case err := <-errChan:
		t.Errorf("获取指标时发生错误: %v", err)
	case metrics := <-metricsChan:
		if len(metrics) == 0 {
			t.Error("未收到任何指标数据")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("测试超时")
	}

	// 计算总执行时间
	duration := time.Since(start)

	// 验证执行时间是否符合并发预期
	if duration > 200*time.Millisecond {
		t.Errorf("查询时间 %v 超过预期，可能未正确并发执行", duration)
	}

	// 验证是否执行了所有查询
	mu.Lock()
	if requestCount != 7 { // 7个指标查询
		t.Errorf("预期执行7个查询，实际执行了 %d 个查询", requestCount)
	}
	mu.Unlock()
}

func TestQueryErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 模拟某些查询失败的情况
		if strings.Contains(r.URL.Query().Get("query"), "cpu") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// 其他查询返回正常结果
		response := `{
			"status":"success",
			"data":{
				"resultType":"matrix",
				"result":[{
					"metric":{"host":"test-host"},
					"values":[[1625097600,"80"]]
				}]
			}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewClient(server.URL, 30)
	opts := QueryOptions{
		Start:  "now-1h",
		End:    "now",
		Labels: []string{"env=prod"},
	}

	_, err := client.GetMetrics(opts)

	// 验证错误处理
	if err == nil {
		t.Error("预期应该返回错误，但没有")
	}

	if !strings.Contains(err.Error(), "cpu") {
		t.Errorf("错误信息应该包含 CPU 查询失败的信息，实际错误: %v", err)
	}
}
