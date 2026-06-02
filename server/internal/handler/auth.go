package handler

import (
	"errors"
	"net/http"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authSvc service.AuthService
}

func NewAuthHandler(authSvc service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

// Register 用户注册
// POST /api/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}

	resp, err := h.authSvc.Register(c.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmailAlreadyExists):
			c.JSON(http.StatusConflict, dto.Fail(409, err.Error()))
		case errors.Is(err, service.ErrUsernameExists):
			c.JSON(http.StatusConflict, dto.Fail(409, err.Error()))
		default:
			c.JSON(http.StatusInternalServerError, dto.Fail(500, "注册失败: "+err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, dto.OK(resp))
}

// Login 用户登录
// POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}

	resp, err := h.authSvc.Login(c.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			c.JSON(http.StatusUnauthorized, dto.Fail(401, err.Error()))
		case errors.Is(err, service.ErrInvalidPassword):
			c.JSON(http.StatusUnauthorized, dto.Fail(401, err.Error()))
		default:
			c.JSON(http.StatusInternalServerError, dto.Fail(500, "登录失败: "+err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, dto.OK(resp))
}
