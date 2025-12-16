// Package cmd implements CLI commands for the inspection tool.
package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"inspection-tool/internal/client/n9e"
	"inspection-tool/internal/client/vm"
	"inspection-tool/internal/config"
	"inspection-tool/internal/model"
	"inspection-tool/internal/report/excel"
	"inspection-tool/internal/report/html"
	"inspection-tool/internal/service"
)

// Command flags
var (
	outputDir        string   // Output directory for reports
	formats          []string // Output formats (excel, html)
	metricsPath      string   // Path to metrics definition file
	mysqlMetricsPath string   // Path to MySQL metrics definition file
	mysqlOnly        bool     // Run MySQL inspection only
	skipMySQL        bool     // Skip MySQL inspection
)

// runCmd represents the run command.
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "执行系统巡检",
	Long: `执行完整的系统巡检流程，包括：
1. 从夜莺（N9E）获取主机元信息
2. 从 VictoriaMetrics 查询监控指标
3. 执行 MySQL 数据库巡检（如果启用）
4. 根据配置的阈值评估告警级别
5. 生成 Excel 和 HTML 格式的巡检报告

示例:
  # 使用默认配置执行巡检（包含 Host 和 MySQL）
  inspect run -c config.yaml

  # 仅执行 MySQL 巡检
  inspect run -c config.yaml --mysql-only

  # 跳过 MySQL 巡检（仅执行 Host 巡检）
  inspect run -c config.yaml --skip-mysql

  # 指定输出格式和目录
  inspect run -c config.yaml -f excel,html -o ./reports

  # 使用自定义指标定义文件
  inspect run -c config.yaml -m custom_metrics.yaml --mysql-metrics custom_mysql_metrics.yaml`,
	Run: runInspection,
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Define command-specific flags
	runCmd.Flags().StringSliceVarP(&formats, "format", "f", nil, "输出格式 (excel,html)，可用逗号分隔多个")
	runCmd.Flags().StringVarP(&outputDir, "output", "o", "", "输出目录")
	runCmd.Flags().StringVarP(&metricsPath, "metrics", "m", "configs/metrics.yaml", "指标定义文件路径")

	// MySQL-specific flags
	runCmd.Flags().StringVar(&mysqlMetricsPath, "mysql-metrics", "configs/mysql-metrics.yaml", "MySQL 指标定义文件路径")
	runCmd.Flags().BoolVar(&mysqlOnly, "mysql-only", false, "仅执行 MySQL 巡检")
	runCmd.Flags().BoolVar(&skipMySQL, "skip-mysql", false, "跳过 MySQL 巡检")
}

