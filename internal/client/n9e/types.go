// Package n9e provides a client for the N9E (Nightingale) API.
package n9e

import (
	"encoding/json"
	"strconv"

	"inspection-tool/internal/model"
)

// TargetResponse represents the API response from N9E /api/n9e/target/:ident endpoint.
type TargetResponse struct {
	Dat TargetData `json:"dat"` // 响应数据
	Err string     `json:"err"` // 错误信息（空字符串表示成功）
}

// TargetsResponse represents the API response from N9E /api/n9e/targets endpoint.
type TargetsResponse struct {
	Dat []TargetData `json:"dat"` // 主机列表
	Err string       `json:"err"` // 错误信息
}

// TargetData contains basic target information.
type TargetData struct {
	Ident      string `json:"ident"`       // 主机标识符（可能为 hostname 或 hostname@IP 格式）
	ExtendInfo string `json:"extend_info"` // JSON 字符串，需要二次解析
}

// ExtendInfo contains detailed host information parsed from the extend_info JSON string.
type ExtendInfo struct {
	CPU        CPUInfo           `json:"cpu"`        // CPU 信息
	Memory     MemoryInfo        `json:"memory"`     // 内存信息
	Network    NetworkInfo       `json:"network"`    // 网络信息
	Platform   PlatformInfo      `json:"platform"`   // 平台/系统信息
	Filesystem []FilesystemInfo  `json:"filesystem"` // 文件系统信息
}

// CPUInfo contains CPU hardware information.
type CPUInfo struct {
	CacheSize            string `json:"cache_size"`              // 缓存大小（如 "16384 KB"）
	CPUCores             string `json:"cpu_cores"`               // CPU 物理核心数
	CPULogicalProcessors string `json:"cpu_logical_processors"`  // CPU 逻辑处理器数
	Family               string `json:"family"`                  // CPU 家族
	MHz                  string `json:"mhz"`                     // CPU 频率（MHz）
	Model                string `json:"model"`                   // CPU 型号代码
	ModelName            string `json:"model_name"`              // CPU 型号名称
	Stepping             string `json:"stepping"`                // CPU stepping
	VendorID             string `json:"vendor_id"`               // CPU 厂商
}

// GetCPUCores returns the number of CPU cores as an integer.
// Returns 0 if parsing fails.
func (c *CPUInfo) GetCPUCores() int {
	cores, err := strconv.Atoi(c.CPUCores)
	if err != nil {
		return 0
	}
	return cores
}

// MemoryInfo contains memory information.
type MemoryInfo struct {
	SwapTotal string `json:"swap_total"` // 交换空间总量（bytes，字符串格式）
	Total     string `json:"total"`      // 内存总量（bytes，字符串格式）
}

// GetTotal returns the total memory in bytes as int64.
// Returns 0 if parsing fails.
func (m *MemoryInfo) GetTotal() int64 {
	total, err := strconv.ParseInt(m.Total, 10, 64)
	if err != nil {
		return 0
	}
	return total
}

// NetworkInfo contains network information.
type NetworkInfo struct {
	Interfaces []NetworkInterface `json:"interfaces"` // 网络接口列表
	IPAddress  string             `json:"ipaddress"`  // 主 IP 地址
	MACAddress string             `json:"macaddress"` // 主 MAC 地址
}

// NetworkInterface represents a single network interface.
type NetworkInterface struct {
	IPv4        string `json:"ipv4"`         // IPv4 地址
	IPv4Network string `json:"ipv4-network"` // IPv4 网段
	MACAddress  string `json:"macaddress"`   // MAC 地址
	Name        string `json:"name"`         // 接口名称（如 eth0）
}

