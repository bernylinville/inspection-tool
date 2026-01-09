// Package service provides business logic services for the inspection tool.
//
// This file defines interfaces for Collectors and Evaluators following Go best practices:
// - Consumer-side interface definition
// - Interface segregation (small, focused interfaces)
// - Compile-time interface satisfaction checks
//
// These interfaces enable:
// - Dependency injection for better testability
// - Loose coupling between components
// - Easy mocking in unit tests
package service

import (
	"context"

	"inspection-tool/internal/model"
)

// =============================================================================
// Host Inspection Interfaces
// =============================================================================

// HostCollector defines the interface for collecting host system metrics.
// Used by Inspector to collect host metadata and metrics from N9E and VictoriaMetrics.
type HostCollector interface {
	// CollectAll executes the complete data collection workflow.
	// Returns host metadata, metrics, and any failed hosts.
	CollectAll(ctx context.Context) (*CollectionResult, error)
}

// HostEvaluator defines the interface for evaluating host metrics against thresholds.
// Used by Inspector to generate alerts based on collected metrics.
type HostEvaluator interface {
	// EvaluateAll evaluates all hosts and returns the complete evaluation result.
	EvaluateAll(hostMetrics map[string]*model.HostMetrics) *EvaluationResult
}

// =============================================================================
// Instance-Based Inspection Interfaces (Generic)
// =============================================================================

// InstanceDiscoverer defines the interface for discovering service instances.
// Type parameter T is the instance type (e.g., MySQLInstance, RedisInstance).
type InstanceDiscoverer[T any] interface {
	// DiscoverInstances discovers all instances by querying metrics.
	// Returns a list of discovered instances.
	DiscoverInstances(ctx context.Context) ([]*T, error)
}

// MetricDefinitionProvider defines the interface for providing metric definitions.
// Type parameter M is the metric definition type.
type MetricDefinitionProvider[M any] interface {
	// GetMetrics returns the list of metric definitions.
	GetMetrics() []*M
}

// =============================================================================
// MySQL Inspection Interfaces
// =============================================================================

// MySQLInstanceCollector defines the interface for MySQL instance collection.
// Combines instance discovery, metric collection, and metric definition access.
type MySQLInstanceCollector interface {
	InstanceDiscoverer[model.MySQLInstance]
	MetricDefinitionProvider[model.MySQLMetricDefinition]

	// CollectMetrics retrieves metric data for all MySQL instances.
	CollectMetrics(
		ctx context.Context,
		instances []*model.MySQLInstance,
		metrics []*model.MySQLMetricDefinition,
	) (map[string]*model.MySQLInspectionResult, error)
}

// MySQLInstanceEvaluator defines the interface for evaluating MySQL metrics.
type MySQLInstanceEvaluator interface {
	// EvaluateAll evaluates all MySQL instances against thresholds.
	EvaluateAll(results map[string]*model.MySQLInspectionResult) []*MySQLEvaluationResult
}

// =============================================================================
// Redis Inspection Interfaces
// =============================================================================

// RedisInstanceCollector defines the interface for Redis instance collection.
type RedisInstanceCollector interface {
	InstanceDiscoverer[model.RedisInstance]
	MetricDefinitionProvider[model.RedisMetricDefinition]

	// CollectMetrics retrieves metric data for all Redis instances.
	CollectMetrics(
		ctx context.Context,
		instances []*model.RedisInstance,
		metrics []*model.RedisMetricDefinition,
	) (map[string]*model.RedisInspectionResult, error)
}

// RedisInstanceEvaluator defines the interface for evaluating Redis metrics.
type RedisInstanceEvaluator interface {
	// EvaluateAll evaluates all Redis instances against thresholds.
	EvaluateAll(results map[string]*model.RedisInspectionResult) []*RedisEvaluationResult
}

// =============================================================================
// Nginx Inspection Interfaces
// =============================================================================

