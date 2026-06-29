package handler

import (
	"net/http"
	"strconv"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/middleware"
	"fandianjizhang/server/internal/service"

	"github.com/gin-gonic/gin"
)

type CreditsHandler struct {
	creditsSvc service.CreditsService
}

func NewCreditsHandler(creditsSvc service.CreditsService) *CreditsHandler {
	return &CreditsHandler{creditsSvc: creditsSvc}
}

// GetBalance 查询当前租户积分余额及最近20条流水
// GET /api/credits
func (h *CreditsHandler) GetBalance(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	resp, err := h.creditsSvc.GetBalance(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "查询积分失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

// GetLogs 分页查询积分流水
// GET /api/credits/logs?page=1&size=20&type=income|expense
func (h *CreditsHandler) GetLogs(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	amountType := c.Query("type") // income / expense / ""
	resp, err := h.creditsSvc.GetLogs(c.Request.Context(), tenantID, page, size, amountType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "查询流水失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}
