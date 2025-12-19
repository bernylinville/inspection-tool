package model

import (
	"testing"
	"time"
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
		Level:      AlertLevelWarning,
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
			{Level: AlertLevelWarning},
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
					{Level: AlertLevelWarning},
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
