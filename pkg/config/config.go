package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// VMConfig represents configuration for VictoriaMetrics.
type VMConfig struct {
	Address string        `yaml:"address"`
	Timeout time.Duration `yaml:"timeout"`
}

// ReportConfig represents the configuration for generating reports.
type ReportConfig struct {
	OutputDir string
	Format    string
}

// Config holds the full configuration structure.
type Config struct {
	VictoriaMetrics VMConfig `yaml:"victoria_metrics"`
	Report          ReportConfig
	Labels          []string
	Projects        []string
	Concurrency     int
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		VictoriaMetrics: VMConfig{
			Address: "http://127.0.0.1:8428",
			Timeout: 30 * time.Second,
		},
		Report: ReportConfig{
			OutputDir: "reports",
			Format:    "xlsx",
		},
		Labels:      []string{},
		Projects:    []string{},
		Concurrency: 10,
	}
}

// LoadConfig 从文件加载配置
func LoadConfig(configFile string) (*Config, error) {
	config := DefaultConfig()

	// 如果未指定配置文件，尝试默认位置
	if configFile == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			configFile = filepath.Join(homeDir, ".inspection-tool.yaml")
		}
	}

	// 如果配置文件存在，则读取
	if configFile != "" {
		if _, err := os.Stat(configFile); err == nil {
			data, err := os.ReadFile(configFile)
			if err != nil {
				return nil, fmt.Errorf("读取配置文件失败: %v", err)
			}

			if err := yaml.Unmarshal(data, config); err != nil {
				return nil, fmt.Errorf("解析配置文件失败: %v", err)
			}
		}
	}

	// 从环境变量覆盖配置
	if addr := os.Getenv("INSPECTION_VM_ADDRESS"); addr != "" {
		config.VictoriaMetrics.Address = addr
	}
	if outDir := os.Getenv("INSPECTION_OUTPUT_DIR"); outDir != "" {
		config.Report.OutputDir = outDir
	}
	if format := os.Getenv("INSPECTION_REPORT_FORMAT"); format != "" {
		config.Report.Format = format
	}

	return config, nil
}

// Validate 验证配置是否有效
func (c *Config) Validate() error {
	if c.VictoriaMetrics.Address == "" {
		return fmt.Errorf("VictoriaMetrics 地址不能为空")
	}
	if c.VictoriaMetrics.Timeout <= 0 {
		return fmt.Errorf("超时时间必须大于0")
	}
	if c.Concurrency <= 0 {
		return fmt.Errorf("并发数必须大于0")
	}
	if c.Report.Format != "xlsx" && c.Report.Format != "pdf" {
		return fmt.Errorf("不支持的报告格式: %s", c.Report.Format)
	}
	return nil
}
