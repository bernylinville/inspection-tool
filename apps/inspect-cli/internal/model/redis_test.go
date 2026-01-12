package model

import (
	"testing"
	"time"

	pkgmodel "inspection-tool/pkg/model"
)

// ============================================================================
// GetNetworkSegment Tests
// ============================================================================

func TestGetNetworkSegment(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected string
	}{
		{
			name:     "valid IPv4",
			ip:       "192.18.102.2",
			expected: "192.18.102",
		},
		{
			name:     "another valid IPv4",
			ip:       "172.18.182.91",
			expected: "172.18.182",
		},
		{
			name:     "different segment",
			ip:       "10.0.0.100",
			expected: "10.0.0",
		},
		{
			name:     "with leading zeros",
			ip:       "192.168.001.100",
			expected: "192.168.001",
		},
		{
			name:     "invalid - only two octets",
			ip:       "192.168",
			expected: "",
		},
		{
			name:     "invalid - empty string",
			ip:       "",
			expected: "",
		},
		{
			name:     "invalid - single number",
			ip:       "192",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetNetworkSegment(tt.ip)
			if result != tt.expected {
				t.Errorf("GetNetworkSegment(%q) = %q, want %q", tt.ip, result, tt.expected)
			}
		})
	}
}

// ============================================================================
// RedisCluster Tests
// ============================================================================

func TestNewRedisCluster(t *testing.T) {
	cluster := NewRedisCluster("192.18.102")

	if cluster.ID != "192.18.102" {
		t.Errorf("ID = %q, want %q", cluster.ID, "192.18.102")
	}
	if cluster.Name != "Redis 集群 - 192.18.102" {
		t.Errorf("Name = %q, want %q", cluster.Name, "Redis 集群 - 192.18.102")
	}
	if len(cluster.Instances) != 0 {
		t.Errorf("Instances should be empty, got %d", len(cluster.Instances))
	}
	if len(cluster.Alerts) != 0 {
		t.Errorf("Alerts should be empty, got %d", len(cluster.Alerts))
	}
}

func TestRedisCluster_AddResult(t *testing.T) {
	cluster := NewRedisCluster("192.18.102")

	instance := &RedisInstance{
		Address: "192.18.102.2:7000",
		IP:      "192.18.102.2",
		Port:    7000,
		Role:    RedisRoleMaster,
	}

	alert := &RedisAlert{
		Address:    "192.18.102.2:7000",
		MetricName: "connection_usage",
		Level:      pkgmodel.AlertLevelWarning,
	}

	result := &RedisInspectionResult{
		Instance: instance,
		Status:   RedisStatusWarning,
		Alerts:   []*RedisAlert{alert},
	}

	cluster.AddResult(result)

	if len(cluster.Instances) != 1 {
		t.Errorf("expected 1 instance, got %d", len(cluster.Instances))
	}
	if len(cluster.Alerts) != 1 {
		t.Errorf("expected 1 alert, got %d", len(cluster.Alerts))
	}
}

func TestRedisCluster_AddResult_NilResult(t *testing.T) {
	cluster := NewRedisCluster("192.18.102")
	cluster.AddResult(nil) // Should not panic

	if len(cluster.Instances) != 0 {
		t.Error("nil result should not add instances")
	}
}

