package model

import (
	"time"
)

// User 用户表
type User struct {
	ID           uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string     `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Email        string     `gorm:"uniqueIndex;size:100;not null" json:"email"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	RegisterIP   string     `gorm:"size:45;default:''" json:"-"`
	LastLoginAt  *time.Time `gorm:"comment:最近登录时间" json:"last_login_at"`
	CreatedAt    time.Time  `json:"created_at"`
	DeletedAt    *time.Time `gorm:"index" json:"-"`
}

func (User) TableName() string { return "users" }

// Tenant 租户表
type Tenant struct {
	ID        uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string     `gorm:"size:100;not null" json:"name"`
	OwnerID   uint64     `gorm:"not null" json:"owner_id"`
	BookType  string     `gorm:"size:20;not null;default:'restaurant'" json:"book_type"`
	CreatedAt time.Time  `json:"created_at"`
	DeletedAt *time.Time `gorm:"index" json:"-"`

	Owner   *User           `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Members []*TenantMember `gorm:"foreignKey:TenantID" json:"members,omitempty"`
}

func (Tenant) TableName() string { return "tenants" }

// 账本类型枚举
const (
	BookTypeRestaurant = "restaurant"
	BookTypeFamily     = "family"
	BookTypeStartup    = "startup"
)

// TenantMember 租户成员表
type TenantMember struct {
	TenantID uint64    `gorm:"primaryKey;not null" json:"tenant_id"`
	UserID   uint64    `gorm:"primaryKey;not null" json:"user_id"`
	Role     string    `gorm:"type:varchar(50);not null;default:'viewer'" json:"role"`
	RoleID   *uint64   `json:"role_id"`
	JoinedAt time.Time `json:"joined_at"`

	Tenant *Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	User   *User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (TenantMember) TableName() string { return "tenant_members" }

// Category 分类表
type Category struct {
	ID        uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID  uint64     `gorm:"not null;index" json:"tenant_id"`
	Name      string     `gorm:"size:50;not null" json:"name"`
	Type      string     `gorm:"type:enum('income','expense');not null" json:"type"`
	Icon      string     `gorm:"size:50;default:''" json:"icon"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"-"`
}

func (Category) TableName() string { return "categories" }

// Transaction 交易记录表
type Transaction struct {
	ID              uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID        uint64     `gorm:"not null;index" json:"tenant_id"`
	UserID          uint64     `gorm:"not null" json:"user_id"`
	Type            string     `gorm:"type:enum('income','expense');not null" json:"type"`
	Amount          float64    `gorm:"type:decimal(12,2);not null" json:"amount"`
	CategoryID      uint64     `gorm:"not null" json:"category_id"`
	MerchantID      uint64     `gorm:"default:0" json:"merchant_id"`
	TransactionDate string     `gorm:"type:datetime;not null" json:"transaction_date"`
	Note            string     `gorm:"size:255;default:''" json:"note"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `gorm:"index" json:"-"`

	Category *Category           `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Merchant *Merchant           `gorm:"foreignKey:MerchantID" json:"merchant,omitempty"`
	Images   []*TransactionImage `gorm:"foreignKey:TransactionID" json:"images,omitempty"`
}

func (Transaction) TableName() string { return "transactions" }

// TransactionImage 交易图片表
type TransactionImage struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TransactionID uint64    `gorm:"not null;index" json:"transaction_id"`
	ImagePath     string    `gorm:"size:500;not null" json:"image_path"`
	OCRAmount     float64   `gorm:"type:decimal(12,2);default:0" json:"ocr_amount"`
	OCRDate       string    `gorm:"size:20;default:''" json:"ocr_date"`
	OCRMerchant   string    `gorm:"size:100;default:''" json:"ocr_merchant"`
	OCRRawTexts   string    `gorm:"type:text" json:"ocr_raw_texts"`
	CreatedAt     time.Time `json:"created_at"`
}

func (TransactionImage) TableName() string { return "transaction_images" }

// OcrRecord OCR识别结果表（按租户隔离，防止跨租户偷 token）
type OcrRecord struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID     uint64    `gorm:"not null;index" json:"tenant_id"`
	ImagePath    string    `gorm:"size:500;not null" json:"image_path"`
	ImageHash    string    `gorm:"size:64;default:''" json:"image_hash"`
	ImageSize    int64     `gorm:"default:0" json:"image_size"`
	AIMode       bool      `gorm:"default:false" json:"ai_mode"`
	Amount       float64   `gorm:"type:decimal(12,2);default:0" json:"amount"`
	Date         string    `gorm:"size:50;default:''" json:"date"`
	MerchantName string    `gorm:"size:200;default:''" json:"merchant_name"`
	RawTexts     string    `gorm:"type:text" json:"raw_texts"` // JSON []string，MySQL 5.7 下 TEXT 列不支持 default，故不加
	CreatedAt    time.Time `json:"created_at"`
}

func (OcrRecord) TableName() string { return "ocr_records" }

// Merchant 商户表（租户级别）
type Merchant struct {
	ID        uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID  uint64     `gorm:"not null;index" json:"tenant_id"`
	Name      string     `gorm:"size:100;not null" json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"-"`
}

func (Merchant) TableName() string { return "merchants" }

// StatsCache 统计缓存表（按租户+类型+周期存商户 Top10）
type StatsCache struct {
	ID         uint64 `gorm:"primaryKey;autoIncrement"`
	TenantID   uint64 `gorm:"not null;uniqueIndex:uniq_stats_cache"`
	CacheType  string `gorm:"type:varchar(20);not null;uniqueIndex:uniq_stats_cache"` // "monthly" | "yearly"
	PeriodKey  string `gorm:"type:varchar(20);not null;uniqueIndex:uniq_stats_cache"` // "2026-06" | "2026"
	Data       string `gorm:"type:text;not null"`
	ComputedAt time.Time
}

func (StatsCache) TableName() string { return "stats_cache" }

// TenantLLMConfig 租户 LLM 配置表（每个租户一条记录）
type TenantLLMConfig struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID     uint64    `gorm:"not null;uniqueIndex" json:"tenant_id"`
	Enabled      bool      `gorm:"default:false" json:"enabled"`
	UsePlatform  bool      `gorm:"default:true" json:"use_platform"` // true=使用平台配置，false=使用租户自己的key
	Provider     string    `gorm:"size:50;default:'openai'" json:"provider"`
	BaseURL      string    `gorm:"size:255;default:''" json:"base_url"`
	APIKey       string    `gorm:"size:500;default:''" json:"-"` // 不直接返回给前端
	Model        string    `gorm:"size:100;default:'gpt-4o'" json:"model"`
	Mode         string    `gorm:"size:20;default:'ocr_text'" json:"mode"` // "vision" | "ocr_text"
	CustomPrompt string    `gorm:"type:text" json:"-"`                     // 自定义识别提示词，空=使用平台默认
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (TenantLLMConfig) TableName() string { return "tenant_llm_configs" }

// LLMTokenUsage 大模型对话 token 消耗记录（按租户隔离，每次 LLM 分析调用写一条）
// (tenant_id, action, ref_type, ref_id) 唯一：与 CreditLog 的幂等键保持一致，
// 同一资源同一动作只落一条 token 记录，防止重复点击/请求竞态导致重复入账
type LLMTokenUsage struct {
	ID                    uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID              uint64    `gorm:"not null;uniqueIndex:uniq_llm_usage_ref,priority:1" json:"tenant_id"`
	Action                string    `gorm:"size:30;not null;uniqueIndex:uniq_llm_usage_ref,priority:2" json:"action"`              // orc2bill 等，与 CreditLog.Action 对应
	RefType               string    `gorm:"size:50;not null;default:'';uniqueIndex:uniq_llm_usage_ref,priority:3" json:"ref_type"` // 关联资源类型，如 "ocr_record"
	RefID                 uint64    `gorm:"not null;default:0;uniqueIndex:uniq_llm_usage_ref,priority:4" json:"ref_id"`
	Provider              string    `gorm:"size:50;default:''" json:"provider"`
	Model                 string    `gorm:"size:100;default:''" json:"model"`
	PromptCacheHitTokens  int       `gorm:"default:0" json:"prompt_cache_hit_tokens"`  // 命中缓存的输入 token 数
	PromptCacheMissTokens int       `gorm:"default:0" json:"prompt_cache_miss_tokens"` // 未命中缓存的输入 token 数
	CompletionTokens      int       `gorm:"default:0" json:"completion_tokens"`
	TotalTokens           int       `gorm:"default:0" json:"total_tokens"`
	CreatedAt             time.Time `gorm:"index" json:"created_at"`
}

func (LLMTokenUsage) TableName() string { return "llm_token_usages" }

// MediaFile 租户媒体文件表（用于去重和资源统计）
type MediaFile struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID     uint64    `gorm:"not null;index" json:"tenant_id"`
	OriginalHash string    `gorm:"size:64;not null;index:idx_hash_tenant,priority:1" json:"original_hash"` // 原始文件 SHA-256
	FileName     string    `gorm:"size:255;not null" json:"file_name"`
	FilePath     string    `gorm:"size:500;not null" json:"file_path"`
	FileSize     int64     `gorm:"not null" json:"file_size"` // 字节
	MimeType     string    `gorm:"size:100;default:''" json:"mime_type"`
	CreatedAt    time.Time `json:"created_at"`
}

func (MediaFile) TableName() string { return "media_files" }

// TenantSettings 租户通用设置表（每个租户一条记录，不存在时用默认值）
type TenantSettings struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID uint64 `gorm:"not null;uniqueIndex" json:"tenant_id"`
	// 不加 gorm default 标签：带 default 的 bool 字段在 INSERT 时零值(false)会被 GORM 跳过，
	// 导致首次保存"关闭"被 DB 默认值覆盖；默认值由 service 层 GetSettings 提供
	RequireExpenseImage bool      `json:"require_expense_image"` // 支出记录是否必须上传图片
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (TenantSettings) TableName() string { return "tenant_settings" }

// PlatformAdmin 平台管理员表
type PlatformAdmin struct {
	ID           uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Email        string     `gorm:"uniqueIndex;size:100;not null" json:"email"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	Name         string     `gorm:"size:50;not null" json:"name"`
	CreatedAt    time.Time  `json:"created_at"`
	DeletedAt    *time.Time `gorm:"index" json:"-"`
}

