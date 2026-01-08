package util

import (
	"testing"
)

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		pattern  string
		expected bool
	}{
		{
			name:     "exact match",
			value:    "172.18.182.91:3306",
			pattern:  "172.18.182.91:3306",
			expected: true,
		},
		{
			name:     "wildcard suffix",
			value:    "172.18.182.91:3306",
			pattern:  "172.18.182.*",
			expected: true,
		},
		{
			name:     "wildcard prefix",
			value:    "172.18.182.91:3306",
			pattern:  "*:3306",
			expected: true,
		},
		{
			name:     "wildcard middle",
			value:    "172.18.182.91:3306",
			pattern:  "172.*.91:3306",
			expected: true,
		},
		{
			name:     "multiple wildcards",
			value:    "172.18.182.91:3306",
			pattern:  "172.*.*:3306",
			expected: true,
		},
		{
			name:     "match all",
			value:    "anything",
			pattern:  "*",
			expected: true,
		},
		{
			name:     "no match - different value",
			value:    "192.168.1.1:3306",
			pattern:  "172.18.182.*",
			expected: false,
		},
		{
			name:     "no match - no wildcard",
			value:    "172.18.182.91:3306",
			pattern:  "172.18.182.92:3306",
			expected: false,
		},
		{
			name:     "hostname pattern",
			value:    "GX-NM-MNS-NGX-01",
			pattern:  "GX-NM-*",
			expected: true,
		},
		{
			name:     "hostname pattern middle",
			value:    "GX-NM-MNS-NGX-01",
			pattern:  "*-NGX-*",
			expected: true,
		},
		{
			name:     "hostname no match",
			value:    "GX-NM-MNS-NGX-01",
			pattern:  "GX-SH-*",
			expected: false,
		},
		{
			name:     "empty value",
			value:    "",
			pattern:  "*",
			expected: true,
		},
		{
			name:     "empty pattern no wildcard",
			value:    "test",
			pattern:  "",
			expected: false,
		},
		{
			name:     "both empty",
			value:    "",
			pattern:  "",
			expected: true,
		},
		{
			name:     "special regex chars escaped",
			value:    "test.value",
			pattern:  "test.value",
			expected: true,
		},
		{
			name:     "special regex chars with wildcard",
			value:    "test.value.extra",
			pattern:  "test.*",
			expected: true,
		},
		{
			name:     "port wildcard",
			value:    "192.168.1.100:3306",
			pattern:  "192.168.1.100:*",
			expected: true,
		},
		{
			name:     "container pattern",
			value:    "tomcat-18001",
			pattern:  "tomcat-*",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchPattern(tt.value, tt.pattern)
			if result != tt.expected {
				t.Errorf("MatchPattern(%q, %q) = %v, want %v", tt.value, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestMatchAnyPattern(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		patterns []string
		expected bool
	}{
		{
			name:     "empty patterns - no filtering",
			value:    "anything",
			patterns: []string{},
			expected: true,
		},
		{
			name:     "nil patterns - no filtering",
			value:    "anything",
			patterns: nil,
			expected: true,
		},
		{
			name:     "single pattern match",
			value:    "172.18.182.91:3306",
			patterns: []string{"172.18.182.*"},
			expected: true,
		},
		{
			name:     "single pattern no match",
			value:    "192.168.1.1:3306",
			patterns: []string{"172.18.182.*"},
			expected: false,
		},
		{
			name:     "multiple patterns - first matches",
			value:    "172.18.182.91:3306",
			patterns: []string{"172.18.182.*", "192.168.*"},
			expected: true,
		},
		{
			name:     "multiple patterns - second matches",
			value:    "192.168.1.1:3306",
			patterns: []string{"172.18.182.*", "192.168.*"},
			expected: true,
		},
		{
			name:     "multiple patterns - none match",
			value:    "10.0.0.1:3306",
			patterns: []string{"172.18.182.*", "192.168.*"},
			expected: false,
		},
		{
			name:     "exact match in list",
			value:    "server-01",
			patterns: []string{"server-01", "server-02"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchAnyPattern(tt.value, tt.patterns)
			if result != tt.expected {
				t.Errorf("MatchAnyPattern(%q, %v) = %v, want %v", tt.value, tt.patterns, result, tt.expected)
			}
		})
	}
}
