package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// RapidOCREngine 通过 HTTP 调用 RapidOCR Python 微服务
type RapidOCREngine struct {
	serviceURL string
	client     *http.Client
	aiMode     bool
}

// NewRapidOCREngine 创建 RapidOCR 引擎实例
func NewRapidOCREngine(serviceURL string, aiMode bool) *RapidOCREngine {
	return &RapidOCREngine{
		serviceURL: serviceURL,
		aiMode:     aiMode,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type rapidOCRResponse struct {
	Texts []struct {
		Text       string  `json:"text"`
		Confidence float64 `json:"confidence"`
		Y          float64 `json:"y"`
		X          float64 `json:"x"`
	} `json:"texts"`
}

// Recognize 调用 RapidOCR 服务识别图片并解析结构化数据
func (e *RapidOCREngine) Recognize(ctx context.Context, imagePath string) (*Result, error) {
	// 读取图片文件
	f, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("打开图片文件失败: %w", err)
	}
	defer f.Close()

	// 构建 multipart 请求
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(imagePath))
	if err != nil {
		return nil, fmt.Errorf("创建 multipart 表单失败: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, fmt.Errorf("写入图片数据失败: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("关闭 multipart writer 失败: %w", err)
	}

	// 发送 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.serviceURL, &body)
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用 OCR 服务失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OCR 服务返回错误状态 %d: %s", resp.StatusCode, string(respBody))
	}

	var ocrResp rapidOCRResponse
	if err := json.NewDecoder(resp.Body).Decode(&ocrResp); err != nil {
		return nil, fmt.Errorf("解析 OCR 响应失败: %w", err)
	}

	// 按视觉行分组，同行内用空格拼接，前端再按空格切成可选块
	lineStrs := groupTextsByLine(ocrResp.Texts)
	rawTexts := make([]OCRText, 0, len(lineStrs))
	for _, line := range lineStrs {
		rawTexts = append(rawTexts, OCRText{Text: line, Confidence: 1.0})
	}

	// 用于结构化提取的逐项文本
	textStrs := make([]string, 0, len(ocrResp.Texts))
	for _, t := range ocrResp.Texts {
		textStrs = append(textStrs, t.Text)
	}

	// 构建结果：非 AI 模式只返回原始文本
	result := &Result{
		RawTexts: rawTexts,
	}
	if e.aiMode {
		result.Amount = extractAmount(textStrs)
		result.Date = extractDate(textStrs)
		result.Merchant = extractMerchant(ocrResp.Texts)
	}

	return result, nil
}

var (
	// 金额正则：¥数字 或 合计数字
	amountPatternYuan  = regexp.MustCompile(`[¥￥]\s*(\d+\.?\d*)`)
	amountPatternTotal = regexp.MustCompile(`合计\s*[：:]?\s*(\d+\.?\d*)`)
	amountPatternNum   = regexp.MustCompile(`(\d{1,6}\.\d{2})`) // 形如 xxx.xx 的金额
	// 日期正则：带时间（含「日」可选）、仅日期两种
	datetimePattern = regexp.MustCompile(`(\d{4})[年\-/](\d{1,2})[月\-/](\d{1,2})[日]?\s*(\d{1,2}):(\d{2})`)
	datePattern     = regexp.MustCompile(`(\d{4})[年\-/](\d{1,2})[月\-/](\d{1,2})`)
)

// extractAmount 从文本中提取金额，取最大值
func extractAmount(texts []string) float64 {
	var maxAmount float64
	combined := strings.Join(texts, " ")

	tryParse := func(s string) {
		s = strings.TrimSpace(s)
		v, err := strconv.ParseFloat(s, 64)
		if err == nil && v > maxAmount {
			maxAmount = v
		}
	}

	// 匹配 ¥数字
	for _, m := range amountPatternYuan.FindAllStringSubmatch(combined, -1) {
		if len(m) > 1 {
			tryParse(m[1])
		}
	}

	// 匹配 合计数字
	for _, m := range amountPatternTotal.FindAllStringSubmatch(combined, -1) {
		if len(m) > 1 {
			tryParse(m[1])
		}
	}

	// 如果没找到，尝试找 xxx.xx 格式的数字
	if maxAmount == 0 {
		for _, m := range amountPatternNum.FindAllStringSubmatch(combined, -1) {
			if len(m) > 1 {
				tryParse(m[1])
			}
		}
	}

	return maxAmount
}

