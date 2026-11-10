package dto

import "time"

type BillingPackageItem struct {
	SKUCode      string           `json:"sku_code"`
	Name         string           `json:"name"`
	WxProductID  string           `json:"wx_product_id"`
	PriceFen     int64            `json:"price_fen"`
	CreditAmount int64            `json:"credit_amount"`
	ValidityDays int              `json:"validity_days"`
	Status       string           `json:"status"`
	SyncedAt     *time.Time       `json:"synced_at"`
	Estimates    BillingEstimates `json:"estimates"`
}

type BillingUnitPrices struct {
	Upload int64 `json:"upload"`
	OCR    int64 `json:"ocr"`
	AI     int64 `json:"ai"`
}

// BillingEstimates 是互斥用途估算：同一笔积分不能同时兑现为多套配额。
type BillingEstimates struct {
	UploadOCR *int64 `json:"upload_ocr_times"`
	AI        *int64 `json:"ai_times"`
	FullChain *int64 `json:"full_chain_times"`
}

type BillingSummaryResp struct {
	Enabled       bool                 `json:"enabled"`
	Packages      []BillingPackageItem `json:"packages"`
	CreditBalance int64                `json:"credit_balance"`
	Unlimited     bool                 `json:"unlimited"`
	UnitPrices    BillingUnitPrices    `json:"unit_prices"`
	Estimates     BillingEstimates     `json:"estimates"`
	SyncedAt      *time.Time           `json:"synced_at"`
	Stale         bool                 `json:"stale"`
	IAPEnabled    bool                 `json:"iap_enabled"`
}

type BillingPurchaseReq struct {
	SKUCode          string `json:"sku_code" binding:"required"`
	ExpectedPriceFen int64  `json:"expected_price_fen"`
	Platform         string `json:"platform"`
}

type BillingIntentResp struct {
	BusinessRef  string    `json:"business_ref"`
	SKUCode      string    `json:"sku_code"`
	Name         string    `json:"name"`
	AmountFen    int64     `json:"amount_fen"`
	CreditAmount int64     `json:"credit_amount"`
	ValidityDays int       `json:"validity_days"`
	ExpiresAt    time.Time `json:"expires_at"`
	Status       string    `json:"status"`
}

type BillingIntentPageResp struct {
	Total int64               `json:"total"`
	Page  int                 `json:"page"`
	Size  int                 `json:"size"`
	Items []BillingIntentResp `json:"items"`
}

type BillingPrepareReq struct {
	Code string `json:"code" binding:"required"`
}

type BillingPrepareResp struct {
	OutTradeNo string `json:"out_trade_no"`
	Mode       string `json:"mode"`
	SignData   string `json:"signData"`
	PaySig     string `json:"paySig"`
	Signature  string `json:"signature"`
	Env        int    `json:"env"`
}

type BillingSeedReq struct {
	Packages []BillingPackageItem `json:"packages"`
}