func (PlatformAdmin) TableName() string { return "platform_admins" }

// PlatformConfig 平台配置表（键值对存储）
type PlatformConfig struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ConfigKey   string    `gorm:"uniqueIndex;size:50;not null" json:"config_key"` // 配置键
	ConfigValue string    `gorm:"size:500;not null" json:"config_value"`          // 配置值
	Description string    `gorm:"size:200;default:''" json:"description"`         // 配置描述
	UpdatedAt   time.Time `json:"updated_at"`
}

func (PlatformConfig) TableName() string { return "platform_configs" }

// 预置配置键常量
const (
	ConfigKeyCacheType       = "cache_type"        // "file" 或 "text"
	ConfigKeyCacheTTLMinutes = "cache_ttl_minutes" // 缓存分钟数
	ConfigKeyOCRCacheEnabled = "ocr_cache_enabled" // "true" 或 "false"
	ConfigKeyLLMCacheEnabled = "llm_cache_enabled" // "true" 或 "false"

	// 积分计费配置键
	ConfigKeyCreditsPerUpload   = "credits_per_upload"          // 每次上传新文件扣积分
	ConfigKeyCreditsPerOCR      = "credits_per_ocr"             // 每次 OCR 识别扣积分
	ConfigKeyCreditsPerOrc2bill = "credits_per_llm"             // 每次 AI 整理（OCR 转记账）扣积分；key 名沿用历史值 credits_per_llm，避免影响已有 platform_configs 数据
	ConfigKeyCreditsPerGBPerDay = "credits_per_gb_per_day"      // 每 G 存储每天扣积分（预留）
	ConfigKeyCreditsNewUserGift = "credits_new_user_gift"       // 新用户注册赠送积分
	ConfigKeyMaxAccountsPerIP   = "credits_max_accounts_per_ip" // 同IP最多多少账号可获赠积分
	ConfigKeyCreditsValidityDays = "credits_validity_days"      // 积分有效期（天），发放时刻起算

	// 平台 LLM 配置键
	ConfigKeyPlatformLLMEnabled  = "platform_llm_enabled"  // "true"/"false"
	ConfigKeyPlatformLLMProvider = "platform_llm_provider" // "openai"/"azure"等
	ConfigKeyPlatformLLMBaseURL  = "platform_llm_base_url" // 自定义端点，空=用默认
	ConfigKeyPlatformLLMAPIKey   = "platform_llm_api_key"  // 平台 API Key
	ConfigKeyPlatformLLMModel    = "platform_llm_model"    // e.g. "gpt-4o"
	ConfigKeyPlatformLLMMode     = "platform_llm_mode"     // "ocr_text"/"image"
)

