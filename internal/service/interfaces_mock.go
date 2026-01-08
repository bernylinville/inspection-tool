package service

import (
	"context"

	"inspection-tool/internal/model"
)

type MockHostCollector struct {
	CollectAllFunc func(ctx context.Context) (*CollectionResult, error)
}

func (m *MockHostCollector) CollectAll(ctx context.Context) (*CollectionResult, error) {
	if m.CollectAllFunc != nil {
		return m.CollectAllFunc(ctx)
	}
	return &CollectionResult{}, nil
}

type MockHostEvaluator struct {
	EvaluateAllFunc func(hostMetrics map[string]*model.HostMetrics) *EvaluationResult
}

func (m *MockHostEvaluator) EvaluateAll(hostMetrics map[string]*model.HostMetrics) *EvaluationResult {
	if m.EvaluateAllFunc != nil {
		return m.EvaluateAllFunc(hostMetrics)
	}
	return &EvaluationResult{}
}

type MockMySQLCollector struct {
	DiscoverInstancesFunc func(ctx context.Context) ([]*model.MySQLInstance, error)
	GetMetricsFunc        func() []*model.MySQLMetricDefinition
	CollectMetricsFunc    func(ctx context.Context, instances []*model.MySQLInstance, metrics []*model.MySQLMetricDefinition) (map[string]*model.MySQLInspectionResult, error)
}

func (m *MockMySQLCollector) DiscoverInstances(ctx context.Context) ([]*model.MySQLInstance, error) {
	if m.DiscoverInstancesFunc != nil {
		return m.DiscoverInstancesFunc(ctx)
	}
	return nil, nil
}

func (m *MockMySQLCollector) GetMetrics() []*model.MySQLMetricDefinition {
	if m.GetMetricsFunc != nil {
		return m.GetMetricsFunc()
	}
	return nil
}

func (m *MockMySQLCollector) CollectMetrics(ctx context.Context, instances []*model.MySQLInstance, metrics []*model.MySQLMetricDefinition) (map[string]*model.MySQLInspectionResult, error) {
	if m.CollectMetricsFunc != nil {
		return m.CollectMetricsFunc(ctx, instances, metrics)
	}
	return make(map[string]*model.MySQLInspectionResult), nil
}

type MockMySQLEvaluator struct {
	EvaluateAllFunc func(results map[string]*model.MySQLInspectionResult) []*MySQLEvaluationResult
}

func (m *MockMySQLEvaluator) EvaluateAll(results map[string]*model.MySQLInspectionResult) []*MySQLEvaluationResult {
	if m.EvaluateAllFunc != nil {
		return m.EvaluateAllFunc(results)
	}
	return nil
}

type MockRedisCollector struct {
	DiscoverInstancesFunc func(ctx context.Context) ([]*model.RedisInstance, error)
	GetMetricsFunc        func() []*model.RedisMetricDefinition
	CollectMetricsFunc    func(ctx context.Context, instances []*model.RedisInstance, metrics []*model.RedisMetricDefinition) (map[string]*model.RedisInspectionResult, error)
}

func (m *MockRedisCollector) DiscoverInstances(ctx context.Context) ([]*model.RedisInstance, error) {
	if m.DiscoverInstancesFunc != nil {
		return m.DiscoverInstancesFunc(ctx)
	}
	return nil, nil
}

func (m *MockRedisCollector) GetMetrics() []*model.RedisMetricDefinition {
	if m.GetMetricsFunc != nil {
		return m.GetMetricsFunc()
	}
	return nil
}

func (m *MockRedisCollector) CollectMetrics(ctx context.Context, instances []*model.RedisInstance, metrics []*model.RedisMetricDefinition) (map[string]*model.RedisInspectionResult, error) {
	if m.CollectMetricsFunc != nil {
		return m.CollectMetricsFunc(ctx, instances, metrics)
	}
	return make(map[string]*model.RedisInspectionResult), nil
}

type MockRedisEvaluator struct {
	EvaluateAllFunc func(results map[string]*model.RedisInspectionResult) []*RedisEvaluationResult
}

func (m *MockRedisEvaluator) EvaluateAll(results map[string]*model.RedisInspectionResult) []*RedisEvaluationResult {
	if m.EvaluateAllFunc != nil {
		return m.EvaluateAllFunc(results)
	}
	return nil
}

