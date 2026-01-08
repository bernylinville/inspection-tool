// Package util provides common utility functions for the inspection tool.
package util

// DefaultAddressLabels defines the default label keys to search for instance addresses.
// The order matters: first match wins.
var DefaultAddressLabels = []string{"address", "instance", "server"}

// ExtractAddress extracts instance address from metric labels.
// Tries labels in order: "address", "instance", "server".
// Returns empty string if no address label is found.
//
// This function is used by MySQL and Redis collectors to extract
// the instance address from VictoriaMetrics query results.
func ExtractAddress(labels map[string]string) string {
	return ExtractAddressWithKeys(labels, DefaultAddressLabels)
}

// ExtractAddressWithKeys extracts instance address from metric labels
// using custom label keys. Tries labels in the provided order.
// Returns empty string if no matching label is found.
func ExtractAddressWithKeys(labels map[string]string, keys []string) string {
	for _, key := range keys {
		if addr, ok := labels[key]; ok && addr != "" {
			return addr
		}
	}
	return ""
}

// ExtractHostname extracts hostname from metric labels.
// Tries labels in order: "agent_hostname", "ident", "host".
// Returns empty string if no hostname label is found.
//
// This function is used by Nginx and Tomcat collectors to extract
// the hostname from VictoriaMetrics query results.
func ExtractHostname(labels map[string]string) string {
	hostnameLabels := []string{"agent_hostname", "ident", "host"}
	for _, key := range hostnameLabels {
		if hostname, ok := labels[key]; ok && hostname != "" {
			return hostname
		}
	}
	return ""
}

// ExtractLabelValue extracts a single label value from metric labels.
// Returns empty string if the label is not found.
func ExtractLabelValue(labels map[string]string, key string) string {
	if val, ok := labels[key]; ok {
		return val
	}
	return ""
}
