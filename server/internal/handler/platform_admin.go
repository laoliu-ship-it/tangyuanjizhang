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
	svc service.PlatformAdminService
}

func NewPlatformAdminHandler(svc service.PlatformAdminService) *PlatformAdminHandler {
	return &PlatformAdminHandler{svc: svc}
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