type MockNginxCollector struct {
	DiscoverInstancesFunc     func(ctx context.Context) ([]*model.NginxInstance, error)
	GetMetricsFunc            func() []*model.NginxMetricDefinition
	CollectMetricsFunc        func(ctx context.Context, instances []*model.NginxInstance, metrics []*model.NginxMetricDefinition) (map[string]*model.NginxInspectionResult, error)
	CollectUpstreamStatusFunc func(ctx context.Context, resultsMap map[string]*model.NginxInspectionResult) error
}

func (m *MockNginxCollector) DiscoverInstances(ctx context.Context) ([]*model.NginxInstance, error) {
	if m.DiscoverInstancesFunc != nil {
		return m.DiscoverInstancesFunc(ctx)
	}
	return nil, nil
}

func (m *MockNginxCollector) GetMetrics() []*model.NginxMetricDefinition {
	if m.GetMetricsFunc != nil {
		return m.GetMetricsFunc()
	}
	return nil
}

func (m *MockNginxCollector) CollectMetrics(ctx context.Context, instances []*model.NginxInstance, metrics []*model.NginxMetricDefinition) (map[string]*model.NginxInspectionResult, error) {
	if m.CollectMetricsFunc != nil {
		return m.CollectMetricsFunc(ctx, instances, metrics)
	}
	return make(map[string]*model.NginxInspectionResult), nil
}

func (m *MockNginxCollector) CollectUpstreamStatus(ctx context.Context, resultsMap map[string]*model.NginxInspectionResult) error {
	if m.CollectUpstreamStatusFunc != nil {
		return m.CollectUpstreamStatusFunc(ctx, resultsMap)
	}
	return nil
}

type MockNginxEvaluator struct {
	EvaluateAllFunc func(results map[string]*model.NginxInspectionResult) []*NginxEvaluationResult
}

func (m *MockNginxEvaluator) EvaluateAll(results map[string]*model.NginxInspectionResult) []*NginxEvaluationResult {
	if m.EvaluateAllFunc != nil {
		return m.EvaluateAllFunc(results)
	}
	return nil
}

type MockTomcatCollector struct {
	DiscoverInstancesFunc func(ctx context.Context) ([]*model.TomcatInstance, error)
	GetMetricsFunc        func() []*model.TomcatMetricDefinition
	CollectMetricsFunc    func(ctx context.Context, instances []*model.TomcatInstance, metrics []*model.TomcatMetricDefinition) (map[string]*model.TomcatInspectionResult, error)
}

func (m *MockTomcatCollector) DiscoverInstances(ctx context.Context) ([]*model.TomcatInstance, error) {
	if m.DiscoverInstancesFunc != nil {
		return m.DiscoverInstancesFunc(ctx)
	}
	return nil, nil
}

func (m *MockTomcatCollector) GetMetrics() []*model.TomcatMetricDefinition {
	if m.GetMetricsFunc != nil {
		return m.GetMetricsFunc()
	}
	return nil
}

func (m *MockTomcatCollector) CollectMetrics(ctx context.Context, instances []*model.TomcatInstance, metrics []*model.TomcatMetricDefinition) (map[string]*model.TomcatInspectionResult, error) {
	if m.CollectMetricsFunc != nil {
		return m.CollectMetricsFunc(ctx, instances, metrics)
	}
	return make(map[string]*model.TomcatInspectionResult), nil
}

type MockTomcatEvaluator struct {
	EvaluateAllFunc func(results map[string]*model.TomcatInspectionResult) []*TomcatEvaluationResult
}

func (m *MockTomcatEvaluator) EvaluateAll(results map[string]*model.TomcatInspectionResult) []*TomcatEvaluationResult {
	if m.EvaluateAllFunc != nil {
		return m.EvaluateAllFunc(results)
	}
	return nil
}

var (
	_ HostCollector = (*MockHostCollector)(nil)
	_ HostEvaluator = (*MockHostEvaluator)(nil)

	_ MySQLInstanceCollector = (*MockMySQLCollector)(nil)
	_ MySQLInstanceEvaluator = (*MockMySQLEvaluator)(nil)

	_ RedisInstanceCollector = (*MockRedisCollector)(nil)
	_ RedisInstanceEvaluator = (*MockRedisEvaluator)(nil)

	_ NginxInstanceCollector = (*MockNginxCollector)(nil)
	_ NginxInstanceEvaluator = (*MockNginxEvaluator)(nil)

	_ TomcatInstanceCollector = (*MockTomcatCollector)(nil)
	_ TomcatInstanceEvaluator = (*MockTomcatEvaluator)(nil)
)
