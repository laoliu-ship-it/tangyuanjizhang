package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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
	ocrSvc        service.OCRService
	merchantSvc   service.MerchantService
	categorySvc   service.CategoryService
	llmSvc        service.LLMService
	creditsSvc    service.CreditsService
	storage       storage.Storage
	mediaRepo     repo.MediaFileRepo
	ocrRecordRepo repo.OcrRecordRepo
	uploadCfg     config.UploadConfig
}

func NewOCRHandler(ocrSvc service.OCRService, merchantSvc service.MerchantService, categorySvc service.CategoryService, llmSvc service.LLMService, creditsSvc service.CreditsService, store storage.Storage, mediaRepo repo.MediaFileRepo, ocrRecordRepo repo.OcrRecordRepo, uploadCfg config.UploadConfig) *OCRHandler {
	return &OCRHandler{ocrSvc: ocrSvc, merchantSvc: merchantSvc, categorySvc: categorySvc, llmSvc: llmSvc, creditsSvc: creditsSvc, storage: store, mediaRepo: mediaRepo, ocrRecordRepo: ocrRecordRepo, uploadCfg: uploadCfg}
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

	// 优先用前端传的 hash；前端未传时服务端自己算，保证去重不依赖前端
	originalHash := c.Request.FormValue("original_hash")
	if originalHash == "" {
		hasher := sha256.New()
		if _, herr := io.Copy(hasher, file); herr != nil {
			c.JSON(http.StatusInternalServerError, dto.Fail(500, "读取文件失败"))
			return
		}
		originalHash = hex.EncodeToString(hasher.Sum(nil))
		if _, serr := file.Seek(0, io.SeekStart); serr != nil {
			c.JSON(http.StatusInternalServerError, dto.Fail(500, "读取文件失败"))
			return
		}
	}
	if originalHash == "" {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "文件 hash 不能为空"))
		return
	}

	// 去重检查：hash 命中 media_file，说明已上传过，不重复扣 upload+ocr
	{
		existing, err := h.mediaRepo.GetByHash(c.Request.Context(), tenantID, originalHash)
		if err != nil {
			log.Printf("[OCR] tenant=%d media lookup error: %v", tenantID, err)
		}
		if existing != nil {
			log.Printf("[OCR] tenant=%d duplicate file detected, hash=%s, file=%s", tenantID, originalHash, existing.FileName)
			// 复用已有的 ocr_record.id，让前端可以继续调 LLM（只扣 LLM 费，不重复扣 upload+ocr）
			var existingOcrID uint64
			var existingRawTexts []string
			if ocrRec, ocrErr := h.ocrRecordRepo.GetByHash(c.Request.Context(), tenantID, originalHash); ocrErr == nil && ocrRec != nil {
				existingOcrID = ocrRec.ID
				_ = json.Unmarshal([]byte(ocrRec.RawTexts), &existingRawTexts)
			}
			c.JSON(http.StatusOK, dto.OK(dto.OCRResponse{
				OcrID:        existingOcrID,
				ImagePath:    "/" + existing.FilePath,
				AIMode:       false,
				Amount:       0,
				Date:         "",
				MerchantID:   0,
				MerchantName: "",
				RawTexts:     existingRawTexts,
				Duplicate:    true,
				FileName:     existing.FileName,
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
	{
		fileSize := header.Size
		if fileSize == 0 {
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
		m, merr := h.merchantSvc.GetOrCreateByName(c.Request.Context(), tenantID, result.Merchant)
		if merr == nil && m != nil {
			merchantID = m.ID
			merchantName = m.Name
		}
	}

	rawTexts := extractRawTextStrings(result.RawTexts)

	// 写入 ocr_records，供后续 LLM 分析使用
	rawTextsJSON, _ := json.Marshal(rawTexts)
	ocrRec := &model.OcrRecord{
		TenantID:     tenantID,
		ImagePath:    savedPath,
		ImageHash:    originalHash,
		ImageSize:    header.Size,
		AIMode:       result.AIMode,
		Amount:       result.Amount,
		Date:         result.Date,
		MerchantName: merchantName,
		RawTexts:     string(rawTextsJSON),
	}
	if err := h.ocrRecordRepo.Create(c.Request.Context(), ocrRec); err != nil {
		log.Printf("[OCR] tenant=%d failed to create ocr_record: %v", tenantID, err)
	}

	// 扣费：新文件 = upload + ocr（重复文件走 early return，不到这里）
	if deductErr := h.creditsSvc.CheckAndDeduct(c.Request.Context(), tenantID, "upload", "ocr_record", ocrRec.ID, "图片上传"); deductErr != nil {
		log.Printf("[credits] tenant=%d upload deduct error: %v", tenantID, deductErr)
		c.JSON(http.StatusOK, dto.Fail(402, "积分不足，请联系管理员充值"))
		return
	}
	if deductErr := h.creditsSvc.CheckAndDeduct(c.Request.Context(), tenantID, "ocr", "ocr_record", ocrRec.ID, "OCR识别"); deductErr != nil {
		log.Printf("[credits] tenant=%d ocr deduct error: %v", tenantID, deductErr)
		c.JSON(http.StatusOK, dto.Fail(402, "积分不足，请联系管理员充值"))
		return
	}

	c.JSON(http.StatusOK, dto.OK(dto.OCRResponse{
		OcrID:        ocrRec.ID,
		ImagePath:    "/" + savedPath,
		AIMode:       result.AIMode,
		Amount:       result.Amount,
		Date:         result.Date,
		MerchantID:   merchantID,
		MerchantName: merchantName,
		RawTexts:     rawTexts,
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
	if originalHash == "" {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "original_hash 不能为空"))
		return
	}

	// 去重检查
	{
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
				llmErrMsg = friendlyLLMError(lerr)
			} else {
				// 重复文件：仅扣 LLM 费（upload+OCR 已在首次上传时扣过）
				if deductErr := h.creditsSvc.CheckAndDeduct(c.Request.Context(), tenantID, "llm", "media_file", existing.ID, "AI整理(重复图片)"); deductErr != nil {
					log.Printf("[credits] tenant=%d llm deduct error: %v", tenantID, deductErr)
					c.JSON(http.StatusOK, dto.Fail(402, "积分不足，请联系管理员充值"))
					return
				}
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
	mf := &model.MediaFile{
		TenantID:     tenantID,
		OriginalHash: originalHash,
		FileName:     header.Filename,
		FilePath:     savedPath,
		FileSize:     header.Size,
		MimeType:     header.Header.Get("Content-Type"),
	}
	if err := h.mediaRepo.Create(c.Request.Context(), mf); err != nil {
		log.Printf("[OCR] tenant=%d failed to create media record: %v", tenantID, err)
	}

	// === OCR 识别 ===
	ocrStartTime := time.Now()
	ocrResult, err := h.ocrSvc.ProcessImage(c.Request.Context(), savedPath)
	ocrDuration := time.Since(ocrStartTime)
	if err != nil {
		log.Printf("[OCR] tenant=%d 耗时=%.2fs 失败: %v", tenantID, ocrDuration.Seconds(), err)
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "OCR识别失败: "+err.Error()))
		return
	}
	log.Printf("[OCR] tenant=%d 耗时=%.2fs 成功", tenantID, ocrDuration.Seconds())

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

	// 写入 ocr_records，供扣费关联和后续追溯使用
	rawTextsJSON, _ := json.Marshal(rawTexts)
	ocrRec := &model.OcrRecord{
		TenantID:     tenantID,
		ImagePath:    savedPath,
		ImageHash:    originalHash,
		ImageSize:    header.Size,
		AIMode:       ocrResult.AIMode,
		Amount:       ocrResult.Amount,
		Date:         ocrResult.Date,
		MerchantName: merchantName,
		RawTexts:     string(rawTextsJSON),
	}
	if err := h.ocrRecordRepo.Create(c.Request.Context(), ocrRec); err != nil {
		log.Printf("[OCR] tenant=%d failed to create ocr_record: %v", tenantID, err)
	}

	// 新文件：OCR 成功后扣 upload + ocr（失败已在上方 return，不会到这里）
	if deductErr := h.creditsSvc.CheckAndDeduct(c.Request.Context(), tenantID, "upload", "ocr_record", ocrRec.ID, "图片上传"); deductErr != nil {
		log.Printf("[credits] tenant=%d upload deduct error: %v", tenantID, deductErr)
		c.JSON(http.StatusOK, dto.Fail(402, "积分不足，请联系管理员充值"))
		return
	}
	if deductErr := h.creditsSvc.CheckAndDeduct(c.Request.Context(), tenantID, "ocr", "ocr_record", ocrRec.ID, "OCR识别"); deductErr != nil {
		log.Printf("[credits] tenant=%d ocr deduct error: %v", tenantID, deductErr)
		c.JSON(http.StatusOK, dto.Fail(402, "积分不足，请联系管理员充值"))
		return
	}

	// === LLM 分析 ===
	llmStartTime := time.Now()
	// LLM 分析（非致命，失败时记录错误继续返回 OCR 结果，LLM 失败不扣 LLM 积分）
	var llmSuggestions []*dto.LLMSuggestion
	var llmErrMsg string
	if suggestions, lerr := h.llmSvc.Analyze(c.Request.Context(), tenantID, savedPath, rawTexts, categoryItems); lerr != nil {
		llmDuration := time.Since(llmStartTime)
		log.Printf("[LLM] tenant=%d 耗时=%.2fs 失败: %v", tenantID, llmDuration.Seconds(), lerr)
		llmErrMsg = friendlyLLMError(lerr)
	} else {
		llmDuration := time.Since(llmStartTime)
		llmSuggestions = suggestions
		log.Printf("[LLM] tenant=%d 耗时=%.2fs 成功，建议数=%d", tenantID, llmDuration.Seconds(), len(suggestions))
		for i, s := range suggestions {
			log.Printf("[LLM] tenant=%d suggestion[%d]: type=%s amount=%.2f category_id=%d category_hint=%s merchant=%s date=%s",
				tenantID, i, s.Type, s.Amount, s.CategoryID, s.CategoryHint, s.MerchantName, s.Date)
		}
		// LLM 成功后才扣 llm 积分
		if deductErr := h.creditsSvc.CheckAndDeduct(c.Request.Context(), tenantID, "llm", "ocr_record", ocrRec.ID, "AI整理"); deductErr != nil {
			log.Printf("[credits] tenant=%d llm deduct error (non-fatal): %v", tenantID, deductErr)
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
