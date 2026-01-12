package model

import (
	"testing"
	"time"

	pkgmodel "inspection-tool/pkg/model"
)

func TestNewInspectionSummary(t *testing.T) {
	tests := []struct {
		name             string
		hosts            []*HostResult
		expectedTotal    int
		expectedNormal   int
		expectedWarning  int
		expectedCritical int
		expectedFailed   int
	}{
		{
			name:             "empty list",
			hosts:            []*HostResult{},
			expectedTotal:    0,
			expectedNormal:   0,
			expectedWarning:  0,
			expectedCritical: 0,
			expectedFailed:   0,
		},
		{
			name:             "nil list",
			hosts:            nil,
			expectedTotal:    0,
			expectedNormal:   0,
			expectedWarning:  0,
			expectedCritical: 0,
			expectedFailed:   0,
		},
		{
			name: "mixed statuses",
			hosts: []*HostResult{
				{Status: pkgmodel.HostStatusNormal},
				{Status: pkgmodel.HostStatusWarning},
				{Status: pkgmodel.HostStatusCritical},
				{Status: pkgmodel.HostStatusFailed},
				{Status: pkgmodel.HostStatusNormal},
			},
			expectedTotal:    5,
			expectedNormal:   2,
			expectedWarning:  1,
			expectedCritical: 1,
			expectedFailed:   1,
		},
		{
			name: "with nil elements",
			hosts: []*HostResult{
				{Status: pkgmodel.HostStatusNormal},
				nil,
				{Status: pkgmodel.HostStatusWarning},
				nil,
			},
			expectedTotal:    2,
			expectedNormal:   1,
			expectedWarning:  1,
			expectedCritical: 0,
			expectedFailed:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := NewInspectionSummary(tt.hosts)

			if summary.TotalHosts != tt.expectedTotal {
				t.Errorf("TotalHosts = %d, want %d", summary.TotalHosts, tt.expectedTotal)
			}
			if summary.NormalHosts != tt.expectedNormal {
				t.Errorf("NormalHosts = %d, want %d", summary.NormalHosts, tt.expectedNormal)
			}
			if summary.WarningHosts != tt.expectedWarning {
				t.Errorf("WarningHosts = %d, want %d", summary.WarningHosts, tt.expectedWarning)
			}
			if summary.CriticalHosts != tt.expectedCritical {
				t.Errorf("CriticalHosts = %d, want %d", summary.CriticalHosts, tt.expectedCritical)
			}
			if summary.FailedHosts != tt.expectedFailed {
				t.Errorf("FailedHosts = %d, want %d", summary.FailedHosts, tt.expectedFailed)
			}
		})
	}
}

func TestNewHostResult(t *testing.T) {
	t.Run("with valid pkgmodel.HostMeta", func(t *testing.T) {
		meta := &pkgmodel.HostMeta{
			Hostname:      "server-01",
			IP:            "192.168.1.1",
			OS:            "linux",
			OSVersion:     "Ubuntu 22.04",
			KernelVersion: "5.15.0",
			CPUCores:      8,
			CPUModel:      "Intel Xeon",
			MemoryTotal:   17179869184,
		}

		result := NewHostResult(meta)

		if result.Hostname != meta.Hostname {
			t.Errorf("Hostname = %q, want %q", result.Hostname, meta.Hostname)
		}
		if result.IP != meta.IP {
			t.Errorf("IP = %q, want %q", result.IP, meta.IP)
		}
		if result.Status != pkgmodel.HostStatusNormal {
			t.Errorf("Status = %v, want %v", result.Status, pkgmodel.HostStatusNormal)
		}
		if result.Metrics == nil {
			t.Error("Metrics should be initialized")
		}
		if result.Alerts == nil {
			t.Error("Alerts should be initialized")
		}
	})

	t.Run("with nil pkgmodel.HostMeta", func(t *testing.T) {
		result := NewHostResult(nil)

		if result.Status != pkgmodel.HostStatusFailed {
			t.Errorf("Status = %v, want %v", result.Status, pkgmodel.HostStatusFailed)
		}
		if result.Metrics == nil {
			t.Error("Metrics should be initialized even for nil meta")
		}
	})
}