func TestRedisCluster_Finalize(t *testing.T) {
	cluster := NewRedisCluster("192.18.102")

	// Add two results: one normal, one warning
	cluster.AddResult(&RedisInspectionResult{
		Instance: &RedisInstance{Address: "192.18.102.2:7000", Role: RedisRoleMaster},
		Status:   RedisStatusNormal,
	})
	cluster.AddResult(&RedisInspectionResult{
		Instance: &RedisInstance{Address: "192.18.102.2:7001", Role: RedisRoleSlave},
		Status:   RedisStatusWarning,
		Alerts: []*RedisAlert{
			{Level: pkgmodel.AlertLevelWarning},
		},
	})

	cluster.Finalize()

	if cluster.Summary == nil {
		t.Fatal("Summary should not be nil after Finalize")
	}
	if cluster.Summary.TotalInstances != 2 {
		t.Errorf("Summary.TotalInstances = %d, want 2", cluster.Summary.TotalInstances)
	}
	if cluster.Summary.NormalInstances != 1 {
		t.Errorf("Summary.NormalInstances = %d, want 1", cluster.Summary.NormalInstances)
	}
	if cluster.Summary.WarningInstances != 1 {
		t.Errorf("Summary.WarningInstances = %d, want 1", cluster.Summary.WarningInstances)
	}

	if cluster.AlertSummary == nil {
		t.Fatal("AlertSummary should not be nil after Finalize")
	}
	if cluster.AlertSummary.TotalAlerts != 1 {
		t.Errorf("AlertSummary.TotalAlerts = %d, want 1", cluster.AlertSummary.TotalAlerts)
	}
}

func TestRedisCluster_GetMasterCount(t *testing.T) {
	cluster := NewRedisCluster("192.18.102")

	cluster.AddResult(&RedisInspectionResult{
		Instance: &RedisInstance{Address: "192.18.102.2:7000", Role: RedisRoleMaster},
	})
	cluster.AddResult(&RedisInspectionResult{
		Instance: &RedisInstance{Address: "192.18.102.2:7001", Role: RedisRoleSlave},
	})
	cluster.AddResult(&RedisInspectionResult{
		Instance: &RedisInstance{Address: "192.18.102.3:7000", Role: RedisRoleMaster},
	})

	count := cluster.GetMasterCount()
	if count != 2 {
		t.Errorf("GetMasterCount() = %d, want 2", count)
	}
}

func TestRedisCluster_GetSlaveCount(t *testing.T) {
	cluster := NewRedisCluster("192.18.102")

	cluster.AddResult(&RedisInspectionResult{
		Instance: &RedisInstance{Address: "192.18.102.2:7000", Role: RedisRoleMaster},
	})
	cluster.AddResult(&RedisInspectionResult{
		Instance: &RedisInstance{Address: "192.18.102.2:7001", Role: RedisRoleSlave},
	})
	cluster.AddResult(&RedisInspectionResult{
		Instance: &RedisInstance{Address: "192.18.102.3:7001", Role: RedisRoleSlave},
	})

	count := cluster.GetSlaveCount()
	if count != 2 {
		t.Errorf("GetSlaveCount() = %d, want 2", count)
	}
}

// ============================================================================
// GroupByClusters Tests
// ============================================================================

func TestRedisInspectionResults_GroupByClusters_Empty(t *testing.T) {
	results := &RedisInspectionResults{
		Results: []*RedisInspectionResult{},
	}

	clusters := results.GroupByClusters()
	if len(clusters) != 0 {
		t.Errorf("expected 0 clusters for empty results, got %d", len(clusters))
	}
}

func TestRedisInspectionResults_GroupByClusters_SingleCluster(t *testing.T) {
	results := &RedisInspectionResults{
		InspectionTime: time.Now(),
		Results: []*RedisInspectionResult{
			{
				Instance: &RedisInstance{Address: "192.18.102.2:7000", IP: "192.18.102.2", Role: RedisRoleMaster},
				Status:   RedisStatusNormal,
			},
			{
				Instance: &RedisInstance{Address: "192.18.102.2:7001", IP: "192.18.102.2", Role: RedisRoleSlave},
				Status:   RedisStatusNormal,
			},
			{
				Instance: &RedisInstance{Address: "192.18.102.3:7000", IP: "192.18.102.3", Role: RedisRoleMaster},
				Status:   RedisStatusNormal,
			},
		},
	}

	clusters := results.GroupByClusters()

	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusters))
	}

	cluster := clusters[0]
	if cluster.ID != "192.18.102" {
		t.Errorf("cluster ID = %q, want %q", cluster.ID, "192.18.102")
	}
	if len(cluster.Instances) != 3 {
		t.Errorf("expected 3 instances in cluster, got %d", len(cluster.Instances))
	}
}

