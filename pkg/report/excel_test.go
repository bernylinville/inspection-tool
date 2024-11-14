package report

import (
	"os"
	"testing"

	"github.com/bernylinville/inspection-tool/pkg/metrics"
	"github.com/xuri/excelize/v2"
)

func TestGenerateExcel(t *testing.T) {
	// 准备测试数据
	testData := []metrics.MetricData{
		{
			Hostname:     "test-project1-host-1",
			IP:           "192.168.1.1",
			CPU:          75.5,
			Memory:       80.2,
			DiskUsage:    60.0,
			NetworkIn:    1.5,
			NetworkOut:   2.0,
			SystemLoad1:  1.5,
			SystemLoad5:  1.4,
			SystemLoad15: 1.3,
			SystemUptime: 3600 * 24, // 24小时
		},
		{
			Hostname:     "test-project1-host-2",
			IP:           "192.168.1.2",
			CPU:          65.5,
			Memory:       70.2,
			DiskUsage:    50.0,
			NetworkIn:    1.2,
			NetworkOut:   1.8,
			SystemLoad1:  1.2,
			SystemLoad5:  1.1,
			SystemLoad15: 1.0,
			SystemUptime: 3600 * 48, // 48小时
		},
		{
			Hostname:     "test-project2-host-1",
			IP:           "192.168.2.1",
			CPU:          85.5,
			Memory:       90.2,
			DiskUsage:    70.0,
			NetworkIn:    2.5,
			NetworkOut:   3.0,
			SystemLoad1:  2.5,
			SystemLoad5:  2.4,
			SystemLoad15: 2.3,
			SystemUptime: 3600 * 72, // 72小时
		},
	}

	// 使用临时文件
	tmpFile := "test_report.xlsx"
	defer os.Remove(tmpFile) // 测试完成后清理

	// 测试基本的Excel生成
	t.Run("基本Excel生成", func(t *testing.T) {
		err := GenerateExcel(testData, tmpFile)
		if err != nil {
			t.Errorf("生成Excel报告时发生错误: %v", err)
		}

		// 验证文件是否创建
		if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
			t.Error("Excel文件未被创建")
		}
	})

	// 测试带进度的Excel生成
	t.Run("带进度的Excel生成", func(t *testing.T) {
		progressCount := 0
		progressCallback := func(stage string, current, total int) {
			progressCount++
		}

		err := GenerateExcelWithProgress(testData, tmpFile, progressCallback)
		if err != nil {
			t.Errorf("生成带进度的Excel报告时发生错误: %v", err)
		}

		if progressCount == 0 {
			t.Error("进度回调未被调用")
		}
	})

	// 测试项目分组功能
	t.Run("项目分组", func(t *testing.T) {
		// 使用 GenerateExcelWithProgress 来测试项目分组
		err := GenerateExcelWithProgress(testData, tmpFile, nil)
		if err != nil {
			t.Errorf("生成带项目分组的Excel报告时发生错误: %v", err)
			return
		}

		f, err := excelize.OpenFile(tmpFile)
		if err != nil {
			t.Errorf("打开生成的Excel文件时发生错误: %v", err)
			return
		}
		defer f.Close()

		// 验证汇总表是否存在
		if _, err := f.GetRows("汇总"); err != nil {
			t.Error("汇总表不存在")
		}

		// 验证项目表是否存在
		expectedSheets := []string{"test-project1", "test-project2"}
		sheets := f.GetSheetList()
		for _, expectedSheet := range expectedSheets {
			found := false
			for _, sheet := range sheets {
				if sheet == expectedSheet {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("项目表 %s 不存在，现有表: %v", expectedSheet, sheets)
			}
		}
	})
}

func TestGroupDataByProject(t *testing.T) {
	testData := []metrics.MetricData{
		{
			Hostname: "project1-host1",
			CPU:      80.0,
		},
		{
			Hostname: "project1-host2",
			CPU:      70.0,
		},
		{
			Hostname: "project2-host1",
			CPU:      90.0,
		},
	}

	projects := groupDataByProject(testData)

	if len(projects) != 2 {
		t.Errorf("期望2个项目，实际得到 %d 个项目", len(projects))
	}

	// 验证项目分组是否正确
	for _, project := range projects {
		switch project.Name {
		case "project1":
			if len(project.Metrics) != 2 {
				t.Errorf("project1 期望2个主机，实际得到 %d 个", len(project.Metrics))
			}
		case "project2":
			if len(project.Metrics) != 1 {
				t.Errorf("project2 期望1个主机，实际得到 %d 个", len(project.Metrics))
			}
		default:
			t.Errorf("未预期的项目名称: %s", project.Name)
		}
	}
}

func TestCalculateProjectSummary(t *testing.T) {
	testData := []metrics.MetricData{
		{
			CPU:          80.0,
			Memory:       70.0,
			DiskUsage:    60.0,
			SystemLoad1:  1.0,
			SystemLoad5:  1.1,
			SystemLoad15: 1.2,
		},
		{
			CPU:          60.0,
			Memory:       50.0,
			DiskUsage:    40.0,
			SystemLoad1:  2.0,
			SystemLoad5:  2.1,
			SystemLoad15: 2.2,
		},
	}

	summary := calculateProjectSummary(testData)

	// 验证计算结果
	if summary.HostCount != 2 {
		t.Errorf("期望主机数量为2，实际得到 %d", summary.HostCount)
	}

	expectedAvgCPU := 70.0
	if summary.AvgCPU != expectedAvgCPU {
		t.Errorf("期望平均CPU为%.2f，实际得到 %.2f", expectedAvgCPU, summary.AvgCPU)
	}

	// 可以添加更多指标的验证...
}
