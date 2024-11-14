package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/bernylinville/inspection-tool/pkg/config"
	"github.com/bernylinville/inspection-tool/pkg/logger"
	"github.com/bernylinville/inspection-tool/pkg/metrics"
	"github.com/bernylinville/inspection-tool/pkg/report"
	"github.com/bernylinville/inspection-tool/pkg/types"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "inspection-tool",
		Usage: "运维巡检工具",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "配置文件路径",
				EnvVars: []string{"INSPECTION_CONFIG"},
			},
			&cli.StringFlag{
				Name:    "vm-addr",
				Usage:   "VictoriaMetrics 地址",
				EnvVars: []string{"INSPECTION_VM_ADDRESS"},
			},
			&cli.StringFlag{
				Name:    "output-dir",
				Usage:   "报告输出目录",
				EnvVars: []string{"INSPECTION_OUTPUT_DIR"},
			},
			&cli.IntFlag{
				Name:    "concurrency",
				Usage:   "并发执行的线程数",
				EnvVars: []string{"INSPECTION_CONCURRENCY"},
			},
			&cli.StringSliceFlag{
				Name:    "projects",
				Usage:   "要生成报告的项目列表",
				EnvVars: []string{"INSPECTION_PROJECTS"},
			},
			&cli.StringFlag{
				Name:  "start",
				Value: "now-1d",
				Usage: "开始时间 (例如: now-1d)",
			},
			&cli.StringFlag{
				Name:  "end",
				Value: "now",
				Usage: "结束时间",
			},
			&cli.StringSliceFlag{
				Name:  "label",
				Usage: "标签过滤(格式: key=value), 可多次使用",
			},
			&cli.BoolFlag{
				Name:  "no-progress",
				Usage: "禁用进度显示",
			},
			&cli.BoolFlag{
				Name:    "debug",
				Usage:   "启用调试模式",
				EnvVars: []string{"INSPECTION_DEBUG"},
			},
		},
		Before: func(c *cli.Context) error {
			// 初始化日志
			logger.InitLogger(c.Bool("debug"))
			return loadConfig(c)
		},
		Action: runInspection,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func loadConfig(c *cli.Context) error {
	// 加载配置
	cfg, err := config.LoadConfig(c.String("config"))
	if err != nil {
		return fmt.Errorf("加载配置失败: %v", err)
	}

	// 命令行参数覆盖配置文件
	if c.IsSet("vm-addr") {
		cfg.VictoriaMetrics.Address = c.String("vm-addr")
	}
	if c.IsSet("output-dir") {
		cfg.Report.OutputDir = c.String("output-dir")
	}
	if c.IsSet("concurrency") {
		cfg.Concurrency = c.Int("concurrency")
	}
	if c.IsSet("projects") {
		cfg.Projects = c.StringSlice("projects")
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("配置验证失败: %v", err)
	}

	// 将配置保存到上下文
	c.Context = context.WithValue(c.Context, types.ConfigKey, cfg)

	logger.Debug().
		Str("vm_addr", cfg.VictoriaMetrics.Address).
		Str("output_dir", cfg.Report.OutputDir).
		Int("concurrency", cfg.Concurrency).
		Msg("加载配置")

	return nil
}

func runInspection(c *cli.Context) error {
	cfg := c.Context.Value(types.ConfigKey).(*config.Config)

	logger.Info().
		Str("start_time", c.String("start")).
		Str("end_time", c.String("end")).
		Strs("labels", c.StringSlice("label")).
		Strs("projects", cfg.Projects).
		Msg("开始巡检任务")

	// 创建输出目录
	if err := os.MkdirAll(cfg.Report.OutputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}

	client := metrics.NewClient(cfg.VictoriaMetrics.Address, cfg.VictoriaMetrics.Timeout)
	showProgress := !c.Bool("no-progress")

	var bar *progressbar.ProgressBar
	var lastStage string

	progressCallback := func(stage string, current, total int) {
		if !showProgress {
			return
		}

		if lastStage != stage {
			if bar != nil {
				bar.Finish()
			}
			bar = progressbar.NewOptions(total,
				progressbar.OptionSetDescription(stage),
				progressbar.OptionSetTheme(progressbar.Theme{
					Saucer:        "=",
					SaucerHead:    ">",
					SaucerPadding: " ",
					BarStart:      "[",
					BarEnd:        "]",
				}),
				progressbar.OptionEnableColorCodes(true),
				progressbar.OptionSetWidth(40),
				progressbar.OptionShowCount(),
				progressbar.OptionSetElapsedTime(true),
			)
			lastStage = stage
		}
		bar.Set(current)
	}

	fmt.Println("开始收集监控数据...")
	startTime := time.Now()

	// 获取监控数据
	data, err := client.GetMetricsWithProgress(metrics.QueryOptions{
		Start:       c.String("start"),
		End:         c.String("end"),
		Labels:      c.StringSlice("label"),
		Projects:    cfg.Projects,
		Concurrency: cfg.Concurrency,
	}, progressCallback)
	if err != nil {
		logger.Error().Err(err).Msg("获取监控数据失败")
		return err
	}

	logger.Info().
		Int("metrics_count", len(data)).
		Msg("成功获取监控数据")

	if len(data) == 0 {
		logger.Warn().Msg("未获取到任何监控数据")
		return fmt.Errorf("未获取到任何监控数据，请检查查询条件和时间范围")
	}

	fmt.Println("\n开始生成报告...")

	// 生成报告文件名
	outputFile := filepath.Join(cfg.Report.OutputDir,
		fmt.Sprintf("inspection_report_%s.%s",
			time.Now().Format("20060102_150405"),
			cfg.Report.Format))

	// 生成报告
	err = report.GenerateExcelWithProgress(data, outputFile, progressCallback)
	if err != nil {
		logger.Error().Err(err).Str("output_file", outputFile).Msg("生成报告失败")
		return err
	}

	duration := time.Since(startTime)
	logger.Info().
		Str("output_file", outputFile).
		Dur("duration", duration).
		Msg("报告生成完成")

	fmt.Printf("\n完成! 耗时: %v\n", duration.Round(time.Second))
	fmt.Printf("报告已生成: %s\n", outputFile)

	logger.Debug().
		Str("vm_addr", cfg.VictoriaMetrics.Address).
		Dur("timeout", cfg.VictoriaMetrics.Timeout).
		Msg("创建 VictoriaMetrics 客户端")

	return nil
}