func TestRedisInspectionResults_GroupByClusters_MultipleClusters(t *testing.T) {
	results := &RedisInspectionResults{
		InspectionTime: time.Now(),
		Results: []*RedisInspectionResult{
			// Cluster 1: 192.18.102.x
			{
				Instance: &RedisInstance{Address: "192.18.102.2:7000", IP: "192.18.102.2", Role: RedisRoleMaster},
				Status:   RedisStatusNormal,
			},
			{
				Instance: &RedisInstance{Address: "192.18.102.2:7001", IP: "192.18.102.2", Role: RedisRoleSlave},
				Status:   RedisStatusNormal,
			},
			// Cluster 2: 192.18.107.x
			{
				Instance: &RedisInstance{Address: "192.18.107.5:7000", IP: "192.18.107.5", Role: RedisRoleMaster},
				Status:   RedisStatusWarning,
				Alerts: []*RedisAlert{
					{Level: pkgmodel.AlertLevelWarning},
				},
			},
			{
				Instance: &RedisInstance{Address: "192.18.107.6:7001", IP: "192.18.107.6", Role: RedisRoleSlave},
				Status:   RedisStatusNormal,
			},
		},
	}

	clusters := results.GroupByClusters()

	if len(clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(clusters))
	}

	// Clusters should be sorted by ID
	if clusters[0].ID != "192.18.102" {
		t.Errorf("first cluster ID = %q, want %q", clusters[0].ID, "192.18.102")
	}
	if clusters[1].ID != "192.18.107" {
		t.Errorf("second cluster ID = %q, want %q", clusters[1].ID, "192.18.107")
	}

	// Verify instance counts
	if len(clusters[0].Instances) != 2 {
		t.Errorf("first cluster should have 2 instances, got %d", len(clusters[0].Instances))
	}
	if len(clusters[1].Instances) != 2 {
		t.Errorf("second cluster should have 2 instances, got %d", len(clusters[1].Instances))
	}

	// Verify second cluster has alert
	if len(clusters[1].Alerts) != 1 {
		t.Errorf("second cluster should have 1 alert, got %d", len(clusters[1].Alerts))
	}
}

func TestRedisInspectionResults_GroupByClusters_NilInstance(t *testing.T) {
	results := &RedisInspectionResults{
		InspectionTime: time.Now(),
		Results: []*RedisInspectionResult{
			{
				Instance: &RedisInstance{Address: "192.18.102.2:7000", IP: "192.18.102.2", Role: RedisRoleMaster},
				Status:   RedisStatusNormal,
			},
			{
				Instance: nil, // nil instance should be skipped
				Status:   RedisStatusFailed,
			},
		},
	}

	clusters := results.GroupByClusters()

	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster (nil instance skipped), got %d", len(clusters))
	}
	if len(clusters[0].Instances) != 1 {
		t.Errorf("expected 1 instance (nil skipped), got %d", len(clusters[0].Instances))
	}
}

func TestRedisInspectionResults_GroupByClusters_PopulatesClustersField(t *testing.T) {
	results := &RedisInspectionResults{
		InspectionTime: time.Now(),
		Results: []*RedisInspectionResult{
			{
				Instance: &RedisInstance{Address: "192.18.102.2:7000", IP: "192.18.102.2", Role: RedisRoleMaster},
				Status:   RedisStatusNormal,
			},
		},
	}

	// Before GroupByClusters
	if results.Clusters != nil {
		t.Error("Clusters should be nil before GroupByClusters")
	}

	clusters := results.GroupByClusters()

	// After GroupByClusters, Clusters field should be populated
	if results.Clusters == nil {
		t.Error("Clusters field should be populated after GroupByClusters")
	}
	if len(results.Clusters) != len(clusters) {
		t.Errorf("Clusters field length = %d, returned length = %d", len(results.Clusters), len(clusters))
	}
}

