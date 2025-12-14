package n9e

import (
	"encoding/json"
	"testing"
)

// Sample extend_info JSON from actual N9E API response
const sampleExtendInfo = `{
	"cpu": {
		"cache_size": "16384 KB",
		"cpu_cores": "4",
		"cpu_logical_processors": "8",
		"family": "6",
		"mhz": "2799.998",
		"model": "106",
		"model_name": "Intel(R) Xeon(R) Platinum 8378C CPU @ 2.80GHz",
		"stepping": "6",
		"vendor_id": "GenuineIntel"
	},
	"memory": {
		"swap_total": "0",
		"total": "16496934912"
	},
	"network": {
		"interfaces": [
			{"ipv4": "192.168.10.24", "ipv4-network": "192.168.10.0/24", "macaddress": "fa:16:3e:a8:64:94", "name": "eth0"},
			{"ipv4": "10.181.192.102", "ipv4-network": "10.181.192.0/18", "macaddress": "fa:16:3e:7d:1e:28", "name": "eth1"}
		],
		"ipaddress": "192.168.10.24",
		"macaddress": "fa:16:3e:a8:64:94"
	},
	"platform": {
		"GOARCH": "amd64",
		"GOOS": "linux",
		"goV": "1.21.13",
		"hardware_platform": "x86_64",
		"hostname": "sd-k8s-master-1",
		"kernel_name": "Linux",
		"kernel_release": "5.14.0-503.38.1.el9_5.x86_64",
		"kernel_version": "#1 SMP PREEMPT_DYNAMIC Fri Apr 18 08:52:10 EDT 2025",
		"machine": "x86_64",
		"os": "GNU/Linux",
		"processor": "x86_64",
		"pythonV": "3.9.21"
	},
	"filesystem": [
		{"kb_size": "4096", "mounted_on": "/dev", "name": "devtmpfs"},
		{"kb_size": "103084600", "mounted_on": "/", "name": "/dev/sda1"},
		{"kb_size": "8055144", "mounted_on": "/dev/shm", "name": "tmpfs"}
	]
}`

func TestParseExtendInfo(t *testing.T) {
	info, err := ParseExtendInfo(sampleExtendInfo)
	if err != nil {
		t.Fatalf("ParseExtendInfo failed: %v", err)
	}

	// Verify CPU info
	if info.CPU.CPUCores != "4" {
		t.Errorf("Expected CPU cores '4', got '%s'", info.CPU.CPUCores)
	}
	if info.CPU.ModelName != "Intel(R) Xeon(R) Platinum 8378C CPU @ 2.80GHz" {
		t.Errorf("Unexpected CPU model name: %s", info.CPU.ModelName)
	}

	// Verify Memory info
	if info.Memory.Total != "16496934912" {
		t.Errorf("Expected memory total '16496934912', got '%s'", info.Memory.Total)
	}

	// Verify Network info
	if info.Network.IPAddress != "192.168.10.24" {
		t.Errorf("Expected IP address '192.168.10.24', got '%s'", info.Network.IPAddress)
	}
	if len(info.Network.Interfaces) != 2 {
		t.Errorf("Expected 2 network interfaces, got %d", len(info.Network.Interfaces))
	}

	// Verify Platform info
	if info.Platform.Hostname != "sd-k8s-master-1" {
		t.Errorf("Expected hostname 'sd-k8s-master-1', got '%s'", info.Platform.Hostname)
	}
	if info.Platform.OS != "GNU/Linux" {
		t.Errorf("Expected OS 'GNU/Linux', got '%s'", info.Platform.OS)
	}
	if info.Platform.KernelRelease != "5.14.0-503.38.1.el9_5.x86_64" {
		t.Errorf("Expected kernel release '5.14.0-503.38.1.el9_5.x86_64', got '%s'", info.Platform.KernelRelease)
	}

	// Verify Filesystem info
	if len(info.Filesystem) != 3 {
		t.Errorf("Expected 3 filesystems, got %d", len(info.Filesystem))
	}
}

func TestParseExtendInfoEmpty(t *testing.T) {
	info, err := ParseExtendInfo("")
	if err != nil {
		t.Fatalf("ParseExtendInfo should not fail on empty string: %v", err)
	}
	if info == nil {
		t.Error("ParseExtendInfo should return non-nil ExtendInfo for empty string")
	}
}

