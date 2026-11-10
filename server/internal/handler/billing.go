package handler

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"fandianjizhang/server/config"
	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/middleware"
	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/service"
	"fandianjizhang/server/internal/xgjclient"

	"github.com/gin-gonic/gin"
)

type BillingHandler struct {
	svc service.BillingService
	cfg config.BillingConfig
}

func NewBillingHandler(svc service.BillingService, cfg config.BillingConfig) *BillingHandler {
	return &BillingHandler{svc: svc, cfg: cfg}
}

func tenantAdmin(c *gin.Context) bool {
	role := middleware.GetUserRole(c)
	if role == "admin" || role == "owner" {
		return true
	}
	c.JSON(http.StatusOK, dto.Fail(403, "仅账本管理员可操作"))
	return false
}

func (h *BillingHandler) Summary(c *gin.Context) {
	refresh := c.Query("refresh") == "1" || c.Query("refresh") == "true"
	resp, err := h.svc.Summary(c.Request.Context(), middleware.GetTenantID(c), refresh)
	if err != nil {
		c.JSON(http.StatusOK, dto.Fail(500, "获取套餐失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

func (h *BillingHandler) CreateIntent(c *gin.Context) {
	if !tenantAdmin(c) {
		return
	}
	if middleware.IsDemo(c) {
		c.JSON(http.StatusOK, dto.Fail(403, "演示账号不可购买"))
		return
	}
	var req dto.BillingPurchaseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}
	resp, err := h.svc.CreatePurchase(c.Request.Context(), middleware.GetTenantID(c), middleware.GetUserID(c), strings.TrimSpace(req.SKUCode), req.ExpectedPriceFen, strings.ToLower(strings.TrimSpace(req.Platform)))
	if err != nil {
		c.JSON(http.StatusOK, dto.Fail(400, err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

func (h *BillingHandler) Prepare(c *gin.Context) {
	if !tenantAdmin(c) {
		return
	}
	if middleware.IsDemo(c) {
		c.JSON(http.StatusOK, dto.Fail(403, "演示账号不可购买"))
		return
	}
	var req dto.BillingPrepareReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}
	resp, err := h.svc.PreparePayment(c.Request.Context(), middleware.GetTenantID(c), middleware.GetUserID(c), c.Param("ref"), req.Code)
	if err != nil {
		c.JSON(http.StatusOK, dto.Fail(400, err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

func (h *BillingHandler) GetIntent(c *gin.Context) {
	resp, err := h.svc.GetIntent(c.Request.Context(), middleware.GetTenantID(c), c.Param("ref"))
	if err != nil {
		c.JSON(http.StatusOK, dto.Fail(404, err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

func (h *BillingHandler) ListIntents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	resp, err := h.svc.ListIntents(c.Request.Context(), middleware.GetTenantID(c), page, size)
	if err != nil {
		c.JSON(http.StatusOK, dto.Fail(500, "获取订单失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(resp))
}

// Webhook 接收小管家发货/退款事件（服务间 HMAC，验签失败关闭流程）。
func (h *BillingHandler) Webhook(c *gin.Context) {
	if strings.TrimSpace(h.cfg.XGJSecret) == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "error": "billing webhook secret not configured"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "read body failed"})
		return
	}
	ts := c.GetHeader("X-VP-Timestamp")
	nonce := c.GetHeader("X-VP-Nonce")
	sig := c.GetHeader("X-VP-Signature")
	if ts == "" || nonce == "" || sig == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "missing signature"})
		return
	}
	sec, err := strconv.ParseInt(ts, 10, 64)
	skew := h.cfg.ClockSkewSeconds
	if skew <= 0 {
		skew = 300
	}
	if err != nil || time.Since(time.Unix(sec, 0)).Abs() > time.Duration(skew)*time.Second {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "timestamp out of window"})
		return
	}
	expect := xgjclient.Sign(h.cfg.XGJSecret, ts, nonce, string(body))
	if subtle.ConstantTimeCompare([]byte(expect), []byte(sig)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "signature mismatch"})
		return
	}
	var event model.BillingDeliveryEvent
	if err := json.Unmarshal(body, &event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid payload"})
		return
	}
	if event.EventKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "missing event_key"})
		return
	}
	already := false
	switch event.EventType {
	case "billing_refund", "xpay_refund_notify":
		already, err = h.svc.ProcessRefund(c.Request.Context(), &event)
	case "billing_deliver", "xpay_goods_deliver_notify":
		already, err = h.svc.ProcessDelivery(c.Request.Context(), &event)
	default:
		// 未知事件类型不能默认按发货处理，否则任何事件都可能触发权益发放。
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "unknown event_type: " + event.EventType})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "already": already})
}

// SeedPackages 平台初始化草稿套餐（显式调用，不自动改线上数据）。
func (h *BillingHandler) SeedPackages(c *gin.Context) {
	var req dto.BillingSeedReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, dto.Fail(400, "参数错误: "+err.Error()))
		return
	}
	if len(req.Packages) == 0 {
		req.Packages = defaultBillingPackages()
	}
	appSlug := strings.TrimSpace(h.cfg.AppSlug)
	if appSlug == "" {
		appSlug = "hehuojizhangzhushou"
	}
	pkgs := make([]*model.BillingPackage, 0, len(req.Packages))
	for i, p := range req.Packages {
		status := p.Status
		if status == "" {
			status = "draft"
		}
		pkgs = append(pkgs, &model.BillingPackage{
			AppSlug: appSlug, SKUCode: p.SKUCode, Name: p.Name, WxProductID: p.WxProductID,
			PriceFen: p.PriceFen, CreditAmount: p.CreditAmount,
			ValidityDays: p.ValidityDays, Status: status, Sort: i,
		})
	}
	if err := h.svc.SeedPackages(c.Request.Context(), pkgs); err != nil {
		c.JSON(http.StatusOK, dto.Fail(500, "初始化失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"count": len(pkgs)}))
}

func defaultBillingPackages() []dto.BillingPackageItem {
	// 本地展示示例：默认 100 积分/元；真实上架以小管家商品 delivery_payload 为准。
	return []dto.BillingPackageItem{
		{SKUCode: "credits_600", Name: "600 积分", PriceFen: 600, CreditAmount: 600, ValidityDays: 365, Status: "draft"},
		{SKUCode: "credits_1200", Name: "1200 积分", PriceFen: 1200, CreditAmount: 1200, ValidityDays: 365, Status: "draft"},
		{SKUCode: "credits_1800", Name: "1800 积分", PriceFen: 1800, CreditAmount: 1800, ValidityDays: 365, Status: "draft"},
		{SKUCode: "credits_3000", Name: "3000 积分", PriceFen: 3000, CreditAmount: 3000, ValidityDays: 365, Status: "draft"},
		{SKUCode: "credits_6000", Name: "6000 积分", PriceFen: 6000, CreditAmount: 6000, ValidityDays: 365, Status: "draft"},
	}
}

var _ = fmt.Sprintf