// PlatformInfo contains operating system and platform information.
type PlatformInfo struct {
	GOARCH           string `json:"GOARCH"`            // Go 架构
	GOOS             string `json:"GOOS"`              // Go 操作系统
	GoVersion        string `json:"goV"`               // Go 版本
	HardwarePlatform string `json:"hardware_platform"` // 硬件平台
	Hostname         string `json:"hostname"`          // 主机名
	KernelName       string `json:"kernel_name"`       // 内核名称（如 Linux）
	KernelRelease    string `json:"kernel_release"`    // 内核版本（如 5.14.0-503.38.1.el9_5.x86_64）
	KernelVersion    string `json:"kernel_version"`    // 完整内核版本字符串
	Machine          string `json:"machine"`           // 机器架构
	OS               string `json:"os"`                // 操作系统类型（如 GNU/Linux）
	Processor        string `json:"processor"`         // 处理器架构
	PythonVersion    string `json:"pythonV"`           // Python 版本
}

// FilesystemInfo contains information about a single filesystem mount.
type FilesystemInfo struct {
	KBSize    string `json:"kb_size"`    // 文件系统大小（KB，字符串格式）
	MountedOn string `json:"mounted_on"` // 挂载点路径
	Name      string `json:"name"`       // 文件系统名称/设备名
}

// GetSizeBytes returns the filesystem size in bytes.
// Returns 0 if parsing fails.
func (f *FilesystemInfo) GetSizeBytes() int64 {
	kb, err := strconv.ParseInt(f.KBSize, 10, 64)
	if err != nil {
		return 0
	}
	return kb * 1024
}

// IsPhysicalDisk returns true if this filesystem appears to be a physical disk.
// It excludes tmpfs, overlay, shm, and other virtual filesystems.
func (f *FilesystemInfo) IsPhysicalDisk() bool {
	// 排除的文件系统类型
	excludedNames := []string{"tmpfs", "overlay", "shm", "devtmpfs"}
	for _, excluded := range excludedNames {
		if f.Name == excluded {
			return false
		}
	}

	// 排除的挂载点模式
	excludedMounts := []string{
		"/dev",
		"/dev/shm",
		"/run",
		"/sys",
		"/proc",
	}
	for _, excluded := range excludedMounts {
		if f.MountedOn == excluded {
			return false
		}
	}

	// 排除 containerd 相关的挂载点
	if len(f.MountedOn) > 20 && (contains(f.MountedOn, "/run/containerd/") ||
		contains(f.MountedOn, "/var/lib/kubelet/pods/")) {
		return false
	}

	return true
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ParseExtendInfo parses the extend_info JSON string into an ExtendInfo struct.
// Returns an error if parsing fails.
func ParseExtendInfo(extendInfoStr string) (*ExtendInfo, error) {
	if extendInfoStr == "" {
		return &ExtendInfo{}, nil
	}

	var info ExtendInfo
	if err := json.Unmarshal([]byte(extendInfoStr), &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// ToHostMeta converts N9E target data to the internal HostMeta model.
// This is a convenience method that combines TargetData and ExtendInfo.
func (t *TargetData) ToHostMeta() (*model.HostMeta, error) {
	extInfo, err := ParseExtendInfo(t.ExtendInfo)
	if err != nil {
		return nil, err
	}

	// 清理 ident 获取主机名
	hostname := model.CleanIdent(t.Ident)

	// 如果 ExtendInfo 中有更准确的主机名，优先使用
	if extInfo.Platform.Hostname != "" {
		hostname = extInfo.Platform.Hostname
	}

	// 收集物理磁盘挂载点
	var diskMounts []model.DiskMountInfo
	for _, fs := range extInfo.Filesystem {
		if fs.IsPhysicalDisk() {
			diskMounts = append(diskMounts, model.DiskMountInfo{
				Path:  fs.MountedOn,
				Total: fs.GetSizeBytes(),
				// Free 和 UsedPercent 将从 VictoriaMetrics 获取
			})
		}
	}

	return &model.HostMeta{
		Ident:         t.Ident,
		Hostname:      hostname,
		IP:            extInfo.Network.IPAddress,
		OS:            extInfo.Platform.OS,
		OSVersion:     "", // N9E 元信息中没有直接的版本号，OS 字段已包含类型
		KernelVersion: extInfo.Platform.KernelRelease,
		CPUCores:      extInfo.CPU.GetCPUCores(),
		CPUModel:      extInfo.CPU.ModelName,
		MemoryTotal:   extInfo.Memory.GetTotal(),
		DiskMounts:    diskMounts,
	}, nil
}
