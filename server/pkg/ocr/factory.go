package ocr

import (
	"fmt"

	"fandianjizhang/server/config"
)

// NewEngine 根据配置创建对应的 OCR 引擎（带内容哈希缓存）
func NewEngine(cfg *config.Config) (Engine, error) {
	var inner Engine
	switch cfg.OCR.Engine {
	case "rapidocr", "":
		if cfg.OCR.RapidOCRURL == "" {
			return nil, fmt.Errorf("OCR_RAPIDOCR_URL 不能为空")
		}
		inner = NewRapidOCREngine(cfg.OCR.RapidOCRURL, cfg.OCR.AIMode)
	case "cloud":
		inner = &CloudEngine{}
	default:
		return nil, fmt.Errorf("未知的 OCR 引擎类型: %s", cfg.OCR.Engine)
	}
	return NewCachedEngine(inner), nil
}
