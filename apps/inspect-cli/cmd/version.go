// Package cmd provides CLI commands for the inspection tool.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Long:  "显示工具的版本号、构建时间、Git 提交哈希、Go 版本和运行平台信息。",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(GetVersionInfo())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
