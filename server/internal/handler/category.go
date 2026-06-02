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

type CategoryHandler struct {
	categorySvc service.CategoryService
}

func NewCategoryHandler(categorySvc service.CategoryService) *CategoryHandler {
	return &CategoryHandler{categorySvc: categorySvc}
}

// List 获取分类列表
// GET /api/categories
func (h *CategoryHandler) List(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	cats, err := h.categorySvc.List(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "获取分类列表失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(cats))
}

// Create 创建分类
// POST /api/categories
func (h *CategoryHandler) Create(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	var req dto.CreateCategoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}

	cat, err := h.categorySvc.Create(c.Request.Context(), tenantID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "创建分类失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(cat))
}

// Update 更新分类
// PUT /api/categories/:id
func (h *CategoryHandler) Update(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "分类ID格式错误"))
		return
	}

	var req dto.UpdateCategoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}

	cat, err := h.categorySvc.Update(c.Request.Context(), id, tenantID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCategoryNotFound):
			c.JSON(http.StatusNotFound, dto.Fail(404, err.Error()))
		default:
			c.JSON(http.StatusInternalServerError, dto.Fail(500, "更新分类失败: "+err.Error()))
		}
		return
	}
	c.JSON(http.StatusOK, dto.OK(cat))
}

// Delete 删除分类
// DELETE /api/categories/:id
func (h *CategoryHandler) Delete(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "分类ID格式错误"))
		return
	}

	if err := h.categorySvc.Delete(c.Request.Context(), id, tenantID); err != nil {
		switch {
		case errors.Is(err, service.ErrCategoryNotFound):
			c.JSON(http.StatusNotFound, dto.Fail(404, err.Error()))
		default:
			c.JSON(http.StatusInternalServerError, dto.Fail(500, "删除分类失败: "+err.Error()))
		}
		return
	}
	c.JSON(http.StatusOK, dto.OK(nil))
}
