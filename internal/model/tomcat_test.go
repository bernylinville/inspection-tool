package model

import (
	"testing"
	"time"
)

func TestGenerateTomcatIdentifier(t *testing.T) {
	tests := []struct {
		name      string
		hostname  string
		port      int
		container string
		expected  string
	}{
		{
			name:      "with container",
			hostname:  "app-server-01",
			port:      8080,
			container: "tomcat-app",
			expected:  "app-server-01:tomcat-app",
		},
		{
			name:      "without container",
			hostname:  "app-server-01",
			port:      8080,
			container: "",
			expected:  "app-server-01:8080",
		},
		{
			name:      "empty container uses port",
			hostname:  "tomcat-host",
			port:      9090,
			container: "",
			expected:  "tomcat-host:9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateTomcatIdentifier(tt.hostname, tt.port, tt.container)
			if got != tt.expected {
				t.Errorf("GenerateTomcatIdentifier() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestTomcatInstanceStatus_Methods(t *testing.T) {
	tests := []struct {
		status     TomcatInstanceStatus
		isHealthy  bool
		isWarning  bool
		isCritical bool
		isFailed   bool
	}{
		{TomcatStatusNormal, true, false, false, false},
		{TomcatStatusWarning, false, true, false, false},
		{TomcatStatusCritical, false, false, true, false},
		{TomcatStatusFailed, false, false, false, true},
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

func TestNewTomcatInstance(t *testing.T) {
	instance := NewTomcatInstance("app-server-01", 8080)

	if instance == nil {
		t.Fatal("expected non-nil instance")
	}
	if instance.Identifier != "app-server-01:8080" {
		t.Errorf("Identifier = %q, want %q", instance.Identifier, "app-server-01:8080")
	}
	if instance.Hostname != "app-server-01" {
		t.Errorf("Hostname = %q, want %q", instance.Hostname, "app-server-01")
	}
	if instance.Port != 8080 {
		t.Errorf("Port = %d, want 8080", instance.Port)
	}
	if instance.ApplicationType != "tomcat" {
		t.Errorf("ApplicationType = %q, want %q", instance.ApplicationType, "tomcat")
	}
}

func TestNewTomcatInstanceWithContainer(t *testing.T) {
	instance := NewTomcatInstanceWithContainer("app-server-01", "tomcat-app")

	if instance == nil {
		t.Fatal("expected non-nil instance")
	}
	if instance.Identifier != "app-server-01:tomcat-app" {
		t.Errorf("Identifier = %q, want %q", instance.Identifier, "app-server-01:tomcat-app")
	}
	if instance.Container != "tomcat-app" {
		t.Errorf("Container = %q, want %q", instance.Container, "tomcat-app")
	}
	if instance.Port != 0 {
		t.Errorf("Port = %d, want 0", instance.Port)
	}
}

func TestTomcatInstance_Methods(t *testing.T) {
	t.Run("SetIP", func(t *testing.T) {
		instance := NewTomcatInstance("app-server", 8080)
		instance.SetIP("192.168.1.1")
		if instance.IP != "192.168.1.1" {
			t.Errorf("IP = %q, want %q", instance.IP, "192.168.1.1")
		}
	})

	t.Run("SetIP nil safe", func(t *testing.T) {
		var instance *TomcatInstance
		instance.SetIP("192.168.1.1")
	})

	t.Run("SetVersion", func(t *testing.T) {
		instance := NewTomcatInstance("app-server", 8080)
		instance.SetVersion("9.0.65")
		if instance.Version != "9.0.65" {
			t.Errorf("Version = %q, want %q", instance.Version, "9.0.65")
		}
	})

	t.Run("SetVersion nil safe", func(t *testing.T) {
		var instance *TomcatInstance
		instance.SetVersion("9.0.65")
	})

	t.Run("SetApplicationType", func(t *testing.T) {
		instance := NewTomcatInstance("app-server", 8080)
		instance.SetApplicationType("jetty")
		if instance.ApplicationType != "jetty" {
			t.Errorf("ApplicationType = %q, want %q", instance.ApplicationType, "jetty")
		}
	})

	t.Run("SetInstallPath", func(t *testing.T) {
		instance := NewTomcatInstance("app-server", 8080)
		instance.SetInstallPath("/opt/tomcat")
		if instance.InstallPath != "/opt/tomcat" {
			t.Errorf("InstallPath = %q, want %q", instance.InstallPath, "/opt/tomcat")
		}
	})

	t.Run("SetLogPath", func(t *testing.T) {
		instance := NewTomcatInstance("app-server", 8080)
		instance.SetLogPath("/var/log/tomcat")
		if instance.LogPath != "/var/log/tomcat" {
			t.Errorf("LogPath = %q, want %q", instance.LogPath, "/var/log/tomcat")
		}
	})

	t.Run("SetJVMConfig", func(t *testing.T) {
		instance := NewTomcatInstance("app-server", 8080)
		instance.SetJVMConfig("-Xmx2g")
		if instance.JVMConfig != "-Xmx2g" {
			t.Errorf("JVMConfig = %q, want %q", instance.JVMConfig, "-Xmx2g")
		}
	})

	t.Run("IsContainerDeployment", func(t *testing.T) {
		containerInstance := NewTomcatInstanceWithContainer("app-server", "tomcat-app")
		if !containerInstance.IsContainerDeployment() {
			t.Error("should return true for container deployment")
		}

		binaryInstance := NewTomcatInstance("app-server", 8080)
		if binaryInstance.IsContainerDeployment() {
			t.Error("should return false for binary deployment")
		}

		var nilInstance *TomcatInstance
		if nilInstance.IsContainerDeployment() {
			t.Error("should return false for nil instance")
		}
	})

	t.Run("String", func(t *testing.T) {
		instance := NewTomcatInstance("app-server", 8080)
		str := instance.String()
		if str == "" {
			t.Error("String() should return non-empty string")
		}
	})

	t.Run("String container", func(t *testing.T) {
		instance := NewTomcatInstanceWithContainer("app-server", "tomcat-app")
		str := instance.String()
		if str == "" {
			t.Error("String() should return non-empty string")
		}
	})

	t.Run("String nil", func(t *testing.T) {
		var instance *TomcatInstance
		str := instance.String()
		if str != "TomcatInstance(nil)" {
			t.Errorf("String() = %q, want %q", str, "TomcatInstance(nil)")
		}
	})
}

func TestNewTomcatAlert(t *testing.T) {
	alert := NewTomcatAlert("app-server:8080", "heap_usage", 85.5, AlertLevelWarning)

	if alert.Identifier != "app-server:8080" {
		t.Errorf("Identifier = %q, want %q", alert.Identifier, "app-server:8080")
	}
	if alert.MetricName != "heap_usage" {
		t.Errorf("MetricName = %q, want %q", alert.MetricName, "heap_usage")
	}
	if alert.CurrentValue != 85.5 {
		t.Errorf("CurrentValue = %v, want 85.5", alert.CurrentValue)
	}
	if alert.Level != AlertLevelWarning {
		t.Errorf("Level = %v, want %v", alert.Level, AlertLevelWarning)
	}
}

func TestTomcatAlert_Methods(t *testing.T) {
	t.Run("IsWarning", func(t *testing.T) {
		alert := &TomcatAlert{Level: AlertLevelWarning}
		if !alert.IsWarning() {
			t.Error("IsWarning() should return true")
		}

		alert2 := &TomcatAlert{Level: AlertLevelCritical}
		if alert2.IsWarning() {
			t.Error("IsWarning() should return false for critical")
		}
	})

	t.Run("IsCritical", func(t *testing.T) {
		alert := &TomcatAlert{Level: AlertLevelCritical}
		if !alert.IsCritical() {
			t.Error("IsCritical() should return true")
		}

		alert2 := &TomcatAlert{Level: AlertLevelWarning}
		if alert2.IsCritical() {
			t.Error("IsCritical() should return false for warning")
		}
	})

	t.Run("nil alert IsWarning", func(t *testing.T) {
		var alert *TomcatAlert
		if alert.IsWarning() {
			t.Error("IsWarning() should return false for nil")
		}
	})

	t.Run("nil alert IsCritical", func(t *testing.T) {
		var alert *TomcatAlert
		if alert.IsCritical() {
			t.Error("IsCritical() should return false for nil")
		}
	})
}

func TestNewTomcatInspectionResult(t *testing.T) {
	instance := NewTomcatInstance("app-server", 8080)
	result := NewTomcatInspectionResult(instance)

	if result.Instance != instance {
		t.Error("Instance should be set")
	}
	if result.Status != TomcatStatusNormal {
		t.Errorf("Status = %v, want %v", result.Status, TomcatStatusNormal)
	}
	if result.Alerts == nil {
		t.Error("Alerts should be initialized")
	}
	if result.LastErrorTimestamp != 0 {
		t.Errorf("LastErrorTimestamp = %d, want 0", result.LastErrorTimestamp)
	}
}

func TestTomcatInspectionResult_AddAlert(t *testing.T) {
	t.Run("nil receiver safe", func(t *testing.T) {
		var result *TomcatInspectionResult
		result.AddAlert(&TomcatAlert{Level: AlertLevelWarning})
	})

	t.Run("nil alert ignored", func(t *testing.T) {
		result := NewTomcatInspectionResult(NewTomcatInstance("app-server", 8080))
		result.AddAlert(nil)

		if len(result.Alerts) != 0 {
			t.Error("nil alert should not be added")
		}
	})

	t.Run("alert added", func(t *testing.T) {
		result := NewTomcatInspectionResult(NewTomcatInstance("app-server", 8080))
		alert := &TomcatAlert{Level: AlertLevelWarning}

		result.AddAlert(alert)

		if len(result.Alerts) != 1 {
			t.Errorf("Alerts count = %d, want 1", len(result.Alerts))
		}
	})
}

func TestTomcatInspectionResult_HasAlerts(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var result *TomcatInspectionResult
		if result.HasAlerts() {
			t.Error("HasAlerts() should return false for nil")
		}
	})

	t.Run("has alerts", func(t *testing.T) {
		result := NewTomcatInspectionResult(NewTomcatInstance("app-server", 8080))
		result.Alerts = []*TomcatAlert{{Level: AlertLevelWarning}}
		if !result.HasAlerts() {
			t.Error("HasAlerts() should return true")
		}
	})

	t.Run("no alerts", func(t *testing.T) {
		result := NewTomcatInspectionResult(NewTomcatInstance("app-server", 8080))
		if result.HasAlerts() {
			t.Error("HasAlerts() should return false")
		}
	})
}

func TestTomcatInspectionResult_FormatUptime(t *testing.T) {
	loc := time.UTC

	tests := []struct {
		name          string
		uptimeSeconds int64
		expected      string
	}{
		{
			name:          "zero seconds",
			uptimeSeconds: 0,
			expected:      "00:00:00",
		},
		{
			name:          "less than a day",
			uptimeSeconds: 3661,
			expected:      "01:01:01",
		},
		{
			name:          "exactly one day",
			uptimeSeconds: 86400,
			expected:      "1天 00:00:00",
		},
		{
			name:          "multiple days",
			uptimeSeconds: 90061,
			expected:      "1天 01:01:01",
		},
		{
			name:          "many days",
			uptimeSeconds: 259200 + 7200 + 180 + 5,
			expected:      "3天 02:03:05",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &TomcatInspectionResult{UptimeSeconds: tt.uptimeSeconds}
			got := result.FormatUptime(loc)
			if got != tt.expected {
				t.Errorf("FormatUptime() = %q, want %q", got, tt.expected)
			}
		})
	}

	t.Run("nil receiver", func(t *testing.T) {
		var result *TomcatInspectionResult
		got := result.FormatUptime(loc)
		if got != "N/A" {
			t.Errorf("FormatUptime() = %q, want %q", got, "N/A")
		}
	})
}

func TestTomcatInspectionResult_FormatLastErrorTime(t *testing.T) {
	loc := time.UTC

	t.Run("no error", func(t *testing.T) {
		result := &TomcatInspectionResult{LastErrorTimestamp: 0}
		got := result.FormatLastErrorTime(loc)
		if got != "无错误" {
			t.Errorf("FormatLastErrorTime() = %q, want %q", got, "无错误")
		}
	})

	t.Run("with timestamp", func(t *testing.T) {
		ts := time.Date(2025, 1, 15, 10, 30, 45, 0, time.UTC).Unix()
		result := &TomcatInspectionResult{LastErrorTimestamp: ts}
		got := result.FormatLastErrorTime(loc)

		expected := "2025-01-15 10:30:45"
		if got != expected {
			t.Errorf("FormatLastErrorTime() = %q, want %q", got, expected)
		}
	})

	t.Run("nil receiver", func(t *testing.T) {
		var result *TomcatInspectionResult
		got := result.FormatLastErrorTime(loc)
		if got != "N/A" {
			t.Errorf("FormatLastErrorTime() = %q, want %q", got, "N/A")
		}
	})

	t.Run("nil location uses UTC", func(t *testing.T) {
		ts := time.Date(2025, 1, 15, 10, 30, 45, 0, time.UTC).Unix()
		result := &TomcatInspectionResult{LastErrorTimestamp: ts}
		got := result.FormatLastErrorTime(nil)

		if got == "" {
			t.Error("FormatLastErrorTime() should return non-empty string")
		}
	})
}

func TestTomcatInspectionResult_GetIdentifier(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var result *TomcatInspectionResult
		if got := result.GetIdentifier(); got != "" {
			t.Errorf("GetIdentifier() = %q, want empty string", got)
		}
	})

	t.Run("nil instance", func(t *testing.T) {
		result := &TomcatInspectionResult{}
		if got := result.GetIdentifier(); got != "" {
			t.Errorf("GetIdentifier() = %q, want empty string", got)
		}
	})

	t.Run("with instance", func(t *testing.T) {
		result := NewTomcatInspectionResult(NewTomcatInstance("app-server", 8080))
		if got := result.GetIdentifier(); got != "app-server:8080" {
			t.Errorf("GetIdentifier() = %q, want %q", got, "app-server:8080")
		}
	})
}

func TestTomcatInspectionResult_SetGetMetric(t *testing.T) {
	t.Run("nil receiver safe", func(t *testing.T) {
		var result *TomcatInspectionResult
		result.SetMetric(&TomcatMetricValue{Name: "test"})
	})

	t.Run("nil value safe", func(t *testing.T) {
		result := NewTomcatInspectionResult(NewTomcatInstance("app-server", 8080))
		result.SetMetric(nil)
	})

	t.Run("set and get", func(t *testing.T) {
		result := NewTomcatInspectionResult(NewTomcatInstance("app-server", 8080))
		value := &TomcatMetricValue{Name: "tomcat_up", RawValue: 1}
		result.SetMetric(value)

		got := result.GetMetric("tomcat_up")
		if got != value {
			t.Error("should return the set metric")
		}
	})

	t.Run("get from nil receiver", func(t *testing.T) {
		var result *TomcatInspectionResult
		got := result.GetMetric("tomcat_up")
		if got != nil {
			t.Error("should return nil for nil receiver")
		}
	})

	t.Run("get from nil metrics", func(t *testing.T) {
		result := &TomcatInspectionResult{}
		got := result.GetMetric("tomcat_up")
		if got != nil {
			t.Error("should return nil for nil metrics map")
		}
	})
}

func TestNewTomcatInspectionSummary(t *testing.T) {
	results := []*TomcatInspectionResult{
		{Status: TomcatStatusNormal},
		{Status: TomcatStatusNormal},
		{Status: TomcatStatusWarning},
		{Status: TomcatStatusCritical},
		{Status: TomcatStatusFailed},
		nil,
	}

	summary := NewTomcatInspectionSummary(results)

	if summary.TotalInstances != 6 {
		t.Errorf("TotalInstances = %d, want 6", summary.TotalInstances)
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

func TestNewTomcatAlertSummary(t *testing.T) {
	alerts := []*TomcatAlert{
		{Level: AlertLevelWarning},
		{Level: AlertLevelCritical},
		{Level: AlertLevelCritical},
		nil,
	}

	summary := NewTomcatAlertSummary(alerts)

	if summary.TotalAlerts != 4 {
		t.Errorf("TotalAlerts = %d, want 4", summary.TotalAlerts)
	}
	if summary.WarningCount != 1 {
		t.Errorf("WarningCount = %d, want 1", summary.WarningCount)
	}
	if summary.CriticalCount != 2 {
		t.Errorf("CriticalCount = %d, want 2", summary.CriticalCount)
	}
}

func TestTomcatInspectionResults_Methods(t *testing.T) {
	now := time.Now()
	results := NewTomcatInspectionResults(now)

	t.Run("AddResult nil receiver safe", func(t *testing.T) {
		var nilResults *TomcatInspectionResults
		nilResults.AddResult(&TomcatInspectionResult{})
	})

	results.AddResult(&TomcatInspectionResult{
		Instance: NewTomcatInstance("app-01", 8080),
		Status:   TomcatStatusNormal,
	})
	results.AddResult(&TomcatInspectionResult{
		Instance: NewTomcatInstance("app-02", 8080),
		Status:   TomcatStatusCritical,
		Alerts:   []*TomcatAlert{{Level: AlertLevelCritical}},
	})
	results.AddResult(nil)

	t.Run("GetResultByIdentifier", func(t *testing.T) {
		found := results.GetResultByIdentifier("app-01:8080")
		if found == nil {
			t.Error("should find existing result")
		}
		notFound := results.GetResultByIdentifier("app-99:8080")
		if notFound != nil {
			t.Error("should return nil for non-existing")
		}
	})

	t.Run("GetResultByIdentifier nil receiver", func(t *testing.T) {
		var nilResults *TomcatInspectionResults
		got := nilResults.GetResultByIdentifier("app-01:8080")
		if got != nil {
			t.Error("should return nil for nil receiver")
		}
	})

	t.Run("GetCriticalResults", func(t *testing.T) {
		critical := results.GetCriticalResults()
		if len(critical) != 1 {
			t.Errorf("GetCriticalResults() count = %d, want 1", len(critical))
		}
	})

	t.Run("GetWarningResults", func(t *testing.T) {
		warning := results.GetWarningResults()
		if len(warning) != 0 {
			t.Errorf("GetWarningResults() count = %d, want 0", len(warning))
		}
	})

	t.Run("GetFailedResults", func(t *testing.T) {
		failed := results.GetFailedResults()
		if len(failed) != 0 {
			t.Errorf("GetFailedResults() count = %d, want 0", len(failed))
		}
	})

	t.Run("Finalize nil receiver safe", func(t *testing.T) {
		var nilResults *TomcatInspectionResults
		nilResults.Finalize(time.Now())
	})

	endTime := now.Add(5 * time.Second)
	results.Finalize(endTime)

	t.Run("HasCritical after finalize", func(t *testing.T) {
		if !results.HasCritical() {
			t.Error("HasCritical() should return true")
		}
	})

	t.Run("HasWarning after finalize", func(t *testing.T) {
		if results.HasWarning() {
			t.Error("HasWarning() should return false")
		}
	})

	t.Run("HasAlerts", func(t *testing.T) {
		if !results.HasAlerts() {
			t.Error("HasAlerts() should return true")
		}
	})

	t.Run("HasCritical nil receiver", func(t *testing.T) {
		var nilResults *TomcatInspectionResults
		if nilResults.HasCritical() {
			t.Error("HasCritical() should return false for nil")
		}
	})

	t.Run("HasWarning nil receiver", func(t *testing.T) {
		var nilResults *TomcatInspectionResults
		if nilResults.HasWarning() {
			t.Error("HasWarning() should return false for nil")
		}
	})

	t.Run("HasAlerts nil receiver", func(t *testing.T) {
		var nilResults *TomcatInspectionResults
		if nilResults.HasAlerts() {
			t.Error("HasAlerts() should return false for nil")
		}
	})
}
