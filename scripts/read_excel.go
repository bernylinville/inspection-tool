//go:build ignore
// +build ignore

// This script reads and displays the contents of an Excel report for verification.
package main

import (
	"fmt"
	"github.com/xuri/excelize/v2"
)

func main() {
	f, err := excelize.OpenFile("sample_inspection_report.xlsx")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer f.Close()

	fmt.Println("📊 Sheets:", f.GetSheetList())
	fmt.Println()

	// Summary sheet
	fmt.Println("═══════════════════════════════════════")
	fmt.Println("  巡检概览")
	fmt.Println("═══════════════════════════════════════")
	for row := 1; row <= 14; row++ {
		a, _ := f.GetCellValue("巡检概览", fmt.Sprintf("A%d", row))
		b, _ := f.GetCellValue("巡检概览", fmt.Sprintf("B%d", row))
		if a != "" || b != "" {
			fmt.Printf("  %-12s %s\n", a, b)
		}
	}
	fmt.Println()

	// Detail sheet - headers
	fmt.Println("═══════════════════════════════════════")
	fmt.Println("  详细数据 (表头)")
	fmt.Println("═══════════════════════════════════════")
	headers := []string{}
	for col := 1; col <= 20; col++ {
		cell := columnName(col) + "1"
		v, _ := f.GetCellValue("详细数据", cell)
		if v == "" {
			break
		}
		headers = append(headers, v)
	}
	for i, h := range headers {
		fmt.Printf("  [%d] %s\n", i+1, h)
	}
	fmt.Println()

	// Detail sheet - data rows
	fmt.Println("═══════════════════════════════════════")
	fmt.Println("  详细数据 (主机列表)")
	fmt.Println("═══════════════════════════════════════")
	for row := 2; row <= 6; row++ {
		hostname, _ := f.GetCellValue("详细数据", fmt.Sprintf("A%d", row))
		ip, _ := f.GetCellValue("详细数据", fmt.Sprintf("B%d", row))
		status, _ := f.GetCellValue("详细数据", fmt.Sprintf("C%d", row))
		cpu, _ := f.GetCellValue("详细数据", fmt.Sprintf("H%d", row))
		mem, _ := f.GetCellValue("详细数据", fmt.Sprintf("I%d", row))
		disk, _ := f.GetCellValue("详细数据", fmt.Sprintf("J%d", row))
		if hostname != "" {
			fmt.Printf("  %-16s %-14s %-6s CPU:%-6s Mem:%-6s Disk:%s\n", hostname, ip, status, cpu, mem, disk)
		}
	}
	fmt.Println()

	// Alerts sheet
	fmt.Println("═══════════════════════════════════════")
	fmt.Println("  异常汇总 (按严重程度排序)")
	fmt.Println("═══════════════════════════════════════")
	fmt.Println("  主机名           | 级别   | 指标          | 当前值")
	fmt.Println("  -----------------+--------+---------------+--------")
	for row := 2; row <= 8; row++ {
		hostname, _ := f.GetCellValue("异常汇总", fmt.Sprintf("A%d", row))
		level, _ := f.GetCellValue("异常汇总", fmt.Sprintf("B%d", row))
		metric, _ := f.GetCellValue("异常汇总", fmt.Sprintf("C%d", row))
		value, _ := f.GetCellValue("异常汇总", fmt.Sprintf("D%d", row))
		if hostname != "" {
			fmt.Printf("  %-16s | %-6s | %-13s | %s\n", hostname, level, metric, value)
		}
	}
	fmt.Println()
	fmt.Println("✅ Excel 报告验证完成！")
	fmt.Println("   请用 Excel/WPS 打开 sample_inspection_report.xlsx 查看完整样式")
}

func columnName(index int) string {
	result := ""
	for index > 0 {
		index--
		result = string(rune('A'+index%26)) + result
		index /= 26
	}
	return result
}
