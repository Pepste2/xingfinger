// Package pkg 提供 xingfinger 的核心功能
// 本文件负责扫描结果的输出和保存
package pkg

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ensureDir 确保文件所在的目录存在
// 如果目录不存在，则自动创建
//
// 参数：
//   - filename: 文件路径
//
// 返回：
//   - error: 创建目录的错误
func ensureDir(filename string) error {
	dir := filepath.Dir(filename)
	if dir == "" || dir == "." {
		return nil // 当前目录，无需创建
	}

	// 检查目录是否存在
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// 创建目录（包括所有父目录）
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录失败: %v", err)
		}
		fmt.Printf("[*] 已创建目录: %s\n", dir)
	}
	return nil
}

// saveResults 保存扫描结果到文件
// 根据文件扩展名自动选择格式：.json, .csv, .xlsx
// 对于 CSV 和 XLSX 格式，只输出存活目标（命中指纹的）
// 支持跨路径输出，自动创建不存在的目录
//
// 参数：
//   - filename: 输出文件路径
//   - results: 指纹识别结果切片
func saveResults(filename string, results []Result) {
	// 确保目录存在
	if err := ensureDir(filename); err != nil {
		fmt.Printf("[!] %v\n", err)
		return
	}

	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".json":
		saveJSON(filename, results)
	case ".csv":
		saveCSV(filename, results)
	case ".xlsx":
		saveXLSX(filename, results)
	default:
		fmt.Printf("[!] 不支持的输出格式: %s (支持: .json, .csv, .xlsx)\n", ext)
	}
}

// saveJSON 将结果保存为 JSON 格式文件
// JSON 格式输出所有结果（包括未命中的）
//
// 参数：
//   - filename: 输出文件路径
//   - results: 指纹识别结果切片
func saveJSON(filename string, results []Result) {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Println("[!] JSON error:", err)
		return
	}

	f, err := os.Create(filename)
	if err != nil {
		fmt.Println("[!] Create error:", err)
		return
	}
	defer f.Close()

	f.Write(data)
	fmt.Printf("[*] 已保存 JSON 结果: %s (%d 条)\n", filename, len(results))
}

// saveCSV 将结果保存为 CSV 格式文件
// CSV 格式只输出存活目标（命中指纹的）
// 列顺序：URL, 状态码, 大小, Server, Title, 指纹信息
//
// 参数：
//   - filename: 输出文件路径
//   - results: 指纹识别结果切片
func saveCSV(filename string, results []Result) {
	// 过滤只保留命中指纹的结果
	hitResults := filterHitResults(results)

	if len(hitResults) == 0 {
		fmt.Println("[!] 无命中结果，不生成 CSV 文件")
		return
	}

	f, err := os.Create(filename)
	if err != nil {
		fmt.Println("[!] Create error:", err)
		return
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// 写入表头
	header := []string{"URL", "状态码", "大小", "Server", "Title", "指纹信息"}
	if err := writer.Write(header); err != nil {
		fmt.Println("[!] CSV write header error:", err)
		return
	}

	// 写入数据行
	for _, r := range hitResults {
		row := []string{
			r.URL,
			fmt.Sprintf("%d", r.StatusCode),
			fmt.Sprintf("%d", r.Length),
			r.Server,
			r.Title,
			r.CMS,
		}
		if err := writer.Write(row); err != nil {
			fmt.Println("[!] CSV write error:", err)
			return
		}
	}

	fmt.Printf("[*] 已保存 CSV 结果: %s (%d 条存活目标)\n", filename, len(hitResults))
}

// saveXLSX 将结果保存为 XLSX 格式文件
// XLSX 格式只输出存活目标（命中指纹的）
// 列顺序：URL, 状态码, 大小, Server, Title, 指纹信息
// 命中指纹的行添加红色高亮（与终端输出一致）
//
// 参数：
//   - filename: 输出文件路径
//   - results: 指纹识别结果切片
func saveXLSX(filename string, results []Result) {
	// 过滤只保留命中指纹的结果
	hitResults := filterHitResults(results)

	if len(hitResults) == 0 {
		fmt.Println("[!] 无命中结果，不生成 XLSX 文件")
		return
	}

	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println("[!] XLSX close error:", err)
		}
	}()

	// 设置工作表名称
	sheetName := "扫描结果"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		fmt.Println("[!] XLSX new sheet error:", err)
		return
	}
	f.SetActiveSheet(index)
	f.DeleteSheet("Sheet1") // 删除默认工作表

	// 设置表头样式
	headers := []string{"URL", "状态码", "大小", "Server", "Title", "指纹信息"}
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#CCCCCC"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
		},
	})

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	// 命中指纹行样式：浅红色背景（与终端红色高亮对应）
	// RGB(237,64,35) → #ED4023，使用更柔和的浅红 #FFE4E1
	hitRowStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#FFE4E1"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
		},
	})

	// 指纹信息列特殊样式：深红背景突出显示
	fingerprintCellStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#ED4023"}, Pattern: 1}, // 与终端红色一致
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},                          // 白色粗体字体
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
		},
	})

	// 写入数据行（所有行都是命中指纹的）
	for rowIdx, r := range hitResults {
		row := rowIdx + 2 // 数据从第 2 行开始

		// 写入前 5 列数据
		data := []interface{}{
			r.URL,
			r.StatusCode,
			r.Length,
			r.Server,
			r.Title,
		}

		for colIdx, value := range data {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, row)
			f.SetCellValue(sheetName, cell, value)
			f.SetCellStyle(sheetName, cell, cell, hitRowStyle)
		}

		// 指纹信息列（第 6 列）特殊高亮
		fingerprintCell, _ := excelize.CoordinatesToCellName(6, row)
		f.SetCellValue(sheetName, fingerprintCell, r.CMS)
		f.SetCellStyle(sheetName, fingerprintCell, fingerprintCell, fingerprintCellStyle)
	}

	// 设置列宽
	f.SetColWidth(sheetName, "A", "A", 50)  // URL 列宽
	f.SetColWidth(sheetName, "B", "C", 12)  // 状态码、大小
	f.SetColWidth(sheetName, "D", "D", 20)  // Server
	f.SetColWidth(sheetName, "E", "E", 30)  // Title
	f.SetColWidth(sheetName, "F", "F", 40)  // 指纹信息

	// 保存文件
	if err := f.SaveAs(filename); err != nil {
		fmt.Println("[!] XLSX save error:", err)
		return
	}

	fmt.Printf("[*] 已保存 XLSX 结果: %s (%d 条存活目标，带高亮)\n", filename, len(hitResults))
}

// filterHitResults 过滤只保留命中指纹的结果
//
// 参数：
//   - results: 所有扫描结果
//
// 返回：
//   - []Result: 只包含命中指纹的结果
func filterHitResults(results []Result) []Result {
	hitResults := []Result{}
	for _, r := range results {
		if r.CMS != "" {
			hitResults = append(hitResults, r)
		}
	}
	return hitResults
}