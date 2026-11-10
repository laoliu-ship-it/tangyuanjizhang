package handler

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authSvc  service.AuthService
	demoMu   sync.Mutex
	demoHits map[string][]time.Time
}

func NewAuthHandler(authSvc service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, demoHits: map[string][]time.Time{}}
}

// DemoConfig 返回体验入口是否开启（登录页据此展示按钮）。
func (h *AuthHandler) DemoConfig(c *gin.Context) {
	c.JSON(http.StatusOK, dto.OK(dto.DemoConfigResp{
		Enabled: h.authSvc.DemoEnabled(), TTLMinutes: h.authSvc.DemoTTLMinutes(),
	}))
}

// DemoLogin 一键体验登录：固定演示账号，短期 token，不赠送积分/不创建新用户。
func (h *AuthHandler) DemoLogin(c *gin.Context) {
	if !h.authSvc.DemoEnabled() {
		c.JSON(http.StatusOK, dto.Fail(403, service.ErrDemoDisabled.Error()))
		return
	}
	if !h.allowDemo(c.ClientIP()) {
		c.JSON(http.StatusTooManyRequests, dto.Fail(429, "体验登录过于频繁，请稍后再试"))
		return
	}
	resp, err := h.authSvc.DemoLogin(c.Request.Context())
	if err != nil {
		if errors.Is(err, service.ErrDemoDisabled) {
			c.JSON(http.StatusOK, dto.Fail(403, err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "体验登录失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

func (h *AuthHandler) allowDemo(ip string) bool {
	h.demoMu.Lock()
	defer h.demoMu.Unlock()
	now := time.Now()
	hits := h.demoHits[ip][:0]
	for _, t := range h.demoHits[ip] {
		if now.Sub(t) < time.Minute {
			hits = append(hits, t)
		}
	}
	if len(hits) >= 5 {
		h.demoHits[ip] = hits
		return false
	}
	h.demoHits[ip] = append(hits, now)
	return true
}

// Register 用户注册
// POST /api/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}
	req.RegisterIP = c.ClientIP()

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

// WeChatLogin 微信小程序登录
// POST /api/auth/wechat/login
func (h *AuthHandler) WeChatLogin(c *gin.Context) {
	var req dto.WeChatLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}

	resp, err := h.authSvc.WeChatLogin(c.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrWeChatNotConfigured):
			c.JSON(http.StatusServiceUnavailable, dto.Fail(503, err.Error()))
		case errors.Is(err, service.ErrWeChatCodeInvalid):
			c.JSON(http.StatusBadRequest, dto.Fail(400, err.Error()))
		default:
			c.JSON(http.StatusInternalServerError, dto.Fail(500, "微信登录失败: "+err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, dto.OK(resp))
}

// WeChatBind 微信登录后关联已有邮箱账号
// POST /api/auth/wechat/bind
func (h *AuthHandler) WeChatBind(c *gin.Context) {
	var req dto.WeChatBindReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}

	resp, err := h.authSvc.WeChatBind(c.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound), errors.Is(err, service.ErrInvalidPassword):
			c.JSON(http.StatusUnauthorized, dto.Fail(401, err.Error()))
		case errors.Is(err, service.ErrWeChatBoundToOther), errors.Is(err, service.ErrAccountWeChatBound):
			c.JSON(http.StatusConflict, dto.Fail(409, err.Error()))
		case errors.Is(err, service.ErrWeChatBindToken):
			c.JSON(http.StatusUnauthorized, dto.Fail(401, err.Error()))
		default:
			c.JSON(http.StatusInternalServerError, dto.Fail(500, "关联账号失败: "+err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, dto.OK(resp))
}

// WeChatRegister 微信登录后创建新账号
// POST /api/auth/wechat/register
func (h *AuthHandler) WeChatRegister(c *gin.Context) {
	var req dto.WeChatRegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}
	req.RegisterIP = c.ClientIP()

	resp, err := h.authSvc.WeChatRegister(c.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUsernameExists):
			c.JSON(http.StatusConflict, dto.Fail(409, err.Error()))
		case errors.Is(err, service.ErrWeChatBindToken):
			c.JSON(http.StatusUnauthorized, dto.Fail(401, err.Error()))
		case errors.Is(err, service.ErrWeChatBoundToOther):
			c.JSON(http.StatusConflict, dto.Fail(409, err.Error()))
		default:
			c.JSON(http.StatusInternalServerError, dto.Fail(500, "创建账号失败: "+err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, dto.OK(resp))
}
