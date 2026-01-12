package util

import (
	"testing"
)

func TestExtractAddress(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		expected string
	}{
		{
			name:     "address label present",
			labels:   map[string]string{"address": "172.18.182.91:3306"},
			expected: "172.18.182.91:3306",
		},
		{
			name:     "instance label present",
			labels:   map[string]string{"instance": "192.168.1.100:3306"},
			expected: "192.168.1.100:3306",
		},
		{
			name:     "server label present",
			labels:   map[string]string{"server": "10.0.0.1:6379"},
			expected: "10.0.0.1:6379",
		},
		{
			name:     "address takes priority over instance",
			labels:   map[string]string{"address": "addr1", "instance": "inst1"},
			expected: "addr1",
		},
		{
			name:     "instance takes priority over server",
			labels:   map[string]string{"instance": "inst1", "server": "srv1"},
			expected: "inst1",
		},
		{
			name:     "empty labels",
			labels:   map[string]string{},
			expected: "",
		},
		{
			name:     "nil labels",
			labels:   nil,
			expected: "",
		},
		{
			name:     "empty address value falls through",
			labels:   map[string]string{"address": "", "instance": "inst1"},
			expected: "inst1",
		},
		{
			name:     "unrelated labels only",
			labels:   map[string]string{"foo": "bar", "baz": "qux"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractAddress(tt.labels)
			if result != tt.expected {
				t.Errorf("ExtractAddress() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractAddressWithKeys(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		keys     []string
		expected string
	}{
		{
			name:     "custom keys",
			labels:   map[string]string{"custom_addr": "192.168.1.1"},
			keys:     []string{"custom_addr"},
			expected: "192.168.1.1",
		},
		{
			name:     "custom keys priority",
			labels:   map[string]string{"first": "val1", "second": "val2"},
			keys:     []string{"first", "second"},
			expected: "val1",
		},
		{
			name:     "empty keys",
			labels:   map[string]string{"address": "addr1"},
			keys:     []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractAddressWithKeys(tt.labels, tt.keys)
			if result != tt.expected {
				t.Errorf("ExtractAddressWithKeys() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractHostname(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		expected string
	}{
		{
			name:     "agent_hostname present",
			labels:   map[string]string{"agent_hostname": "GX-NM-MNS-NGX-01"},
			expected: "GX-NM-MNS-NGX-01",
		},
		{
			name:     "ident present",
			labels:   map[string]string{"ident": "server-01"},
			expected: "server-01",
		},
		{
			name:     "host present",
			labels:   map[string]string{"host": "myhost"},
			expected: "myhost",
		},
		{
			name:     "agent_hostname takes priority",
			labels:   map[string]string{"agent_hostname": "host1", "ident": "host2"},
			expected: "host1",
		},
		{
			name:     "empty labels",
			labels:   map[string]string{},
			expected: "",
		},
		{
			name:     "nil labels",
			labels:   nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractHostname(tt.labels)
			if result != tt.expected {
				t.Errorf("ExtractHostname() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractLabelValue(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		key      string
		expected string
	}{
		{
			name:     "key exists",
			labels:   map[string]string{"version": "8.0.39"},
			key:      "version",
			expected: "8.0.39",
		},
		{
			name:     "key not found",
			labels:   map[string]string{"other": "value"},
			key:      "version",
			expected: "",
		},
		{
			name:     "empty labels",
			labels:   map[string]string{},
			key:      "version",
			expected: "",
		},
		{
			name:     "nil labels",
			labels:   nil,
			key:      "version",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractLabelValue(tt.labels, tt.key)
			if result != tt.expected {
				t.Errorf("ExtractLabelValue() = %q, want %q", result, tt.expected)
			}
		})
	}
}
