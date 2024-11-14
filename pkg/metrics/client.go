package metrics

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bernylinville/inspection-tool/pkg/logger"
)

type Client struct {
	baseURL string
	timeout time.Duration
	client  *http.Client
}

type MetricData struct {
	Hostname     string
	IP           string
	Project      string
	CPU          float64 // cpu_usage_active
	Memory       float64 // mem_used_percent
	DiskUsage    float64 // disk_used_percent
	SystemUptime float64 // system_uptime
	SystemLoad1  float64 // system_load1
	SystemLoad5  float64 // system_load5
	SystemLoad15 float64 // system_load15
	NetworkIn    float64
	NetworkOut   float64
}

// VMQueryResult 用于解析 VictoriaMetrics 的响应
type VMQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Values [][]interface{}   `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

// MetricQueryResult 用于内部传递查询结果
type MetricQueryResult struct {
	Name  string
	Data  map[string]float64
	Error error
}

// QueryOptions 定义查询选项
type QueryOptions struct {
	Start       string   // 开始时间
	End         string   // 结束时间
	Labels      []string // 标签过滤
	Projects    []string // 项目列表
	Concurrency int      // 并发数
}

// MetricQuery 定义单个指标查询
type MetricQuery struct {
	Name  string
	Query string
}

// ProgressCallback 定义进度回调函数类型
type ProgressCallback func(stage string, current, total int)

func NewClient(address string, timeout time.Duration) *Client {
	logger.Debug().
		Str("baseURL", address).
		Dur("timeout", timeout).
		Msg("初始化 VictoriaMetrics 客户端")

	return &Client{
		baseURL: address,
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// buildLabelSelector 构建标签选择器
func buildLabelSelector(labels []string) string {
	if len(labels) == 0 {
		return ""
	}

	// 确保每个标签都有引号
	quotedLabels := make([]string, len(labels))
	for i, label := range labels {
		parts := strings.SplitN(label, "=", 2)
		if len(parts) == 2 {
			quotedLabels[i] = fmt.Sprintf(`%s="%s"`, parts[0], parts[1])
		}
	}

	return fmt.Sprintf("{%s}", strings.Join(quotedLabels, ","))
}

func (c *Client) GetMetrics(opts QueryOptions) ([]MetricData, error) {
	metrics := make(map[string]*MetricData)
	labelSelector := buildLabelSelector(opts.Labels)

	// 修改查询定义
	queries := []MetricQuery{
		{
			Name:  "cpu",
			Query: fmt.Sprintf(`avg(rate(cpu_usage_active%s[5m])) by (host)`, labelSelector),
		},
		{
			Name:  "memory",
			Query: fmt.Sprintf(`avg(mem_used_percent%s) by (host)`, labelSelector),
		},
		{
			Name:  "disk",
			Query: fmt.Sprintf(`max(disk_used_percent%s) by (host)`, strings.Replace(labelSelector, "}", `,fstype!~"tmpfs|devtmpfs"}`, 1)),
		},
		{
			Name:  "uptime",
			Query: fmt.Sprintf(`max(system_uptime%s) by (host)`, labelSelector),
		},
		{
			Name:  "load1",
			Query: fmt.Sprintf(`max(system_load1%s) by (host)`, labelSelector),
		},
		{
			Name:  "load5",
			Query: fmt.Sprintf(`max(system_load5%s) by (host)`, labelSelector),
		},
		{
			Name:  "load15",
			Query: fmt.Sprintf(`max(system_load15%s) by (host)`, labelSelector),
		},
	}

	// 确保并发数大于0
	if opts.Concurrency <= 0 {
		opts.Concurrency = 3 // 默认并发数
	}

	// 创建结果通道
	resultChan := make(chan MetricQueryResult, len(queries))

	// 使用 WaitGroup 确保所有 goroutine 完成
	var wg sync.WaitGroup
	wg.Add(len(queries))

	// 使用并发数
	semaphore := make(chan struct{}, opts.Concurrency)

	// 并发执行查询
	for _, q := range queries {
		go func(query MetricQuery) {
			defer wg.Done()
			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			data, err := c.queryMetric(query.Query, opts.Start, opts.End)
			resultChan <- MetricQueryResult{
				Name:  query.Name,
				Data:  data,
				Error: err,
			}
		}(q)
	}

	// 等待所有查询完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集查询结果
	var errors []error
	results := make(map[string]map[string]float64)

	for i := 0; i < len(queries); i++ {
		result := <-resultChan
		if result.Error != nil {
			errors = append(errors, fmt.Errorf("%s查询错误: %v", result.Name, result.Error))
			continue
		}
		results[result.Name] = result.Data
	}

	// 如果有错误，返回所有错误
	if len(errors) > 0 {
		return nil, fmt.Errorf("查询错误: %v", errors)
	}

	// 整合数据
	for name, data := range results {
		for host, value := range data {
			if _, exists := metrics[host]; !exists {
				// 从主机名提取项目名称
				parts := strings.Split(host, "-")
				projectName := ""
				if len(parts) > 0 {
					projectName = parts[0]
				}

				metrics[host] = &MetricData{
					Hostname: host,
					Project:  projectName,
				}
			}

			switch name {
			case "cpu":
				metrics[host].CPU = value
			case "memory":
				metrics[host].Memory = value
			case "disk":
				metrics[host].DiskUsage = value
			case "uptime":
				metrics[host].SystemUptime = value
			case "load1":
				metrics[host].SystemLoad1 = value
			case "load5":
				metrics[host].SystemLoad5 = value
			case "load15":
				metrics[host].SystemLoad15 = value
			}
		}
	}

	// 转换为切片
	result := make([]MetricData, 0, len(metrics))
	for _, data := range metrics {
		result = append(result, *data)
	}

	// 如果指定了项目列表，过滤数据
	if len(opts.Projects) > 0 {
		projectSet := make(map[string]bool)
		for _, p := range opts.Projects {
			projectSet[p] = true
		}

		filteredResult := make([]MetricData, 0)
		for _, data := range result {
			projectName := strings.Split(data.Hostname, "-")[0]
			if projectSet[projectName] {
				filteredResult = append(filteredResult, data)
			}
		}
		result = filteredResult
	}

	return result, nil
}

func (c *Client) queryMetric(query, start, end string) (map[string]float64, error) {
	url, err := c.buildURL(query, start, end)
	if err != nil {
		return nil, fmt.Errorf("构建URL失败: %v", err)
	}

	// 添加更多调试日志
	logger.Debug().
		Str("url", url).
		Str("query", query).
		Str("start", start).
		Str("end", end).
		Msg("执行查询")

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 添加响应内容的调试日志
	logger.Debug().
		Str("response", string(body)).
		Msg("查询响应")

	var result VMQueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, body: %s", err, string(body))
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("查询失败: %s", string(body))
	}

	// 解析结果
	data := make(map[string]float64)
	for _, r := range result.Data.Result {
		// 使用 instance 或 host 作为标识
		host := r.Metric["instance"]
		if h, ok := r.Metric["host"]; ok {
			host = h
		}

		if len(r.Values) > 0 {
			// 取最后一个值作为当前值
			lastValue := r.Values[len(r.Values)-1]
			if len(lastValue) >= 2 {
				if v, ok := lastValue[1].(string); ok {
					if f, err := parseFloat(v); err == nil {
						data[host] = f
					}
				}
			}
		}
	}

	return data, nil
}

// GetMetricsWithProgress 添加带进度的查询方法
func (c *Client) GetMetricsWithProgress(opts QueryOptions, progress ProgressCallback) ([]MetricData, error) {
	return c.GetMetrics(opts)
}

// 辅助函数：将字符串转换为float64
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// 添加一个时间解析函数
func parseTime(timeStr string) (string, error) {
	if timeStr == "now" {
		// 使用 Unix 时间戳
		return fmt.Sprintf("%d", time.Now().Unix()), nil
	}

	// 处理相对时间，如 "now-1h", "now-1d"
	if strings.HasPrefix(timeStr, "now-") {
		duration := strings.TrimPrefix(timeStr, "now-")
		d, err := parseDuration(duration)
		if err != nil {
			return "", fmt.Errorf("解析时间失败: %v", err)
		}
		// 使用 Unix 时间戳
		return fmt.Sprintf("%d", time.Now().Add(-d).Unix()), nil
	}

	// 尝试解析为 RFC3339 格式
	t, err := time.Parse(time.RFC3339, timeStr)
	if err == nil {
		return fmt.Sprintf("%d", t.Unix()), nil
	}

	return "", fmt.Errorf("不支持的时间格式: %s", timeStr)
}

// 解析持续时间
func parseDuration(s string) (time.Duration, error) {
	// 支持 1d, 1h, 1m 等格式
	multiplier := map[byte]time.Duration{
		'd': 24 * time.Hour,
		'h': time.Hour,
		'm': time.Minute,
		's': time.Second,
	}

	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration: %s", s)
	}

	unit := s[len(s)-1]
	value := s[:len(s)-1]

	mult, ok := multiplier[unit]
	if !ok {
		return 0, fmt.Errorf("unsupported time unit: %c", unit)
	}

	n, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}

	return time.Duration(n * float64(mult)), nil
}

// 修改 buildURL 方法
func (c *Client) buildURL(query string, start, end string) (string, error) {
	startTime, err := parseTime(start)
	if err != nil {
		return "", fmt.Errorf("解析开始时间失败: %v", err)
	}

	endTime, err := parseTime(end)
	if err != nil {
		return "", fmt.Errorf("解析结束时间失败: %v", err)
	}

	u, err := url.Parse(c.baseURL + "/api/v1/query_range")
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("query", query)
	q.Set("start", startTime)
	q.Set("end", endTime)
	q.Set("step", "60") // 添加 step 参数，设置为60秒
	u.RawQuery = q.Encode()

	logger.Debug().
		Str("final_url", u.String()).
		Msg("构建查询URL")

	return u.String(), nil
}
