// Package report provides report generation functionality for the inspection tool.
// It defines the ReportWriter interface and provides implementations for
// different output formats including Excel and HTML.
package report

import (
	"inspection-tool/internal/model"
)

// ReportWriter defines the interface for generating inspection reports.
// Implementations should be able to write inspection results to files
// in their specific format (Excel, HTML, etc.).
type ReportWriter interface {
	// Write generates a report from the inspection result and saves it
	// to the specified output path. The path should include the file
	// extension appropriate for the format.
	//
	// Returns an error if the report generation or file writing fails.
	Write(result *model.InspectionResult, outputPath string) error

	// Format returns the format identifier for this writer.
	// Common values are "excel" and "html".
	Format() string
}