func TestParseExtendInfoInvalid(t *testing.T) {
	_, err := ParseExtendInfo("invalid json")
	if err == nil {
		t.Error("ParseExtendInfo should fail on invalid JSON")
	}
}

func TestCPUInfoGetCPUCores(t *testing.T) {
	tests := []struct {
		name     string
		cpuCores string
		expected int
	}{
		{"Valid cores", "4", 4},
		{"Zero cores", "0", 0},
		{"Invalid string", "invalid", 0},
		{"Empty string", "", 0},
		{"Large number", "128", 128},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CPUInfo{CPUCores: tt.cpuCores}
			if got := c.GetCPUCores(); got != tt.expected {
				t.Errorf("GetCPUCores() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestMemoryInfoGetTotal(t *testing.T) {
	tests := []struct {
		name     string
		total    string
		expected int64
	}{
		{"Valid memory", "16496934912", 16496934912},
		{"Zero memory", "0", 0},
		{"Invalid string", "invalid", 0},
		{"Empty string", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemoryInfo{Total: tt.total}
			if got := m.GetTotal(); got != tt.expected {
				t.Errorf("GetTotal() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestFilesystemInfoGetSizeBytes(t *testing.T) {
	tests := []struct {
		name     string
		kbSize   string
		expected int64
	}{
		{"Valid size", "1024", 1024 * 1024},
		{"Zero size", "0", 0},
		{"Invalid string", "invalid", 0},
		{"Large size", "103084600", 103084600 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FilesystemInfo{KBSize: tt.kbSize}
			if got := f.GetSizeBytes(); got != tt.expected {
				t.Errorf("GetSizeBytes() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestFilesystemInfoIsPhysicalDisk(t *testing.T) {
	tests := []struct {
		name      string
		fs        FilesystemInfo
		isPhysical bool
	}{
		{
			name:      "Physical disk /dev/sda1",
			fs:        FilesystemInfo{Name: "/dev/sda1", MountedOn: "/"},
			isPhysical: true,
		},
		{
			name:      "tmpfs /dev/shm",
			fs:        FilesystemInfo{Name: "tmpfs", MountedOn: "/dev/shm"},
			isPhysical: false,
		},
		{
			name:      "devtmpfs /dev",
			fs:        FilesystemInfo{Name: "devtmpfs", MountedOn: "/dev"},
			isPhysical: false,
		},
		{
			name:      "overlay containerd",
			fs:        FilesystemInfo{Name: "overlay", MountedOn: "/run/containerd/io.containerd.runtime.v2.task/k8s.io/abc123/rootfs"},
			isPhysical: false,
		},
		{
			name:      "tmpfs kubelet pods",
			fs:        FilesystemInfo{Name: "tmpfs", MountedOn: "/var/lib/kubelet/pods/uuid/volumes/kubernetes.io~projected/kube-api-access-xyz"},
			isPhysical: false,
		},
		{
			name:      "shm",
			fs:        FilesystemInfo{Name: "shm", MountedOn: "/run/containerd/io.containerd.grpc.v1.cri/sandboxes/abc/shm"},
			isPhysical: false,
		},
		{
			name:      "Physical disk /data",
			fs:        FilesystemInfo{Name: "/dev/sdb1", MountedOn: "/data"},
			isPhysical: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fs.IsPhysicalDisk(); got != tt.isPhysical {
				t.Errorf("IsPhysicalDisk() = %v, want %v", got, tt.isPhysical)
			}
		})
	}
}

func TestTargetDataToHostMeta(t *testing.T) {
	// Create sample target data with both direct fields and ExtendInfo
	// This matches the real N9E API response format
	target := &TargetData{
		Ident:      "sd-k8s-master-1",
		HostIP:     "192.168.10.24",
		OS:         "centos9",
		CPUNum:     4,
		ExtendInfo: sampleExtendInfo,
	}

	hostMeta, err := target.ToHostMeta()
	if err != nil {
		t.Fatalf("ToHostMeta failed: %v", err)
	}

	// Verify converted HostMeta
	if hostMeta.Ident != "sd-k8s-master-1" {
		t.Errorf("Expected ident 'sd-k8s-master-1', got '%s'", hostMeta.Ident)
	}
	if hostMeta.Hostname != "sd-k8s-master-1" {
		t.Errorf("Expected hostname 'sd-k8s-master-1', got '%s'", hostMeta.Hostname)
	}
	if hostMeta.IP != "192.168.10.24" {
		t.Errorf("Expected IP '192.168.10.24', got '%s'", hostMeta.IP)
	}
	// OS comes from direct field now (centos9), not from ExtendInfo (GNU/Linux)
	if hostMeta.OS != "centos9" {
		t.Errorf("Expected OS 'centos9', got '%s'", hostMeta.OS)
	}
	if hostMeta.KernelVersion != "5.14.0-503.38.1.el9_5.x86_64" {
		t.Errorf("Expected kernel version '5.14.0-503.38.1.el9_5.x86_64', got '%s'", hostMeta.KernelVersion)
	}
	if hostMeta.CPUCores != 4 {
		t.Errorf("Expected 4 CPU cores, got %d", hostMeta.CPUCores)
	}
	if hostMeta.CPUModel != "Intel(R) Xeon(R) Platinum 8378C CPU @ 2.80GHz" {
		t.Errorf("Unexpected CPU model: %s", hostMeta.CPUModel)
	}
	if hostMeta.MemoryTotal != 16496934912 {
		t.Errorf("Expected memory total 16496934912, got %d", hostMeta.MemoryTotal)
	}

	// Should only have physical disk (/ mounted on /dev/sda1)
	if len(hostMeta.DiskMounts) != 1 {
		t.Errorf("Expected 1 physical disk mount, got %d", len(hostMeta.DiskMounts))
	}
	if len(hostMeta.DiskMounts) > 0 && hostMeta.DiskMounts[0].Path != "/" {
		t.Errorf("Expected mount path '/', got '%s'", hostMeta.DiskMounts[0].Path)
	}
}

func TestTargetDataToHostMetaWithIdentClean(t *testing.T) {
	// Test with hostname@IP format
	target := &TargetData{
		Ident:      "server-1@192.168.1.100",
		ExtendInfo: `{"platform": {"hostname": ""}, "cpu": {}, "memory": {}, "network": {}, "filesystem": []}`,
	}

	hostMeta, err := target.ToHostMeta()
	if err != nil {
		t.Fatalf("ToHostMeta failed: %v", err)
	}

	// When ExtendInfo.Platform.Hostname is empty, should use cleaned ident
	if hostMeta.Hostname != "server-1" {
		t.Errorf("Expected hostname 'server-1' (cleaned from ident), got '%s'", hostMeta.Hostname)
	}
}

func TestTargetResponse(t *testing.T) {
	// Test parsing actual API response
	jsonResponse := `{"dat":{"ident":"sd-k8s-master-1","extend_info":"{}"},"err":""}`

	var resp TargetResponse
	if err := json.Unmarshal([]byte(jsonResponse), &resp); err != nil {
		t.Fatalf("Failed to unmarshal TargetResponse: %v", err)
	}

	if resp.Err != "" {
		t.Errorf("Expected empty error, got '%s'", resp.Err)
	}
	if resp.Dat.Ident != "sd-k8s-master-1" {
		t.Errorf("Expected ident 'sd-k8s-master-1', got '%s'", resp.Dat.Ident)
	}
}

func TestTargetsResponse(t *testing.T) {
	// Test parsing targets list response
	jsonResponse := `{"dat":{"list":[{"ident":"host1","extend_info":"{}"},{"ident":"host2","extend_info":"{}"}],"total":2},"err":""}`

	var resp TargetsResponse
	if err := json.Unmarshal([]byte(jsonResponse), &resp); err != nil {
		t.Fatalf("Failed to unmarshal TargetsResponse: %v", err)
	}

	if resp.Err != "" {
		t.Errorf("Expected empty error, got '%s'", resp.Err)
	}
	if len(resp.Dat.List) != 2 {
		t.Errorf("Expected 2 targets, got %d", len(resp.Dat.List))
	}
	if resp.Dat.Total != 2 {
		t.Errorf("Expected total 2, got %d", resp.Dat.Total)
	}
}
