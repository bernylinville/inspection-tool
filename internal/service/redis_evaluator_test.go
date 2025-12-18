package service

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inspection-tool/internal/config"
	"inspection-tool/internal/model"
)

// createTestRedisEvaluator creates a RedisEvaluator with default test thresholds.
func createTestRedisEvaluator() *RedisEvaluator {
	thresholds := &config.RedisThresholds{
		ConnectionUsageWarning:  70,
		ConnectionUsageCritical: 90,
		ReplicationLagWarning:   1048576,  // 1MB
		ReplicationLagCritical:  10485760, // 10MB
	}

	metrics := []*model.RedisMetricDefinition{
		{Name: "connection_usage", DisplayName: "连接使用率"},
		{Name: "replication_lag", DisplayName: "复制延迟"},
		{Name: "master_link_status", DisplayName: "主从链接状态"},
	}

	logger := zerolog.Nop()
	return NewRedisEvaluator(thresholds, metrics, logger)
}

// createTestRedisResult creates a test RedisInspectionResult with given parameters.
func createTestRedisResult(address string, role model.RedisRole, connectedClients, maxClients int) *model.RedisInspectionResult {
	instance := model.NewRedisInstanceWithRole(address, role)
	result := model.NewRedisInspectionResult(instance)
	result.ConnectedClients = connectedClients
	result.MaxClients = maxClients
	result.MasterLinkStatus = true // Default to connected
	return result
}

// =============================================================================
// Constructor Tests
// =============================================================================

func TestNewRedisEvaluator(t *testing.T) {
	evaluator := createTestRedisEvaluator()

	assert.NotNil(t, evaluator)
	assert.NotNil(t, evaluator.thresholds)
	assert.NotNil(t, evaluator.metricDefs)
	assert.Len(t, evaluator.metricDefs, 3)
}

// =============================================================================
// EvaluateAll Tests
// =============================================================================

func TestRedisEvaluator_EvaluateAll_Success(t *testing.T) {
	evaluator := createTestRedisEvaluator()

	results := map[string]*model.RedisInspectionResult{
		"192.18.102.2:7000": createTestRedisResult("192.18.102.2:7000", model.RedisRoleMaster, 50, 1000),
		"192.18.102.2:7001": createTestRedisResult("192.18.102.2:7001", model.RedisRoleSlave, 30, 1000),
		"192.18.102.3:7000": createTestRedisResult("192.18.102.3:7000", model.RedisRoleMaster, 800, 1000), // Warning
	}

	evalResults := evaluator.EvaluateAll(results)

	assert.Len(t, evalResults, 3)

	// Check that at least one has warning status
	hasWarning := false
	for _, r := range evalResults {
		if r.Status == model.RedisStatusWarning {
			hasWarning = true
			break
		}
	}
	assert.True(t, hasWarning, "Expected at least one warning status")
}

func TestRedisEvaluator_EvaluateAll_EmptyResults(t *testing.T) {
	evaluator := createTestRedisEvaluator()

	results := map[string]*model.RedisInspectionResult{}

	evalResults := evaluator.EvaluateAll(results)

	assert.Empty(t, evalResults)
}

// =============================================================================
// Connection Usage Evaluation Tests
// =============================================================================

func TestRedisEvaluator_Evaluate_ConnectionUsage_Normal(t *testing.T) {
	evaluator := createTestRedisEvaluator()
	result := createTestRedisResult("192.18.102.2:7000", model.RedisRoleMaster, 500, 1000)

	evalResult := evaluator.Evaluate(result)

	assert.Equal(t, model.RedisStatusNormal, evalResult.Status)
	assert.Empty(t, evalResult.Alerts)
	assert.Equal(t, model.RedisStatusNormal, result.Status) // In-place update
}