// runInspection executes the complete inspection workflow.
func runInspection(cmd *cobra.Command, args []string) {
	// Print banner first
	printBanner()

	// Step 1: Load configuration
	configPath := GetConfigFile()
	fmt.Printf("📋 加载配置文件: %s\n", configPath)
	cfg, err := config.Load(configPath)
	if err != nil {
		// Use temporary console logger for config loading errors
		tmpLogger := setupLogger("error", "console")
		tmpLogger.Error().Err(err).Str("path", configPath).Msg("failed to load config")
		fmt.Fprintf(os.Stderr, "❌ 加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// Step 2: Initialize logger with configuration
	// Command line --log-level overrides config file setting
	logLevel := cfg.Logging.Level
	if GetLogLevel() != "info" { // If explicitly set via command line
		logLevel = GetLogLevel()
	}
	logger := setupLogger(logLevel, cfg.Logging.Format)
	logger.Debug().
		Str("config_path", configPath).
		Str("log_level", logLevel).
		Str("log_format", cfg.Logging.Format).
		Msg("configuration loaded successfully")

	// Step 2.5: Validate flag mutual exclusion
	if mysqlOnly && skipMySQL {
		fmt.Fprintf(os.Stderr, "❌ --mysql-only 和 --skip-mysql 不能同时使用\n")
		os.Exit(1)
	}

	// Determine execution mode
	runHostInspection := !mysqlOnly
	runMySQLInspection := !skipMySQL && cfg.MySQL.Enabled

	// If --mysql-only but MySQL is not enabled
	if mysqlOnly && !cfg.MySQL.Enabled {
		fmt.Fprintf(os.Stderr, "❌ MySQL 巡检未启用，请在配置文件中设置 mysql.enabled: true\n")
		os.Exit(1)
	}

	logger.Debug().
		Bool("run_host", runHostInspection).
		Bool("run_mysql", runMySQLInspection).
		Bool("mysql_enabled", cfg.MySQL.Enabled).
		Msg("execution mode determined")

	// Step 3: Load Host metrics definitions (if needed)
	var metrics []*model.MetricDefinition
	if runHostInspection {
		fmt.Printf("📊 加载主机指标定义: %s", metricsPath)
		metrics, err = config.LoadMetrics(metricsPath)
		if err != nil {
			logger.Error().Err(err).Str("path", metricsPath).Msg("failed to load metrics")
			fmt.Fprintf(os.Stderr, "\n❌ 加载指标定义失败: %v\n", err)
			os.Exit(1)
		}
		activeCount := config.CountActiveMetrics(metrics)
		fmt.Printf(" (%d 个活跃指标)\n", activeCount)
		logger.Debug().Int("active_metrics", activeCount).Int("total_metrics", len(metrics)).Msg("host metrics loaded")
	}

	// Step 3b: Load MySQL metrics definitions (if needed)
	var mysqlMetrics []*model.MySQLMetricDefinition
	if runMySQLInspection {
		fmt.Printf("📊 加载 MySQL 指标定义: %s", mysqlMetricsPath)
		mysqlMetrics, err = config.LoadMySQLMetrics(mysqlMetricsPath)
		if err != nil {
			logger.Error().Err(err).Str("path", mysqlMetricsPath).Msg("failed to load MySQL metrics")
			fmt.Fprintf(os.Stderr, "\n❌ 加载 MySQL 指标定义失败: %v\n", err)
			os.Exit(1)
		}
		mysqlActiveCount := config.CountActiveMySQLMetrics(mysqlMetrics)
		fmt.Printf(" (%d 个活跃指标)\n", mysqlActiveCount)
		logger.Debug().Int("active_metrics", mysqlActiveCount).Int("total_metrics", len(mysqlMetrics)).Msg("MySQL metrics loaded")
	}

	// Step 4: Determine output settings
	outputFormats := resolveFormats(cfg)
	outputPath := resolveOutputDir(cfg)

	// Ensure output directory exists
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		logger.Error().Err(err).Str("path", outputPath).Msg("failed to create output directory")
		fmt.Fprintf(os.Stderr, "❌ 创建输出目录失败: %v\n", err)
		os.Exit(1)
	}

	// Step 5: Display data source info
	fmt.Println("🔗 连接数据源...")
	if runHostInspection {
		fmt.Printf("   - 夜莺 N9E: %s\n", cfg.Datasources.N9E.Endpoint)
	}
	fmt.Printf("   - VictoriaMetrics: %s\n", cfg.Datasources.VictoriaMetrics.Endpoint)
	fmt.Println()
	logger.Info().
		Str("n9e_endpoint", cfg.Datasources.N9E.Endpoint).
		Str("vm_endpoint", cfg.Datasources.VictoriaMetrics.Endpoint).
		Msg("connecting to data sources")

	// Step 6: Create clients
	var n9eClient *n9e.Client
	if runHostInspection {
		n9eClient = n9e.NewClient(&cfg.Datasources.N9E, &cfg.HTTP.Retry, logger)
	}
	vmClient := vm.NewClient(&cfg.Datasources.VictoriaMetrics, &cfg.HTTP.Retry, logger)
	logger.Debug().Msg("API clients created")

	// Step 7: Create Host services (if needed)
	var inspector *service.Inspector
	if runHostInspection {
		collector := service.NewCollector(cfg, n9eClient, vmClient, metrics, logger)
		evaluator := service.NewEvaluator(&cfg.Thresholds, metrics, logger)
		inspector, err = service.NewInspector(cfg, collector, evaluator, logger, service.WithVersion(Version))
		if err != nil {
			logger.Error().Err(err).Msg("failed to create inspector")
			fmt.Fprintf(os.Stderr, "❌ 创建巡检器失败: %v\n", err)
			os.Exit(1)
		}
		logger.Debug().Msg("host services initialized")
	}

	// Step 7b: Create MySQL services (if needed)
	var mysqlInspector *service.MySQLInspector
	if runMySQLInspection {
		mysqlCollector := service.NewMySQLCollector(&cfg.MySQL, vmClient, mysqlMetrics, logger)
		mysqlEvaluator := service.NewMySQLEvaluator(&cfg.MySQL.Thresholds, mysqlMetrics, logger)
		mysqlInspector, err = service.NewMySQLInspector(cfg, mysqlCollector, mysqlEvaluator, logger,
			service.WithMySQLVersion(Version))
		if err != nil {
			logger.Error().Err(err).Msg("failed to create MySQL inspector")
			fmt.Fprintf(os.Stderr, "❌ 创建 MySQL 巡检器失败: %v\n", err)
			os.Exit(1)
		}
		logger.Debug().Msg("MySQL services initialized")
	}

	// Step 8: Execute inspection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	startTime := time.Now()

	var hostResult *model.InspectionResult
	var mysqlResult *model.MySQLInspectionResults

	// Execute Host inspection
	if runHostInspection {
		fmt.Println("⏳ 开始主机巡检...")
		hostResult, err = inspector.Run(ctx)
		if err != nil {
			logger.Error().Err(err).Msg("host inspection failed")
			fmt.Fprintf(os.Stderr, "❌ 主机巡检执行失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\n📊 主机巡检完成！\n")
		printSummary(hostResult)
	}

	// Execute MySQL inspection
	if runMySQLInspection {
		fmt.Println("\n⏳ 开始 MySQL 巡检...")
		mysqlResult, err = mysqlInspector.Inspect(ctx)
		if err != nil {
			logger.Error().Err(err).Msg("MySQL inspection failed")
			fmt.Fprintf(os.Stderr, "❌ MySQL 巡检执行失败: %v\n", err)
			// Don't exit, continue to generate Host report if available
			if hostResult == nil {
				os.Exit(1)
			}
		} else {
			fmt.Printf("\n📊 MySQL 巡检完成！\n")
			printMySQLSummary(mysqlResult)
		}
	}

	fmt.Printf("\n⏱️  总耗时 %.1fs\n", time.Since(startTime).Seconds())

	// Step 9: Generate reports
	fmt.Println("\n📄 生成报告:")
	logger.Info().
		Strs("formats", outputFormats).
		Str("output_dir", outputPath).
		Msg("starting report generation")

	// Load timezone for report generation
	var timezone *time.Location
	if inspector != nil {
		timezone = inspector.GetTimezone()
	} else if mysqlInspector != nil {
		timezone = mysqlInspector.GetTimezone()
	} else {
		timezone, _ = time.LoadLocation("Asia/Shanghai")
	}

	// Generate filename base
	filenameBase := generateFilename(cfg.Report.FilenameTemplate, timezone)

	// Generate reports for each format
	for _, format := range outputFormats {
		ext := "." + format
		if format == "excel" {
			ext = ".xlsx"
		}
		reportPath := filepath.Join(outputPath, filenameBase+ext)

		var genErr error
		switch format {
		case "excel":
			genErr = generateCombinedExcel(hostResult, mysqlResult, reportPath, timezone, logger)
		case "html":
			genErr = generateCombinedHTML(hostResult, mysqlResult, reportPath, timezone, cfg.Report.HTMLTemplate, logger)
		default:
			logger.Error().Str("format", format).Msg("unsupported format")
			fmt.Fprintf(os.Stderr, "   ❌ 不支持的格式: %s\n", format)
			continue
		}

		if genErr != nil {
			logger.Error().Err(genErr).Str("format", format).Str("path", reportPath).Msg("failed to generate report")
			fmt.Fprintf(os.Stderr, "   ❌ %s 报告生成失败: %v\n", format, genErr)
			continue
		}

		logger.Info().Str("format", format).Str("path", reportPath).Msg("report generated successfully")
		fmt.Printf("   ✅ %s\n", reportPath)
	}

	// Exit with appropriate code based on inspection results
	exitCode := 0
	if hostResult != nil {
		if hostResult.Summary.CriticalHosts > 0 {
			exitCode = 2
		} else if hostResult.Summary.WarningHosts > 0 && exitCode < 1 {
			exitCode = 1
		}
	}
	if mysqlResult != nil && mysqlResult.Summary != nil {
		if mysqlResult.Summary.CriticalInstances > 0 {
			exitCode = 2
		} else if mysqlResult.Summary.WarningInstances > 0 && exitCode < 1 {
			exitCode = 1
		}
	}
	if exitCode > 0 {
		os.Exit(exitCode)
	}
}

// setupLogger creates a zerolog logger with the specified level and format.
// It sets the timezone to Asia/Shanghai for all log timestamps.
func setupLogger(level string, format string) zerolog.Logger {
	// Set log level
	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(logLevel)

	// Load Asia/Shanghai timezone for log timestamps
	tz, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		tz = time.Local
	}

	// Set timezone for all timestamps
	zerolog.TimestampFunc = func() time.Time {
		return time.Now().In(tz)
	}

	// Select output format based on configuration
	var output io.Writer
	if format == "json" {
		// JSON format - structured logging for log aggregation systems
		output = os.Stderr
	} else {
		// Console format - human-readable output for development
		output = zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "15:04:05",
			NoColor:    false,
		}
	}

	return zerolog.New(output).With().Timestamp().Logger()
}

