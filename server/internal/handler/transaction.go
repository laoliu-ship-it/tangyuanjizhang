package handler

import (
	"errors"
	"log"
	"net/http"
	"runtime/debug"
	"strconv"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/middleware"
	"fandianjizhang/server/internal/service"

	"github.com/gin-gonic/gin"
)

type TransactionHandler struct {
	transactionSvc service.TransactionService
}

func NewTransactionHandler(transactionSvc service.TransactionService) *TransactionHandler {
	return &TransactionHandler{transactionSvc: transactionSvc}
}

// List 获取交易记录列表
// GET /api/transactions
func (h *TransactionHandler) List(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	var filter dto.TransactionFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}

	result, err := h.transactionSvc.List(c.Request.Context(), tenantID, filter)
	if err != nil {
		log.Printf("[transaction] List tenant=%d error: %v\n%s", tenantID, err, debug.Stack())
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "获取交易列表失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(result))
}

// Create 创建交易记录
// POST /api/transactions
func (h *TransactionHandler) Create(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)

	var req dto.CreateTransactionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[transaction] Create bind tenant=%d user=%d error: %v", tenantID, userID, err)
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}

	tx, err := h.transactionSvc.Create(c.Request.Context(), tenantID, userID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCategoryNotFound):
			log.Printf("[transaction] Create tenant=%d user=%d category_not_found: %v req=%+v", tenantID, userID, err, req)
			c.JSON(http.StatusBadRequest, dto.Fail(400, "分类不存在，请先在系统中创建对应分类: "+err.Error()))
		default:
			log.Printf("[transaction] Create tenant=%d user=%d error: %v\n%s", tenantID, userID, err, debug.Stack())
			c.JSON(http.StatusInternalServerError, dto.Fail(500, "创建交易失败: "+err.Error()))
		}
		return
	}
	c.JSON(http.StatusOK, dto.OK(tx))
}

// BatchCreate 批量创建交易记录
// POST /api/transactions/batch
func (h *TransactionHandler) BatchCreate(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)

	var req dto.BatchCreateTransactionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[transaction] BatchCreate bind tenant=%d user=%d error: %v", tenantID, userID, err)
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}

	txs, err := h.transactionSvc.BatchCreate(c.Request.Context(), tenantID, userID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCategoryNotFound):
			log.Printf("[transaction] BatchCreate tenant=%d user=%d count=%d category_not_found: %v req=%+v", tenantID, userID, len(req.Transactions), err, req)
			c.JSON(http.StatusBadRequest, dto.Fail(400, "分类不存在，请先在系统中创建对应分类: "+err.Error()))
		default:
			log.Printf("[transaction] BatchCreate tenant=%d user=%d count=%d error: %v\n%s", tenantID, userID, len(req.Transactions), err, debug.Stack())
			c.JSON(http.StatusInternalServerError, dto.Fail(500, "批量创建交易失败: "+err.Error()))
		}
		return
	}
	c.JSON(http.StatusOK, dto.OK(txs))
}

// Update 更新交易记录
// PUT /api/transactions/:id
func (h *TransactionHandler) Update(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "交易ID格式错误"))
		return
	}

	var req dto.UpdateTransactionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}

	tx, err := h.transactionSvc.Update(c.Request.Context(), id, tenantID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTransactionNotFound):
			c.JSON(http.StatusNotFound, dto.Fail(404, err.Error()))
		case errors.Is(err, service.ErrCategoryNotFound):
			c.JSON(http.StatusBadRequest, dto.Fail(400, err.Error()))
		default:
			log.Printf("[transaction] Update id=%d tenant=%d error: %v\n%s", id, tenantID, err, debug.Stack())
			c.JSON(http.StatusInternalServerError, dto.Fail(500, "更新交易失败: "+err.Error()))
		}
		return
	}
	c.JSON(http.StatusOK, dto.OK(tx))
}

// Delete 删除交易记录
// DELETE /api/transactions/:id
func (h *TransactionHandler) Delete(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "交易ID格式错误"))
		return
	}

	if err := h.transactionSvc.Delete(c.Request.Context(), id, tenantID); err != nil {
		switch {
		case errors.Is(err, service.ErrTransactionNotFound):
			c.JSON(http.StatusNotFound, dto.Fail(404, err.Error()))
		default:
			log.Printf("[transaction] Delete id=%d tenant=%d error: %v\n%s", id, tenantID, err, debug.Stack())
			c.JSON(http.StatusInternalServerError, dto.Fail(500, "删除交易失败: "+err.Error()))
		}
		return
	}
	c.JSON(http.StatusOK, dto.OK(nil))
}
