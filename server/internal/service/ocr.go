package service

import (
	"context"
	"fmt"

	"fandianjizhang/server/internal/dto"
	pkgocr "fandianjizhang/server/pkg/ocr"
)

type OCRService interface {
	ProcessImage(ctx context.Context, imagePath string) (*dto.OCRResult, error)
}

type ocrService struct {
	ocrEngine pkgocr.Engine
	aiMode    bool
}

func NewOCRService(ocrEngine pkgocr.Engine, aiMode bool) OCRService {
	return &ocrService{ocrEngine: ocrEngine, aiMode: aiMode}
}

func (s *ocrService) ProcessImage(ctx context.Context, imagePath string) (*dto.OCRResult, error) {
	result, err := s.ocrEngine.Recognize(ctx, imagePath)
	if err != nil {
		return nil, fmt.Errorf("OCR识别失败: %w", err)
	}

	rawTexts := make([]dto.OCRText, 0, len(result.RawTexts))
	for _, t := range result.RawTexts {
		rawTexts = append(rawTexts, dto.OCRText{
			Text:       t.Text,
			Confidence: t.Confidence,
		})
	}

	return &dto.OCRResult{
		AIMode:   s.aiMode,
		Amount:   result.Amount,
		Date:     result.Date,
		Merchant: result.Merchant,
		RawTexts: rawTexts,
	}, nil
}
