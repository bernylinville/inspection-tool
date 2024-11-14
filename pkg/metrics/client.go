package metrics

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	timeout time.Duration
	client  *http.Client
}

type MetricData struct {
	Hostname     string
	IP           string
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

func NewClient(baseURL string, timeout int) *Client {
	return &Client{
		baseURL: baseURL,
		timeout: time.Duration(timeout) * time.Second,
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}
}

// buildLabelSelector 构建标签选择器
func buildLabelSelector(labels []string) string {
	if len(labels) == 0 {
		return ""
	}

	selector := "{"
	for i, label := range labels {
		if i > 0 {
			selector += ","
		}
		selector += label
	}
	selector += "}"
	return selector
}

func (c *Client) GetMetrics(opts QueryOptions) ([]MetricData, error) {
	metrics := make(map[string]*MetricData)
	labelSelector := buildLabelSelector(opts.Labels)

	// 定义所有需要查询的指标
	queries := []MetricQuery{
		{
			Name:  "cpu",
			Query: fmt.Sprintf(`100 - avg(rate(cpu_usage_active%s[5m])) by (host)`, labelSelector),
		},
		{
			Name:  "memory",
			Query: fmt.Sprintf(`mem_used_percent%s`, labelSelector),
		},
		{
			Name:  "disk",
			Query: fmt.Sprintf(`disk_used_percent%s{fstype!~"tmpfs|devtmpfs"}`, labelSelector),
		},
		{
			Name:  "uptime",
			Query: fmt.Sprintf(`system_uptime%s`, labelSelector),
		},
		{
			Name:  "load1",
			Query: fmt.Sprintf(`system_load1%s`, labelSelector),
		},
		{
			Name:  "load5",
			Query: fmt.Sprintf(`system_load5%s`, labelSelector),
		},
		{
			Name:  "load15",
			Query: fmt.Sprintf(`system_load15%s`, labelSelector),
		},
	}

	// 创建结果通道
	resultChan := make(chan MetricQueryResult, len(queries))

	// 并发执行查询
	for _, q := range queries {
		go func(query MetricQuery) {
			data, err := c.queryMetric(query.Query, opts.Start, opts.End)
			resultChan <- MetricQueryResult{
				Name:  query.Name,
				Data:  data,
				Error: err,
			}
		}(q)
	}

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
				metrics[host] = &MetricData{
					Hostname: host,
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

	return result, nil
}

func (c *Client) queryMetric(query, start, end string) (map[string]float64, error) {
	params := url.Values{}
	params.Add("query", query)
	params.Add("start", start)
	params.Add("end", end)

	resp, err := http.Get(fmt.Sprintf("%s/api/v1/query_range?%s", c.baseURL, params.Encode()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result VMQueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
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

// 辅助函数：将字符串转换为float64
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// GetMetricsWithProgress 添加带进度的查询方法
func (c *Client) GetMetricsWithProgress(opts QueryOptions, progress ProgressCallback) ([]MetricData, error) {
	metrics := make(map[string]*MetricData)
	labelSelector := buildLabelSelector(opts.Labels)

	// 定义所有需要查询的指标
	queries := []MetricQuery{
		{
			Name:  "cpu",
			Query: fmt.Sprintf(`100 - avg(rate(cpu_usage_active%s[5m])) by (host)`, labelSelector),
		},
		{
			Name:  "memory",
			Query: fmt.Sprintf(`mem_used_percent%s`, labelSelector),
		},
		{
			Name:  "disk",
			Query: fmt.Sprintf(`disk_used_percent%s{fstype!~"tmpfs|devtmpfs"}`, labelSelector),
		},
		{
			Name:  "uptime",
			Query: fmt.Sprintf(`system_uptime%s`, labelSelector),
		},
		{
			Name:  "load1",
			Query: fmt.Sprintf(`system_load1%s`, labelSelector),
		},
		{
			Name:  "load5",
			Query: fmt.Sprintf(`system_load5%s`, labelSelector),
		},
		{
			Name:  "load15",
			Query: fmt.Sprintf(`system_load15%s`, labelSelector),
		},
	}

	// 创建结果通道
	resultChan := make(chan MetricQueryResult, len(queries))

	// 使用并发数
	semaphore := make(chan struct{}, opts.Concurrency)

	// 并发执行查询
	for _, q := range queries {
		semaphore <- struct{}{} // 获取信号量
		go func(query MetricQuery) {
			defer func() { <-semaphore }() // 释放信号量
			data, err := c.queryMetric(query.Query, opts.Start, opts.End)
			resultChan <- MetricQueryResult{
				Name:  query.Name,
				Data:  data,
				Error: err,
			}
		}(q)
	}

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

		// 更新进度
		if progress != nil {
			progress("查询指标", i+1, len(queries))
		}
	}

	// 如果有错误，返回所有错误
	if len(errors) > 0 {
		return nil, fmt.Errorf("查询错误: %v", errors)
	}

	// 整合数据
	totalHosts := 0
	currentHost := 0

	// 计算总主机数
	for _, data := range results {
		for host := range data {
			if _, exists := metrics[host]; !exists {
				totalHosts++
			}
		}
	}

	// 整合数据并显示进度
	for name, data := range results {
		for host, value := range data {
			if _, exists := metrics[host]; !exists {
				metrics[host] = &MetricData{
					Hostname: host,
				}
				currentHost++
				if progress != nil {
					progress("处理数据", currentHost, totalHosts)
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
