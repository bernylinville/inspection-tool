package model

import (
	"testing"
	"time"

	pkgmodel "inspection-tool/pkg/model"
)

func TestParseAddress(t *testing.T) {
	tests := []struct {
		name         string
		address      string
		expectedIP   string
		expectedPort int
		wantErr      bool
	}{
		{
			name:         "valid address",
			address:      "192.168.1.1:3306",
			expectedIP:   "192.168.1.1",
			expectedPort: 3306,
			wantErr:      false,
		},
		{
			name:         "valid address with different port",
			address:      "10.0.0.100:3307",
			expectedIP:   "10.0.0.100",
			expectedPort: 3307,
			wantErr:      false,
		},
		{
			name:         "missing port",
			address:      "192.168.1.1",
			expectedIP:   "",
			expectedPort: 0,
			wantErr:      true,
		},
		{
			name:         "empty IP",
			address:      ":3306",
			expectedIP:   "",
			expectedPort: 0,
			wantErr:      true,
		},
		{
			name:         "empty string",
			address:      "",
			expectedIP:   "",
			expectedPort: 0,
			wantErr:      true,
		},
		{
			name:         "invalid port - not a number",
			address:      "192.168.1.1:abc",
			expectedIP:   "",
			expectedPort: 0,
			wantErr:      true,
		},
		{
			name:         "port out of range - zero",
			address:      "192.168.1.1:0",
			expectedIP:   "",
			expectedPort: 0,
			wantErr:      true,
		},
		{
			name:         "port out of range - negative",
			address:      "192.168.1.1:-1",
			expectedIP:   "",
			expectedPort: 0,
			wantErr:      true,
		},
		{
			name:         "port out of range - too large",
			address:      "192.168.1.1:65536",
			expectedIP:   "",
			expectedPort: 0,
			wantErr:      true,
		},
		{
			name:         "valid max port",
			address:      "192.168.1.1:65535",
			expectedIP:   "192.168.1.1",
			expectedPort: 65535,
			wantErr:      false,
		},
		{
			name:         "address with spaces",
			address:      " 192.168.1.1 : 3306 ",
			expectedIP:   "192.168.1.1",
			expectedPort: 3306,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip, port, err := ParseAddress(tt.address)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseAddress(%q) expected error, got nil", tt.address)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseAddress(%q) unexpected error: %v", tt.address, err)
				return
			}
			if ip != tt.expectedIP {
				t.Errorf("IP = %q, want %q", ip, tt.expectedIP)
			}
			if port != tt.expectedPort {
				t.Errorf("Port = %d, want %d", port, tt.expectedPort)
			}
		})
	}
}

func TestMySQLInstanceStatus_Methods(t *testing.T) {
	tests := []struct {
		status     MySQLInstanceStatus
		isHealthy  bool
		isWarning  bool
		isCritical bool
		isFailed   bool
	}{
		{MySQLStatusNormal, true, false, false, false},
		{MySQLStatusWarning, false, true, false, false},
		{MySQLStatusCritical, false, false, true, false},
		{MySQLStatusFailed, false, false, false, true},
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

func TestMySQLClusterMode_Methods(t *testing.T) {
	tests := []struct {
		mode          MySQLClusterMode
		isMGR         bool
		isDualMaster  bool
		isMasterSlave bool
	}{
		{ClusterModeMGR, true, false, false},
		{ClusterModeDualMaster, false, true, false},
		{ClusterModeMasterSlave, false, false, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			if got := tt.mode.IsMGR(); got != tt.isMGR {
				t.Errorf("IsMGR() = %v, want %v", got, tt.isMGR)
			}
			if got := tt.mode.IsDualMaster(); got != tt.isDualMaster {
				t.Errorf("IsDualMaster() = %v, want %v", got, tt.isDualMaster)
			}
			if got := tt.mode.IsMasterSlave(); got != tt.isMasterSlave {
				t.Errorf("IsMasterSlave() = %v, want %v", got, tt.isMasterSlave)
			}
		})
	}
}

func TestMySQLMGRRole_Methods(t *testing.T) {
	tests := []struct {
		role        MySQLMGRRole
		isPrimary   bool
		isSecondary bool
	}{
		{MGRRolePrimary, true, false},
		{MGRRoleSecondary, false, true},
		{MGRRoleUnknown, false, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			if got := tt.role.IsPrimary(); got != tt.isPrimary {
				t.Errorf("IsPrimary() = %v, want %v", got, tt.isPrimary)
			}
			if got := tt.role.IsSecondary(); got != tt.isSecondary {
				t.Errorf("IsSecondary() = %v, want %v", got, tt.isSecondary)
			}
		})
	}
}

