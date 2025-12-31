// Package report provides report generation functionality for inspection results.
// It defines the ReportWriter interface and provides a registry for managing
// different report formats (Excel, HTML, etc.).
package report

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"inspection-tool/internal/report/excel"
	"inspection-tool/internal/report/html"
)

// Registry manages report writers for different formats.
// It provides a centralized way to access report writers by format name.
type Registry struct {
	writers map[string]ReportWriter
}

// NewRegistry creates a new report registry with pre-registered Excel and HTML writers.
// If timezone is nil, defaults to Asia/Shanghai.
// htmlTemplatePath is optional; if empty, the HTML writer will use the embedded default template.
// excelTemplatePath is optional; if empty, the Excel writer will create reports from scratch.
func NewRegistry(timezone *time.Location, htmlTemplatePath, excelTemplatePath string) *Registry {
	// Set default timezone if not provided
	if timezone == nil {
		timezone, _ = time.LoadLocation("Asia/Shanghai")
	}

	// Create writers
	excelWriter := excel.NewWriter(timezone, excelTemplatePath)
	htmlWriter := html.NewWriter(timezone, htmlTemplatePath)

	// Build registry
	r := &Registry{
		writers: make(map[string]ReportWriter),
	}

	// Register writers using their Format() return values
	r.writers[excelWriter.Format()] = excelWriter
	r.writers[htmlWriter.Format()] = htmlWriter

	return r
}

// Get returns a writer for the specified format.
// Format names are case-insensitive (e.g., "Excel", "EXCEL", "excel" all work).
// Returns an error if the format is not supported.
func (r *Registry) Get(format string) (ReportWriter, error) {
	// Normalize format to lowercase for case-insensitive lookup
	normalizedFormat := strings.ToLower(strings.TrimSpace(format))

	writer, ok := r.writers[normalizedFormat]
	if !ok {
		supported := r.GetAll()
		return nil, fmt.Errorf("unsupported report format %q, supported formats: %s",
			format, strings.Join(supported, ", "))
	}

	return writer, nil
}

// GetAll returns all supported format names in sorted order.
func (r *Registry) GetAll() []string {
	formats := make([]string, 0, len(r.writers))
	for format := range r.writers {
		formats = append(formats, format)
	}
	sort.Strings(formats)
	return formats
}

// Has checks if the specified format is supported.
// Format names are case-insensitive.
func (r *Registry) Has(format string) bool {
	normalizedFormat := strings.ToLower(strings.TrimSpace(format))
	_, ok := r.writers[normalizedFormat]
	return ok
}
