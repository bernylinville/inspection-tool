package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestRunInspection(t *testing.T) {
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

	// 创建临时输出文件
	tmpFile := "test_report.xlsx"
	defer os.Remove(tmpFile)

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
			&cli.IntFlag{
				Name:  "timeout",
				Value: 30,
			},
		},
	}

	// 准备测试上下文
	args := []string{
		"inspection-tool",
		"--vm-addr", server.URL,
		"--start", "now-1h",
		"--end", "now",
		"--output", tmpFile,
		"--no-progress",
		"--timeout", "30",
	}
	if err := app.Run(args); err != nil {
		t.Errorf("运行巡检时发生错误: %v", err)
	}

	// 验证输出文件是否创建
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("未生成报告文件")
	}
}
