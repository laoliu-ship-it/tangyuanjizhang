package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/middleware"
	"fandianjizhang/server/internal/repo"
	"fandianjizhang/server/internal/service"
	"fandianjizhang/server/pkg/excel"

	"github.com/gin-gonic/gin"
)

type ExportHandler struct {
	transactionSvc  service.TransactionService
	exporter        *excel.Exporter
	categoryRepo    repo.CategoryRepo
	transactionRepo repo.TransactionRepo
}

func NewExportHandler(
	transactionSvc service.TransactionService,
	exporter *excel.Exporter,
	categoryRepo repo.CategoryRepo,
	transactionRepo repo.TransactionRepo,
) *ExportHandler {
	return &ExportHandler{
		transactionSvc:  transactionSvc,
		exporter:        exporter,
		categoryRepo:    categoryRepo,
		transactionRepo: transactionRepo,
	}
}

// Excel 导出交易记录为 Excel 文件
// GET /api/export/excel?start_date=2024-01-01&end_date=2024-01-31&type=expense
func (h *ExportHandler) Excel(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	start := c.Query("start_date")
	end := c.Query("end_date")
	txType := c.Query("type")

	// 前端不传日期时导出全部数据
	filter := dto.TransactionFilter{
		StartDate: start,
		EndDate:   end,
		Type:      txType,
		Page:      1,
		PageSize:  100000,
	}

	fmt.Printf("[Export] 请求参数: tenantID=%d, start_date=%q, end_date=%q, type=%q\n",
		tenantID, start, end, txType)

	result, err := h.transactionSvc.List(c.Request.Context(), tenantID, filter)
	if err != nil {
		fmt.Printf("[Export] 查询失败: %v\n", err)
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "查询交易记录失败: "+err.Error()))
		return
	}

	fmt.Printf("[Export] 查询结果: total=%d, items=%d\n", result.Total, len(result.Items))
	if len(result.Items) > 0 {
		fmt.Printf("[Export] 第一条数据: ID=%d, Date=%q, Amount=%.2f, Type=%q\n",
			result.Items[0].ID, result.Items[0].TransactionDate, result.Items[0].Amount, result.Items[0].Type)
	}

	buf, err := h.exporter.ExportTransactions(result.Items)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "导出Excel失败: "+err.Error()))
		return
	}

	filename := fmt.Sprintf("transactions_%s_%s.xlsx", start, end)
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf)
}

// DownloadTemplate 下载导入模板
// GET /api/import/template
func (h *ExportHandler) DownloadTemplate(c *gin.Context) {
	data, err := excel.GenerateTemplate()
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "生成模板失败: "+err.Error()))
		return
	}
	c.Header("Content-Disposition", "attachment; filename=import_template.xlsx")
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

// ParseHeaders 解析 Excel 表头，返回列映射建议
// POST /api/import/parse-headers?sheet=0
func (h *ExportHandler) ParseHeaders(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "请上传 file 文件"))
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "读取文件失败: "+err.Error()))
		return
	}

	sheetIndex := -1
	if s := c.Query("sheet"); s != "" {
		if idx, e := strconv.Atoi(s); e == nil {
			sheetIndex = idx
		}
	}

	result, err := excel.ParseHeaders(data, sheetIndex)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "解析表头失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.OK(result))
}

// ImportExcel 导入 Excel 交易记录
// POST /api/import/excel
// Form fields: file (required), mapping (JSON, optional), sheet (int, optional)
func (h *ExportHandler) ImportExcel(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "请上传 file 文件"))
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "读取文件失败: "+err.Error()))
		return
	}

	// 解析可选的列映射
	var mapping *excel.ColumnMapping
	if mappingJSON := c.PostForm("mapping"); mappingJSON != "" {
		var m excel.ColumnMapping
		if err := json.Unmarshal([]byte(mappingJSON), &m); err != nil {
			c.JSON(http.StatusBadRequest, dto.Fail(400, "mapping 参数格式错误: "+err.Error()))
			return
		}
		mapping = &m
	}

	// 解析可选的 sheet 下标
	sheetIndex := -1
	if s := c.PostForm("sheet"); s != "" {
		if idx, e := strconv.Atoi(s); e == nil {
			sheetIndex = idx
		}
	}

	dryRun := c.PostForm("dry_run") == "true"

	categories, err := h.categoryRepo.List(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail(500, "获取分类列表失败: "+err.Error()))
		return
	}

	transactions, result, err := excel.ImportFromExcel(data, tenantID, userID, categories, mapping, sheetIndex, dryRun)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(400, "解析Excel失败: "+err.Error()))
		return
	}

	importedCount := 0
	if !dryRun && len(transactions) > 0 {
		if err := h.transactionRepo.BatchCreate(c.Request.Context(), transactions); err != nil {
			c.JSON(http.StatusInternalServerError, dto.Fail(500, "批量创建交易记录失败: "+err.Error()))
			return
		}
		importedCount = len(transactions)
	}

	c.JSON(http.StatusOK, dto.OK(gin.H{
		"dry_run":       dryRun,
		"valid_count":   result.ValidCount,
		"skipped_count": result.SkippedCount,
		"imported":      importedCount,
		"issues":        result.Issues,
	}))
}
