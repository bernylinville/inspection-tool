package vm

import (
	"encoding/json"
	"testing"
)

// TestQueryResponse_IsSuccess tests the IsSuccess method.
func TestQueryResponse_IsSuccess(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"success status", "success", true},
		{"error status", "error", false},
		{"empty status", "", false},
		{"unknown status", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &QueryResponse{Status: tt.status}
			if got := resp.IsSuccess(); got != tt.expected {
				t.Errorf("IsSuccess() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestQueryData_ResultType tests IsVector and IsMatrix methods.
func TestQueryData_ResultType(t *testing.T) {
	tests := []struct {
		name       string
		resultType string
		isVector   bool
		isMatrix   bool
	}{
		{"vector type", "vector", true, false},
		{"matrix type", "matrix", false, true},
		{"scalar type", "scalar", false, false},
		{"string type", "string", false, false},
		{"empty type", "", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &QueryData{ResultType: tt.resultType}
			if got := data.IsVector(); got != tt.isVector {
				t.Errorf("IsVector() = %v, want %v", got, tt.isVector)
			}
			if got := data.IsMatrix(); got != tt.isMatrix {
				t.Errorf("IsMatrix() = %v, want %v", got, tt.isMatrix)
			}
		})
	}
}

// TestSample_GetIdent tests host identifier extraction from labels.
func TestSample_GetIdent(t *testing.T) {
	tests := []struct {
		name     string
		metric   Metric
		expected string
	}{
		{
			"ident label",
			Metric{"ident": "server-1", "host": "server-host"},
			"server-1",
		},
		{
			"host label (no ident)",
			Metric{"host": "server-host", "instance": "192.168.1.1:9100"},
			"server-host",
		},
		{
			"instance label (no ident or host)",
			Metric{"instance": "192.168.1.1:9100", "job": "node"},
			"192.168.1.1:9100",
		},
		{
			"no identifier labels",
			Metric{"job": "node", "cpu": "cpu-total"},
			"",
		},
		{
			"empty metric",
			Metric{},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sample := &Sample{Metric: tt.metric}
			if got := sample.GetIdent(); got != tt.expected {
				t.Errorf("GetIdent() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestSample_GetLabel tests label value retrieval.
func TestSample_GetLabel(t *testing.T) {
	sample := &Sample{
		Metric: Metric{
			"__name__": "cpu_usage_active",
			"cpu":      "cpu-total",
			"ident":    "server-1",
		},
	}

	tests := []struct {
		label    string
		expected string
	}{
		{"__name__", "cpu_usage_active"},
		{"cpu", "cpu-total"},
		{"ident", "server-1"},
		{"nonexistent", ""},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			if got := sample.GetLabel(tt.label); got != tt.expected {
				t.Errorf("GetLabel(%q) = %q, want %q", tt.label, got, tt.expected)
			}
		})
	}
}

// TestMetric_Name tests the metric name retrieval.
func TestMetric_Name(t *testing.T) {
	tests := []struct {
		name     string
		metric   Metric
		expected string
	}{
		{"with name", Metric{"__name__": "cpu_usage"}, "cpu_usage"},
		{"without name", Metric{"cpu": "cpu-total"}, ""},
		{"empty metric", Metric{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.metric.Name(); got != tt.expected {
				t.Errorf("Name() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestSampleValue_Timestamp tests timestamp extraction.
func TestSampleValue_Timestamp(t *testing.T) {
	tests := []struct {
		name      string
		value     SampleValue
		timestamp float64
		unix      int64
	}{
		{
			"valid timestamp",
			SampleValue{1702451234.567, "75.5"},
			1702451234.567,
			1702451234,
		},
		{
			"integer timestamp",
			SampleValue{float64(1702451234), "50.0"},
			1702451234.0,
			1702451234,
		},
		{
			"empty value",
			SampleValue{},
			0,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.Timestamp(); got != tt.timestamp {
				t.Errorf("Timestamp() = %v, want %v", got, tt.timestamp)
			}
			if got := tt.value.TimestampUnix(); got != tt.unix {
				t.Errorf("TimestampUnix() = %v, want %v", got, tt.unix)
			}
		})
	}
}

// TestSampleValue_Value tests value parsing.
func TestSampleValue_Value(t *testing.T) {
	tests := []struct {
		name      string
		value     SampleValue
		expected  float64
		shouldErr bool
	}{
		{
			"string value",
			SampleValue{1702451234.567, "75.5"},
			75.5,
			false,
		},
		{
			"integer string value",
			SampleValue{1702451234.567, "100"},
			100.0,
			false,
		},
		{
			"float64 value",
			SampleValue{1702451234.567, float64(42.5)},
			42.5,
			false,
		},
		{
			"zero value",
			SampleValue{1702451234.567, "0"},
			0,
			false,
		},
		{
			"negative value",
			SampleValue{1702451234.567, "-15.5"},
			-15.5,
			false,
		},
		{
			"empty sample",
			SampleValue{},
			0,
			true,
		},
		{
			"invalid string",
			SampleValue{1702451234.567, "invalid"},
			0,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.value.Value()
			if tt.shouldErr {
				if err == nil {
					t.Errorf("Value() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Value() unexpected error: %v", err)
				}
				if got != tt.expected {
					t.Errorf("Value() = %v, want %v", got, tt.expected)
				}
			}
		})
	}
}

// TestSampleValue_MustValue tests MustValue method.
func TestSampleValue_MustValue(t *testing.T) {
	// Valid value
	v1 := SampleValue{1702451234.567, "75.5"}
	if got := v1.MustValue(); got != 75.5 {
		t.Errorf("MustValue() = %v, want 75.5", got)
	}

	// Invalid value returns 0
	v2 := SampleValue{1702451234.567, "invalid"}
	if got := v2.MustValue(); got != 0 {
		t.Errorf("MustValue() = %v, want 0", got)
	}
}

// TestSampleValue_IsNaN tests NaN detection.
func TestSampleValue_IsNaN(t *testing.T) {
	tests := []struct {
		name     string
		value    SampleValue
		expected bool
	}{
		{"NaN string", SampleValue{1702451234.567, "NaN"}, true},
		{"+Inf string", SampleValue{1702451234.567, "+Inf"}, true},
		{"-Inf string", SampleValue{1702451234.567, "-Inf"}, true},
		{"normal value", SampleValue{1702451234.567, "75.5"}, false},
		{"empty sample", SampleValue{}, true},
		{"single element", SampleValue{1702451234.567}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.IsNaN(); got != tt.expected {
				t.Errorf("IsNaN() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestParseQueryResults tests the result parsing function.
func TestParseQueryResults(t *testing.T) {
	t.Run("success response", func(t *testing.T) {
		resp := &QueryResponse{
			Status: "success",
			Data: QueryData{
				ResultType: "vector",
				Result: []Sample{
					{
						Metric: Metric{"ident": "server-1", "__name__": "cpu_usage"},
						Value:  SampleValue{1702451234.567, "75.5"},
					},
					{
						Metric: Metric{"ident": "server-2", "__name__": "cpu_usage"},
						Value:  SampleValue{1702451234.567, "50.0"},
					},
				},
			},
		}

		results, err := ParseQueryResults(resp)
		if err != nil {
			t.Fatalf("ParseQueryResults() error: %v", err)
		}

		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}

		if results[0].Ident != "server-1" {
			t.Errorf("results[0].Ident = %q, want %q", results[0].Ident, "server-1")
		}
		if results[0].Value != 75.5 {
			t.Errorf("results[0].Value = %v, want %v", results[0].Value, 75.5)
		}
	})

	t.Run("error response", func(t *testing.T) {
		resp := &QueryResponse{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     "invalid query",
		}

		_, err := ParseQueryResults(resp)
		if err == nil {
			t.Error("ParseQueryResults() expected error for error response")
		}
	})

	t.Run("non-vector result type", func(t *testing.T) {
		resp := &QueryResponse{
			Status: "success",
			Data: QueryData{
				ResultType: "matrix",
				Result:     []Sample{},
			},
		}

		_, err := ParseQueryResults(resp)
		if err == nil {
			t.Error("ParseQueryResults() expected error for non-vector result")
		}
	})

	t.Run("skip NaN values", func(t *testing.T) {
		resp := &QueryResponse{
			Status: "success",
			Data: QueryData{
				ResultType: "vector",
				Result: []Sample{
					{
						Metric: Metric{"ident": "server-1"},
						Value:  SampleValue{1702451234.567, "NaN"},
					},
					{
						Metric: Metric{"ident": "server-2"},
						Value:  SampleValue{1702451234.567, "50.0"},
					},
				},
			},
		}

		results, err := ParseQueryResults(resp)
		if err != nil {
			t.Fatalf("ParseQueryResults() error: %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("expected 1 result (NaN skipped), got %d", len(results))
		}

		if results[0].Ident != "server-2" {
			t.Errorf("results[0].Ident = %q, want %q", results[0].Ident, "server-2")
		}
	})
}

// TestGroupResultsByIdent tests the grouping function.
func TestGroupResultsByIdent(t *testing.T) {
	results := []QueryResult{
		{Ident: "server-1", Value: 75.5},
		{Ident: "server-2", Value: 50.0},
		{Ident: "", Value: 30.0}, // Should be skipped
		{Ident: "server-1", Value: 80.0}, // Should override the first one
	}

	grouped := GroupResultsByIdent(results)

	if len(grouped) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(grouped))
	}

	if r, ok := grouped["server-1"]; !ok {
		t.Error("server-1 not found in grouped results")
	} else if r.Value != 80.0 {
		t.Errorf("server-1 value = %v, want %v (last one should win)", r.Value, 80.0)
	}

	if r, ok := grouped["server-2"]; !ok {
		t.Error("server-2 not found in grouped results")
	} else if r.Value != 50.0 {
		t.Errorf("server-2 value = %v, want %v", r.Value, 50.0)
	}
}

// TestHostFilter_IsEmpty tests the filter emptiness check.
func TestHostFilter_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		filter   *HostFilter
		expected bool
	}{
		{"nil filter", nil, true},
		{"empty filter", &HostFilter{}, true},
		{"with business groups", &HostFilter{BusinessGroups: []string{"group1"}}, false},
		{"with tags", &HostFilter{Tags: map[string]string{"env": "prod"}}, false},
		{"with both", &HostFilter{
			BusinessGroups: []string{"group1"},
			Tags:           map[string]string{"env": "prod"},
		}, false},
		{"empty arrays", &HostFilter{
			BusinessGroups: []string{},
			Tags:           map[string]string{},
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.filter.IsEmpty(); got != tt.expected {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestQueryResponse_JSONParsing tests JSON unmarshalling of API response.
func TestQueryResponse_JSONParsing(t *testing.T) {
	jsonData := `{
		"status": "success",
		"data": {
			"resultType": "vector",
			"result": [
				{
					"metric": {
						"__name__": "cpu_usage_active",
						"cpu": "cpu-total",
						"ident": "sd-k8s-master-1"
					},
					"value": [1702451234.567, "75.5"]
				},
				{
					"metric": {
						"__name__": "cpu_usage_active",
						"cpu": "cpu-total",
						"ident": "sd-k8s-node-1"
					},
					"value": [1702451234.567, "42.3"]
				}
			]
		}
	}`

	var resp QueryResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	if !resp.IsSuccess() {
		t.Error("expected success status")
	}

	if !resp.Data.IsVector() {
		t.Error("expected vector result type")
	}

	if len(resp.Data.Result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Data.Result))
	}

	// Check first sample
	sample := resp.Data.Result[0]
	if sample.GetIdent() != "sd-k8s-master-1" {
		t.Errorf("first sample ident = %q, want %q", sample.GetIdent(), "sd-k8s-master-1")
	}

	value, err := sample.Value.Value()
	if err != nil {
		t.Errorf("Value() error: %v", err)
	}
	if value != 75.5 {
		t.Errorf("first sample value = %v, want %v", value, 75.5)
	}

	if sample.Metric.Name() != "cpu_usage_active" {
		t.Errorf("metric name = %q, want %q", sample.Metric.Name(), "cpu_usage_active")
	}
}

// TestQueryResponse_ErrorParsing tests JSON unmarshalling of error response.
func TestQueryResponse_ErrorParsing(t *testing.T) {
	jsonData := `{
		"status": "error",
		"errorType": "bad_data",
		"error": "invalid query expression"
	}`

	var resp QueryResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	if resp.IsSuccess() {
		t.Error("expected error status")
	}

	if resp.ErrorType != "bad_data" {
		t.Errorf("errorType = %q, want %q", resp.ErrorType, "bad_data")
	}

	if resp.Error != "invalid query expression" {
		t.Errorf("error = %q, want %q", resp.Error, "invalid query expression")
	}
}
