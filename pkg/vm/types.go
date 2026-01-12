// Package vm provides a client for VictoriaMetrics/Prometheus API.
package vm

import (
	"fmt"
	"strconv"
)

// QueryResponse represents the API response from /api/v1/query endpoint.
// This structure follows the Prometheus HTTP API specification.
type QueryResponse struct {
	Status    string    `json:"status"`    // 响应状态：success 或 error
	Data      QueryData `json:"data"`      // 查询数据
	ErrorType string    `json:"errorType"` // 错误类型（仅在 status=error 时存在）
	Error     string    `json:"error"`     // 错误信息（仅在 status=error 时存在）
	Warnings  []string  `json:"warnings"`  // 警告信息列表
}

// IsSuccess returns true if the query was successful.
func (r *QueryResponse) IsSuccess() bool {
	return r.Status == "success"
}

// QueryData contains the result data from a query.
type QueryData struct {
	ResultType string   `json:"resultType"` // 结果类型：vector, matrix, scalar, string
	Result     []Sample `json:"result"`     // 结果样本列表
}

// IsVector returns true if the result type is "vector" (instant vector).
func (d *QueryData) IsVector() bool {
	return d.ResultType == "vector"
}

// IsMatrix returns true if the result type is "matrix" (range vector).
func (d *QueryData) IsMatrix() bool {
	return d.ResultType == "matrix"
}

// Sample represents a single sample in the query result.
// For instant queries (vector), use Value field.
// For range queries (matrix), use Values field.
type Sample struct {
	Metric Metric        `json:"metric"` // 指标标签
	Value  SampleValue   `json:"value"`  // 即时查询值 [timestamp, value]
	Values []SampleValue `json:"values"` // 范围查询值列表 [[timestamp, value], ...]
}

// GetIdent returns the host identifier from metric labels.
// It tries "ident" first, then "host", then "instance".
func (s *Sample) GetIdent() string {
	if ident, ok := s.Metric["ident"]; ok {
		return ident
	}
	if host, ok := s.Metric["host"]; ok {
		return host
	}
	if instance, ok := s.Metric["instance"]; ok {
		return instance
	}
	return ""
}

// GetLabel returns the value of a specific label, or empty string if not found.
func (s *Sample) GetLabel(name string) string {
	if value, ok := s.Metric[name]; ok {
		return value
	}
	return ""
}

// Metric represents a set of label-value pairs for a time series.
type Metric map[string]string

// Name returns the metric name (__name__ label).
func (m Metric) Name() string {
	if name, ok := m["__name__"]; ok {
		return name
	}
	return ""
}

// SampleValue represents a single [timestamp, value] pair.
// In Prometheus API, this is represented as a two-element array:
// [unix_timestamp_float, "value_string"]
type SampleValue [2]interface{}

// Timestamp returns the Unix timestamp as float64.
// Returns 0 if parsing fails.
func (v SampleValue) Timestamp() float64 {
	if len(v) < 2 {
		return 0
	}
	if ts, ok := v[0].(float64); ok {
		return ts
	}
	return 0
}

// TimestampUnix returns the Unix timestamp as int64 (seconds).
func (v SampleValue) TimestampUnix() int64 {
	return int64(v.Timestamp())
}

// Value returns the sample value as float64.
// Returns 0 and error if parsing fails.
func (v SampleValue) Value() (float64, error) {
	if len(v) < 2 {
		return 0, fmt.Errorf("invalid sample value: length %d", len(v))
	}

	switch val := v[1].(type) {
	case string:
		// Prometheus API 返回的值是字符串格式
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse value %q: %w", val, err)
		}
		return f, nil
	case float64:
		return val, nil
	default:
		return 0, fmt.Errorf("unexpected value type: %T", v[1])
	}
}

// MustValue returns the sample value as float64.
// Returns 0 if parsing fails (use Value() for error handling).
func (v SampleValue) MustValue() float64 {
	val, _ := v.Value()
	return val
}

// IsNaN returns true if the value is NaN, Inf, or invalid.
// In Prometheus, NaN is represented as the string "NaN".
func (v SampleValue) IsNaN() bool {
	// Check if value element is nil or not present
	if v[1] == nil {
		return true
	}
	if str, ok := v[1].(string); ok {
		return str == "NaN" || str == "+Inf" || str == "-Inf"
	}
	return false
}

// QueryResult is a convenience wrapper for processing query results.
type QueryResult struct {
	Ident  string            // 主机标识符
	Value  float64           // 指标值
	Labels map[string]string // 所有标签
}

// ParseQueryResults converts QueryResponse to a slice of QueryResult.
// This is a convenience function for processing vector query results.
func ParseQueryResults(resp *QueryResponse) ([]QueryResult, error) {
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("query failed: %s - %s", resp.ErrorType, resp.Error)
	}

	if !resp.Data.IsVector() {
		return nil, fmt.Errorf("unexpected result type: %s (expected vector)", resp.Data.ResultType)
	}

	results := make([]QueryResult, 0, len(resp.Data.Result))
	for _, sample := range resp.Data.Result {
		if sample.Value.IsNaN() {
			continue // 跳过 NaN 值
		}

		value, err := sample.Value.Value()
		if err != nil {
			continue // 跳过无法解析的值
		}

		results = append(results, QueryResult{
			Ident:  sample.GetIdent(),
			Value:  value,
			Labels: sample.Metric,
		})
	}

	return results, nil
}

// GroupResultsByIdent groups query results by host identifier.
// Returns a map where key is ident and value is the QueryResult.
// If multiple samples have the same ident, the last one wins.
func GroupResultsByIdent(results []QueryResult) map[string]QueryResult {
	grouped := make(map[string]QueryResult, len(results))
	for _, r := range results {
		if r.Ident != "" {
			grouped[r.Ident] = r
		}
	}
	return grouped
}

// HostFilter defines filters for querying specific hosts.
type HostFilter struct {
	BusinessGroups []string          // 业务组（OR 关系）
	Tags           map[string]string // 标签（AND 关系）
}

// IsEmpty returns true if no filters are set.
func (f *HostFilter) IsEmpty() bool {
	return f == nil || (len(f.BusinessGroups) == 0 && len(f.Tags) == 0)
}