func TestHostResult_SetMetric(t *testing.T) {
	t.Run("normal set", func(t *testing.T) {
		result := &HostResult{Metrics: make(map[string]*MetricValue)}
		value := &MetricValue{Name: "cpu_usage", RawValue: 75.5}

		result.SetMetric(value)

		if result.Metrics["cpu_usage"] != value {
			t.Error("metric should be set")
		}
	})

	t.Run("nil metrics map initialized", func(t *testing.T) {
		result := &HostResult{}
		value := &MetricValue{Name: "cpu_usage", RawValue: 75.5}

		result.SetMetric(value)

		if result.Metrics == nil {
			t.Error("Metrics map should be initialized")
		}
		if result.Metrics["cpu_usage"] != value {
			t.Error("metric should be set")
		}
	})

	t.Run("nil value ignored", func(t *testing.T) {
		result := &HostResult{Metrics: make(map[string]*MetricValue)}
		result.SetMetric(nil)

		if len(result.Metrics) != 0 {
			t.Error("nil value should not add to metrics")
		}
	})
}

func TestHostResult_GetMetric(t *testing.T) {
	t.Run("existing metric", func(t *testing.T) {
		value := &MetricValue{Name: "cpu_usage", RawValue: 75.5}
		result := &HostResult{Metrics: map[string]*MetricValue{"cpu_usage": value}}

		got := result.GetMetric("cpu_usage")
		if got != value {
			t.Error("should return the metric")
		}
	})

	t.Run("non-existing metric", func(t *testing.T) {
		result := &HostResult{Metrics: make(map[string]*MetricValue)}

		got := result.GetMetric("not_exists")
		if got != nil {
			t.Error("should return nil for non-existing metric")
		}
	})

	t.Run("nil metrics map", func(t *testing.T) {
		result := &HostResult{}

		got := result.GetMetric("cpu_usage")
		if got != nil {
			t.Error("should return nil for nil metrics map")
		}
	})
}

func TestHostResult_AddAlert(t *testing.T) {
	t.Run("nil alert ignored", func(t *testing.T) {
		result := &HostResult{Status: pkgmodel.HostStatusNormal, Alerts: make([]*pkgmodel.Alert, 0)}
		result.AddAlert(nil)

		if len(result.Alerts) != 0 {
			t.Error("nil alert should not be added")
		}
	})

	t.Run("warning upgrades status from normal", func(t *testing.T) {
		result := &HostResult{Status: pkgmodel.HostStatusNormal, Alerts: make([]*pkgmodel.Alert, 0)}
		alert := &pkgmodel.Alert{Level: pkgmodel.AlertLevelWarning}

		result.AddAlert(alert)

		if result.Status != pkgmodel.HostStatusWarning {
			t.Errorf("Status = %v, want %v", result.Status, pkgmodel.HostStatusWarning)
		}
		if len(result.Alerts) != 1 {
			t.Errorf("Alerts count = %d, want 1", len(result.Alerts))
		}
	})

	t.Run("critical upgrades status from normal", func(t *testing.T) {
		result := &HostResult{Status: pkgmodel.HostStatusNormal, Alerts: make([]*pkgmodel.Alert, 0)}
		alert := &pkgmodel.Alert{Level: pkgmodel.AlertLevelCritical}

		result.AddAlert(alert)

		if result.Status != pkgmodel.HostStatusCritical {
			t.Errorf("Status = %v, want %v", result.Status, pkgmodel.HostStatusCritical)
		}
	})

	t.Run("critical upgrades status from warning", func(t *testing.T) {
		result := &HostResult{Status: pkgmodel.HostStatusWarning, Alerts: make([]*pkgmodel.Alert, 0)}
		alert := &pkgmodel.Alert{Level: pkgmodel.AlertLevelCritical}

		result.AddAlert(alert)

		if result.Status != pkgmodel.HostStatusCritical {
			t.Errorf("Status = %v, want %v", result.Status, pkgmodel.HostStatusCritical)
		}
	})

	t.Run("warning does not downgrade critical", func(t *testing.T) {
		result := &HostResult{Status: pkgmodel.HostStatusCritical, Alerts: make([]*pkgmodel.Alert, 0)}
		alert := &pkgmodel.Alert{Level: pkgmodel.AlertLevelWarning}

		result.AddAlert(alert)

		if result.Status != pkgmodel.HostStatusCritical {
			t.Errorf("Status = %v, want %v (should not downgrade)", result.Status, pkgmodel.HostStatusCritical)
		}
	})

	t.Run("multiple alerts accumulate", func(t *testing.T) {
		result := &HostResult{Status: pkgmodel.HostStatusNormal, Alerts: make([]*pkgmodel.Alert, 0)}

		result.AddAlert(&pkgmodel.Alert{Level: pkgmodel.AlertLevelWarning})
		result.AddAlert(&pkgmodel.Alert{Level: pkgmodel.AlertLevelWarning})
		result.AddAlert(&pkgmodel.Alert{Level: pkgmodel.AlertLevelCritical})

		if len(result.Alerts) != 3 {
			t.Errorf("Alerts count = %d, want 3", len(result.Alerts))
		}
		if result.Status != pkgmodel.HostStatusCritical {
			t.Errorf("Status = %v, want %v", result.Status, pkgmodel.HostStatusCritical)
		}
	})
}

