// Package cmd implements CLI commands for the inspection tool.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"inspection-tool/internal/config"
)

// validateCmd represents the validate command.
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "验证配置文件",
	Long:  "加载并验证配置文件，检查格式、必填字段、数值范围和业务逻辑约束。",
	Run:   runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

// runValidate executes the validate command logic.
func runValidate(cmd *cobra.Command, args []string) {
	configPath := GetConfigFile()

	// Load and validate configuration (Load internally calls Validate)
	_, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 配置验证失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ 配置文件验证通过: %s\n", configPath)
}