// ============================================================================
// HasMultipleClusters Tests
// ============================================================================

func TestRedisInspectionResults_HasMultipleClusters_True(t *testing.T) {
	results := &RedisInspectionResults{
		Clusters: []*RedisCluster{
			NewRedisCluster("192.18.102"),
			NewRedisCluster("192.18.107"),
		},
	}

	if !results.HasMultipleClusters() {
		t.Error("HasMultipleClusters() should return true for 2 clusters")
	}
}

func TestRedisInspectionResults_HasMultipleClusters_False_SingleCluster(t *testing.T) {
	results := &RedisInspectionResults{
		Clusters: []*RedisCluster{
			NewRedisCluster("192.18.102"),
		},
	}

	if results.HasMultipleClusters() {
		t.Error("HasMultipleClusters() should return false for 1 cluster")
	}
}

func TestRedisInspectionResults_HasMultipleClusters_False_Empty(t *testing.T) {
	results := &RedisInspectionResults{
		Clusters: []*RedisCluster{},
	}

	if results.HasMultipleClusters() {
		t.Error("HasMultipleClusters() should return false for empty clusters")
	}
}

func TestRedisInspectionResults_HasMultipleClusters_False_Nil(t *testing.T) {
	results := &RedisInspectionResults{
		Clusters: nil,
	}

	if results.HasMultipleClusters() {
		t.Error("HasMultipleClusters() should return false for nil clusters")
	}
}

// ============================================================================
// Integration Test: Full Scenario (陕西项目场景)
// ============================================================================

func TestRedisInstanceStatus_Methods(t *testing.T) {
	tests := []struct {
		status     RedisInstanceStatus
		isHealthy  bool
		isWarning  bool
		isCritical bool
		isFailed   bool
	}{
		{RedisStatusNormal, true, false, false, false},
		{RedisStatusWarning, false, true, false, false},
		{RedisStatusCritical, false, false, true, false},
		{RedisStatusFailed, false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsHealthy(); got != tt.isHealthy {
				t.Errorf("IsHealthy() = %v, want %v", got, tt.isHealthy)
			}
			if got := tt.status.IsWarning(); got != tt.isWarning {
				t.Errorf("IsWarning() = %v, want %v", got, tt.isWarning)
			}
			if got := tt.status.IsCritical(); got != tt.isCritical {
				t.Errorf("IsCritical() = %v, want %v", got, tt.isCritical)
			}
			if got := tt.status.IsFailed(); got != tt.isFailed {
				t.Errorf("IsFailed() = %v, want %v", got, tt.isFailed)
			}
		})
	}
}

func TestRedisRole_Methods(t *testing.T) {
	tests := []struct {
		role     RedisRole
		isMaster bool
		isSlave  bool
	}{
		{RedisRoleMaster, true, false},
		{RedisRoleSlave, false, true},
		{RedisRoleUnknown, false, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			if got := tt.role.IsMaster(); got != tt.isMaster {
				t.Errorf("IsMaster() = %v, want %v", got, tt.isMaster)
			}
			if got := tt.role.IsSlave(); got != tt.isSlave {
				t.Errorf("IsSlave() = %v, want %v", got, tt.isSlave)
			}
		})
	}
}

func TestRedisClusterMode_Methods(t *testing.T) {
	tests := []struct {
		mode               RedisClusterMode
		is3M3S             bool
		is3M6S             bool
		expectedSlaveCount int
	}{
		{ClusterMode3M3S, true, false, 1},
		{ClusterMode3M6S, false, true, 2},
		{RedisClusterMode("unknown"), false, false, 0},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			if got := tt.mode.Is3M3S(); got != tt.is3M3S {
				t.Errorf("Is3M3S() = %v, want %v", got, tt.is3M3S)
			}
			if got := tt.mode.Is3M6S(); got != tt.is3M6S {
				t.Errorf("Is3M6S() = %v, want %v", got, tt.is3M6S)
			}
			if got := tt.mode.GetExpectedSlaveCount(); got != tt.expectedSlaveCount {
				t.Errorf("GetExpectedSlaveCount() = %v, want %v", got, tt.expectedSlaveCount)
			}
		})
	}
}

