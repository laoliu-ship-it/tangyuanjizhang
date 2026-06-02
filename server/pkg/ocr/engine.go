package ocr

import (
	"context"
	"errors"
)

// OCRText 单条 OCR 识别文本
type OCRText struct {
	Text       string
	Confidence float64
}

// Result OCR 识别结果（解析后的结构化数据）
type Result struct {
	Amount   float64
	Date     string
	Merchant string
	RawTexts []OCRText
}

// Engine OCR 引擎接口
type Engine interface {
	Recognize(ctx context.Context, imagePath string) (*Result, error)
}

// ErrNotImplemented 表示该 OCR 引擎未实现
var ErrNotImplemented = errors.New("ocr engine not implemented")
