package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/middleware"
	"fandianjizhang/server/internal/repo"
	"fandianjizhang/server/internal/service"

	"github.com/gin-gonic/gin"
)

type LLMHandler struct {
	llmSvc        service.LLMService
	categorySvc   service.CategoryService
	ocrRecordRepo repo.OcrRecordRepo
}

func NewLLMHandler(llmSvc service.LLMService, categorySvc service.CategoryService, ocrRecordRepo repo.OcrRecordRepo) *LLMHandler {
	return &LLMHandler{llmSvc: llmSvc, categorySvc: categorySvc, ocrRecordRepo: ocrRecordRepo}
}

// GetConfig 获取租户 LLM 配置
// GET /api/llm/config
func (h *LLMHandler) GetConfig(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	resp, err := h.llmSvc.GetConfig(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "获取 LLM 配置失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

// SaveConfig 保存租户 LLM 配置（管理员操作）
// PUT /api/llm/config
func (h *LLMHandler) SaveConfig(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	var req dto.SaveTenantLLMConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}

	resp, err := h.llmSvc.SaveConfig(c.Request.Context(), tenantID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "保存 LLM 配置失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

// Analyze 通过 ocr_id 调用 LLM 分析（OCR 完成后调用）
// POST /api/llm/analyze
func (h *LLMHandler) Analyze(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	var req dto.LLMAnalyzeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}

	// 从 ocr_records 取内容，同时校验 tenant_id 防止跨租户偷 token
	ocrRec, err := h.ocrRecordRepo.GetByID(c.Request.Context(), req.OcrID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "查询 OCR 记录失败: "+err.Error()))
		return
	}
	if ocrRec == nil {
		c.JSON(http.StatusForbidden, dto.Fail(403, "OCR 记录不存在或无权限访问"))
		return
	}

	// 解析存储的 raw_texts
	var rawTexts []string
	if ocrRec.RawTexts != "" {
		_ = json.Unmarshal([]byte(ocrRec.RawTexts), &rawTexts)
	}

	// 从数据库加载分类列表
	cats, cerr := h.categorySvc.List(c.Request.Context(), tenantID)
	if cerr != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "获取分类失败: "+cerr.Error()))
		return
	}
	categories := make([]dto.CategoryItem, 0, len(cats))
	for _, cat := range cats {
		categories = append(categories, dto.CategoryItem{ID: cat.ID, Name: cat.Name, Type: cat.Type})
	}

	// 调用 LLM 分析
	llmStartTime := time.Now()
	suggestions, err := h.llmSvc.Analyze(c.Request.Context(), tenantID, ocrRec.ImagePath, rawTexts, categories)
	llmDuration := time.Since(llmStartTime)

	if err != nil {
		log.Printf("[LLM] tenant=%d ocr_id=%d 耗时=%.2fs 失败: %v", tenantID, req.OcrID, llmDuration.Seconds(), err)
		c.JSON(http.StatusOK, dto.OK(dto.LLMAnalyzeResp{
			Suggestions: nil,
			Error:       friendlyLLMError(err),
		}))
		return
	}

	log.Printf("[LLM] tenant=%d ocr_id=%d 耗时=%.2fs 成功，建议数=%d", tenantID, req.OcrID, llmDuration.Seconds(), len(suggestions))
	c.JSON(http.StatusOK, dto.OK(dto.LLMAnalyzeResp{
		Suggestions: suggestions,
	}))
}

// friendlyLLMError 将 LLM 技术性错误转换为用户友好的中文提示
func friendlyLLMError(err error) string {
	msg := err.Error()

	// 检测视觉模式不支持的情况
	if containsStr(msg, "unknown variant") || containsStr(msg, "image_url") || containsStr(msg, "binary") {
		return "当前模型不支持图片视觉识别，请在设置-AI设置-识别模式改为文字模式（将 OCR 文本发给 AI）或更换为支持视觉的模型"
	}

	// 检测 API 调用失败
	if containsStr(msg, "400") || containsStr(msg, "401") || containsStr(msg, "403") {
		return "LLM API 调用失败，请检查 API Key 和模型配置是否正确"
	}

	if containsStr(msg, "timeout") || containsStr(msg, "超时") {
		return "LLM 分析超时，请稍后重试"
	}

	// 默认提示
	return "AI 分析失败：" + msg
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
