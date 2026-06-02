package handler

import (
	"net/http"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/middleware"
	"fandianjizhang/server/internal/service"

	"github.com/gin-gonic/gin"
)

type LLMHandler struct {
	llmSvc service.LLMService
}

func NewLLMHandler(llmSvc service.LLMService) *LLMHandler {
	return &LLMHandler{llmSvc: llmSvc}
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