// printBanner prints the application banner.
func printBanner() {
	fmt.Printf("🔍 系统巡检工具 %s\n", Version)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// printSummary prints the inspection result summary.
func printSummary(result *model.InspectionResult) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if result.Summary != nil {
		fmt.Printf("   主机总数: %d\n", result.Summary.TotalHosts)
		fmt.Printf("   正常主机: %d\n", result.Summary.NormalHosts)
		fmt.Printf("   警告主机: %d\n", result.Summary.WarningHosts)
		fmt.Printf("   严重主机: %d\n", result.Summary.CriticalHosts)
		fmt.Printf("   失败主机: %d\n", result.Summary.FailedHosts)
	}
	fmt.Println()
	if result.AlertSummary != nil {
		fmt.Printf("   告警总数: %d\n", result.AlertSummary.TotalAlerts)
		fmt.Printf("   警告级别: %d\n", result.AlertSummary.WarningCount)
		fmt.Printf("   严重级别: %d\n", result.AlertSummary.CriticalCount)
	}
}

// resolveFormats determines the output formats to use.
// Command line flags take precedence over config file.
func resolveFormats(cfg *config.Config) []string {
	if len(formats) > 0 {
		return formats
	}
	if len(cfg.Report.Formats) > 0 {
		return cfg.Report.Formats
	}
	return []string{"excel", "html"} // default
}

