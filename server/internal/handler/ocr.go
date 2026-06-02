package handler

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/middleware"
	"fandianjizhang/server/internal/service"
	"fandianjizhang/server/pkg/storage"

	"github.com/gin-gonic/gin"
)

type OCRHandler struct {
	ocrSvc      service.OCRService
	merchantSvc service.MerchantService
	categorySvc service.CategoryService
	llmSvc      service.LLMService
	storage     storage.Storage
}

func NewOCRHandler(ocrSvc service.OCRService, merchantSvc service.MerchantService, categorySvc service.CategoryService, llmSvc service.LLMService, store storage.Storage) *OCRHandler {
	return &OCRHandler{ocrSvc: ocrSvc, merchantSvc: merchantSvc, categorySvc: categorySvc, llmSvc: llmSvc, storage: store}
}

// Upload 上传图片并进行 OCR 识别
// POST /api/upload/ocr
func (h *OCRHandler) Upload(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "请上传 file 文件"))
		return
	}
	defer file.Close()

	ext := filepath.Ext(header.Filename)
	allowedExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true,
		".bmp": true, ".webp": true,
	}
	if !allowedExts[ext] {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "不支持的图片格式，请上传 jpg/jpeg/png/bmp/webp"))
		return
	}

	date := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("%d_ocr%s", time.Now().UnixNano(), ext)
	savedPath, err := h.storage.Save(tenantID, date, filename, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "保存图片失败: "+err.Error()))
		return
	}

	result, err := h.ocrSvc.ProcessImage(c.Request.Context(), savedPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "OCR识别失败: "+err.Error()))
		return
	}

	var merchantID uint64 = 0
	var merchantName string = ""
	if result.Merchant != "" {
		m, err := h.merchantSvc.GetOrCreateByName(c.Request.Context(), tenantID, result.Merchant)
		if err == nil && m != nil {
			merchantID = m.ID
			merchantName = m.Name
		}
	}

	c.JSON(http.StatusOK, dto.OK(gin.H{
		"image_path":    "/" + savedPath,
		"ai_mode":       result.AIMode,
		"amount":        result.Amount,
		"date":          result.Date,
		"merchant_id":   merchantID,
		"merchant_name": merchantName,
		"raw_texts":     extractRawTextStrings(result.RawTexts),
	}))
}

// Analyze 上传图片，进行 OCR 识别 + LLM 分析，返回记账建议
// POST /api/upload/ocr/analyze
func (h *OCRHandler) Analyze(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "请上传 file 文件"))
		return
	}
	defer file.Close()

	ext := filepath.Ext(header.Filename)
	allowedExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true,
		".bmp": true, ".webp": true,
	}
	if !allowedExts[ext] {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "不支持的图片格式，请上传 jpg/jpeg/png/bmp/webp"))
		return
	}

	date := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("%d_analyze%s", time.Now().UnixNano(), ext)
	savedPath, err := h.storage.Save(tenantID, date, filename, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "保存图片失败: "+err.Error()))
		return
	}

	ocrResult, err := h.ocrSvc.ProcessImage(c.Request.Context(), savedPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "OCR识别失败: "+err.Error()))
		return
	}

	var merchantID uint64
	var merchantName string
	if ocrResult.Merchant != "" {
		m, merr := h.merchantSvc.GetOrCreateByName(c.Request.Context(), tenantID, ocrResult.Merchant)
		if merr == nil && m != nil {
			merchantID = m.ID
			merchantName = m.Name
		}
	}

	// 查询租户分类列表传给 LLM，让其直接匹配 category_id
	var categoryItems []dto.CategoryItem
	if cats, cerr := h.categorySvc.List(c.Request.Context(), tenantID); cerr == nil {
		categoryItems = make([]dto.CategoryItem, 0, len(cats))
		for _, cat := range cats {
			categoryItems = append(categoryItems, dto.CategoryItem{
				ID:   cat.ID,
				Name: cat.Name,
				Type: cat.Type,
			})
		}
		log.Printf("[LLM] tenant=%d passing %d categories to LLM", tenantID, len(categoryItems))
	} else {
		log.Printf("[LLM] tenant=%d failed to load categories: %v", tenantID, cerr)
	}

	rawTexts := extractRawTextStrings(ocrResult.RawTexts)

	// LLM 分析（非致命，失败时记录错误继续返回 OCR 结果）
	var llmSuggestions []*dto.LLMSuggestion
	var llmErrMsg string
	if suggestions, lerr := h.llmSvc.Analyze(c.Request.Context(), tenantID, savedPath, rawTexts, categoryItems); lerr != nil {
		log.Printf("[LLM] tenant=%d analyze error: %v", tenantID, lerr)
		llmErrMsg = lerr.Error()
	} else {
		llmSuggestions = suggestions
		for i, s := range suggestions {
			log.Printf("[LLM] tenant=%d suggestion[%d]: type=%s amount=%.2f category_id=%d category_hint=%s merchant=%s date=%s",
				tenantID, i, s.Type, s.Amount, s.CategoryID, s.CategoryHint, s.MerchantName, s.Date)
		}
	}

	c.JSON(http.StatusOK, dto.OK(dto.OCRAnalyzeResponse{
		ImagePath:    "/" + savedPath,
		AIMode:       ocrResult.AIMode,
		Amount:       ocrResult.Amount,
		Date:         ocrResult.Date,
		MerchantID:   merchantID,
		MerchantName: merchantName,
		RawTexts:     rawTexts,
		LLM:          llmSuggestions,
		LLMError:     llmErrMsg,
	}))
}

func extractRawTextStrings(texts []dto.OCRText) []string {
	if len(texts) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(texts))
	for _, t := range texts {
		result = append(result, t.Text)
	}
	return result
}
