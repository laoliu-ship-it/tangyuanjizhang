package handler

import (
	"net/http"
	"strconv"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/service"

	"github.com/gin-gonic/gin"
)

type RBACHandler struct {
	svc service.RBACService
}

func NewRBACHandler(svc service.RBACService) *RBACHandler {
	return &RBACHandler{svc: svc}
}

func (h *RBACHandler) ListPermissions(c *gin.Context) {
	c.JSON(http.StatusOK, dto.OK(h.svc.AvailablePermissions()))
}

func (h *RBACHandler) ListRoles(c *gin.Context) {
	tenantID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if tenantID == 0 {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "租户ID格式错误"))
		return
	}

	roles, err := h.svc.ListRoles(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "服务器内部错误"))
		return
	}
	c.JSON(http.StatusOK, dto.OK(roles))
}

func (h *RBACHandler) CreateRole(c *gin.Context) {
	tenantID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if tenantID == 0 {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "租户ID格式错误"))
		return
	}

	var req dto.CreateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, err.Error()))
		return
	}

	role, err := h.svc.CreateRole(c.Request.Context(), tenantID, req)
	if err != nil {
		if err == service.ErrRoleAlreadyName {
			c.JSON(http.StatusBadRequest, dto.Fail(400, err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "服务器内部错误"))
		return
	}
	c.JSON(http.StatusOK, dto.OK(role))
}

func (h *RBACHandler) UpdateRole(c *gin.Context) {
	tenantID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	roleID, _ := strconv.ParseUint(c.Param("roleId"), 10, 64)
	if tenantID == 0 || roleID == 0 {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数格式错误"))
		return
	}

	var req dto.UpdateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, err.Error()))
		return
	}

	err := h.svc.UpdateRolePermissions(c.Request.Context(), tenantID, roleID, req.Permissions)
	if err != nil {
		switch err {
		case service.ErrRoleNotFound:
			c.JSON(http.StatusNotFound, dto.Fail(404, err.Error()))
		case service.ErrRoleIsSystem:
			c.JSON(http.StatusBadRequest, dto.Fail(400, err.Error()))
		default:
			c.JSON(http.StatusInternalServerError, dto.Fail(500, "服务器内部错误"))
		}
		return
	}
	c.JSON(http.StatusOK, dto.OK(nil))
}

func (h *RBACHandler) DeleteRole(c *gin.Context) {
	tenantID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	roleID, _ := strconv.ParseUint(c.Param("roleId"), 10, 64)
	if tenantID == 0 || roleID == 0 {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数格式错误"))
		return
	}

	err := h.svc.DeleteRole(c.Request.Context(), tenantID, roleID)
	if err != nil {
		switch err {
		case service.ErrRoleNotFound:
			c.JSON(http.StatusNotFound, dto.Fail(404, err.Error()))
		case service.ErrRoleIsSystem:
			c.JSON(http.StatusBadRequest, dto.Fail(400, err.Error()))
		default:
			c.JSON(http.StatusInternalServerError, dto.Fail(500, "服务器内部错误"))
		}
		return
	}
	c.JSON(http.StatusOK, dto.OK(nil))
}
