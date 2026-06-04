package ocr

import (
	"fmt"

	"fandianjizhang/server/config"
	"fandianjizhang/server/internal/repo"
)

// NewEngine 根据配置创建对应的 OCR 引擎（带可配置缓存）
func NewEngine(cfg *config.Config, configRepo repo.PlatformConfigRepo) (Engine, error) {
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
	return NewCachedEngine(inner, configRepo), nil
}