// 默认配置值
var DefaultPlatformConfigs = map[string]struct {
	Value       string
	Description string
}{
	ConfigKeyCacheType:       {Value: "file", Description: "缓存类型：file（文件SHA256）或 text（文本内容）"},
	ConfigKeyCacheTTLMinutes: {Value: "30", Description: "缓存有效期（分钟）"},
	ConfigKeyOCRCacheEnabled: {Value: "true", Description: "是否启用 OCR 缓存"},
	ConfigKeyLLMCacheEnabled: {Value: "true", Description: "是否启用 LLM 缓存"},
	// 积分计费默认值
	ConfigKeyCreditsPerUpload:   {Value: "5", Description: "每次上传新文件扣除积分数"},
	ConfigKeyCreditsPerOCR:      {Value: "10", Description: "每次 OCR 识别扣除积分数"},
	ConfigKeyCreditsPerOrc2bill: {Value: "20", Description: "每次 AI 整理（OCR 转记账）扣除积分数"},
	ConfigKeyCreditsPerGBPerDay: {Value: "50", Description: "每 G 存储每天扣除积分数（预留）"},
	ConfigKeyCreditsNewUserGift: {Value: "1000", Description: "新用户注册赠送积分数"},
	ConfigKeyMaxAccountsPerIP:   {Value: "3", Description: "同一 IP 最多多少账号可获赠注册积分（0=不限制）"},
	ConfigKeyCreditsValidityDays: {Value: "365", Description: "积分有效期（天），从发放时刻起算"},
	// 平台 LLM 配置默认值（空值=未配置）
	ConfigKeyPlatformLLMEnabled:  {Value: "false", Description: "平台 LLM 是否启用"},
	ConfigKeyPlatformLLMProvider: {Value: "openai", Description: "平台 LLM Provider（openai/azure等）"},
	ConfigKeyPlatformLLMBaseURL:  {Value: "", Description: "平台 LLM 自定义 BaseURL（空=使用默认）"},
	ConfigKeyPlatformLLMAPIKey:   {Value: "", Description: "平台 LLM API Key"},
	ConfigKeyPlatformLLMModel:    {Value: "gpt-4o", Description: "平台 LLM 默认模型"},
	ConfigKeyPlatformLLMMode:     {Value: "ocr_text", Description: "平台 LLM 调用模式：ocr_text 或 image"},
}

