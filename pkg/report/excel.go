package report

import (
	"fmt"
	"log"
	"math"
	"sort"
	"strings"

	"github.com/bernylinville/inspection-tool/pkg/metrics"
	"github.com/xuri/excelize/v2"
)

// ProjectData 定义项目数据结构
type ProjectData struct {
	Name    string               // 项目名称
	Metrics []metrics.MetricData // 项目的监控数据
	Summary ProjectSummary       // 项目汇总数据
}

// ProjectSummary 定义项目汇总数据
type ProjectSummary struct {
	HostCount    int     // 主机数量
	AvgCPU       float64 // 平均CPU使用率
	AvgMemory    float64 // 平均内存使用率
	AvgDisk      float64 // 平均磁盘使用率
	AvgLoad1     float64 // 平均1分钟负载
	AvgLoad5     float64 // 平均5分钟负载
	AvgLoad15    float64 // 平均15分钟负载
	HealthStatus struct {
		Healthy  int
		Warning  int
		Critical int
	}
}

// GenerateExcel 生成Excel报告
func GenerateExcel(data []metrics.MetricData, outputFile string) error {
	f := excelize.NewFile()
	defer f.Close()

	// 创建工作表
	sheetName := "巡检报告"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return err
	}

	// 设置标题
	titles := []string{
		"主机名",
		"IP地址",
		"CPU使用率(%)",
		"内存使用率(%)",
		"磁盘使用率(%)",
		"系统运行时间(小时)",
		"系统负载(1分钟)",
		"系统负载(5分钟)",
		"系统负载(15分钟)",
		"网络入流量(MB/s)",
		"网络出流量(MB/s)",
	}

	// 设置标题样式
	style, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#CCCCCC"}, Pattern: 1},
	})
	if err != nil {
		return err
	}

	// 写入标题
	for i, title := range titles {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, title)
		f.SetCellStyle(sheetName, cell, cell, style)
	}

	// 写入数据
	for i, item := range data {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), item.Hostname)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), item.IP)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), item.CPU)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), item.Memory)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), item.DiskUsage)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), item.SystemUptime/3600) // 转换为小时
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), item.SystemLoad1)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), item.SystemLoad5)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), item.SystemLoad15)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), item.NetworkIn)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), item.NetworkOut)
	}

	// 调整列宽
	for i := 0; i < len(titles); i++ {
		col := string(rune('A' + i))
		f.SetColWidth(sheetName, col, col, 15)
	}

	// 设置活动工作表
	f.SetActiveSheet(index)

	// 保存文件
	return f.SaveAs(outputFile)
}

// GenerateExcelWithProgress 生成带进度显示的Excel报告
func GenerateExcelWithProgress(data []metrics.MetricData, outputFile string, progress metrics.ProgressCallback) error {
	// 按项目分组数据
	projects := groupDataByProject(data)

	f := excelize.NewFile()
	defer f.Close()

	// 删除默认的 Sheet1
	if sheets := f.GetSheetList(); len(sheets) > 0 {
		for _, sheet := range sheets {
			if sheet != "Sheet1" { // 不要删除默认的 Sheet1，因为它是必需的
				f.DeleteSheet(sheet)
			}
		}
	}

	if progress != nil {
		progress("创建汇总表", 1, len(projects)+2)
	}

	// 创建汇总表
	err := createSummarySheet(f, projects)
	if err != nil {
		return fmt.Errorf("创建汇总表失败: %v", err)
	}

	// 为每个项目创建工作表
	for i, project := range projects {
		log.Printf("正在创建项目工作表: %s", project.Name)
		if progress != nil {
			progress(fmt.Sprintf("创建项目表: %s", project.Name), i+2, len(projects)+2)
		}

		err := createProjectSheet(f, project)
		if err != nil {
			return fmt.Errorf("创建项目表失败 [%s]: %v", project.Name, err)
		}
	}

	// 删除默认的 Sheet1（如果还存在）
	if sheets := f.GetSheetList(); len(sheets) > 1 {
		// 只有在有其他工作表时才删除 Sheet1
		f.DeleteSheet("Sheet1")
	}

	// 设置第一个工作表为活动工作表
	if sheets := f.GetSheetList(); len(sheets) > 0 {
		index, err := f.GetSheetIndex(sheets[0])
		if err != nil {
			return fmt.Errorf("获取工作表索引失败: %v", err)
		}
		f.SetActiveSheet(index)
	}

	// 保存文件
	if progress != nil {
		progress("保存文件", len(projects)+2, len(projects)+2)
	}

	return f.SaveAs(outputFile)
}

// groupDataByProject 按项目分组数据
func groupDataByProject(data []metrics.MetricData) []ProjectData {
	projectMap := make(map[string]*ProjectData)

	// 按项目分组
	for _, metric := range data {
		projectName := metric.Project
		if projectName == "" {
			// 如果没有设置项目字段，则从主机名中提取
			parts := strings.Split(metric.Hostname, "-")
			if len(parts) > 0 {
				projectName = parts[0]
				// 更新 metric 的项目字段
				metric.Project = projectName
			}
		}

		if _, exists := projectMap[projectName]; !exists {
			projectMap[projectName] = &ProjectData{
				Name:    projectName,
				Metrics: make([]metrics.MetricData, 0),
			}
		}
		projectMap[projectName].Metrics = append(projectMap[projectName].Metrics, metric)
	}

	// 计算每个项目的汇总数据
	for _, project := range projectMap {
		project.Summary = calculateProjectSummary(project.Metrics)
	}

	// 转换为切片
	projects := make([]ProjectData, 0, len(projectMap))
	for _, project := range projectMap {
		projects = append(projects, *project)
	}

	// 按项目名称排序
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})

	return projects
}