func TestHostResult_HasAlerts(t *testing.T) {
	t.Run("has alerts", func(t *testing.T) {
		result := &HostResult{Alerts: []*pkgmodel.Alert{{Level: pkgmodel.AlertLevelWarning}}}
		if !result.HasAlerts() {
			t.Error("HasAlerts() should return true")
		}
	})

	t.Run("no alerts", func(t *testing.T) {
		result := &HostResult{Alerts: []*pkgmodel.Alert{}}
		if result.HasAlerts() {
			t.Error("HasAlerts() should return false")
		}
	})

	t.Run("nil alerts", func(t *testing.T) {
		result := &HostResult{}
		if result.HasAlerts() {
			t.Error("HasAlerts() should return false for nil alerts")
		}
	})
}

func TestNewInspectionResult(t *testing.T) {
	now := time.Now()
	result := NewInspectionResult(now)

	if !result.InspectionTime.Equal(now) {
		t.Errorf("InspectionTime = %v, want %v", result.InspectionTime, now)
	}
	if result.Hosts == nil {
		t.Error("Hosts should be initialized")
	}
	if result.Alerts == nil {
		t.Error("Alerts should be initialized")
	}
}

func TestInspectionResult_AddHost(t *testing.T) {
	t.Run("nil host ignored", func(t *testing.T) {
		result := NewInspectionResult(time.Now())
		result.AddHost(nil)

		if len(result.Hosts) != 0 {
			t.Error("nil host should not be added")
		}
	})

	t.Run("host added with alerts collected", func(t *testing.T) {
		result := NewInspectionResult(time.Now())
		host := &HostResult{
			Hostname: "server-01",
			Alerts: []*pkgmodel.Alert{
				{Level: pkgmodel.AlertLevelWarning},
				{Level: pkgmodel.AlertLevelCritical},
			},
		}

		result.AddHost(host)

		if len(result.Hosts) != 1 {
			t.Errorf("Hosts count = %d, want 1", len(result.Hosts))
		}
		if len(result.Alerts) != 2 {
			t.Errorf("Alerts count = %d, want 2", len(result.Alerts))
		}
	})
}

func TestInspectionResult_Finalize(t *testing.T) {
	startTime := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 1, 1, 10, 5, 30, 0, time.UTC)

	result := NewInspectionResult(startTime)
	result.AddHost(&HostResult{Status: pkgmodel.HostStatusNormal})
	result.AddHost(&HostResult{Status: pkgmodel.HostStatusWarning, Alerts: []*pkgmodel.Alert{{Level: pkgmodel.AlertLevelWarning}}})

	result.Finalize(endTime)

	expectedDuration := 5*time.Minute + 30*time.Second
	if result.Duration != expectedDuration {
		t.Errorf("Duration = %v, want %v", result.Duration, expectedDuration)
	}
	if result.Summary == nil {
		t.Error("Summary should be set")
	}
	if result.Summary.TotalHosts != 2 {
		t.Errorf("Summary.TotalHosts = %d, want 2", result.Summary.TotalHosts)
	}
	if result.pkgmodel.AlertSummary == nil {
		t.Error("pkgmodel.AlertSummary should be set")
	}
}

