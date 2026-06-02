package handler

import (
	"net/http"
	"strconv"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/middleware"
	"fandianjizhang/server/internal/service"

	"github.com/gin-gonic/gin"
)

type MerchantHandler struct {
	merchantSvc service.MerchantService
}

func NewMerchantHandler(merchantSvc service.MerchantService) *MerchantHandler {
	return &MerchantHandler{merchantSvc: merchantSvc}
}

// List 获取商户列表
// GET /api/merchants
func (h *MerchantHandler) List(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	merchants, err := h.merchantSvc.List(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "获取商户列表失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(merchants))
}

// Create 创建商户
// POST /api/merchants
func (h *MerchantHandler) Create(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	var req struct {
		Name string `json:"name" binding:"required,min=1,max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}
	m, err := h.merchantSvc.Create(c.Request.Context(), tenantID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "创建商户失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(&dto.MerchantResp{
		ID:        m.ID,
		Name:      m.Name,
		CreatedAt: m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}))
}

// Update 更新商户
// PUT /api/merchants/:id
func (h *MerchantHandler) Update(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "商户ID格式错误"))
		return
	}
	var req struct {
		Name string `json:"name" binding:"required,min=1,max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}
	m, err := h.merchantSvc.Update(c.Request.Context(), id, tenantID, req.Name)
	if err != nil {
		switch err {
		case service.ErrMerchantNotFound:
			c.JSON(http.StatusNotFound, dto.Fail(404, err.Error()))
		default:
			c.JSON(http.StatusInternalServerError, dto.Fail(500, "更新商户失败: "+err.Error()))
		}
		return
	}
	c.JSON(http.StatusOK, dto.OK(&dto.MerchantResp{
		ID:        m.ID,
		Name:      m.Name,
		CreatedAt: m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}))
}

// Delete 删除商户
// DELETE /api/merchants/:id
func (h *MerchantHandler) Delete(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "商户ID格式错误"))
		return
	}
	if err := h.merchantSvc.Delete(c.Request.Context(), id, tenantID); err != nil {
		switch err {
		case service.ErrMerchantNotFound:
			c.JSON(http.StatusNotFound, dto.Fail(404, err.Error()))
		default:
			c.JSON(http.StatusInternalServerError, dto.Fail(500, "删除商户失败: "+err.Error()))
		}
		return
	}
	c.JSON(http.StatusOK, dto.OK(nil))
}