// NginxInstanceCollector defines the interface for Nginx instance collection.
type NginxInstanceCollector interface {
	InstanceDiscoverer[model.NginxInstance]
	MetricDefinitionProvider[model.NginxMetricDefinition]

	// CollectMetrics retrieves metric data for all Nginx instances.
	CollectMetrics(
		ctx context.Context,
		instances []*model.NginxInstance,
		metrics []*model.NginxMetricDefinition,
	) (map[string]*model.NginxInspectionResult, error)

	// CollectUpstreamStatus collects upstream backend status for all instances.
	CollectUpstreamStatus(
		ctx context.Context,
		resultsMap map[string]*model.NginxInspectionResult,
	) error
}

// NginxInstanceEvaluator defines the interface for evaluating Nginx metrics.
type NginxInstanceEvaluator interface {
	// EvaluateAll evaluates all Nginx instances against thresholds.
	EvaluateAll(results map[string]*model.NginxInspectionResult) []*NginxEvaluationResult
}

// =============================================================================
// Tomcat Inspection Interfaces
// =============================================================================

// TomcatInstanceCollector defines the interface for Tomcat instance collection.
type TomcatInstanceCollector interface {
	InstanceDiscoverer[model.TomcatInstance]
	MetricDefinitionProvider[model.TomcatMetricDefinition]

	// CollectMetrics retrieves metric data for all Tomcat instances.
	CollectMetrics(
		ctx context.Context,
		instances []*model.TomcatInstance,
		metrics []*model.TomcatMetricDefinition,
	) (map[string]*model.TomcatInspectionResult, error)
}

// TomcatInstanceEvaluator defines the interface for evaluating Tomcat metrics.
type TomcatInstanceEvaluator interface {
	// EvaluateAll evaluates all Tomcat instances against thresholds.
	EvaluateAll(results map[string]*model.TomcatInspectionResult) []*TomcatEvaluationResult
}

// =============================================================================
// Elasticsearch Inspection Interfaces
// =============================================================================

type ElasticsearchInstanceCollector interface {
	InstanceDiscoverer[model.ElasticsearchInstance]
	MetricDefinitionProvider[model.ElasticsearchMetricDefinition]

	CollectMetrics(
		ctx context.Context,
		instances []*model.ElasticsearchInstance,
		metrics []*model.ElasticsearchMetricDefinition,
	) (map[string]*model.ElasticsearchInspectionResult, error)
}

type ElasticsearchInstanceEvaluator interface {
	EvaluateAll(results map[string]*model.ElasticsearchInspectionResult) []*ElasticsearchEvaluationResult
}

// =============================================================================
// Compile-Time Interface Satisfaction Checks
// =============================================================================

// These compile-time checks ensure that concrete types implement their interfaces.
// If a type doesn't implement an interface, compilation will fail with a clear error.

// Host inspection
var _ HostCollector = (*Collector)(nil)
var _ HostEvaluator = (*Evaluator)(nil)

// MySQL inspection
var _ MySQLInstanceCollector = (*MySQLCollector)(nil)
var _ MySQLInstanceEvaluator = (*MySQLEvaluator)(nil)

// Redis inspection
var _ RedisInstanceCollector = (*RedisCollector)(nil)
var _ RedisInstanceEvaluator = (*RedisEvaluator)(nil)

// Nginx inspection
var _ NginxInstanceCollector = (*NginxCollector)(nil)
var _ NginxInstanceEvaluator = (*NginxEvaluator)(nil)

// Tomcat inspection
var _ TomcatInstanceCollector = (*TomcatCollector)(nil)
var _ TomcatInstanceEvaluator = (*TomcatEvaluator)(nil)

// Elasticsearch inspection
var _ ElasticsearchInstanceCollector = (*ElasticsearchCollector)(nil)
var _ ElasticsearchInstanceEvaluator = (*ElasticsearchEvaluator)(nil)