// extractDate 从文本中提取日期，有时间则一并返回（格式 YYYY-MM-DDTHH:mm）
func extractDate(texts []string) string {
	combined := strings.Join(texts, " ")
	pad := func(s string) string {
		if len(s) == 1 {
			return "0" + s
		}
		return s
	}
	// 优先匹配带时间
	if m := datetimePattern.FindStringSubmatch(combined); len(m) >= 6 {
		return fmt.Sprintf("%s-%s-%sT%s:%s", m[1], pad(m[2]), pad(m[3]), pad(m[4]), m[5])
	}
	// 退而仅匹配日期
	if m := datePattern.FindStringSubmatch(combined); len(m) >= 4 {
		return fmt.Sprintf("%s-%s-%s", m[1], pad(m[2]), pad(m[3]))
	}
	return ""
}

// extractMerchant 取第一行置信度最高的文本作为商户名
func extractMerchant(texts []struct {
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
	Y          float64 `json:"y"`
	X          float64 `json:"x"`
}) string {
	if len(texts) == 0 {
		return ""
	}
	// 取前3条中置信度最高的
	limit := 3
	if len(texts) < limit {
		limit = len(texts)
	}
	best := texts[0]
	for i := 1; i < limit; i++ {
		if texts[i].Confidence > best.Confidence {
			best = texts[i]
		}
	}
	return best.Text
}

// groupTextsByLine 按 y 坐标将识别结果分组为行，同行内按 x 从左到右排列
func groupTextsByLine(texts []struct {
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
	Y          float64 `json:"y"`
	X          float64 `json:"x"`
}) []string {
	if len(texts) == 0 {
		return nil
	}

	// 按 Y 排序（从上到下）
	sorted := make([]struct {
		Text string
		Y    float64
		X    float64
	}, len(texts))
	for i, t := range texts {
		sorted[i] = struct {
			Text string
			Y    float64
			X    float64
		}{t.Text, t.Y, t.X}
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Y < sorted[j].Y })

	// 动态阈值：用第 25 百分位间距 × 3 区分行内抖动与行间距
	// median 容易被大量行内小间距拉低，导致 fallback 过大；p25 更稳定
	threshold := 8.0
	if len(sorted) > 1 {
		var gaps []float64
		for i := 1; i < len(sorted); i++ {
			g := sorted[i].Y - sorted[i-1].Y
			if g > 0 {
				gaps = append(gaps, g)
			}
		}
		if len(gaps) > 0 {
			sort.Float64s(gaps)
			p25 := gaps[len(gaps)/4]
			threshold = math.Max(p25*3, 5)
		}
	}

	type item struct{ text string; x float64 }
	var lines [][]item
	var cur []item
	curY := sorted[0].Y

	for _, t := range sorted {
		if math.Abs(t.Y-curY) <= threshold {
			cur = append(cur, item{t.Text, t.X})
		} else {
			if len(cur) > 0 {
				sort.Slice(cur, func(i, j int) bool { return cur[i].x < cur[j].x })
				lines = append(lines, cur)
			}
			cur = []item{{t.Text, t.X}}
			curY = t.Y
		}
	}
	if len(cur) > 0 {
		sort.Slice(cur, func(i, j int) bool { return cur[i].x < cur[j].x })
		lines = append(lines, cur)
	}

	result := make([]string, 0, len(lines))
	for _, line := range lines {
		parts := make([]string, 0, len(line))
		for _, it := range line {
			parts = append(parts, it.text)
		}
		result = append(result, strings.Join(parts, " "))
	}
	return result
}
