package excel

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/model"

	"github.com/xuri/excelize/v2"
)

// Exporter Excel 导出器
type Exporter struct{}

// NewExporter 创建导出器实例
func NewExporter() *Exporter {
	return &Exporter{}
}

// ExportTransactions 将交易记录导出为 Excel 字节数据
func (e *Exporter) ExportTransactions(items []*dto.TransactionResp) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "交易记录"
	f.SetSheetName("Sheet1", sheet)

	// 设置表头样式
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 11},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#4472C4"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	if err != nil {
		// 如果样式创建失败，使用默认样式继续
		headerStyle = 0
	}

	// 写入表头
	headers := []string{"日期", "类型", "金额", "分类", "备注", "创建时间"}
	cols := []string{"A", "B", "C", "D", "E", "F"}
	for i, h := range headers {
		cell := cols[i] + "1"
		f.SetCellValue(sheet, cell, h)
		if headerStyle != 0 {
			f.SetCellStyle(sheet, cell, cell, headerStyle)
		}
	}

	// 设置列宽
	colWidths := map[string]float64{
		"A": 15, // 日期
		"B": 10, // 类型
		"C": 12, // 金额
		"D": 15, // 分类
		"E": 30, // 备注
		"F": 22, // 创建时间
	}
	for col, width := range colWidths {
		f.SetColWidth(sheet, col, col, width)
	}

	// 金额样式
	amountStyle, _ := f.NewStyle(&excelize.Style{
		NumFmt: 2, // 数字格式：0.00
	})

	// 写入数据行
	typeMap := map[string]string{
		"income":  "收入",
		"expense": "支出",
	}

	for i, item := range items {
		row := i + 2
		typeCN := typeMap[item.Type]
		if typeCN == "" {
			typeCN = item.Type
		}

		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), item.TransactionDate)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), typeCN)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), item.Amount)
		f.SetCellStyle(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), amountStyle)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), item.CategoryName)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), item.Note)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), item.CreatedAt)
	}

	// 添加汇总行
	if len(items) > 0 {
		summaryRow := len(items) + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", summaryRow), "合计")
		f.SetCellFormula(sheet, fmt.Sprintf("C%d", summaryRow),
			fmt.Sprintf("=SUMIF(B2:B%d,\"收入\",C2:C%d)-SUMIF(B2:B%d,\"支出\",C2:C%d)",
				summaryRow-1, summaryRow-1, summaryRow-1, summaryRow-1))
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("生成Excel缓冲区失败: %w", err)
	}
	return buf.Bytes(), nil
}

// GenerateTemplate 生成导入模板 xlsx 文件
func GenerateTemplate() ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "导入模板"
	f.SetSheetName("Sheet1", sheet)

	// 表头（第1行）：日期、类型、金额、分类、商户、备注
	headers := []string{"日期", "类型", "金额", "分类", "商户", "备注"}
	cols := []string{"A", "B", "C", "D", "E", "F"}
	for i, h := range headers {
		f.SetCellValue(sheet, cols[i]+"1", h)
	}

	// 第2行示例（支出）
	f.SetCellValue(sheet, "A2", "2024-01-15")
	f.SetCellValue(sheet, "B2", "支出")
	f.SetCellValue(sheet, "C2", 128.50)
	f.SetCellValue(sheet, "D2", "食材采购")
	f.SetCellValue(sheet, "E2", "菜市场")
	f.SetCellValue(sheet, "F2", "今日蔬菜")

	// 第3行示例（收入）
	f.SetCellValue(sheet, "A3", "2024-01-15")
	f.SetCellValue(sheet, "B3", "收入")
	f.SetCellValue(sheet, "C3", 3500.00)
	f.SetCellValue(sheet, "D3", "堂食收入")
	f.SetCellValue(sheet, "E3", "")
	f.SetCellValue(sheet, "F3", "午市堂食")

	// 第4行说明
	f.SetCellValue(sheet, "A4", "说明：日期格式 YYYY-MM-DD，类型只填\"收入\"或\"支出\"")

	// 设置列宽
	colWidths := map[string]float64{
		"A": 15,
		"B": 10,
		"C": 12,
		"D": 15,
		"E": 15,
		"F": 30,
	}
	for col, width := range colWidths {
		f.SetColWidth(sheet, col, col, width)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("生成模板失败: %w", err)
	}
	return buf.Bytes(), nil
}

