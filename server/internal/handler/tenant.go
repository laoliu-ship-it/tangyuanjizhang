package handler

import (
	"errors"
	"net/http"
	"strconv"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/middleware"
	"fandianjizhang/server/internal/service"

	"github.com/gin-gonic/gin"
)

type TenantHandler struct {
	tenantSvc service.TenantService
}

func NewTenantHandler(tenantSvc service.TenantService) *TenantHandler {
	return &TenantHandler{tenantSvc: tenantSvc}
}

// List 获取当前用户的租户列表
// GET /api/tenants
func (h *TenantHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	tenants, err := h.tenantSvc.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "获取租户列表失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(tenants))
}

// Create 创建租户
// POST /api/tenants
func (h *TenantHandler) Create(c *gin.Context) {
	var req dto.CreateTenantReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}

	userID := middleware.GetUserID(c)
	tenant, err := h.tenantSvc.Create(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "创建租户失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(tenant))
}

// Update 更新租户信息
// PUT /api/tenants/:id
func (h *TenantHandler) Update(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "租户ID格式错误"))
		return
	}

	var req dto.UpdateTenantReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}

	userID := middleware.GetUserID(c)
	tenant, err := h.tenantSvc.Update(c.Request.Context(), tenantID, userID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTenantNotFound):
			c.JSON(http.StatusNotFound, dto.Fail(404, err.Error()))
		case errors.Is(err, service.ErrPermissionDenied):
			c.JSON(http.StatusForbidden, dto.Fail(403, err.Error()))
		default:
			c.JSON(http.StatusInternalServerError, dto.Fail(500, "更新租户失败: "+err.Error()))
		}
		return
	}
	c.JSON(http.StatusOK, dto.OK(tenant))
}

// InviteMember 邀请成员
// POST /api/tenants/:id/members
func (h *TenantHandler) InviteMember(c *gin.Context) {
	tenantIDStr := c.Param("id")
	tenantID, err := strconv.ParseUint(tenantIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "租户ID格式错误"))
		return
	}

	// 验证用户是该租户的成员（Tenant 中间件已校验，此处确保路径与 header 一致）
	headerTenantID := middleware.GetTenantID(c)
	if headerTenantID != tenantID {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "租户ID不匹配"))
		return
	}

	var req dto.InviteMemberReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}

	if err := h.tenantSvc.InviteMember(c.Request.Context(), tenantID, req); err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			c.JSON(http.StatusNotFound, dto.Fail(404, err.Error()))
		default:
			c.JSON(http.StatusInternalServerError, dto.Fail(500, "邀请成员失败: "+err.Error()))
		}
		return
	}
	c.JSON(http.StatusOK, dto.OK(nil))
}

// RemoveMember 移除成员
// DELETE /api/tenants/:id/members/:userId
func (h *TenantHandler) RemoveMember(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	targetUserID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "用户ID格式错误"))
		return
	}

	if err := h.tenantSvc.RemoveMember(c.Request.Context(), tenantID, targetUserID); err != nil {
		switch {
		case errors.Is(err, service.ErrTenantNotFound):
			c.JSON(http.StatusNotFound, dto.Fail(404, err.Error()))
		case errors.Is(err, service.ErrCannotRemoveOwner):
			c.JSON(http.StatusBadRequest, dto.Fail(400, err.Error()))
		default:
			c.JSON(http.StatusInternalServerError, dto.Fail(500, "移除成员失败: "+err.Error()))
		}
		return
	}
	c.JSON(http.StatusOK, dto.OK(nil))
}

// UpdateMemberRole 修改成员角色
// PUT /api/tenants/:id/members/:userId
func (h *TenantHandler) UpdateMemberRole(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	targetUserID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "用户ID格式错误"))
		return
	}

	var req dto.UpdateMemberRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}

	if err := h.tenantSvc.UpdateMemberRole(c.Request.Context(), tenantID, targetUserID, req.Role); err != nil {
		switch {
		case errors.Is(err, service.ErrMemberNotFound):
			c.JSON(http.StatusNotFound, dto.Fail(404, err.Error()))
		default:
			c.JSON(http.StatusBadRequest, dto.Fail(400, err.Error()))
		}
		return
	}
	c.JSON(http.StatusOK, dto.OK(nil))
}

// ListMembers 获取成员列表
// GET /api/tenants/:id/members
func (h *TenantHandler) ListMembers(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	members, err := h.tenantSvc.ListMembers(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "获取成员列表失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(members))
}
