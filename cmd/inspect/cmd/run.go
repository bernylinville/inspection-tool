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
	redisMetricsPath string   // Path to Redis metrics definition file
	redisOnly        bool     // Run Redis inspection only
	skipRedis        bool     // Skip Redis inspection
	nginxMetricsPath string   // Path to Nginx metrics definition file
	nginxOnly        bool     // Run Nginx inspection only
	skipNginx        bool     // Skip Nginx inspection
	tomcatMetricsPath string  // Path to Tomcat metrics definition file
	tomcatOnly        bool    // Run Tomcat inspection only
	skipTomcat        bool    // Skip Tomcat inspection
	excelTemplatePath string  // Path to Excel template file (optional)
)

const (
	// Default Excel template path
	defaultExcelTemplate = "templates/excel/inspection_template.xlsx"
)

// runCmd represents the run command.
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "执行系统巡检",
	Long: `执行完整的系统巡检流程，包括：
1. 从夜莺（N9E）获取主机元信息
2. 从 VictoriaMetrics 查询监控指标
3. 执行 MySQL 数据库巡检（如果启用）
4. 执行 Redis 集群巡检（如果启用）
5. 执行 Nginx/OpenResty 巡检（如果启用）
6. 执行 Tomcat 应用巡检（如果启用）
7. 根据配置的阈值评估告警级别
8. 生成 Excel 和 HTML 格式的巡检报告

示例:
  # 使用默认配置执行巡检（包含 Host、MySQL、Redis、Nginx 和 Tomcat）
  inspect run -c config.yaml

  # 仅执行 MySQL 巡检
  inspect run -c config.yaml --mysql-only

  # 仅执行 Redis 巡检
  inspect run -c config.yaml --redis-only

  # 仅执行 Nginx 巡检
  inspect run -c config.yaml --nginx-only

  # 仅执行 Tomcat 巡检
  inspect run -c config.yaml --tomcat-only

  # 跳过 MySQL 巡检
  inspect run -c config.yaml --skip-mysql

  # 跳过 Redis 巡检
  inspect run -c config.yaml --skip-redis

  # 跳过 Nginx 巡检
  inspect run -c config.yaml --skip-nginx

  # 跳过 Tomcat 巡检
  inspect run -c config.yaml --skip-tomcat

  # 仅执行 Host 巡检（跳过 MySQL、Redis、Nginx 和 Tomcat）
  inspect run -c config.yaml --skip-mysql --skip-redis --skip-nginx --skip-tomcat

  # 指定输出格式和目录
  inspect run -c config.yaml -f excel,html -o ./reports

  # 使用自定义指标定义文件
  inspect run -c config.yaml -m custom_metrics.yaml --mysql-metrics custom_mysql_metrics.yaml --redis-metrics custom_redis_metrics.yaml --nginx-metrics custom_nginx_metrics.yaml --tomcat-metrics custom_tomcat_metrics.yaml`,
	Run: runInspection,
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Define command-specific flags
	runCmd.Flags().StringSliceVarP(&formats, "format", "f", nil, "输出格式 (excel,html)，可用逗号分隔多个")
	runCmd.Flags().StringVarP(&outputDir, "output", "o", "", "输出目录")
	runCmd.Flags().StringVarP(&metricsPath, "metrics", "m", "configs/metrics.yaml", "指标定义文件路径")
	runCmd.Flags().StringVar(&excelTemplatePath, "excel-template", "", "Excel 模板文件路径（默认使用 templates/excel/inspection_template.xlsx）")

	// MySQL-specific flags
	runCmd.Flags().StringVar(&mysqlMetricsPath, "mysql-metrics", "configs/mysql-metrics.yaml", "MySQL 指标定义文件路径")
	runCmd.Flags().BoolVar(&mysqlOnly, "mysql-only", false, "仅执行 MySQL 巡检")
	runCmd.Flags().BoolVar(&skipMySQL, "skip-mysql", false, "跳过 MySQL 巡检")

	// Redis-specific flags
	runCmd.Flags().StringVar(&redisMetricsPath, "redis-metrics", "configs/redis-metrics.yaml", "Redis 指标定义文件路径")
	runCmd.Flags().BoolVar(&redisOnly, "redis-only", false, "仅执行 Redis 巡检")
	runCmd.Flags().BoolVar(&skipRedis, "skip-redis", false, "跳过 Redis 巡检")

	// Nginx-specific flags
	runCmd.Flags().StringVar(&nginxMetricsPath, "nginx-metrics", "configs/nginx-metrics.yaml", "Nginx 指标定义文件路径")
	runCmd.Flags().BoolVar(&nginxOnly, "nginx-only", false, "仅执行 Nginx 巡检")
	runCmd.Flags().BoolVar(&skipNginx, "skip-nginx", false, "跳过 Nginx 巡检")

	// Tomcat-specific flags
	runCmd.Flags().StringVar(&tomcatMetricsPath, "tomcat-metrics", "configs/tomcat-metrics.yaml", "Tomcat 指标定义文件路径")
	runCmd.Flags().BoolVar(&tomcatOnly, "tomcat-only", false, "仅执行 Tomcat 巡检")
	runCmd.Flags().BoolVar(&skipTomcat, "skip-tomcat", false, "跳过 Tomcat 巡检")
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
	if redisOnly && skipRedis {
		fmt.Fprintf(os.Stderr, "❌ --redis-only 和 --skip-redis 不能同时使用\n")
		os.Exit(1)
	}
	if redisOnly && mysqlOnly {
		fmt.Fprintf(os.Stderr, "❌ --redis-only 和 --mysql-only 不能同时使用\n")
		os.Exit(1)
	}
	if nginxOnly && skipNginx {
		fmt.Fprintf(os.Stderr, "❌ --nginx-only 和 --skip-nginx 不能同时使用\n")
		os.Exit(1)
	}
	if nginxOnly && mysqlOnly {
		fmt.Fprintf(os.Stderr, "❌ --nginx-only 和 --mysql-only 不能同时使用\n")
		os.Exit(1)
	}
	if nginxOnly && redisOnly {
		fmt.Fprintf(os.Stderr, "❌ --nginx-only 和 --redis-only 不能同时使用\n")
		os.Exit(1)
	}

	// Tomcat flag validation
	if tomcatOnly && skipTomcat {
		fmt.Fprintf(os.Stderr, "❌ --tomcat-only 和 --skip-tomcat 不能同时使用\n")
		os.Exit(1)
	}
	if tomcatOnly && mysqlOnly {
		fmt.Fprintf(os.Stderr, "❌ --tomcat-only 和 --mysql-only 不能同时使用\n")
		os.Exit(1)
	}
	if tomcatOnly && redisOnly {
		fmt.Fprintf(os.Stderr, "❌ --tomcat-only 和 --redis-only 不能同时使用\n")
		os.Exit(1)
	}
	if tomcatOnly && nginxOnly {
		fmt.Fprintf(os.Stderr, "❌ --tomcat-only 和 --nginx-only 不能同时使用\n")
		os.Exit(1)
	}

	// Determine execution mode
	runHostInspection := !mysqlOnly && !redisOnly && !nginxOnly && !tomcatOnly
	runMySQLInspection := !skipMySQL && !redisOnly && !nginxOnly && !tomcatOnly && cfg.MySQL.Enabled
	runRedisInspection := !skipRedis && !mysqlOnly && !nginxOnly && !tomcatOnly && cfg.Redis.Enabled
	runNginxInspection := !skipNginx && !mysqlOnly && !redisOnly && !tomcatOnly && cfg.Nginx.Enabled
	runTomcatInspection := !skipTomcat && !mysqlOnly && !redisOnly && !nginxOnly && cfg.Tomcat.Enabled

	// If --mysql-only but MySQL is not enabled
	if mysqlOnly && !cfg.MySQL.Enabled {
		fmt.Fprintf(os.Stderr, "❌ MySQL 巡检未启用，请在配置文件中设置 mysql.enabled: true\n")
		os.Exit(1)
	}

	// If --redis-only but Redis is not enabled
	if redisOnly && !cfg.Redis.Enabled {
		fmt.Fprintf(os.Stderr, "❌ Redis 巡检未启用，请在配置文件中设置 redis.enabled: true\n")
		os.Exit(1)
	}

	// If --nginx-only but Nginx is not enabled
	if nginxOnly && !cfg.Nginx.Enabled {
		fmt.Fprintf(os.Stderr, "❌ Nginx 巡检未启用，请在配置文件中设置 nginx.enabled: true\n")
		os.Exit(1)
	}

	// If --tomcat-only but Tomcat is not enabled
	if tomcatOnly && !cfg.Tomcat.Enabled {
		fmt.Fprintf(os.Stderr, "❌ Tomcat 巡检未启用，请在配置文件中设置 tomcat.enabled: true\n")
		os.Exit(1)
	}

	logger.Debug().
		Bool("run_host", runHostInspection).
		Bool("run_mysql", runMySQLInspection).
		Bool("run_redis", runRedisInspection).
		Bool("run_nginx", runNginxInspection).
		Bool("run_tomcat", runTomcatInspection).
		Bool("mysql_enabled", cfg.MySQL.Enabled).
		Bool("redis_enabled", cfg.Redis.Enabled).
		Bool("nginx_enabled", cfg.Nginx.Enabled).
		Bool("tomcat_enabled", cfg.Tomcat.Enabled).
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

	// Step 3c: Load Redis metrics definitions (if needed)
	var redisMetrics []*model.RedisMetricDefinition
	if runRedisInspection {
		fmt.Printf("📊 加载 Redis 指标定义: %s", redisMetricsPath)
		redisMetrics, err = config.LoadRedisMetrics(redisMetricsPath)
		if err != nil {
			logger.Error().Err(err).Str("path", redisMetricsPath).Msg("failed to load Redis metrics")
			fmt.Fprintf(os.Stderr, "\n❌ 加载 Redis 指标定义失败: %v\n", err)
			os.Exit(1)
		}
		redisActiveCount := config.CountActiveRedisMetrics(redisMetrics)
		fmt.Printf(" (%d 个活跃指标)\n", redisActiveCount)
		logger.Debug().Int("active_metrics", redisActiveCount).Int("total_metrics", len(redisMetrics)).Msg("Redis metrics loaded")
	}

	// Step 3d: Load Nginx metrics definitions (if needed)
	var nginxMetrics []*model.NginxMetricDefinition
	if runNginxInspection {
		fmt.Printf("📊 加载 Nginx 指标定义: %s", nginxMetricsPath)
		nginxMetrics, err = config.LoadNginxMetrics(nginxMetricsPath)
		if err != nil {
			logger.Error().Err(err).Str("path", nginxMetricsPath).Msg("failed to load Nginx metrics")
			fmt.Fprintf(os.Stderr, "\n❌ 加载 Nginx 指标定义失败: %v\n", err)
			os.Exit(1)
		}
		nginxActiveCount := config.CountActiveNginxMetrics(nginxMetrics)
		fmt.Printf(" (%d 个活跃指标)\n", nginxActiveCount)
		logger.Debug().Int("active_metrics", nginxActiveCount).Int("total_metrics", len(nginxMetrics)).Msg("Nginx metrics loaded")
	}

	// Step 3e: Load Tomcat metrics definitions (if needed)
	var tomcatMetrics []*model.TomcatMetricDefinition
	if runTomcatInspection {
		fmt.Printf("📊 加载 Tomcat 指标定义: %s", tomcatMetricsPath)
		tomcatMetrics, err = config.LoadTomcatMetrics(tomcatMetricsPath)
		if err != nil {
			logger.Error().Err(err).Str("path", tomcatMetricsPath).Msg("failed to load Tomcat metrics")
			fmt.Fprintf(os.Stderr, "\n❌ 加载 Tomcat 指标定义失败: %v\n", err)
			os.Exit(1)
		}
		tomcatActiveCount := config.CountActiveTomcatMetrics(tomcatMetrics)
		fmt.Printf(" (%d 个活跃指标)\n", tomcatActiveCount)
		logger.Debug().Int("active_metrics", tomcatActiveCount).Int("total_metrics", len(tomcatMetrics)).Msg("Tomcat metrics loaded")
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

	// Load timezone for evaluators that need it
	timezone, _ := time.LoadLocation("Asia/Shanghai")

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

	// Step 7c: Create Redis services (if needed)
	var redisInspector *service.RedisInspector
	if runRedisInspection {
		redisCollector := service.NewRedisCollector(&cfg.Redis, vmClient, redisMetrics, logger)
		redisEvaluator := service.NewRedisEvaluator(&cfg.Redis.Thresholds, redisMetrics, logger)
		redisInspector, err = service.NewRedisInspector(cfg, redisCollector, redisEvaluator, logger,
			service.WithRedisVersion(Version))
		if err != nil {
			logger.Error().Err(err).Msg("failed to create Redis inspector")
			fmt.Fprintf(os.Stderr, "❌ 创建 Redis 巡检器失败: %v\n", err)
			os.Exit(1)
		}
		logger.Debug().Msg("Redis services initialized")
	}

	// Step 7d: Create Nginx services (if needed)
	var nginxInspector *service.NginxInspector
	if runNginxInspection {
		nginxCollector := service.NewNginxCollector(&cfg.Nginx, vmClient, n9eClient, nginxMetrics, logger)
		nginxEvaluator := service.NewNginxEvaluator(&cfg.Nginx.Thresholds, nginxMetrics, timezone, logger)
		nginxInspector, err = service.NewNginxInspector(cfg, nginxCollector, nginxEvaluator, logger,
			service.WithNginxVersion(Version))
		if err != nil {
			logger.Error().Err(err).Msg("failed to create Nginx inspector")
			fmt.Fprintf(os.Stderr, "❌ 创建 Nginx 巡检器失败: %v\n", err)
			os.Exit(1)
		}
		logger.Debug().Msg("Nginx services initialized")
	}

	// Step 7e: Create Tomcat services (if needed)
	var tomcatInspector *service.TomcatInspector
	if runTomcatInspection {
		tomcatCollector := service.NewTomcatCollector(&cfg.Tomcat, vmClient, n9eClient, tomcatMetrics, logger)
		tomcatEvaluator := service.NewTomcatEvaluator(&cfg.Tomcat.Thresholds, tomcatMetrics, timezone, logger)
		tomcatInspector, err = service.NewTomcatInspector(cfg, tomcatCollector, tomcatEvaluator, logger,
			service.WithTomcatVersion(Version))
		if err != nil {
			logger.Error().Err(err).Msg("failed to create Tomcat inspector")
			fmt.Fprintf(os.Stderr, "❌ 创建 Tomcat 巡检器失败: %v\n", err)
			os.Exit(1)
		}
		logger.Debug().Msg("Tomcat services initialized")
	}

	// Step 8: Execute inspection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	startTime := time.Now()

	var hostResult *model.InspectionResult
	var mysqlResult *model.MySQLInspectionResults
	var redisResult *model.RedisInspectionResults
	var nginxResult *model.NginxInspectionResults
	var tomcatResult *model.TomcatInspectionResults

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

	// Execute Redis inspection
	if runRedisInspection {
		fmt.Println("\n⏳ 开始 Redis 巡检...")
		redisResult, err = redisInspector.Inspect(ctx)
		if err != nil {
			logger.Error().Err(err).Msg("Redis inspection failed")
			fmt.Fprintf(os.Stderr, "❌ Redis 巡检执行失败: %v\n", err)
			// Don't exit, continue to generate Host/MySQL report if available
			if hostResult == nil && mysqlResult == nil && nginxResult == nil {
				os.Exit(1)
			}
		} else {
			fmt.Printf("\n📊 Redis 巡检完成！\n")
			printRedisSummary(redisResult)
		}
	}

	// Execute Nginx inspection
	if runNginxInspection {
		fmt.Println("\n⏳ 开始 Nginx 巡检...")
		nginxResult, err = nginxInspector.Inspect(ctx)
		if err != nil {
			logger.Error().Err(err).Msg("Nginx inspection failed")
			fmt.Fprintf(os.Stderr, "❌ Nginx 巡检执行失败: %v\n", err)
			// Don't exit, continue to generate Host/MySQL/Redis report if available
			if hostResult == nil && mysqlResult == nil && redisResult == nil {
				os.Exit(1)
			}
		} else {
			fmt.Printf("\n📊 Nginx 巡检完成！\n")
			printNginxSummary(nginxResult)
		}
	}

	// Execute Tomcat inspection
	if runTomcatInspection {
		fmt.Println("\n⏳ 开始 Tomcat 巡检...")
		tomcatResult, err = tomcatInspector.Inspect(ctx)
		if err != nil {
			logger.Error().Err(err).Msg("Tomcat inspection failed")
			fmt.Fprintf(os.Stderr, "❌ Tomcat 巡检执行失败: %v\n", err)
			// Don't exit, continue to generate other reports if available
			if hostResult == nil && mysqlResult == nil && redisResult == nil && nginxResult == nil {
				os.Exit(1)
			}
		} else {
			fmt.Printf("\n📊 Tomcat 巡检完成！\n")
			printTomcatSummary(tomcatResult)
		}
	}

	fmt.Printf("\n⏱️  总耗时 %.1fs\n", time.Since(startTime).Seconds())

	// Step 9: Generate reports
	fmt.Println("\n📄 生成报告:")
	logger.Info().
		Strs("formats", outputFormats).
		Str("output_dir", outputPath).
		Msg("starting report generation")

	// Use timezone for report generation
	if inspector != nil {
		timezone = inspector.GetTimezone()
	} else if mysqlInspector != nil {
		timezone = mysqlInspector.GetTimezone()
	} else if redisInspector != nil {
		timezone = redisInspector.GetTimezone()
	} else if nginxInspector != nil {
		timezone = nginxInspector.GetTimezone()
	} else if tomcatInspector != nil {
		timezone = tomcatInspector.GetTimezone()
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
			genErr = generateCombinedExcel(hostResult, mysqlResult, redisResult, nginxResult, tomcatResult, reportPath, timezone, excelTemplatePath, logger)
		case "html":
			genErr = generateCombinedHTML(hostResult, mysqlResult, redisResult, nginxResult, tomcatResult, reportPath, timezone, cfg.Report.HTMLTemplate, logger)
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
	if redisResult != nil && redisResult.Summary != nil {
		if redisResult.Summary.CriticalInstances > 0 {
			exitCode = 2
		} else if redisResult.Summary.WarningInstances > 0 && exitCode < 1 {
			exitCode = 1
		}
	}
	if nginxResult != nil && nginxResult.Summary != nil {
		if nginxResult.Summary.CriticalInstances > 0 {
			exitCode = 2
		} else if nginxResult.Summary.WarningInstances > 0 && exitCode < 1 {
			exitCode = 1
		}
	}
	if tomcatResult != nil && tomcatResult.Summary != nil {
		if tomcatResult.Summary.CriticalInstances > 0 {
			exitCode = 2
		} else if tomcatResult.Summary.WarningInstances > 0 && exitCode < 1 {
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

// printRedisSummary prints the Redis inspection result summary.
func printRedisSummary(result *model.RedisInspectionResults) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if result.Summary != nil {
		fmt.Printf("   Redis 实例总数: %d\n", result.Summary.TotalInstances)
		fmt.Printf("   正常实例: %d\n", result.Summary.NormalInstances)
		fmt.Printf("   警告实例: %d\n", result.Summary.WarningInstances)
		fmt.Printf("   严重实例: %d\n", result.Summary.CriticalInstances)
		fmt.Printf("   失败实例: %d\n", result.Summary.FailedInstances)
	}
	fmt.Println()
	if result.AlertSummary != nil {
		fmt.Printf("   Redis 告警总数: %d\n", result.AlertSummary.TotalAlerts)
		fmt.Printf("   警告级别: %d\n", result.AlertSummary.WarningCount)
		fmt.Printf("   严重级别: %d\n", result.AlertSummary.CriticalCount)
	}
}

// printNginxSummary prints the Nginx inspection result summary.
func printNginxSummary(result *model.NginxInspectionResults) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if result.Summary != nil {
		fmt.Printf("   Nginx 实例总数: %d\n", result.Summary.TotalInstances)
		fmt.Printf("   正常实例: %d\n", result.Summary.NormalInstances)
		fmt.Printf("   警告实例: %d\n", result.Summary.WarningInstances)
		fmt.Printf("   严重实例: %d\n", result.Summary.CriticalInstances)
		fmt.Printf("   失败实例: %d\n", result.Summary.FailedInstances)
	}
	fmt.Println()
	if result.AlertSummary != nil {
		fmt.Printf("   Nginx 告警总数: %d\n", result.AlertSummary.TotalAlerts)
		fmt.Printf("   警告级别: %d\n", result.AlertSummary.WarningCount)
		fmt.Printf("   严重级别: %d\n", result.AlertSummary.CriticalCount)
	}
}

// printTomcatSummary prints the Tomcat inspection result summary.
func printTomcatSummary(result *model.TomcatInspectionResults) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if result.Summary != nil {
		fmt.Printf("   Tomcat 实例总数: %d\n", result.Summary.TotalInstances)
		fmt.Printf("   正常实例: %d\n", result.Summary.NormalInstances)
		fmt.Printf("   警告实例: %d\n", result.Summary.WarningInstances)
		fmt.Printf("   严重实例: %d\n", result.Summary.CriticalInstances)
		fmt.Printf("   失败实例: %d\n", result.Summary.FailedInstances)
	}
	fmt.Println()
	if result.AlertSummary != nil {
		fmt.Printf("   Tomcat 告警总数: %d\n", result.AlertSummary.TotalAlerts)
		fmt.Printf("   警告级别: %d\n", result.AlertSummary.WarningCount)
		fmt.Printf("   严重级别: %d\n", result.AlertSummary.CriticalCount)
	}
}

// generateCombinedExcel creates Excel report with Host, MySQL, Redis, Nginx and Tomcat data in same file.
func generateCombinedExcel(hostResult *model.InspectionResult, mysqlResult *model.MySQLInspectionResults, redisResult *model.RedisInspectionResults, nginxResult *model.NginxInspectionResults, tomcatResult *model.TomcatInspectionResults, outputPath string, timezone *time.Location, templatePath string, logger zerolog.Logger) error {
	// Resolve template path: use flag, config, or default
	if templatePath == "" {
		templatePath = defaultExcelTemplate
	}
	w := excel.NewWriter(timezone, templatePath)

	// Only Nginx mode
	if hostResult == nil && mysqlResult == nil && redisResult == nil && tomcatResult == nil && nginxResult != nil {
		return w.WriteNginxInspection(nginxResult, outputPath)
	}

	// Only Tomcat mode
	if hostResult == nil && mysqlResult == nil && redisResult == nil && tomcatResult != nil && nginxResult == nil {
		return w.WriteTomcatInspection(tomcatResult, outputPath)
	}

	// Only Redis mode
	if hostResult == nil && mysqlResult == nil && redisResult != nil && nginxResult == nil && tomcatResult == nil {
		return w.WriteRedisInspection(redisResult, outputPath)
	}

	// Only MySQL mode
	if hostResult == nil && mysqlResult != nil && redisResult == nil && nginxResult == nil && tomcatResult == nil {
		return w.WriteMySQLInspection(mysqlResult, outputPath)
	}

	// Only Host mode
	if hostResult != nil && mysqlResult == nil && redisResult == nil && nginxResult == nil && tomcatResult == nil {
		return w.Write(hostResult, outputPath)
	}

	// Combined mode: use WriteCombined to generate unified alerts sheet
	if err := w.WriteCombined(hostResult, mysqlResult, redisResult, nginxResult, tomcatResult, outputPath); err != nil {
		return fmt.Errorf("failed to write combined report: %w", err)
	}

	logger.Debug().
		Bool("has_host", hostResult != nil).
		Bool("has_mysql", mysqlResult != nil).
		Bool("has_redis", redisResult != nil).
		Bool("has_nginx", nginxResult != nil).
		Bool("has_tomcat", tomcatResult != nil).
		Str("path", outputPath).
		Msg("combined Excel report generated")

	return nil
}

// generateCombinedHTML creates HTML report with Host, MySQL, Redis, Nginx and Tomcat data.
func generateCombinedHTML(hostResult *model.InspectionResult, mysqlResult *model.MySQLInspectionResults, redisResult *model.RedisInspectionResults, nginxResult *model.NginxInspectionResults, tomcatResult *model.TomcatInspectionResults, outputPath string, timezone *time.Location, templatePath string, logger zerolog.Logger) error {
	w := html.NewWriter(timezone, templatePath)

	// Only Redis mode
	if hostResult == nil && mysqlResult == nil && redisResult != nil && nginxResult == nil && tomcatResult == nil {
		return w.WriteRedisInspection(redisResult, outputPath)
	}

	// Only MySQL mode
	if hostResult == nil && mysqlResult != nil && redisResult == nil && nginxResult == nil && tomcatResult == nil {
		return w.WriteMySQLInspection(mysqlResult, outputPath)
	}

	// Only Nginx mode
	if hostResult == nil && mysqlResult == nil && redisResult == nil && nginxResult != nil && tomcatResult == nil {
		return w.WriteNginxInspection(nginxResult, outputPath)
	}

	// Only Tomcat mode
	if hostResult == nil && mysqlResult == nil && redisResult == nil && nginxResult == nil && tomcatResult != nil {
		return w.WriteTomcatInspection(tomcatResult, outputPath)
	}

	// Only Host mode
	if hostResult != nil && mysqlResult == nil && redisResult == nil && nginxResult == nil && tomcatResult == nil {
		return w.Write(hostResult, outputPath)
	}

	// Combined mode
	if err := w.WriteCombined(hostResult, mysqlResult, redisResult, nginxResult, tomcatResult, outputPath); err != nil {
		return fmt.Errorf("failed to write combined HTML report: %w", err)
	}

	logger.Debug().
		Bool("has_host", hostResult != nil).
		Bool("has_mysql", mysqlResult != nil).
		Bool("has_redis", redisResult != nil).
		Bool("has_nginx", nginxResult != nil).
		Bool("has_tomcat", tomcatResult != nil).
		Str("path", outputPath).
		Msg("combined HTML report generated")

	return nil
}
