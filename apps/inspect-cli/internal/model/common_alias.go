// Package model provides data models for the inspection tool.
package model

import pkgmodel "inspection-tool/pkg/model"

// Host and alert aliases.
type HostStatus = pkgmodel.HostStatus

type DiskMountInfo = pkgmodel.DiskMountInfo

type HostMeta = pkgmodel.HostMeta

type AlertLevel = pkgmodel.AlertLevel

type Alert = pkgmodel.Alert

type AlertSummary = pkgmodel.AlertSummary

// Metric aliases.
type MetricStatus = pkgmodel.MetricStatus

type MetricCategory = pkgmodel.MetricCategory

type MetricFormat = pkgmodel.MetricFormat

type AggregateType = pkgmodel.AggregateType

type MetricDefinition = pkgmodel.MetricDefinition

type MetricValue = pkgmodel.MetricValue

type HostMetrics = pkgmodel.HostMetrics

type MetricsConfig = pkgmodel.MetricsConfig

// CleanIdent extracts the hostname from an ident string.
func CleanIdent(ident string) string {
	return pkgmodel.CleanIdent(ident)
}

// NewAlert creates a new Alert with the given parameters.
func NewAlert(hostname, metricName string, currentValue float64, level AlertLevel) *Alert {
	return pkgmodel.NewAlert(hostname, metricName, currentValue, level)
}

// NewAlertSummary creates a new AlertSummary from a list of alerts.
func NewAlertSummary(alerts []*Alert) *AlertSummary {
	return pkgmodel.NewAlertSummary(alerts)
}

// NewMetricValue creates a new MetricValue with the given raw value.
func NewMetricValue(name string, rawValue float64) *MetricValue {
	return pkgmodel.NewMetricValue(name, rawValue)
}

// NewNAMetricValue creates a MetricValue representing "N/A" for pending metrics.
func NewNAMetricValue(name string) *MetricValue {
	return pkgmodel.NewNAMetricValue(name)
}

// NewHostMetrics creates a new HostMetrics instance for the given hostname.
func NewHostMetrics(hostname string) *HostMetrics {
	return pkgmodel.NewHostMetrics(hostname)
}

// HostStatus constants.
const (
	HostStatusNormal   = pkgmodel.HostStatusNormal
	HostStatusWarning  = pkgmodel.HostStatusWarning
	HostStatusCritical = pkgmodel.HostStatusCritical
	HostStatusFailed   = pkgmodel.HostStatusFailed
)

// AlertLevel constants.
const (
	AlertLevelNormal   = pkgmodel.AlertLevelNormal
	AlertLevelWarning  = pkgmodel.AlertLevelWarning
	AlertLevelCritical = pkgmodel.AlertLevelCritical
)

// MetricStatus constants.
const (
	MetricStatusNormal   = pkgmodel.MetricStatusNormal
	MetricStatusWarning  = pkgmodel.MetricStatusWarning
	MetricStatusCritical = pkgmodel.MetricStatusCritical
	MetricStatusPending  = pkgmodel.MetricStatusPending
)

// MetricFormat constants.
const (
	MetricFormatPercent   = pkgmodel.MetricFormatPercent
	MetricFormatSize      = pkgmodel.MetricFormatSize
	MetricFormatDuration  = pkgmodel.MetricFormatDuration
	MetricFormatNumber    = pkgmodel.MetricFormatNumber
	MetricFormatNTPOffset = pkgmodel.MetricFormatNTPOffset
)

// AggregateType constants.
const (
	AggregateMax = pkgmodel.AggregateMax
	AggregateMin = pkgmodel.AggregateMin
	AggregateAvg = pkgmodel.AggregateAvg
)
