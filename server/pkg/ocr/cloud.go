package ocr

import "context"

// CloudEngine 云 OCR 引擎（预留接口，未实现）
type CloudEngine struct{}

// Recognize 云 OCR 识别（未实现）
func (e *CloudEngine) Recognize(_ context.Context, _ string) (*Result, error) {
	return nil, ErrNotImplemented
}