func TestRedisEvaluator_Evaluate_ConnectionUsage_Warning(t *testing.T) {
	evaluator := createTestRedisEvaluator()
	result := createTestRedisResult("192.18.102.2:7000", model.RedisRoleMaster, 750, 1000) // 75%

	evalResult := evaluator.Evaluate(result)

	assert.Equal(t, model.RedisStatusWarning, evalResult.Status)
	require.Len(t, evalResult.Alerts, 1)
	assert.Equal(t, "connection_usage", evalResult.Alerts[0].MetricName)
	assert.Equal(t, model.AlertLevelWarning, evalResult.Alerts[0].Level)
	assert.Contains(t, evalResult.Alerts[0].Message, "75.0%")
	assert.Contains(t, evalResult.Alerts[0].Message, "警告阈值")
}

func TestRedisEvaluator_Evaluate_ConnectionUsage_Critical(t *testing.T) {
	evaluator := createTestRedisEvaluator()
	result := createTestRedisResult("192.18.102.2:7000", model.RedisRoleMaster, 950, 1000) // 95%

	evalResult := evaluator.Evaluate(result)

	assert.Equal(t, model.RedisStatusCritical, evalResult.Status)
	require.Len(t, evalResult.Alerts, 1)
	assert.Equal(t, "connection_usage", evalResult.Alerts[0].MetricName)
	assert.Equal(t, model.AlertLevelCritical, evalResult.Alerts[0].Level)
	assert.Contains(t, evalResult.Alerts[0].Message, "95.0%")
	assert.Contains(t, evalResult.Alerts[0].Message, "严重阈值")
}

func TestRedisEvaluator_Evaluate_ConnectionUsage_ExactWarningThreshold(t *testing.T) {
	evaluator := createTestRedisEvaluator()
	result := createTestRedisResult("192.18.102.2:7000", model.RedisRoleMaster, 700, 1000) // Exactly 70%

	evalResult := evaluator.Evaluate(result)

	assert.Equal(t, model.RedisStatusWarning, evalResult.Status)
	require.Len(t, evalResult.Alerts, 1)
	assert.Equal(t, model.AlertLevelWarning, evalResult.Alerts[0].Level)
}

func TestRedisEvaluator_Evaluate_ConnectionUsage_ExactCriticalThreshold(t *testing.T) {
	evaluator := createTestRedisEvaluator()
	result := createTestRedisResult("192.18.102.2:7000", model.RedisRoleMaster, 900, 1000) // Exactly 90%

	evalResult := evaluator.Evaluate(result)

	assert.Equal(t, model.RedisStatusCritical, evalResult.Status)
	require.Len(t, evalResult.Alerts, 1)
	assert.Equal(t, model.AlertLevelCritical, evalResult.Alerts[0].Level)
}

func TestRedisEvaluator_Evaluate_ConnectionUsage_ZeroMaxClients(t *testing.T) {
	evaluator := createTestRedisEvaluator()
	result := createTestRedisResult("192.18.102.2:7000", model.RedisRoleMaster, 100, 0) // MaxClients = 0

	evalResult := evaluator.Evaluate(result)

	// Should not trigger alert (usage = 0 due to division protection)
	assert.Equal(t, model.RedisStatusNormal, evalResult.Status)
	assert.Empty(t, evalResult.Alerts)
}

// =============================================================================
// Replication Lag Evaluation Tests
// =============================================================================

func TestRedisEvaluator_Evaluate_ReplicationLag_Normal(t *testing.T) {
	evaluator := createTestRedisEvaluator()
	result := createTestRedisResult("192.18.102.2:7001", model.RedisRoleSlave, 50, 1000)
	result.ReplicationLag = 512 * 1024 // 512 KB (< 1MB)

	evalResult := evaluator.Evaluate(result)

	assert.Equal(t, model.RedisStatusNormal, evalResult.Status)
	assert.Empty(t, evalResult.Alerts)
}

