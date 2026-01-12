package model

import "testing"

func TestCleanIdent(t *testing.T) {
	tests := []struct {
		name     string
		ident    string
		expected string
	}{
		{
			name:     "hostname@IP format",
			ident:    "server-01@192.168.1.1",
			expected: "server-01",
		},
		{
			name:     "hostname only",
			ident:    "server-01",
			expected: "server-01",
		},
		{
			name:     "empty string",
			ident:    "",
			expected: "",
		},
		{
			name:     "@ at start returns empty",
			ident:    "@192.168.1.1",
			expected: "@192.168.1.1",
		},
		{
			name:     "hostname with multiple @ symbols",
			ident:    "server-01@192.168.1.1@extra",
			expected: "server-01",
		},
		{
			name:     "complex hostname",
			ident:    "prod-web-server-001@10.0.0.100",
			expected: "prod-web-server-001",
		},
		{
			name:     "hostname with hyphen and numbers",
			ident:    "db-mysql-3306@172.16.0.50",
			expected: "db-mysql-3306",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanIdent(tt.ident)
			if result != tt.expected {
				t.Errorf("CleanIdent(%q) = %q, want %q", tt.ident, result, tt.expected)
			}
		})
	}
}
