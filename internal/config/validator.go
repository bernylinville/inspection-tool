// Package config provides configuration management for the inspection tool.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

// ValidationError represents a single validation error with user-friendly message.
type ValidationError struct {
	Field   string      // Field path (e.g., "datasources.n9e.endpoint")
	Tag     string      // Validation tag that failed (e.g., "required", "url")
	Value   interface{} // Actual value that failed validation
	Message string      // User-friendly error message
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return e.Message
}

// ValidationErrors is a collection of validation errors.
type ValidationErrors []*ValidationError

// Error implements the error interface.
func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("config validation failed:\n")
	for _, err := range e {
		sb.WriteString(fmt.Sprintf("  - %s: %s\n", err.Field, err.Message))
	}
	return sb.String()
}

// validate is the package-level validator instance.
var validate *validator.Validate

// init initializes the validator with custom validations.
func init() {
	validate = validator.New()

	// Register custom validation for timezone
	validate.RegisterValidation("timezone", validateTimezone)
}

// Validate validates the configuration and returns user-friendly error messages.
func Validate(cfg *Config) error {
	var validationErrors ValidationErrors

	// Run struct validation
	if err := validate.Struct(cfg); err != nil {
		if fieldErrors, ok := err.(validator.ValidationErrors); ok {
			for _, fe := range fieldErrors {
				validationErrors = append(validationErrors, &ValidationError{
					Field:   formatFieldName(fe.Namespace()),
					Tag:     fe.Tag(),
					Value:   fe.Value(),
					Message: translateError(fe),
				})
			}
		}
	}

	// Run custom business logic validations
	if errs := validateThresholds(cfg); len(errs) > 0 {
		validationErrors = append(validationErrors, errs...)
	}

	if errs := validateTimezoneConfig(cfg); len(errs) > 0 {
		validationErrors = append(validationErrors, errs...)
	}

	if errs := validateMySQLThresholds(cfg); len(errs) > 0 {
		validationErrors = append(validationErrors, errs...)
	}

	if len(validationErrors) > 0 {
		return validationErrors
	}

	return nil
}

// validateTimezone is a custom validator for timezone strings.
func validateTimezone(fl validator.FieldLevel) bool {
	tz := fl.Field().String()
	if tz == "" {
		return true // Empty is allowed, will use default
	}
	_, err := time.LoadLocation(tz)
	return err == nil
}

// validateThresholds validates that warning thresholds are less than critical thresholds.
func validateThresholds(cfg *Config) ValidationErrors {
	var errors ValidationErrors

	thresholdPairs := []struct {
		name     string
		warning  float64
		critical float64
	}{
		{"thresholds.cpu_usage", cfg.Thresholds.CPUUsage.Warning, cfg.Thresholds.CPUUsage.Critical},
		{"thresholds.memory_usage", cfg.Thresholds.MemoryUsage.Warning, cfg.Thresholds.MemoryUsage.Critical},
		{"thresholds.disk_usage", cfg.Thresholds.DiskUsage.Warning, cfg.Thresholds.DiskUsage.Critical},
		{"thresholds.zombie_processes", cfg.Thresholds.ZombieProcesses.Warning, cfg.Thresholds.ZombieProcesses.Critical},
		{"thresholds.load_per_core", cfg.Thresholds.LoadPerCore.Warning, cfg.Thresholds.LoadPerCore.Critical},
	}

	for _, tp := range thresholdPairs {
		if tp.warning >= tp.critical {
			errors = append(errors, &ValidationError{
				Field:   tp.name,
				Tag:     "threshold_order",
				Value:   fmt.Sprintf("warning=%v, critical=%v", tp.warning, tp.critical),
				Message: fmt.Sprintf("warning threshold (%.2f) must be less than critical threshold (%.2f)", tp.warning, tp.critical),
			})
		}
	}

	return errors
}

// validateTimezoneConfig validates the timezone configuration.
func validateTimezoneConfig(cfg *Config) ValidationErrors {
	var errors ValidationErrors

	if cfg.Report.Timezone != "" {
		if _, err := time.LoadLocation(cfg.Report.Timezone); err != nil {
			errors = append(errors, &ValidationError{
				Field:   "report.timezone",
				Tag:     "timezone",
				Value:   cfg.Report.Timezone,
				Message: fmt.Sprintf("invalid timezone: %s", cfg.Report.Timezone),
			})
		}
	}

	return errors
}

// validateMySQLThresholds validates MySQL threshold configuration.
func validateMySQLThresholds(cfg *Config) ValidationErrors {
	var errors ValidationErrors

	// Skip validation if MySQL inspection is disabled
	if !cfg.MySQL.Enabled {
		return errors
	}

	// Validate connection usage thresholds (warning < critical)
	if cfg.MySQL.Thresholds.ConnectionUsageWarning >= cfg.MySQL.Thresholds.ConnectionUsageCritical {
		errors = append(errors, &ValidationError{
			Field:   "mysql.thresholds.connection_usage",
			Tag:     "threshold_order",
			Value:   fmt.Sprintf("warning=%v, critical=%v", cfg.MySQL.Thresholds.ConnectionUsageWarning, cfg.MySQL.Thresholds.ConnectionUsageCritical),
			Message: fmt.Sprintf("warning threshold (%.2f) must be less than critical threshold (%.2f)", cfg.MySQL.Thresholds.ConnectionUsageWarning, cfg.MySQL.Thresholds.ConnectionUsageCritical),
		})
	}

	// Validate cluster_mode is set when enabled
	if cfg.MySQL.ClusterMode == "" {
		errors = append(errors, &ValidationError{
			Field:   "mysql.cluster_mode",
			Tag:     "required_when_enabled",
			Value:   "",
			Message: "cluster_mode is required when MySQL inspection is enabled",
		})
	}

	return errors
}

// formatFieldName converts the validator field namespace to a user-friendly format.
// Example: "Config.Datasources.N9E.Endpoint" -> "datasources.n9e.endpoint"
func formatFieldName(namespace string) string {
	// Remove the root struct name (e.g., "Config.")
	parts := strings.Split(namespace, ".")
	if len(parts) > 1 {
		parts = parts[1:] // Remove "Config"
	}

	// Convert to lowercase and join
	for i, part := range parts {
		parts[i] = strings.ToLower(part)
	}

	return strings.Join(parts, ".")
}

// translateError converts a validator.FieldError to a user-friendly message.
func translateError(fe validator.FieldError) string {
	field := formatFieldName(fe.Namespace())

	switch fe.Tag() {
	case "required":
		return "this field is required"
	case "url":
		return fmt.Sprintf("invalid URL format: %v", fe.Value())
	case "gte":
		return fmt.Sprintf("value must be greater than or equal to %s", fe.Param())
	case "lte":
		return fmt.Sprintf("value must be less than or equal to %s", fe.Param())
	case "oneof":
		return fmt.Sprintf("value must be one of: %s", fe.Param())
	case "dive":
		return fmt.Sprintf("invalid value in list: %v", fe.Value())
	case "timezone":
		return fmt.Sprintf("invalid timezone: %v", fe.Value())
	default:
		return fmt.Sprintf("validation failed on '%s' tag for field '%s'", fe.Tag(), field)
	}
}