func TestNewRedisInstance(t *testing.T) {
	t.Run("valid address", func(t *testing.T) {
		instance := NewRedisInstance("192.18.102.2:7000")

		if instance == nil {
			t.Fatal("expected non-nil instance")
		}
		if instance.Address != "192.18.102.2:7000" {
			t.Errorf("Address = %q, want %q", instance.Address, "192.18.102.2:7000")
		}
		if instance.IP != "192.18.102.2" {
			t.Errorf("IP = %q, want %q", instance.IP, "192.18.102.2")
		}
		if instance.Port != 7000 {
			t.Errorf("Port = %d, want 7000", instance.Port)
		}
		if instance.ApplicationType != "Redis" {
			t.Errorf("ApplicationType = %q, want %q", instance.ApplicationType, "Redis")
		}
		if instance.Role != RedisRoleUnknown {
			t.Errorf("Role = %v, want %v", instance.Role, RedisRoleUnknown)
		}
	})

	t.Run("invalid address returns nil", func(t *testing.T) {
		instance := NewRedisInstance("invalid")
		if instance != nil {
			t.Error("expected nil for invalid address")
		}
	})
}

func TestNewRedisInstanceWithRole(t *testing.T) {
	instance := NewRedisInstanceWithRole("192.18.102.2:7000", RedisRoleMaster)

	if instance == nil {
		t.Fatal("expected non-nil instance")
	}
	if instance.Role != RedisRoleMaster {
		t.Errorf("Role = %v, want %v", instance.Role, RedisRoleMaster)
	}
}

func TestRedisInstance_Methods(t *testing.T) {
	instance := NewRedisInstance("192.18.102.2:7000")

	t.Run("SetVersion", func(t *testing.T) {
		instance.SetVersion("6.2.6")
		if instance.Version != "6.2.6" {
			t.Errorf("Version = %q, want %q", instance.Version, "6.2.6")
		}
	})

	t.Run("SetClusterEnabled", func(t *testing.T) {
		instance.SetClusterEnabled(true)
		if !instance.ClusterEnabled {
			t.Error("ClusterEnabled should be true")
		}
	})

	t.Run("String", func(t *testing.T) {
		str := instance.String()
		if str == "" {
			t.Error("String() should return non-empty string")
		}
	})

	t.Run("String nil", func(t *testing.T) {
		var nilInstance *RedisInstance
		if nilInstance.String() != "<nil>" {
			t.Errorf("String() = %q, want %q", nilInstance.String(), "<nil>")
		}
	})
}

func TestNewRedisAlert(t *testing.T) {
	alert := NewRedisAlert("192.18.102.2:7000", "connection_usage", 85.5, pkgmodel.AlertLevelWarning)

	if alert.Address != "192.18.102.2:7000" {
		t.Errorf("Address = %q, want %q", alert.Address, "192.18.102.2:7000")
	}
	if alert.MetricName != "connection_usage" {
		t.Errorf("MetricName = %q, want %q", alert.MetricName, "connection_usage")
	}
	if alert.CurrentValue != 85.5 {
		t.Errorf("CurrentValue = %v, want 85.5", alert.CurrentValue)
	}
	if alert.Level != pkgmodel.AlertLevelWarning {
		t.Errorf("Level = %v, want %v", alert.Level, pkgmodel.AlertLevelWarning)
	}
}

func TestRedisAlert_Methods(t *testing.T) {
	t.Run("IsWarning", func(t *testing.T) {
		alert := &RedisAlert{Level: pkgmodel.AlertLevelWarning}
		if !alert.IsWarning() {
			t.Error("IsWarning() should return true")
		}
	})

	t.Run("IsCritical", func(t *testing.T) {
		alert := &RedisAlert{Level: pkgmodel.AlertLevelCritical}
		if !alert.IsCritical() {
			t.Error("IsCritical() should return true")
		}
	})
}