// calculateProjectSummary 计算项目汇总数据
func calculateProjectSummary(data []metrics.MetricData) ProjectSummary {
	var summary ProjectSummary
	count := float64(len(data))
	summary.HostCount = len(data)

	for _, metric := range data {
		summary.AvgCPU += metric.CPU
		summary.AvgMemory += metric.Memory
		summary.AvgDisk += metric.DiskUsage
		summary.AvgLoad1 += metric.SystemLoad1
		summary.AvgLoad5 += metric.SystemLoad5
		summary.AvgLoad15 += metric.SystemLoad15
	}

	if count > 0 {
		summary.AvgCPU /= count
		summary.AvgMemory /= count
		summary.AvgDisk /= count
		summary.AvgLoad1 /= count
		summary.AvgLoad5 /= count
		summary.AvgLoad15 /= count
	}

	return summary
}

// createSummarySheet 创建汇总表
func createSummarySheet(f *excelize.File, projects []ProjectData) error {
	sheetName := "汇总"
	_, err := f.NewSheet(sheetName)
	if err != nil {
		return err
	}

	// 设置标题
	titles := []string{
		"项目名称",
		"主机数量",
		"平均CPU使用率(%)",
		"平均内存使用率(%)",
		"平均磁盘使用率(%)",
		"平均负载(1分钟)",
		"平均负载(5分钟)",
		"平均负载(15分钟)",
	}

	// 设置标题样式
	style, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#CCCCCC"}, Pattern: 1},
	})
	if err != nil {
		return err
	}

	// 写入标题
	for i, title := range titles {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, title)
		f.SetCellStyle(sheetName, cell, cell, style)
	}

	// 写入项目汇总数据
	for i, project := range projects {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), project.Name)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), project.Summary.HostCount)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), formatFloat(project.Summary.AvgCPU))
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), formatFloat(project.Summary.AvgMemory))
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), formatFloat(project.Summary.AvgDisk))
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), formatFloat(project.Summary.AvgLoad1))
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), formatFloat(project.Summary.AvgLoad5))
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), formatFloat(project.Summary.AvgLoad15))
	}

	// 调整列宽
	for i := 0; i < len(titles); i++ {
		col := string(rune('A' + i))
		f.SetColWidth(sheetName, col, col, 15)
	}

	return nil
}

// createProjectSheet 创建项目工作表
func createProjectSheet(f *excelize.File, project ProjectData) error {
	sheetName := project.Name
	_, err := f.NewSheet(sheetName)
	if err != nil {
		return err
	}

	// 设置标题
	titles := []string{
		"主机名",
		"IP地址",
		"CPU使用率(%)",
		"内存使用率(%)",
		"磁盘使用率(%)",
		"系统运行时间(小时)",
		"系统负载(1分钟)",
		"系统负载(5分钟)",
		"系统负载(15分钟)",
		"网络入流量(MB/s)",
		"网络出流量(MB/s)",
		"健康状态",
		"健康检查信息",
	}

	// 设置标题样式
	style, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#CCCCCC"}, Pattern: 1},
	})
	if err != nil {
		return err
	}

	// 写入标题
	for i, title := range titles {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, title)
		f.SetCellStyle(sheetName, cell, cell, style)
	}

	// 写入数据
	for i, item := range project.Metrics {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), item.Hostname)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), item.IP)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), formatFloat(item.CPU))
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), formatFloat(item.Memory))
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), formatFloat(item.DiskUsage))
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), formatFloat(item.SystemUptime/3600))
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), formatFloat(item.SystemLoad1))
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), formatFloat(item.SystemLoad5))
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), formatFloat(item.SystemLoad15))
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), formatFloat(item.NetworkIn))
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), formatFloat(item.NetworkOut))

		health := metrics.CheckHealth(item)
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", row), health.Status)
		f.SetCellValue(sheetName, fmt.Sprintf("M%d", row), health.Message)

		// 设置健康状态的颜色
		style, _ := f.NewStyle(&excelize.Style{
			Fill: excelize.Fill{
				Type:    "pattern",
				Color:   []string{getHealthColor(health.Status)},
				Pattern: 1,
			},
		})
		f.SetCellStyle(sheetName, fmt.Sprintf("L%d", row), fmt.Sprintf("L%d", row), style)
	}

	// 调整列宽
	for i := 0; i < len(titles); i++ {
		col := string(rune('A' + i))
		f.SetColWidth(sheetName, col, col, 15)
	}

	return nil
}

// formatFloat 格式化浮点数，保留2位小数
func formatFloat(f float64) float64 {
	return math.Round(f*100) / 100
}

// getHealthColor 根据健康状态返回颜色
func getHealthColor(status metrics.HealthStatusType) string {
	switch status {
	case metrics.HealthStatusHealthy:
		return "#90EE90" // 浅绿色
	case metrics.HealthStatusWarning:
		return "#FFD700" // 金色
	case metrics.HealthStatusCritical:
		return "#FF6B6B" // 红色
	default:
		return "#FFFFFF" // 白色
	}
}