func TestRedisEvaluator_Evaluate_ReplicationLag_Warning(t *testing.T) {
	evaluator := createTestRedisEvaluator()
	result := createTestRedisResult("192.18.102.2:7001", model.RedisRoleSlave, 50, 1000)
	result.ReplicationLag = 5 * 1048576 // 5 MB (>= 1MB, < 10MB)

	evalResult := evaluator.Evaluate(result)

	assert.Equal(t, model.RedisStatusWarning, evalResult.Status)
	require.Len(t, evalResult.Alerts, 1)
	assert.Equal(t, "replication_lag", evalResult.Alerts[0].MetricName)
	assert.Equal(t, model.AlertLevelWarning, evalResult.Alerts[0].Level)
	assert.Contains(t, evalResult.Alerts[0].Message, "复制延迟")
	assert.Contains(t, evalResult.Alerts[0].Message, "警告阈值")
}

func TestRedisEvaluator_Evaluate_ReplicationLag_Critical(t *testing.T) {
	evaluator := createTestRedisEvaluator()
	result := createTestRedisResult("192.18.102.2:7001", model.RedisRoleSlave, 50, 1000)
	result.ReplicationLag = 15 * 1048576 // 15 MB (>= 10MB)

	evalResult := evaluator.Evaluate(result)

	assert.Equal(t, model.RedisStatusCritical, evalResult.Status)
	require.Len(t, evalResult.Alerts, 1)
	assert.Equal(t, "replication_lag", evalResult.Alerts[0].MetricName)
	assert.Equal(t, model.AlertLevelCritical, evalResult.Alerts[0].Level)
	assert.Contains(t, evalResult.Alerts[0].Message, "复制延迟")
	assert.Contains(t, evalResult.Alerts[0].Message, "严重阈值")
}

func TestRedisEvaluator_Evaluate_ReplicationLag_ExactWarningThreshold(t *testing.T) {
	evaluator := createTestRedisEvaluator()
	result := createTestRedisResult("192.18.102.2:7001", model.RedisRoleSlave, 50, 1000)
	result.ReplicationLag = 1048576 // Exactly 1MB

	evalResult := evaluator.Evaluate(result)

	assert.Equal(t, model.RedisStatusWarning, evalResult.Status)
	require.Len(t, evalResult.Alerts, 1)
	assert.Equal(t, model.AlertLevelWarning, evalResult.Alerts[0].Level)
}

func TestRedisEvaluator_Evaluate_ReplicationLag_ExactCriticalThreshold(t *testing.T) {
	evaluator := createTestRedisEvaluator()
	result := createTestRedisResult("192.18.102.2:7001", model.RedisRoleSlave, 50, 1000)
	result.ReplicationLag = 10485760 // Exactly 10MB

	evalResult := evaluator.Evaluate(result)

	assert.Equal(t, model.RedisStatusCritical, evalResult.Status)
	require.Len(t, evalResult.Alerts, 1)
	assert.Equal(t, model.AlertLevelCritical, evalResult.Alerts[0].Level)
}

// =============================================================================
// Master Link Status Evaluation Tests
// =============================================================================

func TestRedisEvaluator_Evaluate_MasterLinkStatus_Normal(t *testing.T) {
	evaluator := createTestRedisEvaluator()
	result := createTestRedisResult("192.18.102.2:7001", model.RedisRoleSlave, 50, 1000)
	result.MasterLinkStatus = true // Link up

	evalResult := evaluator.Evaluate(result)

	assert.Equal(t, model.RedisStatusNormal, evalResult.Status)
	assert.Empty(t, evalResult.Alerts)
}

func TestRedisEvaluator_Evaluate_MasterLinkStatus_Critical(t *testing.T) {
	evaluator := createTestRedisEvaluator()
	result := createTestRedisResult("192.18.102.2:7001", model.RedisRoleSlave, 50, 1000)
	result.MasterLinkStatus = false // Link down

	evalResult := evaluator.Evaluate(result)

	assert.Equal(t, model.RedisStatusCritical, evalResult.Status)
	require.Len(t, evalResult.Alerts, 1)
	assert.Equal(t, "master_link_status", evalResult.Alerts[0].MetricName)
	assert.Equal(t, model.AlertLevelCritical, evalResult.Alerts[0].Level)
	assert.Contains(t, evalResult.Alerts[0].Message, "主从链接断开")
}