// ColumnMapping 列映射：字段名 → 列下标（-1 表示未映射）
type ColumnMapping struct {
	Date     int `json:"date"`
	Type     int `json:"type"`
	Amount   int `json:"amount"`
	Category int `json:"category"`
	Merchant int `json:"merchant"`
	Note     int `json:"note"`
}

// ParseHeadersResult 解析表头结果
type ParseHeadersResult struct {
	Sheets      []string      `json:"sheets"`
	SheetIndex  int           `json:"sheet_index"`
	Headers     []string      `json:"headers"`
	SampleRows  [][]string    `json:"sample_rows"`
	Suggestions ColumnMapping `json:"suggestions"`
}

// parseAmount 解析金额字符串，兼容全角逗号小数点和千分位分隔符
func parseAmount(s string) (float64, error) {
	// 全角逗号 → 小数点（如 "16，75"）
	s = strings.ReplaceAll(s, "，", ".")
	// 若含 ASCII 逗号且不像千分位（逗号后不是恰好3位数字），视为小数点
	if strings.Count(s, ",") == 1 {
		parts := strings.SplitN(s, ",", 2)
		if len(parts[1]) != 3 {
			s = parts[0] + "." + parts[1]
		} else {
			s = strings.ReplaceAll(s, ",", "")
		}
	} else {
		// 多个 ASCII 逗号均视为千分位，直接去掉
		s = strings.ReplaceAll(s, ",", "")
	}
	return strconv.ParseFloat(strings.TrimSpace(s), 64)
}

// fieldAliases 字段名与常见表头别名的映射
var fieldAliases = map[string][]string{
	"date":     {"日期", "交易日期", "时间", "交易时间", "date"},
	"type":     {"类型", "收支类型", "收支", "交易类型", "type", "借贷标志"},
	"amount":   {"金额", "交易金额", "付款金额", "收款金额", "amount", "价格", "金额(元)"},
	"category": {"分类", "类别", "category", "科目", "用途"},
	"merchant": {"商户", "商家", "店铺", "merchant", "商户名称", "摘要"},
	"note":     {"备注", "说明", "note", "remark", "描述", "备注信息"},
}

// suggestMapping 根据表头自动推断列映射
func suggestMapping(headers []string) ColumnMapping {
	m := ColumnMapping{Date: -1, Type: -1, Amount: -1, Category: -1, Merchant: -1, Note: -1}
	for i, h := range headers {
		normalized := strings.ToLower(strings.TrimSpace(h))
		for field, aliases := range fieldAliases {
			for _, alias := range aliases {
				if strings.ToLower(alias) == normalized {
					switch field {
					case "date":
						if m.Date == -1 {
							m.Date = i
						}
					case "type":
						if m.Type == -1 {
							m.Type = i
						}
					case "amount":
						if m.Amount == -1 {
							m.Amount = i
						}
					case "category":
						if m.Category == -1 {
							m.Category = i
						}
					case "merchant":
						if m.Merchant == -1 {
							m.Merchant = i
						}
					case "note":
						if m.Note == -1 {
							m.Note = i
						}
					}
				}
			}
		}
	}
	return m
}

// ParseHeaders 解析 xlsx 文件的表头和样本行，并自动推断列映射
// bestSheetIndex 在多个 sheet 中自动选择表头匹配度最高的
// 评分规则：date/type/amount/category 各占 3 分，merchant/note 各占 1 分
func bestSheetIndex(f *excelize.File, sheets []string) int {
	best, bestScore := 0, -1
	for i, s := range sheets {
		rows, err := f.GetRows(s)
		if err != nil || len(rows) == 0 {
			continue
		}
		m := suggestMapping(rows[0])
		score := 0
		if m.Date >= 0 {
			score += 3
		}
		if m.Type >= 0 {
			score += 3
		}
		if m.Amount >= 0 {
			score += 3
		}
		if m.Category >= 0 {
			score += 3
		}
		if m.Merchant >= 0 {
			score++
		}
		if m.Note >= 0 {
			score++
		}
		if score > bestScore {
			bestScore = score
			best = i
		}
	}
	return best
}

