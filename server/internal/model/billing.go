package model

import (
	"encoding/json"
	"time"
)

// 账本计费：积分是唯一实际权益；购买道具只发放通用积分，OCR/AI 继续走既有扣分配置。
const (
	BillingIntentCreated   = "created"
	BillingIntentPaid      = "paid"
	BillingIntentDelivered = "delivered"
	BillingIntentCancelled = "cancelled"
	BillingIntentRefunded  = "refunded"

	BillingProcessedDeliver = "deliver"
	BillingProcessedRefund  = "refund"
)

// BillingCapability 是小管家虚拟支付能力（iOS IAP 等）在记账侧的缓存。
type BillingCapability struct {
	ID         uint64     `gorm:"primaryKey" json:"id"`
	AppSlug    string     `gorm:"size:64;not null;uniqueIndex:uk_billing_cap_app" json:"app_slug"`
	IAPEnabled bool       `gorm:"not null;default:false;comment:iOS IAP 是否开通" json:"iap_enabled"`
	Env        int        `gorm:"not null;default:0;comment:0正式/1沙箱" json:"env"`
	SyncedAt   *time.Time `gorm:"comment:同步时间" json:"synced_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// DemoAccount 是系统演示账号与演示账本的唯一关联记录。
type DemoAccount struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	UserID    uint64    `gorm:"not null;uniqueIndex:uk_demo_user;comment:演示用户ID" json:"user_id"`
	TenantID  uint64    `gorm:"not null;uniqueIndex:uk_demo_tenant;comment:演示账本ID" json:"tenant_id"`
	CreatedAt time.Time `json:"created_at"`
}

// BillingPackage 是记账侧展示的积分商品（从小管家商品同步或平台初始化）。
type BillingPackage struct {
	ID           uint64     `gorm:"primaryKey" json:"id"`
	AppSlug      string     `gorm:"size:64;not null;uniqueIndex:uk_billing_pkg_sku,priority:1;comment:小程序slug" json:"app_slug"`
	SKUCode      string     `gorm:"size:64;not null;uniqueIndex:uk_billing_pkg_sku,priority:2;comment:内部SKU" json:"sku_code"`
	Name         string     `gorm:"size:128;not null;comment:商品名称" json:"name"`
	WxProductID  string     `gorm:"size:64;comment:微信道具productId" json:"wx_product_id"`
	PriceFen     int64      `gorm:"not null;comment:价格(分)" json:"price_fen"`
	CreditAmount int64      `gorm:"not null;comment:到账通用积分" json:"credit_amount"`
	ValidityDays int        `gorm:"not null;default:365;comment:积分有效天数" json:"validity_days"`
	Status       string     `gorm:"size:16;not null;default:draft;comment:商品状态" json:"status"`
	Sort         int        `gorm:"not null;default:0;comment:排序" json:"sort"`
	SyncedAt     *time.Time `gorm:"comment:最近一次权威同步时间" json:"synced_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// BillingPurchaseIntent 是账本购买意图；business_ref 唯一，积分额是下单快照。
type BillingPurchaseIntent struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`
	TenantID     uint64    `gorm:"not null;index;comment:账本ID" json:"tenant_id"`
	UserID       uint64    `gorm:"not null;index;comment:购买用户" json:"user_id"`
	AppSlug      string    `gorm:"size:64;not null;comment:小程序slug" json:"app_slug"`
	BusinessRef  string    `gorm:"size:64;not null;uniqueIndex:uk_billing_intent_ref;comment:业务引用" json:"business_ref"`
	XgjIntentRef string    `gorm:"size:64;index;comment:小管家意图ref" json:"xgj_intent_ref"`
	SKUCode      string    `gorm:"size:64;not null;comment:内部SKU" json:"sku_code"`
	AmountFen    int64     `gorm:"not null;comment:金额(分)" json:"amount_fen"`
	CreditAmount int64     `gorm:"not null;comment:到账积分快照" json:"credit_amount"`
	ValidityDays int       `gorm:"not null;default:365;comment:积分有效期快照" json:"validity_days"`
	Status       string    `gorm:"size:16;not null;default:created;index;comment:意图状态" json:"status"`
	ExpiresAt    time.Time `gorm:"not null;comment:意图过期时间" json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// BillingProcessedEvent 是发货/退款回调幂等表。
type BillingProcessedEvent struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	EventKey  string    `gorm:"size:160;not null;uniqueIndex;comment:事件幂等键" json:"event_key"`
	EventType string    `gorm:"size:32;not null;comment:事件类型" json:"event_type"`
	OrderRef  string    `gorm:"size:64;index;comment:订单/业务引用" json:"order_ref"`
	Result    string    `gorm:"size:24;comment:处理结果" json:"result"`
	CreatedAt time.Time `json:"created_at"`
}

// BillingRefund 记录退款回收结果；已消耗部分进入人工核查，不制造负余额。
type BillingRefund struct {
	ID               uint64    `gorm:"primaryKey" json:"id"`
	TenantID         uint64    `gorm:"not null;index;comment:账本ID" json:"tenant_id"`
	RefundKey        string    `gorm:"size:160;not null;uniqueIndex;comment:退款幂等键" json:"refund_key"`
	OrderRef         string    `gorm:"size:64;index;comment:订单/业务引用" json:"order_ref"`
	MchRefundID      string    `gorm:"size:128;comment:商户退款单号" json:"mch_refund_id"`
	Status           string    `gorm:"size:24;not null;comment:退款状态" json:"status"`
	RefundFeeFen     int64     `gorm:"not null;default:0;comment:退款金额(分)" json:"refund_fee_fen"`
	ReclaimedCredits int64     `gorm:"not null;default:0;comment:已回收积分" json:"reclaimed_credits"`
	ConsumedCredits  int64     `gorm:"not null;default:0;comment:已消耗待人工核查积分" json:"consumed_credits"`
	Reason           string    `gorm:"size:255;comment:原因" json:"reason"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// BillingDeliveryEvent 是服务间发货/退款事件 DTO（handler 验签后传入 repo）。
type BillingDeliveryEvent struct {
	EventKey          string          `json:"event_key"`
	EventType         string          `json:"event_type"`
	AppID             uint            `json:"app_id"`
	OutTradeNo        string          `json:"out_trade_no"`
	WxOrderID         string          `json:"wx_order_id"`
	OpenID            string          `json:"open_id"`
	BusinessReference string          `json:"business_reference"`
	SKUCode           string          `json:"sku_code"`
	WxProductID       string          `json:"wx_product_id"`
	Quantity          int             `json:"quantity"`
	AmountFen         int64           `json:"amount_fen"`
	DeliveryPayload   json.RawMessage `json:"delivery_payload"`
	CreditAmount      int64           `json:"credit_amount"`
	ValidityDays      int             `json:"validity_days"`
	MchRefundID       string          `json:"mch_refund_id"`
	WxRefundID        string          `json:"wx_refund_id"`
	RefundFeeFen      int64           `json:"refund_fee_fen"`
	OccurredAt        string          `json:"occurred_at"`
}
