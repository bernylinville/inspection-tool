// Package cmd provides CLI commands for the inspection tool.
package cmd

import (
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

// Version information, injected at build time via -ldflags.
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// Global flags
var (
	cfgFile  string // Config file path
	logLevel string // Log level
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "inspect",
	Short: "系统巡检工具 - 基于监控数据的无侵入式巡检",
	Long: `系统巡检工具通过调用夜莺（N9E）和 VictoriaMetrics API
查询监控数据，生成 Excel 和 HTML 格式的巡检报告。

数据流: Categraf → 夜莺 N9E → VictoriaMetrics → 本工具 → Excel/HTML 报告

主要功能:
  - 从夜莺获取主机元信息（主机名、IP、操作系统、CPU 核心数等）
  - 从 VictoriaMetrics 查询指标数据（CPU、内存、磁盘、负载等）
  - 根据配置的阈值评估告警级别
  - 生成 Excel 和 HTML 格式的巡检报告`,
	Version: Version,
	// Run displays help when called without any subcommands
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// init initializes the root command and its flags.
func init() {
	// Global flags available to all commands
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "config.yaml", "配置文件路径")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "日志级别 (debug, info, warn, error)")

	// Customize version template
	rootCmd.SetVersionTemplate(`{{printf "%s\n" .Version}}`)
}

// GetConfigFile returns the config file path from command line flag.
func GetConfigFile() string {
	return cfgFile
}

// GetLogLevel returns the log level from command line flag.
func GetLogLevel() string {
	return logLevel
}

// GetVersionInfo returns formatted version information.
func GetVersionInfo() string {
	return Version + "\n" +
		"Build Time: " + BuildTime + "\n" +
		"Git Commit: " + GitCommit + "\n" +
		"Go Version: " + runtime.Version() + "\n" +
		"OS/Arch: " + runtime.GOOS + "/" + runtime.GOARCH
}