// TenantCredits 租户积分余额表（每租户一条）
type TenantCredits struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID  uint64    `gorm:"uniqueIndex;not null" json:"tenant_id"`
	Balance   int64     `gorm:"not null;default:0" json:"balance"`       // 当前积分余额
	Unlimited bool      `gorm:"not null;default:false" json:"unlimited"` // true=无限制白名单
	UpdatedAt time.Time `json:"updated_at"`
}

func (TenantCredits) TableName() string { return "tenant_credits" }

// CreditLog 积分流水对账表
type CreditLog struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID     uint64    `gorm:"not null;index" json:"tenant_id"`
	Amount       int64     `gorm:"not null" json:"amount"`             // 正=充值/赠送, 负=消耗
	Action       string    `gorm:"size:30;not null" json:"action"`     // upload/ocr/orc2bill/storage/gift/admin_adjust（llm 为历史动作名，已由 orc2bill 取代）
	BalanceAfter int64     `gorm:"not null" json:"balance_after"`      // 操作后余额快照；unlimited 时为 -1
	RefType      string    `gorm:"size:50;default:''" json:"ref_type"` // 关联资源类型，如 "ocr_record"
	RefID        uint64    `gorm:"default:0" json:"ref_id"`            // 关联资源 ID
	Note         string    `gorm:"size:200;default:''" json:"note"`
	CreatedAt    time.Time `json:"created_at"`
}

func (CreditLog) TableName() string { return "credit_logs" }

// CreditLot 积分批次表：每次充值/赠送生成一条，扣费按 expires_at 升序逐批消耗（最近过期优先）
type CreditLot struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement;comment:批次ID" json:"id"`
	TenantID  uint64    `gorm:"not null;index:idx_lot_deduct,priority:1;comment:租户ID" json:"tenant_id"`
	Amount    int64     `gorm:"not null;comment:原始发放数量（不变，追溯用）" json:"amount"`
	Remaining int64     `gorm:"not null;comment:剩余可用数量" json:"remaining"`
	Source    string    `gorm:"size:30;not null;comment:来源：gift/admin_adjust/migration，与发放流水action一致" json:"source"`
	ExpiresAt time.Time `gorm:"not null;index:idx_lot_deduct,priority:2;index:idx_lot_expire;comment:过期时间" json:"expires_at"`
	CreatedAt time.Time `gorm:"comment:发放时间" json:"created_at"`
}

func (CreditLot) TableName() string { return "credit_lots" }

// CreditLogDetail 扣费明细表：一条扣费流水从哪些批次各扣了多少（追溯用）
type CreditLogDetail struct {
	ID     uint64 `gorm:"primaryKey;autoIncrement;comment:明细ID" json:"id"`
	LogID  uint64 `gorm:"not null;index;comment:流水ID，关联credit_logs.id" json:"log_id"`
	LotID  uint64 `gorm:"not null;index;comment:批次ID，关联credit_lots.id" json:"lot_id"`
	Amount int64  `gorm:"not null;comment:从该批次扣除的数量（正数）" json:"amount"`
}

func (CreditLogDetail) TableName() string { return "credit_log_details" }