func TestInspectionResult_GetHostByName(t *testing.T) {
	result := NewInspectionResult(time.Now())
	result.AddHost(&HostResult{Hostname: "server-01"})
	result.AddHost(&HostResult{Hostname: "server-02"})
	result.Hosts = append(result.Hosts, nil)

	t.Run("existing host", func(t *testing.T) {
		host := result.GetHostByName("server-01")
		if host == nil || host.Hostname != "server-01" {
			t.Error("should find existing host")
		}
	})

	t.Run("non-existing host", func(t *testing.T) {
		host := result.GetHostByName("server-99")
		if host != nil {
			t.Error("should return nil for non-existing host")
		}
	})
}

func TestInspectionResult_GetCriticalHosts(t *testing.T) {
	result := NewInspectionResult(time.Now())
	result.AddHost(&HostResult{Hostname: "normal", Status: pkgmodel.HostStatusNormal})
	result.AddHost(&HostResult{Hostname: "critical-1", Status: pkgmodel.HostStatusCritical})
	result.AddHost(&HostResult{Hostname: "warning", Status: pkgmodel.HostStatusWarning})
	result.AddHost(&HostResult{Hostname: "critical-2", Status: pkgmodel.HostStatusCritical})
	result.Hosts = append(result.Hosts, nil)

	critical := result.GetCriticalHosts()

	if len(critical) != 2 {
		t.Errorf("GetCriticalHosts() count = %d, want 2", len(critical))
	}
}

func TestInspectionResult_GetWarningHosts(t *testing.T) {
	result := NewInspectionResult(time.Now())
	result.AddHost(&HostResult{Hostname: "normal", Status: pkgmodel.HostStatusNormal})
	result.AddHost(&HostResult{Hostname: "warning-1", Status: pkgmodel.HostStatusWarning})
	result.AddHost(&HostResult{Hostname: "warning-2", Status: pkgmodel.HostStatusWarning})
	result.Hosts = append(result.Hosts, nil)

	warning := result.GetWarningHosts()

	if len(warning) != 2 {
		t.Errorf("GetWarningHosts() count = %d, want 2", len(warning))
	}
}

func TestInspectionResult_GetFailedHosts(t *testing.T) {
	result := NewInspectionResult(time.Now())
	result.AddHost(&HostResult{Hostname: "normal", Status: pkgmodel.HostStatusNormal})
	result.AddHost(&HostResult{Hostname: "failed-1", Status: pkgmodel.HostStatusFailed})
	result.Hosts = append(result.Hosts, nil)

	failed := result.GetFailedHosts()

	if len(failed) != 1 {
		t.Errorf("GetFailedHosts() count = %d, want 1", len(failed))
	}
}

func TestInspectionResult_HasCritical(t *testing.T) {
	t.Run("nil summary returns false", func(t *testing.T) {
		result := &InspectionResult{}
		if result.HasCritical() {
			t.Error("should return false for nil summary")
		}
	})

	t.Run("has critical", func(t *testing.T) {
		result := &InspectionResult{Summary: &InspectionSummary{CriticalHosts: 1}}
		if !result.HasCritical() {
			t.Error("should return true")
		}
	})

	t.Run("no critical", func(t *testing.T) {
		result := &InspectionResult{Summary: &InspectionSummary{CriticalHosts: 0}}
		if result.HasCritical() {
			t.Error("should return false")
		}
	})
}

func TestInspectionResult_HasWarning(t *testing.T) {
	t.Run("nil summary returns false", func(t *testing.T) {
		result := &InspectionResult{}
		if result.HasWarning() {
			t.Error("should return false for nil summary")
		}
	})

	t.Run("has warning", func(t *testing.T) {
		result := &InspectionResult{Summary: &InspectionSummary{WarningHosts: 1}}
		if !result.HasWarning() {
			t.Error("should return true")
		}
	})
}

func TestInspectionResult_HasAlerts(t *testing.T) {
	t.Run("has alerts", func(t *testing.T) {
		result := &InspectionResult{Alerts: []*pkgmodel.Alert{{Level: pkgmodel.AlertLevelWarning}}}
		if !result.HasAlerts() {
			t.Error("should return true")
		}
	})

	t.Run("no alerts", func(t *testing.T) {
		result := &InspectionResult{Alerts: []*pkgmodel.Alert{}}
		if result.HasAlerts() {
			t.Error("should return false")
		}
	})
}