// =============================================================================
// Skip Master Metrics Tests
// =============================================================================

func TestRedisEvaluator_Evaluate_SkipMasterMetrics(t *testing.T) {
	evaluator := createTestRedisEvaluator()
	result := createTestRedisResult("192.18.102.2:7000", model.RedisRoleMaster, 50, 1000)
	result.MasterLinkStatus = false      // Would trigger alert for slave
	result.ReplicationLag = 15 * 1048576 // Would trigger alert for slave

	evalResult := evaluator.Evaluate(result)

	// Master should not have replication lag or link status alerts
	assert.Equal(t, model.RedisStatusNormal, evalResult.Status)
	assert.Empty(t, evalResult.Alerts)
}

// =============================================================================
// Multiple Alerts Tests
// =============================================================================

func TestRedisEvaluator_Evaluate_MultipleAlerts_MostSevereStatus(t *testing.T) {
	evaluator := createTestRedisEvaluator()
	result := createTestRedisResult("192.18.102.2:7001", model.RedisRoleSlave, 750, 1000) // 75% - Warning
	result.MasterLinkStatus = false // Critical

	evalResult := evaluator.Evaluate(result)

	assert.Equal(t, model.RedisStatusCritical, evalResult.Status) // Critical wins
	assert.Len(t, evalResult.Alerts, 2)                           // Both alerts present
}

func TestRedisEvaluator_Evaluate_MultipleAlerts_AllCritical(t *testing.T) {
	evaluator := createTestRedisEvaluator()
	result := createTestRedisResult("192.18.102.2:7001", model.RedisRoleSlave, 950, 1000) // 95% - Critical
	result.MasterLinkStatus = false                                                       // Critical
	result.ReplicationLag = 15 * 1048576                                                  // Critical

	evalResult := evaluator.Evaluate(result)

	assert.Equal(t, model.RedisStatusCritical, evalResult.Status)
	assert.Len(t, evalResult.Alerts, 3) // All three alerts
}

func TestRedisEvaluator_Evaluate_MultipleAlerts_AllWarning(t *testing.T) {
	evaluator := createTestRedisEvaluator()
	result := createTestRedisResult("192.18.102.2:7001", model.RedisRoleSlave, 750, 1000) // 75% - Warning
	result.ReplicationLag = 5 * 1048576                                                   // Warning

	evalResult := evaluator.Evaluate(result)

	assert.Equal(t, model.RedisStatusWarning, evalResult.Status)
	assert.Len(t, evalResult.Alerts, 2) // Both warnings
}

// =============================================================================
// Failed Instance Tests
// =============================================================================

func TestRedisEvaluator_Evaluate_FailedInstance(t *testing.T) {
	evaluator := createTestRedisEvaluator()
	result := createTestRedisResult("192.18.102.2:7000", model.RedisRoleMaster, 950, 1000)
	result.Error = "connection refused"

	evalResult := evaluator.Evaluate(result)

	assert.Equal(t, model.RedisStatusFailed, evalResult.Status)
	assert.Empty(t, evalResult.Alerts) // No evaluation for failed instances
}

// =============================================================================
// Nil Instance Tests
// =============================================================================

func TestRedisEvaluator_Evaluate_NilInstance(t *testing.T) {
	evaluator := createTestRedisEvaluator()
	result := model.NewRedisInspectionResult(nil)
	result.ConnectedClients = 750
	result.MaxClients = 1000

	evalResult := evaluator.Evaluate(result)

	// Should still evaluate connection usage
	assert.Equal(t, model.RedisStatusWarning, evalResult.Status)
	require.Len(t, evalResult.Alerts, 1)
}

// =============================================================================
// Helper Method Tests: formatValue
// =============================================================================