func TestNewMySQLInstance(t *testing.T) {
	t.Run("valid address", func(t *testing.T) {
		instance := NewMySQLInstance("192.168.1.1:3306")

		if instance == nil {
			t.Fatal("expected non-nil instance")
		}
		if instance.Address != "192.168.1.1:3306" {
			t.Errorf("Address = %q, want %q", instance.Address, "192.168.1.1:3306")
		}
		if instance.IP != "192.168.1.1" {
			t.Errorf("IP = %q, want %q", instance.IP, "192.168.1.1")
		}
		if instance.Port != 3306 {
			t.Errorf("Port = %d, want 3306", instance.Port)
		}
		if instance.DatabaseType != "MySQL" {
			t.Errorf("DatabaseType = %q, want %q", instance.DatabaseType, "MySQL")
		}
	})

	t.Run("invalid address returns nil", func(t *testing.T) {
		instance := NewMySQLInstance("invalid")
		if instance != nil {
			t.Error("expected nil for invalid address")
		}
	})
}

func TestNewMySQLInstanceWithClusterMode(t *testing.T) {
	instance := NewMySQLInstanceWithClusterMode("192.168.1.1:3306", ClusterModeMGR)

	if instance == nil {
		t.Fatal("expected non-nil instance")
	}
	if instance.ClusterMode != ClusterModeMGR {
		t.Errorf("ClusterMode = %v, want %v", instance.ClusterMode, ClusterModeMGR)
	}
}

func TestMySQLInstance_SetVersion(t *testing.T) {
	instance := NewMySQLInstance("192.168.1.1:3306")
	instance.SetVersion("8.0.39", "8.0.39")

	if instance.Version != "8.0.39" {
		t.Errorf("Version = %q, want %q", instance.Version, "8.0.39")
	}
	if instance.InnoDBVersion != "8.0.39" {
		t.Errorf("InnoDBVersion = %q, want %q", instance.InnoDBVersion, "8.0.39")
	}
}

func TestMySQLInstance_String(t *testing.T) {
	t.Run("normal instance", func(t *testing.T) {
		instance := NewMySQLInstance("192.168.1.1:3306")
		instance.Version = "8.0.39"
		instance.ServerID = "1"
		instance.ClusterMode = ClusterModeMGR

		str := instance.String()
		if str == "" {
			t.Error("String() should return non-empty string")
		}
	})

	t.Run("nil instance", func(t *testing.T) {
		var instance *MySQLInstance
		str := instance.String()
		if str != "<nil>" {
			t.Errorf("String() = %q, want %q", str, "<nil>")
		}
	})
}

