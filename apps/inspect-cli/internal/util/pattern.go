package util

import (
	"regexp"
	"strings"
)

// MatchPattern checks if a value matches a pattern with wildcard support.
// Supports wildcard '*' which matches any sequence of characters.
//
// Examples:
//   - "172.18.182.*" matches "172.18.182.91:3306"
//   - "192.168.1.100:*" matches "192.168.1.100:3306"
//   - "GX-NM-*" matches "GX-NM-MNS-NGX-01"
//   - "*" matches all values
func MatchPattern(value, pattern string) bool {
	if value == pattern {
		return true
	}

	if !strings.Contains(pattern, "*") {
		return false
	}

	regexPattern := regexp.QuoteMeta(pattern)
	regexPattern = strings.ReplaceAll(regexPattern, "\\*", ".*")
	regexPattern = "^" + regexPattern + "$"

	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return false
	}

	return re.MatchString(value)
}

// MatchAnyPattern checks if a value matches any of the provided patterns.
// Returns true if patterns is empty (no filtering) or value matches at least one pattern.
func MatchAnyPattern(value string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}

	for _, pattern := range patterns {
		if MatchPattern(value, pattern) {
			return true
		}
	}

	return false
}
