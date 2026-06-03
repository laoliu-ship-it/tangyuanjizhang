package handler

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"fandianjizhang/server/config"
	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/middleware"
	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/repo"
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
	mediaRepo   repo.MediaFileRepo
	uploadCfg   config.UploadConfig
}

func NewOCRHandler(ocrSvc service.OCRService, merchantSvc service.MerchantService, categorySvc service.CategoryService, llmSvc service.LLMService, store storage.Storage, mediaRepo repo.MediaFileRepo, uploadCfg config.UploadConfig) *OCRHandler {
	return &OCRHandler{ocrSvc: ocrSvc, merchantSvc: merchantSvc, categorySvc: categorySvc, llmSvc: llmSvc, storage: store, mediaRepo: mediaRepo, uploadCfg: uploadCfg}
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

	if header.Size > h.uploadCfg.MaxSize {
		c.JSON(http.StatusBadRequest, dto.Fail(400, fmt.Sprintf("图片文件过大，最大支持 %dMB", h.uploadCfg.MaxSize/1024/1024)))
		return
	}

	originalHash := c.Request.FormValue("original_hash")

	// 去重检查：如果提供了 hash，先查是否已存在
	if originalHash != "" {
		existing, err := h.mediaRepo.GetByHash(c.Request.Context(), tenantID, originalHash)
		if err != nil {
			log.Printf("[OCR] tenant=%d media lookup error: %v", tenantID, err)
		}
		if existing != nil {
			// 文件已存在，直接返回已有信息
			log.Printf("[OCR] tenant=%d duplicate file detected, hash=%s, file=%s", tenantID, originalHash, existing.FileName)
			c.JSON(http.StatusOK, dto.OK(gin.H{
				"image_path":    "/" + existing.FilePath,
				"ai_mode":       false,
				"amount":        float64(0),
				"date":          "",
				"merchant_id":   0,
				"merchant_name": "",
				"raw_texts":     []string{},
				"duplicate":     true,
				"file_name":     existing.FileName,
			}))
			return
		}
	}

	date := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("%d_ocr%s", time.Now().UnixNano(), ext)
	savedPath, err := h.storage.Save(tenantID, date, filename, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "保存图片失败: "+err.Error()))
		return
	}

	// 记录媒体文件
	if originalHash != "" {
		fileSize := header.Size
		if fileSize == 0 {
			// 如果 FormFile header 没有 size，尝试从文件头推断
			fileSize = -1
		}
		mf := &model.MediaFile{
			TenantID:     tenantID,
			OriginalHash: originalHash,
			FileName:     header.Filename,
			FilePath:     savedPath,
			FileSize:     fileSize,
			MimeType:     header.Header.Get("Content-Type"),
		}
		if err := h.mediaRepo.Create(c.Request.Context(), mf); err != nil {
			log.Printf("[OCR] tenant=%d failed to create media record: %v", tenantID, err)
		}
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

	if header.Size > h.uploadCfg.MaxSize {
		c.JSON(http.StatusBadRequest, dto.Fail(400, fmt.Sprintf("图片文件过大，最大支持 %dMB", h.uploadCfg.MaxSize/1024/1024)))
		return
	}

	originalHash := c.Request.FormValue("original_hash")

	// 去重检查
	if originalHash != "" {
		existing, err := h.mediaRepo.GetByHash(c.Request.Context(), tenantID, originalHash)
		if err != nil {
			log.Printf("[OCR] tenant=%d media lookup error: %v", tenantID, err)
		}
		if existing != nil {
			log.Printf("[OCR] tenant=%d duplicate file detected (analyze), hash=%s, file=%s", tenantID, originalHash, existing.FileName)
			// 对已存在文件也做 LLM 分析
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
			}

			rawTexts := []string{}
			var llmSuggestions []*dto.LLMSuggestion
			var llmErrMsg string
			if suggestions, lerr := h.llmSvc.Analyze(c.Request.Context(), tenantID, existing.FilePath, rawTexts, categoryItems); lerr != nil {
				log.Printf("[LLM] tenant=%d analyze error: %v", tenantID, lerr)
				llmErrMsg = lerr.Error()
			} else {
				llmSuggestions = suggestions
			}

			c.JSON(http.StatusOK, dto.OK(dto.OCRAnalyzeResponse{
				ImagePath:    "/" + existing.FilePath,
				AIMode:       false,
				Amount:       0,
				Date:         "",
				MerchantID:   0,
				MerchantName: "",
				RawTexts:     rawTexts,
				LLM:          llmSuggestions,
				LLMError:     llmErrMsg,
				Duplicate:    true,
				FileName:     existing.FileName,
			}))
			return
		}
	}

	date := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("%d_analyze%s", time.Now().UnixNano(), ext)
	savedPath, err := h.storage.Save(tenantID, date, filename, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "保存图片失败: "+err.Error()))
		return
	}

	// 记录媒体文件
	if originalHash != "" {
		fileSize := header.Size
		mf := &model.MediaFile{
			TenantID:     tenantID,
			OriginalHash: originalHash,
			FileName:     header.Filename,
			FilePath:     savedPath,
			FileSize:     fileSize,
			MimeType:     header.Header.Get("Content-Type"),
		}
		if err := h.mediaRepo.Create(c.Request.Context(), mf); err != nil {
			log.Printf("[OCR] tenant=%d failed to create media record: %v", tenantID, err)
		}
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
