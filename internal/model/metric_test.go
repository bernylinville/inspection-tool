package model

import "testing"

func TestNewNAMetricValue(t *testing.T) {
	value := NewNAMetricValue("ntp_check")

	if value.Name != "ntp_check" {
		t.Errorf("Name = %q, want %q", value.Name, "ntp_check")
	}
	if value.RawValue != 0 {
		t.Errorf("RawValue = %v, want 0", value.RawValue)
	}
	if value.FormattedValue != "N/A" {
		t.Errorf("FormattedValue = %q, want %q", value.FormattedValue, "N/A")
	}
	if value.Status != MetricStatusPending {
		t.Errorf("Status = %v, want %v", value.Status, MetricStatusPending)
	}
	if !value.IsNA {
		t.Error("IsNA should be true")
	}
}

func TestNewMetricValue(t *testing.T) {
	value := NewMetricValue("cpu_usage", 75.5)

	if value.Name != "cpu_usage" {
		t.Errorf("Name = %q, want %q", value.Name, "cpu_usage")
	}
	if value.RawValue != 75.5 {
		t.Errorf("RawValue = %v, want 75.5", value.RawValue)
	}
	if value.Status != MetricStatusNormal {
		t.Errorf("Status = %v, want %v", value.Status, MetricStatusNormal)
	}
	if value.IsNA {
		t.Error("IsNA should be false")
	}
}

func TestMetricDefinition_IsPending(t *testing.T) {
	tests := []struct {
		name     string
		def      MetricDefinition
		expected bool
	}{
		{
			name:     "status is pending",
			def:      MetricDefinition{Name: "test", Query: "some_query", Status: "pending"},
			expected: true,
		},
		{
			name:     "query is empty",
			def:      MetricDefinition{Name: "test", Query: "", Status: ""},
			expected: true,
		},
		{
			name:     "normal metric",
			def:      MetricDefinition{Name: "test", Query: "some_query", Status: ""},
			expected: false,
		},
		{
			name:     "both pending status and empty query",
			def:      MetricDefinition{Name: "test", Query: "", Status: "pending"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.def.IsPending(); got != tt.expected {
				t.Errorf("IsPending() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMetricDefinition_HasExpandLabel(t *testing.T) {
	tests := []struct {
		name     string
		def      MetricDefinition
		expected bool
	}{
		{
			name:     "has expand label",
			def:      MetricDefinition{ExpandByLabel: "path"},
			expected: true,
		},
		{
			name:     "no expand label",
			def:      MetricDefinition{ExpandByLabel: ""},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.def.HasExpandLabel(); got != tt.expected {
				t.Errorf("HasExpandLabel() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewHostMetrics(t *testing.T) {
	metrics := NewHostMetrics("server-01")

	if metrics.Hostname != "server-01" {
		t.Errorf("Hostname = %q, want %q", metrics.Hostname, "server-01")
	}
	if metrics.Metrics == nil {
		t.Error("Metrics map should be initialized")
	}
}

func TestHostMetrics_SetMetric(t *testing.T) {
	t.Run("normal set", func(t *testing.T) {
		metrics := NewHostMetrics("server-01")
		value := &MetricValue{Name: "cpu_usage", RawValue: 75.5}

		metrics.SetMetric(value)

		if metrics.Metrics["cpu_usage"] != value {
			t.Error("metric should be set")
		}
	})

	t.Run("nil metrics map initialized", func(t *testing.T) {
		metrics := &HostMetrics{Hostname: "server-01"}
		value := &MetricValue{Name: "cpu_usage", RawValue: 75.5}

		metrics.SetMetric(value)

		if metrics.Metrics == nil {
			t.Error("Metrics map should be initialized")
		}
	})
}

func TestHostMetrics_GetMetric(t *testing.T) {
	t.Run("existing metric", func(t *testing.T) {
		value := &MetricValue{Name: "cpu_usage", RawValue: 75.5}
		metrics := &HostMetrics{
			Hostname: "server-01",
			Metrics:  map[string]*MetricValue{"cpu_usage": value},
		}

		got := metrics.GetMetric("cpu_usage")
		if got != value {
			t.Error("should return the metric")
		}
	})

	t.Run("non-existing metric", func(t *testing.T) {
		metrics := NewHostMetrics("server-01")

		got := metrics.GetMetric("not_exists")
		if got != nil {
			t.Error("should return nil for non-existing metric")
		}
	})

	t.Run("nil metrics map", func(t *testing.T) {
		metrics := &HostMetrics{Hostname: "server-01"}

		got := metrics.GetMetric("cpu_usage")
		if got != nil {
			t.Error("should return nil for nil metrics map")
		}
	})
}