// resolveOutputDir determines the output directory to use.
// Command line flags take precedence over config file.
func resolveOutputDir(cfg *config.Config) string {
	if outputDir != "" {
		return outputDir
	}
	if cfg.Report.OutputDir != "" {
		return cfg.Report.OutputDir
	}
	return "./reports" // default
}

// generateFilename creates a filename from the template.
// Supports {{.Date}} placeholder for current date.
func generateFilename(template string, tz *time.Location) string {
	if template == "" {
		template = "inspection_report_{{.Date}}"
	}

	// Get current date in the configured timezone
	now := time.Now().In(tz)
	dateStr := now.Format("2006-01-02")

	// Replace placeholders
	filename := strings.ReplaceAll(template, "{{.Date}}", dateStr)
	filename = strings.ReplaceAll(filename, "{{ .Date }}", dateStr)

	return filename
}

// printMySQLSummary prints the MySQL inspection result summary.
func printMySQLSummary(result *model.MySQLInspectionResults) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if result.Summary != nil {
		fmt.Printf("   MySQL 实例总数: %d\n", result.Summary.TotalInstances)
		fmt.Printf("   正常实例: %d\n", result.Summary.NormalInstances)
		fmt.Printf("   警告实例: %d\n", result.Summary.WarningInstances)
		fmt.Printf("   严重实例: %d\n", result.Summary.CriticalInstances)
		fmt.Printf("   失败实例: %d\n", result.Summary.FailedInstances)
	}
	fmt.Println()
	if result.AlertSummary != nil {
		fmt.Printf("   MySQL 告警总数: %d\n", result.AlertSummary.TotalAlerts)
		fmt.Printf("   警告级别: %d\n", result.AlertSummary.WarningCount)
		fmt.Printf("   严重级别: %d\n", result.AlertSummary.CriticalCount)
	}
}

// generateCombinedExcel creates Excel report with Host and MySQL data in same file.
func generateCombinedExcel(hostResult *model.InspectionResult, mysqlResult *model.MySQLInspectionResults, outputPath string, timezone *time.Location, logger zerolog.Logger) error {
	w := excel.NewWriter(timezone)

	// Only MySQL mode
	if hostResult == nil && mysqlResult != nil {
		return w.WriteMySQLInspection(mysqlResult, outputPath)
	}

	// Only Host mode
	if hostResult != nil && mysqlResult == nil {
		return w.Write(hostResult, outputPath)
	}

	// Combined mode: write Host first, then append MySQL
	if hostResult != nil {
		if err := w.Write(hostResult, outputPath); err != nil {
			return fmt.Errorf("failed to write host report: %w", err)
		}
	}
	if mysqlResult != nil {
		if err := w.AppendMySQLInspection(mysqlResult, outputPath); err != nil {
			return fmt.Errorf("failed to append MySQL report: %w", err)
		}
	}

	logger.Debug().
		Bool("has_host", hostResult != nil).
		Bool("has_mysql", mysqlResult != nil).
		Str("path", outputPath).
		Msg("combined Excel report generated")

	return nil
}

// generateCombinedHTML creates HTML report with Host and MySQL data.
func generateCombinedHTML(hostResult *model.InspectionResult, mysqlResult *model.MySQLInspectionResults, outputPath string, timezone *time.Location, templatePath string, logger zerolog.Logger) error {
	w := html.NewWriter(timezone, templatePath)

	// Only MySQL mode
	if hostResult == nil && mysqlResult != nil {
		return w.WriteMySQLInspection(mysqlResult, outputPath)
	}

	// Only Host mode
	if hostResult != nil && mysqlResult == nil {
		return w.Write(hostResult, outputPath)
	}

	// Combined mode
	if err := w.WriteCombined(hostResult, mysqlResult, outputPath); err != nil {
		return fmt.Errorf("failed to write combined HTML report: %w", err)
	}

	logger.Debug().
		Bool("has_host", hostResult != nil).
		Bool("has_mysql", mysqlResult != nil).
		Str("path", outputPath).
		Msg("combined HTML report generated")

	return nil
}
