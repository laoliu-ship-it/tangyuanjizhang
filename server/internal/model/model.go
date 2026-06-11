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
	CreatedAt    time.Time  `json:"created_at"`
	DeletedAt    *time.Time `gorm:"index" json:"-"`
}

func (User) TableName() string { return "users" }

// Tenant 租户表
type Tenant struct {
	ID        uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string     `gorm:"size:100;not null" json:"name"`
	OwnerID   uint64     `gorm:"not null" json:"owner_id"`
	CreatedAt time.Time  `json:"created_at"`
	DeletedAt *time.Time `gorm:"index" json:"-"`

	Owner   *User           `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Members []*TenantMember `gorm:"foreignKey:TenantID" json:"members,omitempty"`
}

func (Tenant) TableName() string { return "tenants" }

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
	RawTexts     string    `gorm:"type:text;default:''" json:"raw_texts"` // JSON []string
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
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID    uint64    `gorm:"not null;uniqueIndex" json:"tenant_id"`
	Enabled     bool      `gorm:"default:false" json:"enabled"`
	UsePlatform bool      `gorm:"default:true" json:"use_platform"` // true=使用平台配置，false=使用租户自己的key
	Provider    string    `gorm:"size:50;default:'openai'" json:"provider"`
	BaseURL     string    `gorm:"size:255;default:''" json:"base_url"`
	APIKey      string    `gorm:"size:500;default:''" json:"-"` // 不直接返回给前端
	Model       string    `gorm:"size:100;default:'gpt-4o'" json:"model"`
	Mode        string    `gorm:"size:20;default:'ocr_text'" json:"mode"` // "vision" | "ocr_text"
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (TenantLLMConfig) TableName() string { return "tenant_llm_configs" }

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
	ID                  uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID            uint64    `gorm:"not null;uniqueIndex" json:"tenant_id"`
	RequireExpenseImage bool      `gorm:"default:true" json:"require_expense_image"` // 支出记录是否必须上传图片
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
}