func TestNewRedisInspectionResult(t *testing.T) {
	t.Run("with valid instance", func(t *testing.T) {
		instance := NewRedisInstance("192.18.102.2:7000")
		result := NewRedisInspectionResult(instance)

		if result.Instance != instance {
			t.Error("Instance should be set")
		}
		if result.Status != RedisStatusNormal {
			t.Errorf("Status = %v, want %v", result.Status, RedisStatusNormal)
		}
		if result.NonRootUser != "N/A" {
			t.Errorf("NonRootUser = %q, want %q", result.NonRootUser, "N/A")
		}
		if result.Alerts == nil {
			t.Error("Alerts should be initialized")
		}
	})

	t.Run("with nil instance", func(t *testing.T) {
		result := NewRedisInspectionResult(nil)

		if result.Status != RedisStatusFailed {
			t.Errorf("Status = %v, want %v", result.Status, RedisStatusFailed)
		}
	})
}

func TestRedisInspectionResult_AddAlert(t *testing.T) {
	t.Run("nil alert ignored", func(t *testing.T) {
		result := NewRedisInspectionResult(NewRedisInstance("192.18.102.2:7000"))
		result.AddAlert(nil)

		if len(result.Alerts) != 0 {
			t.Error("nil alert should not be added")
		}
	})

	t.Run("warning upgrades status from normal", func(t *testing.T) {
		result := NewRedisInspectionResult(NewRedisInstance("192.18.102.2:7000"))
		alert := &RedisAlert{Level: pkgmodel.AlertLevelWarning}

		result.AddAlert(alert)

		if result.Status != RedisStatusWarning {
			t.Errorf("Status = %v, want %v", result.Status, RedisStatusWarning)
		}
	})

	t.Run("critical upgrades status from warning", func(t *testing.T) {
		result := NewRedisInspectionResult(NewRedisInstance("192.18.102.2:7000"))
		result.Status = RedisStatusWarning
		alert := &RedisAlert{Level: pkgmodel.AlertLevelCritical}

		result.AddAlert(alert)

		if result.Status != RedisStatusCritical {
			t.Errorf("Status = %v, want %v", result.Status, RedisStatusCritical)
		}
	})

	t.Run("warning does not downgrade critical", func(t *testing.T) {
		result := NewRedisInspectionResult(NewRedisInstance("192.18.102.2:7000"))
		result.Status = RedisStatusCritical
		alert := &RedisAlert{Level: pkgmodel.AlertLevelWarning}

		result.AddAlert(alert)

		if result.Status != RedisStatusCritical {
			t.Errorf("Status = %v, want %v (should not downgrade)", result.Status, RedisStatusCritical)
		}
	})
}