func TestNewMySQLAlert(t *testing.T) {
	alert := NewMySQLAlert("192.168.1.1:3306", "connection_usage", 85.5, pkgmodel.AlertLevelWarning)

	if alert.Address != "192.168.1.1:3306" {
		t.Errorf("Address = %q, want %q", alert.Address, "192.168.1.1:3306")
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

func TestMySQLAlert_Methods(t *testing.T) {
	t.Run("IsWarning", func(t *testing.T) {
		alert := &MySQLAlert{Level: pkgmodel.AlertLevelWarning}
		if !alert.IsWarning() {
			t.Error("IsWarning() should return true")
		}
		alert.Level = pkgmodel.AlertLevelCritical
		if alert.IsWarning() {
			t.Error("IsWarning() should return false for critical")
		}
	})

	t.Run("IsCritical", func(t *testing.T) {
		alert := &MySQLAlert{Level: pkgmodel.AlertLevelCritical}
		if !alert.IsCritical() {
			t.Error("IsCritical() should return true")
		}
		alert.Level = pkgmodel.AlertLevelWarning
		if alert.IsCritical() {
			t.Error("IsCritical() should return false for warning")
		}
	})
}

func TestNewMySQLInspectionResult(t *testing.T) {
	t.Run("with valid instance", func(t *testing.T) {
		instance := NewMySQLInstance("192.168.1.1:3306")
		result := NewMySQLInspectionResult(instance)

		if result.Instance != instance {
			t.Error("Instance should be set")
		}
		if result.Status != MySQLStatusNormal {
			t.Errorf("Status = %v, want %v", result.Status, MySQLStatusNormal)
		}
		if result.NonRootUser != "N/A" {
			t.Errorf("NonRootUser = %q, want %q", result.NonRootUser, "N/A")
		}
		if result.Alerts == nil {
			t.Error("Alerts should be initialized")
		}
	})

	t.Run("with nil instance", func(t *testing.T) {
		result := NewMySQLInspectionResult(nil)

		if result.Status != MySQLStatusFailed {
			t.Errorf("Status = %v, want %v", result.Status, MySQLStatusFailed)
		}
	})
}

func TestMySQLInspectionResult_AddAlert(t *testing.T) {
	t.Run("nil alert ignored", func(t *testing.T) {
		result := NewMySQLInspectionResult(NewMySQLInstance("192.168.1.1:3306"))
		result.AddAlert(nil)

		if len(result.Alerts) != 0 {
			t.Error("nil alert should not be added")
		}
	})

	t.Run("warning upgrades status from normal", func(t *testing.T) {
		result := NewMySQLInspectionResult(NewMySQLInstance("192.168.1.1:3306"))
		alert := &MySQLAlert{Level: pkgmodel.AlertLevelWarning}

		result.AddAlert(alert)

		if result.Status != MySQLStatusWarning {
			t.Errorf("Status = %v, want %v", result.Status, MySQLStatusWarning)
		}
	})

	t.Run("critical upgrades status from warning", func(t *testing.T) {
		result := NewMySQLInspectionResult(NewMySQLInstance("192.168.1.1:3306"))
		result.Status = MySQLStatusWarning
		alert := &MySQLAlert{Level: pkgmodel.AlertLevelCritical}

		result.AddAlert(alert)

		if result.Status != MySQLStatusCritical {
			t.Errorf("Status = %v, want %v", result.Status, MySQLStatusCritical)
		}
	})

	t.Run("warning does not downgrade critical", func(t *testing.T) {
		result := NewMySQLInspectionResult(NewMySQLInstance("192.168.1.1:3306"))
		result.Status = MySQLStatusCritical
		alert := &MySQLAlert{Level: pkgmodel.AlertLevelWarning}

		result.AddAlert(alert)

		if result.Status != MySQLStatusCritical {
			t.Errorf("Status = %v, want %v (should not downgrade)", result.Status, MySQLStatusCritical)
		}
	})
}

func TestMySQLInspectionResult_GetConnectionUsagePercent(t *testing.T) {
	tests := []struct {
		name               string
		maxConnections     int
		currentConnections int
		expected           float64
	}{
		{
			name:               "normal calculation",
			maxConnections:     1000,
			currentConnections: 750,
			expected:           75.0,
		},
		{
			name:               "zero max connections",
			maxConnections:     0,
			currentConnections: 100,
			expected:           0,
		},
		{
			name:               "zero current connections",
			maxConnections:     1000,
			currentConnections: 0,
			expected:           0,
		},
		{
			name:               "full capacity",
			maxConnections:     100,
			currentConnections: 100,
			expected:           100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &MySQLInspectionResult{
				MaxConnections:     tt.maxConnections,
				CurrentConnections: tt.currentConnections,
			}

			got := result.GetConnectionUsagePercent()
			if got != tt.expected {
				t.Errorf("GetConnectionUsagePercent() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMySQLInspectionResult_GetAddress(t *testing.T) {
	t.Run("with instance", func(t *testing.T) {
		result := NewMySQLInspectionResult(NewMySQLInstance("192.168.1.1:3306"))
		if got := result.GetAddress(); got != "192.168.1.1:3306" {
			t.Errorf("GetAddress() = %q, want %q", got, "192.168.1.1:3306")
		}
	})

	t.Run("nil instance", func(t *testing.T) {
		result := &MySQLInspectionResult{}
		if got := result.GetAddress(); got != "" {
			t.Errorf("GetAddress() = %q, want empty string", got)
		}
	})
}

func TestMySQLInspectionResult_SetGetMetric(t *testing.T) {
	result := NewMySQLInspectionResult(NewMySQLInstance("192.168.1.1:3306"))

	t.Run("set and get", func(t *testing.T) {
		value := &MySQLMetricValue{Name: "mysql_up", RawValue: 1}
		result.SetMetric(value)

		got := result.GetMetric("mysql_up")
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
}

func TestNewMySQLInspectionSummary(t *testing.T) {
	results := []*MySQLInspectionResult{
		{Status: MySQLStatusNormal},
		{Status: MySQLStatusNormal},
		{Status: MySQLStatusWarning},
		{Status: MySQLStatusCritical},
		{Status: MySQLStatusFailed},
		nil,
	}

	summary := NewMySQLInspectionSummary(results)

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

func TestNewMySQLAlertSummary(t *testing.T) {
	alerts := []*MySQLAlert{
		{Level: pkgmodel.AlertLevelWarning},
		{Level: pkgmodel.AlertLevelWarning},
		{Level: pkgmodel.AlertLevelCritical},
		nil,
	}

	summary := NewMySQLAlertSummary(alerts)

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

func TestMySQLInspectionResults_Methods(t *testing.T) {
	now := time.Now()
	results := NewMySQLInspectionResults(now)

	results.AddResult(&MySQLInspectionResult{
		Instance: NewMySQLInstance("192.168.1.1:3306"),
		Status:   MySQLStatusNormal,
	})
	results.AddResult(&MySQLInspectionResult{
		Instance: NewMySQLInstance("192.168.1.2:3306"),
		Status:   MySQLStatusCritical,
		Alerts:   []*MySQLAlert{{Level: pkgmodel.AlertLevelCritical}},
	})
	results.AddResult(nil)

	t.Run("GetResultByAddress", func(t *testing.T) {
		found := results.GetResultByAddress("192.168.1.1:3306")
		if found == nil {
			t.Error("should find existing result")
		}
		notFound := results.GetResultByAddress("192.168.1.99:3306")
		if notFound != nil {
			t.Error("should return nil for non-existing")
		}
	})

	t.Run("GetCriticalResults", func(t *testing.T) {
		critical := results.GetCriticalResults()
		if len(critical) != 1 {
			t.Errorf("GetCriticalResults() count = %d, want 1", len(critical))
		}
	})

	endTime := now.Add(5 * time.Second)
	results.Finalize(endTime)

	t.Run("HasCritical after finalize", func(t *testing.T) {
		if !results.HasCritical() {
			t.Error("HasCritical() should return true")
		}
	})

	t.Run("HasAlerts", func(t *testing.T) {
		if !results.HasAlerts() {
			t.Error("HasAlerts() should return true")
		}
	})
}

func TestMySQLMetricDefinition_Methods(t *testing.T) {
	t.Run("IsPending", func(t *testing.T) {
		def := &MySQLMetricDefinition{Status: "pending"}
		if !def.IsPending() {
			t.Error("IsPending() should return true for pending status")
		}

		def2 := &MySQLMetricDefinition{Query: ""}
		if !def2.IsPending() {
			t.Error("IsPending() should return true for empty query")
		}

		def3 := &MySQLMetricDefinition{Query: "some_query", Status: ""}
		if def3.IsPending() {
			t.Error("IsPending() should return false for normal metric")
		}
	})

	t.Run("HasLabelExtract", func(t *testing.T) {
		def := &MySQLMetricDefinition{LabelExtract: "version"}
		if !def.HasLabelExtract() {
			t.Error("HasLabelExtract() should return true")
		}

		def2 := &MySQLMetricDefinition{}
		if def2.HasLabelExtract() {
			t.Error("HasLabelExtract() should return false")
		}
	})

	t.Run("IsForClusterMode", func(t *testing.T) {
		def := &MySQLMetricDefinition{ClusterMode: "mgr"}
		if !def.IsForClusterMode(ClusterModeMGR) {
			t.Error("should return true for matching mode")
		}
		if def.IsForClusterMode(ClusterModeDualMaster) {
			t.Error("should return false for non-matching mode")
		}

		defAll := &MySQLMetricDefinition{}
		if !defAll.IsForClusterMode(ClusterModeMGR) {
			t.Error("should return true for empty cluster mode (applies to all)")
		}
	})

	t.Run("GetDisplayName", func(t *testing.T) {
		def := &MySQLMetricDefinition{Name: "mysql_up", DisplayName: "连接状态"}
		if got := def.GetDisplayName(); got != "连接状态" {
			t.Errorf("GetDisplayName() = %q, want %q", got, "连接状态")
		}

		def2 := &MySQLMetricDefinition{Name: "mysql_up"}
		if got := def2.GetDisplayName(); got != "mysql_up" {
			t.Errorf("GetDisplayName() = %q, want %q", got, "mysql_up")
		}
	})
}
