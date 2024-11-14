package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bernylinville/inspection-tool/pkg/config"
	"github.com/bernylinville/inspection-tool/pkg/types"
	"github.com/urfave/cli/v2"
)

func TestRunInspection(t *testing.T) {
	// 设置测试超时
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 创建模拟的 VictoriaMetrics 服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"status":"success",
			"data":{
				"resultType":"matrix",
				"result":[{
					"metric":{"instance":"test-host-1"},
					"values":[[1625097600,"80"]]
				}]
			}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	}))
	defer server.Close()

	// 创建临时输出目录和文件
	tmpDir := "test_reports"
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := "test_report.xlsx"

	// 创建测试配置
	cfg := &config.Config{
		VictoriaMetrics: config.VMConfig{
			Address: server.URL,
			Timeout: 30,
		},
		Report: config.ReportConfig{
			OutputDir: tmpDir,
			Format:    "xlsx",
		},
		Concurrency: 10,
	}

	// 设置测试参数
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "vm-addr",
				Value: server.URL,
			},
			&cli.StringFlag{
				Name:  "start",
				Value: "now-1h",
			},
			&cli.StringFlag{
				Name:  "end",
				Value: "now",
			},
			&cli.StringFlag{
				Name:  "output",
				Value: tmpFile,
			},
			&cli.BoolFlag{
				Name:  "no-progress",
				Value: true,
			},
		},
	}

	// 准备测试上下文
	ctx := context.WithValue(context.Background(), types.ConfigKey, cfg)
	cliCtx := cli.NewContext(app, nil, nil)
	cliCtx.Context = ctx

	// 修改：使用 errChan 来传递错误
	errChan := make(chan error, 1)

	go func() {
		if err := runInspection(cliCtx); err != nil {
			errChan <- err
			return
		}
		close(errChan)
	}()

	// 等待测试完成或超时
	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("运行巡检时发生错误: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("测试超时")
	}

	// 验证输出文件是否创建
	time.Sleep(100 * time.Millisecond) // 给文件系统一点时间
	expectedFile := filepath.Join(tmpDir, "inspection_report_*.xlsx")
	matches, err := filepath.Glob(expectedFile)
	if err != nil {
		t.Errorf("查找输出文件失败: %v", err)
	}
	if len(matches) == 0 {
		t.Error("未生成报告文件")
	}
}