// ParseHeaders 解析 xlsx 文件的 sheet 列表、表头和样本数据，自动推断列映射
// sheetIndex=-1 时自动选择表头匹配度最高的 sheet
func ParseHeaders(data []byte, sheetIndex int) (*ParseHeadersResult, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("文件内容为空")
	}

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("解析Excel文件失败: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("Excel文件中没有工作表")
	}

	if sheetIndex < 0 || sheetIndex >= len(sheets) {
		sheetIndex = bestSheetIndex(f, sheets)
	}

	rows, err := f.GetRows(sheets[sheetIndex])
	if err != nil {
		return nil, fmt.Errorf("读取工作表失败: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("工作表为空")
	}

	headers := rows[0]

	// 收集最多 3 条样本行（跳过"说明"行和空行）
	var sampleRows [][]string
	for i := 1; i < len(rows) && len(sampleRows) < 3; i++ {
		row := rows[i]
		isEmpty := true
		for _, cell := range row {
			if strings.TrimSpace(cell) != "" {
				isEmpty = false
				break
			}
		}
		if isEmpty {
			continue
		}
		if len(row) > 0 && strings.Contains(row[0], "说明") {
			continue
		}
		for len(row) < len(headers) {
			row = append(row, "")
		}
		sampleRows = append(sampleRows, row[:len(headers)])
	}

	return &ParseHeadersResult{
		Sheets:      sheets,
		SheetIndex:  sheetIndex,
		Headers:     headers,
		SampleRows:  sampleRows,
		Suggestions: suggestMapping(headers),
	}, nil
}

// ImportResult 导入/预检结果
type ImportResult struct {
	ValidCount   int      // 可导入行数
	SkippedCount int      // 跳过行数
	Issues       []string // 问题说明（含修复建议）
}

