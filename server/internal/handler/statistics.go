package handler

import (
	"net/http"
	"strconv"
	"time"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/middleware"
	"fandianjizhang/server/internal/service"

	"github.com/gin-gonic/gin"
)

type StatisticsHandler struct {
	statisticsSvc service.StatisticsService
}

func NewStatisticsHandler(statisticsSvc service.StatisticsService) *StatisticsHandler {
	return &StatisticsHandler{statisticsSvc: statisticsSvc}
}

// Daily 获取某天的统计数据
// GET /api/statistics/daily?date=2024-01-01
func (h *StatisticsHandler) Daily(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	date := c.Query("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	// 验证日期格式
	if _, err := time.Parse("2006-01-02", date); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "日期格式错误，应为 YYYY-MM-DD"))
		return
	}

	result, err := h.statisticsSvc.Daily(c.Request.Context(), tenantID, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "获取日统计失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(result))
}

// Monthly 获取某月的统计数据
// GET /api/statistics/monthly?year=2024&month=1
func (h *StatisticsHandler) Monthly(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	now := time.Now()
	yearStr := c.DefaultQuery("year", strconv.Itoa(now.Year()))
	monthStr := c.DefaultQuery("month", strconv.Itoa(int(now.Month())))

	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 2000 || year > 2100 {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "year 参数错误"))
		return
	}
	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "month 参数错误，应为 1-12"))
		return
	}

	result, err := h.statisticsSvc.Monthly(c.Request.Context(), tenantID, year, month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "获取月统计失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(result))
}

// Yearly 获取某年的统计数据
// GET /api/statistics/yearly?year=2024
func (h *StatisticsHandler) Yearly(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	now := time.Now()
	yearStr := c.DefaultQuery("year", strconv.Itoa(now.Year()))

	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 2000 || year > 2100 {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "year 参数错误"))
		return
	}

	result, err := h.statisticsSvc.Yearly(c.Request.Context(), tenantID, year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "获取年统计失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(result))
}

// Range 获取日期范围统计数据
// GET /api/statistics/range?start=2024-01-01&end=2024-01-31
func (h *StatisticsHandler) Range(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	start := c.Query("start")
	end := c.Query("end")

	if start == "" || end == "" {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "start 和 end 参数不能为空"))
		return
	}

	if _, err := time.Parse("2006-01-02", start); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "start 日期格式错误，应为 YYYY-MM-DD"))
		return
	}
	if _, err := time.Parse("2006-01-02", end); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "end 日期格式错误，应为 YYYY-MM-DD"))
		return
	}
	if start > end {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "start 不能大于 end"))
		return
	}

	result, err := h.statisticsSvc.Range(c.Request.Context(), tenantID, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "获取范围统计失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(result))
}
