package model

import (
	"testing"
	"time"
)

func TestGenerateNginxIdentifier(t *testing.T) {
	tests := []struct {
		name      string
		hostname  string
		port      int
		container string
		expected  string
	}{
		{
			name:      "with container",
			hostname:  "web-server-01",
			port:      80,
			container: "nginx-proxy",
			expected:  "web-server-01:nginx-proxy",
		},
		{
			name:      "without container",
			hostname:  "web-server-01",
			port:      8080,
			container: "",
			expected:  "web-server-01:8080",
		},
		{
			name:      "empty container uses port",
			hostname:  "nginx-host",
			port:      443,
			container: "",
			expected:  "nginx-host:443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateNginxIdentifier(tt.hostname, tt.port, tt.container)
			if got != tt.expected {
				t.Errorf("GenerateNginxIdentifier() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNginxInstanceStatus_Methods(t *testing.T) {
	tests := []struct {
		status     NginxInstanceStatus
		isHealthy  bool
		isWarning  bool
		isCritical bool
		isFailed   bool
	}{
		{NginxStatusNormal, true, false, false, false},
		{NginxStatusWarning, false, true, false, false},
		{NginxStatusCritical, false, false, true, false},
		{NginxStatusFailed, false, false, false, true},
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

func TestNewNginxInstance(t *testing.T) {
	t.Run("valid instance", func(t *testing.T) {
		instance := NewNginxInstance("web-server-01", 80)

		if instance == nil {
			t.Fatal("expected non-nil instance")
		}
		if instance.Identifier != "web-server-01:80" {
			t.Errorf("Identifier = %q, want %q", instance.Identifier, "web-server-01:80")
		}
		if instance.Hostname != "web-server-01" {
			t.Errorf("Hostname = %q, want %q", instance.Hostname, "web-server-01")
		}
		if instance.Port != 80 {
			t.Errorf("Port = %d, want 80", instance.Port)
		}
		if instance.ApplicationType != "nginx" {
			t.Errorf("ApplicationType = %q, want %q", instance.ApplicationType, "nginx")
		}
	})

	t.Run("empty hostname returns nil", func(t *testing.T) {
		instance := NewNginxInstance("", 80)
		if instance != nil {
			t.Error("expected nil for empty hostname")
		}
	})

	t.Run("invalid port zero returns nil", func(t *testing.T) {
		instance := NewNginxInstance("web-server", 0)
		if instance != nil {
			t.Error("expected nil for port 0")
		}
	})

	t.Run("invalid port negative returns nil", func(t *testing.T) {
		instance := NewNginxInstance("web-server", -1)
		if instance != nil {
			t.Error("expected nil for negative port")
		}
	})

	t.Run("invalid port too large returns nil", func(t *testing.T) {
		instance := NewNginxInstance("web-server", 65536)
		if instance != nil {
			t.Error("expected nil for port > 65535")
		}
	})
}

func TestNewNginxInstanceWithContainer(t *testing.T) {
	t.Run("valid container instance", func(t *testing.T) {
		instance := NewNginxInstanceWithContainer("web-server-01", "nginx-proxy")

		if instance == nil {
			t.Fatal("expected non-nil instance")
		}
		if instance.Identifier != "web-server-01:nginx-proxy" {
			t.Errorf("Identifier = %q, want %q", instance.Identifier, "web-server-01:nginx-proxy")
		}
		if instance.Container != "nginx-proxy" {
			t.Errorf("Container = %q, want %q", instance.Container, "nginx-proxy")
		}
	})

	t.Run("empty hostname returns nil", func(t *testing.T) {
		instance := NewNginxInstanceWithContainer("", "nginx-proxy")
		if instance != nil {
			t.Error("expected nil for empty hostname")
		}
	})

	t.Run("empty container returns nil", func(t *testing.T) {
		instance := NewNginxInstanceWithContainer("web-server", "")
		if instance != nil {
			t.Error("expected nil for empty container")
		}
	})
}

func TestNginxInstance_Methods(t *testing.T) {
	t.Run("IsContainerDeployment", func(t *testing.T) {
		containerInstance := &NginxInstance{Container: "nginx-proxy"}
		if !containerInstance.IsContainerDeployment() {
			t.Error("should return true for container deployment")
		}

		binaryInstance := &NginxInstance{Port: 80}
		if binaryInstance.IsContainerDeployment() {
			t.Error("should return false for binary deployment")
		}
	})

	t.Run("String", func(t *testing.T) {
		instance := NewNginxInstance("web-server", 80)
		instance.SetApplicationType("openresty")
		instance.SetVersion("1.21.4")

		str := instance.String()
		if str == "" {
			t.Error("String() should return non-empty string")
		}
	})

	t.Run("String nil", func(t *testing.T) {
		var instance *NginxInstance
		if instance.String() != "<nil>" {
			t.Errorf("String() = %q, want %q", instance.String(), "<nil>")
		}
	})

	t.Run("SetAccessLogPath", func(t *testing.T) {
		instance := NewNginxInstance("web-server", 80)
		instance.SetAccessLogPath("/var/log/nginx/access.log")
		if instance.AccessLogPath != "/var/log/nginx/access.log" {
			t.Errorf("AccessLogPath = %q, want %q", instance.AccessLogPath, "/var/log/nginx/access.log")
		}
	})
}

func TestNewNginxUpstreamStatus(t *testing.T) {
	status := NewNginxUpstreamStatus("backend-pool", "192.168.1.1:8080", true)

	if status.UpstreamName != "backend-pool" {
		t.Errorf("UpstreamName = %q, want %q", status.UpstreamName, "backend-pool")
	}
	if status.BackendAddress != "192.168.1.1:8080" {
		t.Errorf("BackendAddress = %q, want %q", status.BackendAddress, "192.168.1.1:8080")
	}
	if !status.Status {
		t.Error("Status should be true")
	}
	if !status.IsHealthy() {
		t.Error("IsHealthy() should return true")
	}
}

func TestNewNginxAlert(t *testing.T) {
	alert := NewNginxAlert("web-server:80", "connection_usage", 85.5, AlertLevelWarning)

	if alert.Identifier != "web-server:80" {
		t.Errorf("Identifier = %q, want %q", alert.Identifier, "web-server:80")
	}
	if alert.MetricName != "connection_usage" {
		t.Errorf("MetricName = %q, want %q", alert.MetricName, "connection_usage")
	}
	if alert.CurrentValue != 85.5 {
		t.Errorf("CurrentValue = %v, want 85.5", alert.CurrentValue)
	}
	if alert.Level != AlertLevelWarning {
		t.Errorf("Level = %v, want %v", alert.Level, AlertLevelWarning)
	}
}

func TestNginxAlert_Methods(t *testing.T) {
	t.Run("IsWarning", func(t *testing.T) {
		alert := &NginxAlert{Level: AlertLevelWarning}
		if !alert.IsWarning() {
			t.Error("IsWarning() should return true")
		}
	})

	t.Run("IsCritical", func(t *testing.T) {
		alert := &NginxAlert{Level: AlertLevelCritical}
		if !alert.IsCritical() {
			t.Error("IsCritical() should return true")
		}
	})
}

func TestNewNginxInspectionResult(t *testing.T) {
	t.Run("with valid instance", func(t *testing.T) {
		instance := NewNginxInstance("web-server", 80)
		result := NewNginxInspectionResult(instance)

		if result.Instance != instance {
			t.Error("Instance should be set")
		}
		if result.Status != NginxStatusNormal {
			t.Errorf("Status = %v, want %v", result.Status, NginxStatusNormal)
		}
		if result.ConnectionUsagePercent != -1 {
			t.Errorf("ConnectionUsagePercent = %v, want -1", result.ConnectionUsagePercent)
		}
		if result.Alerts == nil {
			t.Error("Alerts should be initialized")
		}
		if result.UpstreamStatus == nil {
			t.Error("UpstreamStatus should be initialized")
		}
	})

	t.Run("with nil instance", func(t *testing.T) {
		result := NewNginxInspectionResult(nil)

		if result.Status != NginxStatusFailed {
			t.Errorf("Status = %v, want %v", result.Status, NginxStatusFailed)
		}
	})
}

func TestNginxInspectionResult_AddAlert(t *testing.T) {
	t.Run("nil alert ignored", func(t *testing.T) {
		result := NewNginxInspectionResult(NewNginxInstance("web-server", 80))
		result.AddAlert(nil)

		if len(result.Alerts) != 0 {
			t.Error("nil alert should not be added")
		}
	})

	t.Run("warning upgrades status", func(t *testing.T) {
		result := NewNginxInspectionResult(NewNginxInstance("web-server", 80))
		result.AddAlert(&NginxAlert{Level: AlertLevelWarning})

		if result.Status != NginxStatusWarning {
			t.Errorf("Status = %v, want %v", result.Status, NginxStatusWarning)
		}
	})

	t.Run("critical upgrades status", func(t *testing.T) {
		result := NewNginxInspectionResult(NewNginxInstance("web-server", 80))
		result.AddAlert(&NginxAlert{Level: AlertLevelCritical})

		if result.Status != NginxStatusCritical {
			t.Errorf("Status = %v, want %v", result.Status, NginxStatusCritical)
		}
	})

	t.Run("warning does not downgrade critical", func(t *testing.T) {
		result := NewNginxInspectionResult(NewNginxInstance("web-server", 80))
		result.Status = NginxStatusCritical
		result.AddAlert(&NginxAlert{Level: AlertLevelWarning})

		if result.Status != NginxStatusCritical {
			t.Errorf("Status = %v, want %v (should not downgrade)", result.Status, NginxStatusCritical)
		}
	})
}

func TestNginxInspectionResult_GetConnectionUsagePercent(t *testing.T) {
	tests := []struct {
		name              string
		workerProcesses   int
		workerConnections int
		activeConnections int
		expected          float64
	}{
		{
			name:              "normal calculation",
			workerProcesses:   4,
			workerConnections: 1024,
			activeConnections: 2048,
			expected:          50.0,
		},
		{
			name:              "zero worker processes",
			workerProcesses:   0,
			workerConnections: 1024,
			activeConnections: 100,
			expected:          -1,
		},
		{
			name:              "zero worker connections",
			workerProcesses:   4,
			workerConnections: 0,
			activeConnections: 100,
			expected:          -1,
		},
		{
			name:              "zero active connections",
			workerProcesses:   4,
			workerConnections: 1024,
			activeConnections: 0,
			expected:          0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &NginxInspectionResult{
				WorkerProcesses:   tt.workerProcesses,
				WorkerConnections: tt.workerConnections,
				ActiveConnections: tt.activeConnections,
			}

			got := result.GetConnectionUsagePercent()
			if got != tt.expected {
				t.Errorf("GetConnectionUsagePercent() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNginxInspectionResult_FormatLastErrorTime(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")

	t.Run("no error", func(t *testing.T) {
		result := &NginxInspectionResult{LastErrorTimestamp: 0}
		got := result.FormatLastErrorTime(loc)
		if got != "无错误" {
			t.Errorf("FormatLastErrorTime() = %q, want %q", got, "无错误")
		}
	})

	t.Run("with timestamp", func(t *testing.T) {
		ts := time.Date(2025, 1, 15, 10, 30, 45, 0, time.UTC).Unix()
		result := &NginxInspectionResult{LastErrorTimestamp: ts}
		got := result.FormatLastErrorTime(time.UTC)

		expected := "2025-01-15 10:30:45"
		if got != expected {
			t.Errorf("FormatLastErrorTime() = %q, want %q", got, expected)
		}
	})

	t.Run("nil location uses UTC", func(t *testing.T) {
		ts := time.Date(2025, 1, 15, 10, 30, 45, 0, time.UTC).Unix()
		result := &NginxInspectionResult{LastErrorTimestamp: ts}
		got := result.FormatLastErrorTime(nil)

		if got == "" {
			t.Error("FormatLastErrorTime() should return non-empty string")
		}
	})
}

func TestNginxInspectionResult_UpstreamMethods(t *testing.T) {
	result := NewNginxInspectionResult(NewNginxInstance("web-server", 80))

	t.Run("HasUpstreamStatus false initially", func(t *testing.T) {
		if result.HasUpstreamStatus() {
			t.Error("HasUpstreamStatus() should return false initially")
		}
	})

	result.AddUpstreamStatus(NginxUpstreamStatus{UpstreamName: "backend", BackendAddress: "192.168.1.1:8080", Status: true})
	result.AddUpstreamStatus(NginxUpstreamStatus{UpstreamName: "backend", BackendAddress: "192.168.1.2:8080", Status: false})

	t.Run("HasUpstreamStatus true after adding", func(t *testing.T) {
		if !result.HasUpstreamStatus() {
			t.Error("HasUpstreamStatus() should return true")
		}
	})

	t.Run("GetUnhealthyUpstreams", func(t *testing.T) {
		unhealthy := result.GetUnhealthyUpstreams()
		if len(unhealthy) != 1 {
			t.Errorf("GetUnhealthyUpstreams() count = %d, want 1", len(unhealthy))
		}
	})
}

func TestNewNginxInspectionSummary(t *testing.T) {
	results := []*NginxInspectionResult{
		{Status: NginxStatusNormal},
		{Status: NginxStatusNormal},
		{Status: NginxStatusWarning},
		{Status: NginxStatusCritical},
		{Status: NginxStatusFailed},
		nil,
	}

	summary := NewNginxInspectionSummary(results)

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

func TestNewNginxAlertSummary(t *testing.T) {
	alerts := []*NginxAlert{
		{Level: AlertLevelWarning},
		{Level: AlertLevelCritical},
		{Level: AlertLevelCritical},
		nil,
	}

	summary := NewNginxAlertSummary(alerts)

	if summary.TotalAlerts != 3 {
		t.Errorf("TotalAlerts = %d, want 3", summary.TotalAlerts)
	}
	if summary.WarningCount != 1 {
		t.Errorf("WarningCount = %d, want 1", summary.WarningCount)
	}
	if summary.CriticalCount != 2 {
		t.Errorf("CriticalCount = %d, want 2", summary.CriticalCount)
	}
}

func TestNginxInspectionResults_Methods(t *testing.T) {
	now := time.Now()
	results := NewNginxInspectionResults(now)

	results.AddResult(&NginxInspectionResult{
		Instance: NewNginxInstance("web-01", 80),
		Status:   NginxStatusNormal,
	})
	results.AddResult(&NginxInspectionResult{
		Instance: NewNginxInstance("web-02", 80),
		Status:   NginxStatusCritical,
		Alerts:   []*NginxAlert{{Level: AlertLevelCritical}},
	})
	results.AddResult(nil)

	t.Run("GetResultByIdentifier", func(t *testing.T) {
		found := results.GetResultByIdentifier("web-01:80")
		if found == nil {
			t.Error("should find existing result")
		}
		notFound := results.GetResultByIdentifier("web-99:80")
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
}

func TestNginxMetricDefinition_Methods(t *testing.T) {
	t.Run("IsPending", func(t *testing.T) {
		def := &NginxMetricDefinition{Status: "pending"}
		if !def.IsPending() {
			t.Error("IsPending() should return true for pending status")
		}

		def2 := &NginxMetricDefinition{Query: ""}
		if !def2.IsPending() {
			t.Error("IsPending() should return true for empty query")
		}

		def3 := &NginxMetricDefinition{Query: "some_query", Status: ""}
		if def3.IsPending() {
			t.Error("IsPending() should return false for normal metric")
		}
	})

	t.Run("HasLabelExtract", func(t *testing.T) {
		def := &NginxMetricDefinition{LabelExtract: []string{"version"}}
		if !def.HasLabelExtract() {
			t.Error("HasLabelExtract() should return true")
		}

		def2 := &NginxMetricDefinition{}
		if def2.HasLabelExtract() {
			t.Error("HasLabelExtract() should return false")
		}
	})

	t.Run("GetDisplayName", func(t *testing.T) {
		def := &NginxMetricDefinition{Name: "nginx_up", DisplayName: "运行状态"}
		if got := def.GetDisplayName(); got != "运行状态" {
			t.Errorf("GetDisplayName() = %q, want %q", got, "运行状态")
		}

		def2 := &NginxMetricDefinition{Name: "nginx_up"}
		if got := def2.GetDisplayName(); got != "nginx_up" {
			t.Errorf("GetDisplayName() = %q, want %q", got, "nginx_up")
		}
	})
}
