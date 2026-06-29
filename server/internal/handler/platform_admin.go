package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/service"

	"github.com/gin-gonic/gin"
)

type PlatformAdminHandler struct {
	svc    service.PlatformAdminService
	llmSvc service.LLMService
}

func NewPlatformAdminHandler(svc service.PlatformAdminService, llmSvc service.LLMService) *PlatformAdminHandler {
	return &PlatformAdminHandler{svc: svc, llmSvc: llmSvc}
}

// Login 平台管理员登录 POST /api/platform/auth/login
func (h *PlatformAdminHandler) Login(c *gin.Context) {
	var req dto.PlatformLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}

	resp, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPlatformAdminNotFound):
			c.JSON(http.StatusUnauthorized, dto.Fail(401, "账号或密码错误"))
		case errors.Is(err, service.ErrPlatformInvalidPassword):
			c.JSON(http.StatusUnauthorized, dto.Fail(401, "账号或密码错误"))
		default:
			c.JSON(http.StatusInternalServerError, dto.Fail(500, "登录失败: "+err.Error()))
		}
		return
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

// Dashboard 平台仪表盘 GET /api/platform/dashboard
func (h *PlatformAdminHandler) Dashboard(c *gin.Context) {
	resp, err := h.svc.GetDashboard(c.Request.Context())
	if err != nil {
		log.Printf("[platform] Dashboard error: %v", err)
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "获取仪表盘数据失败"))
		return
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

// ListUsers 用户列表 GET /api/platform/users
func (h *PlatformAdminHandler) ListUsers(c *gin.Context) {
	var filter dto.PlatformUserFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}
	resp, err := h.svc.ListUsers(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "获取用户列表失败"))
		return
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

// GetUserDetail 用户详情 GET /api/platform/users/:id
func (h *PlatformAdminHandler) GetUserDetail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "用户ID格式错误"))
		return
	}
	resp, err := h.svc.GetUserDetail(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, dto.Fail(404, "用户不存在"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "获取用户详情失败"))
		return
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

// GetConfigs 获取所有平台配置 GET /api/platform/configs
func (h *PlatformAdminHandler) GetConfigs(c *gin.Context) {
	resp, err := h.svc.GetAllConfigs(c.Request.Context())
	if err != nil {
		log.Printf("[platform] GetConfigs error: %v", err)
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "获取配置失败"))
		return
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

// UpdateConfig 更新单个配置 PUT /api/platform/configs/:key
func (h *PlatformAdminHandler) UpdateConfig(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "配置键不能为空"))
		return
	}

	var req dto.PlatformConfigUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}

	resp, err := h.svc.UpdateConfig(c.Request.Context(), key, req.Value)
	if err != nil {
		log.Printf("[platform] UpdateConfig error: %v", err)
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "更新配置失败"))
		return
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

// ListUserTenantCredits 查看用户下各账本积分 GET /api/platform/users/:id/tenants/credits
func (h *PlatformAdminHandler) ListUserTenantCredits(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "用户ID格式错误"))
		return
	}
	resp, err := h.svc.ListUserTenantCredits(c.Request.Context(), id)
	if err != nil {
		log.Printf("[platform] ListUserTenantCredits error: %v", err)
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "查询积分失败"))
		return
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

// GetTenantCredits 查看租户积分详情 GET /api/platform/tenants/:id/credits
func (h *PlatformAdminHandler) GetTenantCredits(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "账本ID格式错误"))
		return
	}
	resp, err := h.svc.GetTenantCredits(c.Request.Context(), id)
	if err != nil {
		log.Printf("[platform] GetTenantCredits error: %v", err)
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "查询积分失败"))
		return
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

// AdjustTenantCredits 充值/扣减积分 POST /api/platform/tenants/:id/credits/adjust
func (h *PlatformAdminHandler) AdjustTenantCredits(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "账本ID格式错误"))
		return
	}
	var req dto.CreditsAdjustReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}
	resp, err := h.svc.AdjustTenantCredits(c.Request.Context(), id, req)
	if err != nil {
		log.Printf("[platform] AdjustTenantCredits error: %v", err)
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "调整积分失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

// SetTenantUnlimited 设置/取消无限制 PUT /api/platform/tenants/:id/credits/unlimited
func (h *PlatformAdminHandler) SetTenantUnlimited(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "账本ID格式错误"))
		return
	}
	var req dto.CreditsUnlimitedReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}
	resp, err := h.svc.SetTenantUnlimited(c.Request.Context(), id, req.Unlimited)
	if err != nil {
		log.Printf("[platform] SetTenantUnlimited error: %v", err)
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "设置失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

// GetPlatformLLMConfig 获取平台 LLM 配置 GET /api/platform/llm-config
func (h *PlatformAdminHandler) GetPlatformLLMConfig(c *gin.Context) {
	resp, err := h.svc.GetPlatformLLMConfig(c.Request.Context())
	if err != nil {
		log.Printf("[platform] GetPlatformLLMConfig error: %v", err)
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "获取平台 LLM 配置失败"))
		return
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

// UpdatePlatformLLMConfig 更新平台 LLM 配置 PUT /api/platform/llm-config
func (h *PlatformAdminHandler) UpdatePlatformLLMConfig(c *gin.Context) {
	var req dto.UpdatePlatformLLMConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}
	log.Printf("[platform] UpdatePlatformLLMConfig req: enabled=%v provider=%s model=%s apiKey=%v baseURL=%q mode=%s",
		req.Enabled, req.Provider, req.Model, req.APIKey != "", req.BaseURL, req.Mode)
	resp, err := h.svc.UpdatePlatformLLMConfig(c.Request.Context(), req)
	if err != nil {
		log.Printf("[platform] UpdatePlatformLLMConfig error: %v", err)
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "更新平台 LLM 配置失败"))
		return
	}
	log.Printf("[platform] UpdatePlatformLLMConfig saved: enabled=%v", resp.Enabled)
	// 清除所有租户 LLM 客户端缓存，使新配置立即生效
	h.llmSvc.InvalidatePlatformClients()
	c.JSON(http.StatusOK, dto.OK(resp))
}

// GetTenantPrompt 获取账本识别提示词 GET /api/platform/tenants/:id/prompt
func (h *PlatformAdminHandler) GetTenantPrompt(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "账本ID格式错误"))
		return
	}
	resp, err := h.llmSvc.GetPrompt(c.Request.Context(), id)
	if err != nil {
		log.Printf("[platform] GetTenantPrompt error: %v", err)
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "获取提示词失败"))
		return
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

// UpdateTenantPrompt 修改账本识别提示词 PUT /api/platform/tenants/:id/prompt
func (h *PlatformAdminHandler) UpdateTenantPrompt(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "账本ID格式错误"))
		return
	}
	var req dto.SaveTenantPromptReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}
	resp, err := h.llmSvc.SavePrompt(c.Request.Context(), id, req)
	if err != nil {
		log.Printf("[platform] UpdateTenantPrompt error: %v", err)
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "保存提示词失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}