func TestRedisInspectionResult_GetConnectionUsagePercent(t *testing.T) {
	tests := []struct {
		name             string
		maxClients       int
		connectedClients int
		expected         float64
	}{
		{
			name:             "normal calculation",
			maxClients:       10000,
			connectedClients: 7500,
			expected:         75.0,
		},
		{
			name:             "zero max clients",
			maxClients:       0,
			connectedClients: 100,
			expected:         0,
		},
		{
			name:             "zero connected clients",
			maxClients:       10000,
			connectedClients: 0,
			expected:         0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &RedisInspectionResult{
				MaxClients:       tt.maxClients,
				ConnectedClients: tt.connectedClients,
			}

			got := result.GetConnectionUsagePercent()
			if got != tt.expected {
				t.Errorf("GetConnectionUsagePercent() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRedisInspectionResult_GetAddress(t *testing.T) {
	t.Run("with instance", func(t *testing.T) {
		result := NewRedisInspectionResult(NewRedisInstance("192.18.102.2:7000"))
		if got := result.GetAddress(); got != "192.18.102.2:7000" {
			t.Errorf("GetAddress() = %q, want %q", got, "192.18.102.2:7000")
		}
	})

	t.Run("nil instance", func(t *testing.T) {
		result := &RedisInspectionResult{}
		if got := result.GetAddress(); got != "" {
			t.Errorf("GetAddress() = %q, want empty string", got)
		}
	})
}

func TestRedisInspectionResult_SetGetMetric(t *testing.T) {
	result := NewRedisInspectionResult(NewRedisInstance("192.18.102.2:7000"))

	t.Run("set and get", func(t *testing.T) {
		value := &RedisMetricValue{Name: "redis_up", RawValue: 1}
		result.SetMetric(value)

		got := result.GetMetric("redis_up")
		if got != value {
			t.Error("should return the set metric")
		}
	})

	t.Run("get non-existing", func(t *testing.T) {
		got := result.GetMetric("not_exists")
		if got != nil {
			t.Error("should return nil for non-existing metric")
		}
	})

	t.Run("get from nil metrics", func(t *testing.T) {
		emptyResult := &RedisInspectionResult{}
		got := emptyResult.GetMetric("redis_up")
		if got != nil {
			t.Error("should return nil for nil metrics map")
		}
	})
}

func TestNewRedisInspectionSummary(t *testing.T) {
	results := []*RedisInspectionResult{
		{Status: RedisStatusNormal},
		{Status: RedisStatusNormal},
		{Status: RedisStatusWarning},
		{Status: RedisStatusCritical},
		{Status: RedisStatusFailed},
		nil,
	}

	summary := NewRedisInspectionSummary(results)

	if summary.TotalInstances != 5 {
		t.Errorf("TotalInstances = %d, want 5", summary.TotalInstances)
	}
	if summary.NormalInstances != 2 {
		t.Errorf("NormalInstances = %d, want 2", summary.NormalInstances)
	}
	if summary.WarningInstances != 1 {
		t.Errorf("WarningInstances = %d, want 1", summary.WarningInstances)
	}
	if summary.CriticalInstances != 1 {
		t.Errorf("CriticalInstances = %d, want 1", summary.CriticalInstances)
	}
	if summary.FailedInstances != 1 {
		t.Errorf("FailedInstances = %d, want 1", summary.FailedInstances)
	}
}

func TestNewRedisAlertSummary(t *testing.T) {
	alerts := []*RedisAlert{
		{Level: pkgmodel.AlertLevelWarning},
		{Level: pkgmodel.AlertLevelWarning},
		{Level: pkgmodel.AlertLevelCritical},
		nil,
	}

	summary := NewRedisAlertSummary(alerts)

	if summary.TotalAlerts != 3 {
		t.Errorf("TotalAlerts = %d, want 3", summary.TotalAlerts)
	}
	if summary.WarningCount != 2 {
		t.Errorf("WarningCount = %d, want 2", summary.WarningCount)
	}
	if summary.CriticalCount != 1 {
		t.Errorf("CriticalCount = %d, want 1", summary.CriticalCount)
	}
}

func TestRedisMetricDefinition_Methods(t *testing.T) {
	t.Run("IsPending", func(t *testing.T) {
		def := &RedisMetricDefinition{Status: "pending"}
		if !def.IsPending() {
			t.Error("IsPending() should return true for pending status")
		}

		def2 := &RedisMetricDefinition{Query: ""}
		if !def2.IsPending() {
			t.Error("IsPending() should return true for empty query")
		}

		def3 := &RedisMetricDefinition{Query: "some_query", Status: ""}
		if def3.IsPending() {
			t.Error("IsPending() should return false for normal metric")
		}
	})

	t.Run("GetDisplayName", func(t *testing.T) {
		def := &RedisMetricDefinition{Name: "redis_up", DisplayName: "连接状态"}
		if got := def.GetDisplayName(); got != "连接状态" {
			t.Errorf("GetDisplayName() = %q, want %q", got, "连接状态")
		}

		def2 := &RedisMetricDefinition{Name: "redis_up"}
		if got := def2.GetDisplayName(); got != "redis_up" {
			t.Errorf("GetDisplayName() = %q, want %q", got, "redis_up")
		}
	})
}

func TestRedisInspectionResults_GroupByClusters_ShanxiScenario(t *testing.T) {
	// Simulate 陕西项目 scenario: 2 clusters × 3主3从 = 12 nodes
	results := &RedisInspectionResults{
		InspectionTime: time.Now(),
		Results:        make([]*RedisInspectionResult, 0, 12),
	}

	// Cluster 1: 192.18.102.x - 3 masters, 3 slaves
	for i := 2; i <= 4; i++ {
		// Master
		results.Results = append(results.Results, &RedisInspectionResult{
			Instance: &RedisInstance{
				Address: "192.18.102." + string(rune('0'+i)) + ":7000",
				IP:      "192.18.102." + string(rune('0'+i)),
				Port:    7000,
				Role:    RedisRoleMaster,
			},
			Status: RedisStatusNormal,
		})
		// Slave
		results.Results = append(results.Results, &RedisInspectionResult{
			Instance: &RedisInstance{
				Address: "192.18.102." + string(rune('0'+i)) + ":7001",
				IP:      "192.18.102." + string(rune('0'+i)),
				Port:    7001,
				Role:    RedisRoleSlave,
			},
			Status: RedisStatusNormal,
		})
	}

	// Cluster 2: 192.18.107.x - 3 masters, 3 slaves
	for i := 5; i <= 7; i++ {
		// Master
		results.Results = append(results.Results, &RedisInspectionResult{
			Instance: &RedisInstance{
				Address: "192.18.107." + string(rune('0'+i)) + ":7000",
				IP:      "192.18.107." + string(rune('0'+i)),
				Port:    7000,
				Role:    RedisRoleMaster,
			},
			Status: RedisStatusNormal,
		})
		// Slave
		results.Results = append(results.Results, &RedisInspectionResult{
			Instance: &RedisInstance{
				Address: "192.18.107." + string(rune('0'+i)) + ":7001",
				IP:      "192.18.107." + string(rune('0'+i)),
				Port:    7001,
				Role:    RedisRoleSlave,
			},
			Status: RedisStatusNormal,
		})
	}

	// Execute grouping
	clusters := results.GroupByClusters()

	// Verify: should have exactly 2 clusters
	if len(clusters) != 2 {
		t.Fatalf("陕西项目场景: expected 2 clusters, got %d", len(clusters))
	}

	// Verify HasMultipleClusters
	if !results.HasMultipleClusters() {
		t.Error("陕西项目场景: HasMultipleClusters() should return true")
	}

	// Verify cluster 1
	cluster1 := clusters[0]
	if cluster1.ID != "192.18.102" {
		t.Errorf("cluster 1 ID = %q, want %q", cluster1.ID, "192.18.102")
	}
	if len(cluster1.Instances) != 6 {
		t.Errorf("cluster 1 should have 6 instances (3主3从), got %d", len(cluster1.Instances))
	}
	if cluster1.GetMasterCount() != 3 {
		t.Errorf("cluster 1 should have 3 masters, got %d", cluster1.GetMasterCount())
	}
	if cluster1.GetSlaveCount() != 3 {
		t.Errorf("cluster 1 should have 3 slaves, got %d", cluster1.GetSlaveCount())
	}

	// Verify cluster 2
	cluster2 := clusters[1]
	if cluster2.ID != "192.18.107" {
		t.Errorf("cluster 2 ID = %q, want %q", cluster2.ID, "192.18.107")
	}
	if len(cluster2.Instances) != 6 {
		t.Errorf("cluster 2 should have 6 instances (3主3从), got %d", len(cluster2.Instances))
	}
	if cluster2.GetMasterCount() != 3 {
		t.Errorf("cluster 2 should have 3 masters, got %d", cluster2.GetMasterCount())
	}
	if cluster2.GetSlaveCount() != 3 {
		t.Errorf("cluster 2 should have 3 slaves, got %d", cluster2.GetSlaveCount())
	}
}