func TestRedisEvaluator_formatValue(t *testing.T) {
	evaluator := createTestRedisEvaluator()

	tests := []struct {
		name       string
		metricName string
		value      float64
		expected   string
	}{
		{
			name:       "connection_usage",
			metricName: "connection_usage",
			value:      75.5,
			expected:   "75.5%",
		},
		{
			name:       "replication_lag_bytes",
			metricName: "replication_lag",
			value:      512,
			expected:   "512 B",
		},
		{
			name:       "replication_lag_kb",
			metricName: "replication_lag",
			value:      2048, // 2KB
			expected:   "2.00 KB",
		},
		{
			name:       "replication_lag_mb",
			metricName: "replication_lag",
			value:      5242880, // 5MB
			expected:   "5.00 MB",
		},
		{
			name:       "replication_lag_gb",
			metricName: "replication_lag",
			value:      2147483648, // 2GB
			expected:   "2.00 GB",
		},
		{
			name:       "master_link_status_disconnected",
			metricName: "master_link_status",
			value:      0,
			expected:   "断开",
		},
		{
			name:       "master_link_status_connected",
			metricName: "master_link_status",
			value:      1,
			expected:   "正常",
		},
		{
			name:       "unknown_metric",
			metricName: "unknown",
			value:      123.456,
			expected:   "123.46",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluator.formatValue(tt.value, tt.metricName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Helper Method Tests: getThresholds
// =============================================================================

func TestRedisEvaluator_getThresholds(t *testing.T) {
	evaluator := createTestRedisEvaluator()

	tests := []struct {
		name             string
		metricName       string
		expectedWarning  float64
		expectedCritical float64
	}{
		{
			name:             "connection_usage",
			metricName:       "connection_usage",
			expectedWarning:  70,
			expectedCritical: 90,
		},
		{
			name:             "replication_lag",
			metricName:       "replication_lag",
			expectedWarning:  1048576,
			expectedCritical: 10485760,
		},
		{
			name:             "master_link_status",
			metricName:       "master_link_status",
			expectedWarning:  1,
			expectedCritical: 1,
		},
		{
			name:             "unknown_metric",
			metricName:       "unknown",
			expectedWarning:  0,
			expectedCritical: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warning, critical := evaluator.getThresholds(tt.metricName)
			assert.Equal(t, tt.expectedWarning, warning)
			assert.Equal(t, tt.expectedCritical, critical)
		})
	}
}

// =============================================================================
// Helper Method Tests: generateAlertMessage
// =============================================================================

func TestRedisEvaluator_generateAlertMessage(t *testing.T) {
	evaluator := createTestRedisEvaluator()

	tests := []struct {
		name         string
		metricName   string
		currentValue float64
		level        model.AlertLevel
		contains     []string
	}{
		{
			name:         "connection_usage_warning",
			metricName:   "connection_usage",
			currentValue: 75.0,
			level:        model.AlertLevelWarning,
			contains:     []string{"75.0%", "警告阈值", "70.0%"},
		},
		{
			name:         "connection_usage_critical",
			metricName:   "connection_usage",
			currentValue: 95.0,
			level:        model.AlertLevelCritical,
			contains:     []string{"95.0%", "严重阈值", "90.0%"},
		},
		{
			name:         "replication_lag_warning",
			metricName:   "replication_lag",
			currentValue: 5242880, // 5MB
			level:        model.AlertLevelWarning,
			contains:     []string{"复制延迟", "警告阈值", "1 MB"},
		},
		{
			name:         "replication_lag_critical",
			metricName:   "replication_lag",
			currentValue: 15728640, // 15MB
			level:        model.AlertLevelCritical,
			contains:     []string{"复制延迟", "严重阈值", "10 MB"},
		},
		{
			name:         "master_link_status",
			metricName:   "master_link_status",
			currentValue: 0,
			level:        model.AlertLevelCritical,
			contains:     []string{"主从链接断开"},
		},
		{
			name:         "unknown_metric",
			metricName:   "unknown",
			currentValue: 123.45,
			level:        model.AlertLevelWarning,
			contains:     []string{"指标异常", "123.45"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := evaluator.generateAlertMessage(tt.metricName, tt.currentValue, tt.level)
			for _, substr := range tt.contains {
				assert.Contains(t, message, substr, "Message should contain: %s", substr)
			}
		})
	}
}

// =============================================================================
// Helper Method Tests: determineInstanceStatus
// =============================================================================

func TestRedisEvaluator_determineInstanceStatus(t *testing.T) {
	evaluator := createTestRedisEvaluator()

	tests := []struct {
		name     string
		alerts   []*model.RedisAlert
		expected model.RedisInstanceStatus
	}{
		{
			name:     "no_alerts",
			alerts:   []*model.RedisAlert{},
			expected: model.RedisStatusNormal,
		},
		{
			name: "single_warning",
			alerts: []*model.RedisAlert{
				{Level: model.AlertLevelWarning},
			},
			expected: model.RedisStatusWarning,
		},
		{
			name: "single_critical",
			alerts: []*model.RedisAlert{
				{Level: model.AlertLevelCritical},
			},
			expected: model.RedisStatusCritical,
		},
		{
			name: "mixed_warning_critical",
			alerts: []*model.RedisAlert{
				{Level: model.AlertLevelWarning},
				{Level: model.AlertLevelCritical},
			},
			expected: model.RedisStatusCritical,
		},
		{
			name: "multiple_warnings",
			alerts: []*model.RedisAlert{
				{Level: model.AlertLevelWarning},
				{Level: model.AlertLevelWarning},
			},
			expected: model.RedisStatusWarning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := evaluator.determineInstanceStatus(tt.alerts)
			assert.Equal(t, tt.expected, status)
		})
	}
}

// =============================================================================
// createAlert Tests
// =============================================================================

func TestRedisEvaluator_createAlert(t *testing.T) {
	evaluator := createTestRedisEvaluator()

	alert := evaluator.createAlert(
		"192.18.102.2:7001",
		"connection_usage",
		75.5,
		model.AlertLevelWarning,
	)

	assert.Equal(t, "192.18.102.2:7001", alert.Address)
	assert.Equal(t, "connection_usage", alert.MetricName)
	assert.Equal(t, "连接使用率", alert.MetricDisplayName) // From metric definition
	assert.Equal(t, 75.5, alert.CurrentValue)
	assert.Equal(t, "75.5%", alert.FormattedValue)
	assert.Equal(t, float64(70), alert.WarningThreshold)
	assert.Equal(t, float64(90), alert.CriticalThreshold)
	assert.Equal(t, model.AlertLevelWarning, alert.Level)
	assert.NotEmpty(t, alert.Message)
}

func TestRedisEvaluator_createAlert_UnknownMetric(t *testing.T) {
	evaluator := createTestRedisEvaluator()

	alert := evaluator.createAlert(
		"192.18.102.2:7001",
		"unknown_metric",
		99.9,
		model.AlertLevelCritical,
	)

	// Should fall back to metric name as display name
	assert.Equal(t, "unknown_metric", alert.MetricDisplayName)
	assert.Equal(t, "99.90", alert.FormattedValue)
}

// =============================================================================
// In-place Update Tests
// =============================================================================

func TestRedisEvaluator_Evaluate_InPlaceUpdate(t *testing.T) {
	evaluator := createTestRedisEvaluator()
	result := createTestRedisResult("192.18.102.2:7000", model.RedisRoleMaster, 950, 1000)

	// Before evaluation
	assert.Equal(t, model.RedisStatusNormal, result.Status)
	assert.Empty(t, result.Alerts)

	evaluator.Evaluate(result)

	// After evaluation - result should be updated in-place
	assert.Equal(t, model.RedisStatusCritical, result.Status)
	assert.Len(t, result.Alerts, 1)
}
