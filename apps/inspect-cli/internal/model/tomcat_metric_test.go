package model

import "testing"

func TestTomcatMetricDefinition_IsPending(t *testing.T) {
	tests := []struct {
		name     string
		def      TomcatMetricDefinition
		expected bool
	}{
		{
			name:     "status is pending",
			def:      TomcatMetricDefinition{Name: "test", Query: "some_query", Status: "pending"},
			expected: true,
		},
		{
			name:     "query is empty",
			def:      TomcatMetricDefinition{Name: "test", Query: "", Status: ""},
			expected: true,
		},
		{
			name:     "normal metric",
			def:      TomcatMetricDefinition{Name: "test", Query: "some_query", Status: ""},
			expected: false,
		},
		{
			name:     "both pending and empty query",
			def:      TomcatMetricDefinition{Name: "test", Query: "", Status: "pending"},
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

func TestTomcatMetricDefinition_HasLabelExtract(t *testing.T) {
	tests := []struct {
		name     string
		def      TomcatMetricDefinition
		expected bool
	}{
		{
			name:     "has label extract",
			def:      TomcatMetricDefinition{LabelExtract: []string{"version", "path"}},
			expected: true,
		},
		{
			name:     "empty label extract",
			def:      TomcatMetricDefinition{LabelExtract: []string{}},
			expected: false,
		},
		{
			name:     "nil label extract",
			def:      TomcatMetricDefinition{LabelExtract: nil},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.def.HasLabelExtract(); got != tt.expected {
				t.Errorf("HasLabelExtract() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTomcatMetricDefinition_GetDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		def      TomcatMetricDefinition
		expected string
	}{
		{
			name:     "has display name",
			def:      TomcatMetricDefinition{Name: "tomcat_up", DisplayName: "运行状态"},
			expected: "运行状态",
		},
		{
			name:     "no display name falls back to name",
			def:      TomcatMetricDefinition{Name: "tomcat_up", DisplayName: ""},
			expected: "tomcat_up",
		},
		{
			name:     "both empty returns empty",
			def:      TomcatMetricDefinition{Name: "", DisplayName: ""},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.def.GetDisplayName(); got != tt.expected {
				t.Errorf("GetDisplayName() = %v, want %v", got, tt.expected)
			}
		})
	}
}