// ImportFromExcel 从 xlsx 数据中解析交易记录
// mapping 指定各字段对应的列下标；若 mapping 为 nil，则自动从表头推断
// sheetIndex=-1 时自动选最佳 sheet
// dryRun=true 时只验证不返回 Transaction，用于预检
// 返回：可导入的 Transaction 列表（dryRun=true 时为 nil）、ImportResult、error
func ImportFromExcel(data []byte, tenantID uint64, userID uint64, categoryList []*model.Category, mapping *ColumnMapping, sheetIndex int, dryRun bool) ([]*model.Transaction, *ImportResult, error) {
	if len(data) == 0 {
		return nil, nil, fmt.Errorf("文件内容为空")
	}

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("解析Excel文件失败: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, nil, fmt.Errorf("Excel文件中没有工作表")
	}

	if sheetIndex < 0 || sheetIndex >= len(sheets) {
		sheetIndex = bestSheetIndex(f, sheets)
	}

	rows, err := f.GetRows(sheets[sheetIndex])
	if err != nil {
		return nil, nil, fmt.Errorf("读取工作表失败: %w", err)
	}

	// 若未提供列映射，从表头自动推断
	if mapping == nil {
		if len(rows) == 0 {
			return nil, nil, fmt.Errorf("工作表为空")
		}
		suggested := suggestMapping(rows[0])
		mapping = &suggested
	}

	// 检查必填字段是否已映射
	if mapping.Date < 0 {
		return nil, nil, fmt.Errorf("未找到【日期】列，请检查表头或手动指定列映射")
	}
	if mapping.Type < 0 {
		return nil, nil, fmt.Errorf("未找到【类型】列，请检查表头或手动指定列映射")
	}
	if mapping.Amount < 0 {
		return nil, nil, fmt.Errorf("未找到【金额】列，请检查表头或手动指定列映射")
	}
	if mapping.Category < 0 {
		return nil, nil, fmt.Errorf("未找到【分类】列，请检查表头或手动指定列映射")
	}

	// 构建分类名称 → Category 的映射
	categoryMap := make(map[string]*model.Category, len(categoryList))
	for _, cat := range categoryList {
		if cat != nil {
			categoryMap[cat.Name] = cat
		}
	}

	getCell := func(row []string, idx int) string {
		if idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	// colName 返回列字母标识（0→A, 1→B, 25→Z, 26→AA …）
	colName := func(idx int) string {
		if idx < 0 {
			return "?"
		}
		s := ""
		for n := idx; ; {
			s = string(rune('A'+n%26)) + s
			n = n/26 - 1
			if n < 0 {
				break
			}
		}
		return s
	}

	var transactions []*model.Transaction
	res := &ImportResult{}

	for rowIdx, row := range rows {
		lineNum := rowIdx + 1
		if rowIdx == 0 {
			continue // 跳过表头
		}

		// 跳过说明行
		if getCell(row, mapping.Date) != "" && strings.Contains(getCell(row, mapping.Date), "说明") {
			continue
		}

		// 跳过完全空行
		isEmpty := true
		for _, cell := range row {
			if strings.TrimSpace(cell) != "" {
				isEmpty = false
				break
			}
		}
		if isEmpty {
			continue
		}

		dateStr := getCell(row, mapping.Date)
		typeStr := getCell(row, mapping.Type)
		amountStr := getCell(row, mapping.Amount)
		categoryName := getCell(row, mapping.Category)
		merchant := getCell(row, mapping.Merchant)
		note := getCell(row, mapping.Note)

		// 验证日期
		if dateStr == "" {
			res.Issues = append(res.Issues, fmt.Sprintf(
				"第%d行 %s列(日期)：单元格为空 → 请填写日期，格式 YYYY-MM-DD（如 2026-01-15）",
				lineNum, colName(mapping.Date)))
			res.SkippedCount++
			continue
		}
		if _, err := time.Parse("2006-01-02", dateStr); err != nil {
			res.Issues = append(res.Issues, fmt.Sprintf(
				"第%d行 %s列(日期)：格式错误，当前值 %q → 请改为 YYYY-MM-DD 格式（如 2026-01-15）",
				lineNum, colName(mapping.Date), dateStr))
			res.SkippedCount++
			continue
		}

		// 解析类型
		var txType string
		switch typeStr {
		case "收入":
			txType = "income"
		case "支出":
			txType = "expense"
		default:
			res.Issues = append(res.Issues, fmt.Sprintf(
				"第%d行 %s列(类型)：当前值 %q 无法识别 → 请将单元格改为「收入」或「支出」",
				lineNum, colName(mapping.Type), typeStr))
			res.SkippedCount++
			continue
		}

		// 解析金额（处理全角逗号小数点和千分位逗号）
		if amountStr == "" {
			res.Issues = append(res.Issues, fmt.Sprintf(
				"第%d行 %s列(金额)：单元格为空 → 请填写金额数字（如 128.50）",
				lineNum, colName(mapping.Amount)))
			res.SkippedCount++
			continue
		}
		amount, err := parseAmount(amountStr)
		if err != nil || amount <= 0 {
			res.Issues = append(res.Issues, fmt.Sprintf(
				"第%d行 %s列(金额)：当前值 %q 无法解析 → 请确保是纯数字，去掉货币符号和非数字字符（如改为 128.50）",
				lineNum, colName(mapping.Amount), amountStr))
			res.SkippedCount++
			continue
		}

		// 查找分类（精确匹配优先，再模糊匹配）
		var categoryID uint64
		if cat, ok := categoryMap[categoryName]; ok {
			categoryID = cat.ID
		} else {
			found := false
			for name, cat := range categoryMap {
				if strings.Contains(name, categoryName) || strings.Contains(categoryName, name) {
					categoryID = cat.ID
					found = true
					break
				}
			}
			if !found {
				catNames := make([]string, 0, len(categoryMap))
				for n := range categoryMap {
					catNames = append(catNames, n)
				}
				res.Issues = append(res.Issues, fmt.Sprintf(
					"第%d行 %s列(分类)：当前值 %q 未找到 → 请改为系统已有分类之一：%s",
					lineNum, colName(mapping.Category), categoryName,
					strings.Join(catNames, "、")))
				res.SkippedCount++
				continue
			}
		}

		res.ValidCount++
		if !dryRun {
			// 拼接 Note：商户名前置
			fullNote := note
			if merchant != "" && note != "" {
				fullNote = merchant + " - " + note
			} else if merchant != "" {
				fullNote = merchant
			}

			transactions = append(transactions, &model.Transaction{
				TenantID:        tenantID,
				UserID:          userID,
				Type:            txType,
				Amount:          amount,
				CategoryID:      categoryID,
				TransactionDate: dateStr,
				Note:            fullNote,
			})
		}
	}

	return transactions, res, nil
}
