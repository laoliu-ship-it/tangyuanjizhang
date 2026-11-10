package handler

import (
	"time"

	"fandianjizhang/server/internal/dto"
)

func demoDate() string { return time.Now().Format("2006-01-02") }

func demoOCRResponse() dto.OCRResponse {
	return dto.OCRResponse{
		AIMode:       true,
		Amount:       128.50,
		Date:         demoDate(),
		MerchantName: "演示商户（示例数据）",
		RawTexts:     []string{"演示识别结果：午餐 128.50 元（不调用真实 OCR/付费上游）"},
		FileName:     "demo.jpg",
		Demo:         true,
	}
}

func demoAnalyzeResponse() dto.OCRAnalyzeResponse {
	return dto.OCRAnalyzeResponse{
		AIMode:       true,
		Amount:       128.50,
		Date:         demoDate(),
		MerchantName: "演示商户（示例数据）",
		RawTexts:     []string{"演示识别结果：午餐 128.50 元（不调用真实 OCR/付费上游）"},
		LLM:          demoLLMSuggestions(),
		FileName:     "demo.jpg",
		Demo:         true,
	}
}

func demoLLMSuggestions() []*dto.LLMSuggestion {
	return []*dto.LLMSuggestion{{
		Type:         "expense",
		Amount:       128.50,
		MerchantName: "演示商户（示例数据）",
		Date:         demoDate(),
		CategoryHint: "餐饮",
		Note:         "演示：AI 分析结果不访问真实 LLM",
		SourceLines:  []int{0},
	}}
}
