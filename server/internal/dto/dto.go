package dto

// ========== 通用响应 ==========

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func OK(data any) Response {
	return Response{Code: 0, Message: "ok", Data: data}
}

func Fail(code int, message string) Response {
	return Response{Code: code, Message: message, Data: nil}
}

// ========== 认证 ==========

type RegisterReq struct {
	Username string `json:"username" binding:"required,min=2,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type TenantSummary struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

type LoginResp struct {
	Token    string          `json:"token"`
	UserID   uint64          `json:"user_id"`
	Username string          `json:"username"`
	Email    string          `json:"email"`
	Tenants  []TenantSummary `json:"tenants"`
}

// ========== 租户 ==========

type CreateTenantReq struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
}

type UpdateTenantReq struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
}

type InviteMemberReq struct {
	Username string `json:"username" binding:"required,min=2,max=50"`
	Role     string `json:"role" binding:"required,min=1,max=50"`
}

type RemoveMemberReq struct {
	UserID uint64 `json:"user_id" binding:"required"`
}

type UpdateMemberRoleReq struct {
	Role string `json:"role" binding:"required,min=1,max=50"`
}

type TenantSettingsResp struct {
	RequireExpenseImage bool `json:"require_expense_image"`
}

type UpdateTenantSettingsReq struct {
	RequireExpenseImage bool `json:"require_expense_image"`
}

type TenantResp struct {
	ID        uint64      `json:"id"`
	Name      string      `json:"name"`
	OwnerID   uint64      `json:"owner_id"`
	CreatedAt string      `json:"created_at"`
}

type MemberResp struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	JoinedAt string `json:"joined_at"`
}

// ========== 分类 ==========

type CreateCategoryReq struct {
	Name string `json:"name" binding:"required,min=1,max=50"`
	Type string `json:"type" binding:"required,oneof=income expense"`
	Icon string `json:"icon"`
}

type UpdateCategoryReq struct {
	Name string `json:"name" binding:"required,min=1,max=50"`
	Type string `json:"type" binding:"required,oneof=income expense"`
	Icon string `json:"icon"`
}

// ========== 交易 ==========

type CreateTransactionReq struct {
	Type            string  `json:"type" binding:"required,oneof=income expense"`
	Amount          float64 `json:"amount" binding:"required,gt=0"`
	CategoryID      uint64  `json:"category_id" binding:"required"`
	MerchantID      uint64  `json:"merchant_id"`
	MerchantName    string  `json:"merchant_name"`
	TransactionDate string  `json:"transaction_date" binding:"required"`
	Note            string  `json:"note"`
	ImagePath       string  `json:"image_path"`
	OCRAmount       float64 `json:"ocr_amount"`
	OCRDate         string  `json:"ocr_date"`
	OCRMerchant     string  `json:"ocr_merchant"`
	OCRRawTexts     string  `json:"ocr_raw_texts"` // JSON数组
}

type UpdateTransactionReq struct {
	Type            string  `json:"type" binding:"required,oneof=income expense"`
	Amount          float64 `json:"amount" binding:"required,gt=0"`
	CategoryID      uint64  `json:"category_id" binding:"required"`
	MerchantID      uint64  `json:"merchant_id"`
	TransactionDate string  `json:"transaction_date" binding:"required"`
	Note            string  `json:"note"`
	ImagePath       string  `json:"image_path"`
	OCRAmount       float64 `json:"ocr_amount"`
	OCRDate         string  `json:"ocr_date"`
	OCRMerchant     string  `json:"ocr_merchant"`
	OCRRawTexts     string  `json:"ocr_raw_texts"`
}

type BatchCreateTransactionReq struct {
	Transactions []CreateTransactionReq `json:"transactions" binding:"required,min=1"`
}

type TransactionFilter struct {
	Type        string `form:"type"`
	CategoryID  uint64 `form:"category_id"`
	StartDate   string `form:"start_date"`
	EndDate     string `form:"end_date"`
	SortBy      string `form:"sort_by"` // "transaction_date" | "created_at"
	Page        int    `form:"page"`
	PageSize    int    `form:"page_size"`
}

func (f *TransactionFilter) Normalize() {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	if f.PageSize > 100000 {
		f.PageSize = 100000
	}
	if f.SortBy != "created_at" {
		f.SortBy = "transaction_date"
	}
}

type TransactionImageResp struct {
	ImagePath   string  `json:"image_path"`
	OCRAmount   float64 `json:"ocr_amount"`
	OCRDate     string  `json:"ocr_date"`
	OCRMerchant string  `json:"ocr_merchant"`
	OCRRawTexts string  `json:"ocr_raw_texts"`
}

type TransactionResp struct {
	ID              uint64                 `json:"id"`
	TenantID        uint64                 `json:"tenant_id"`
	UserID          uint64                 `json:"user_id"`
	Type            string                 `json:"type"`
	Amount          float64                `json:"amount"`
	CategoryID      uint64                 `json:"category_id"`
	CategoryName    string                 `json:"category_name,omitempty"`
	MerchantID      uint64                 `json:"merchant_id"`
	MerchantName    string                 `json:"merchant_name"`
	TransactionDate string                 `json:"transaction_date"`
	Note            string                 `json:"note"`
	CreatedAt       string                 `json:"created_at"`
	Images          []*TransactionImageResp `json:"images,omitempty"`
}

type TransactionListResp struct {
	Total int64              `json:"total"`
	Page  int                `json:"page"`
	Size  int                `json:"size"`
	Items []*TransactionResp `json:"items"`
}

// ========== 商户 ==========

type MerchantResp struct {
	ID        uint64 `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

// ========== OCR ==========

type OCRResult struct {
	AIMode   bool      `json:"ai_mode"`
	Amount   float64   `json:"amount"`
	Date     string    `json:"date"`
	Merchant string    `json:"merchant"`
	RawTexts []OCRText `json:"raw_texts"`
}

type OCRText struct {
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
}

// OCRResponse OCR识别接口响应（/upload/ocr），前端用 ocr_id 再请求 LLM
type OCRResponse struct {
	OcrID        uint64   `json:"ocr_id"`
	ImagePath    string   `json:"image_path"`
	AIMode       bool     `json:"ai_mode"`
	Amount       float64  `json:"amount"`
	Date         string   `json:"date"`
	MerchantID   uint64   `json:"merchant_id"`
	MerchantName string   `json:"merchant_name"`
	RawTexts     []string `json:"raw_texts"`
	Duplicate    bool     `json:"duplicate,omitempty"`
	FileName     string   `json:"file_name,omitempty"`
}

// OCRAnalyzeResponse OCR+LLM 合并接口响应（/upload/ocr/analyze，不改动）
type OCRAnalyzeResponse struct {
	ImagePath    string           `json:"image_path"`
	AIMode       bool             `json:"ai_mode"`
	Amount       float64          `json:"amount"`
	Date         string           `json:"date"`
	MerchantID   uint64           `json:"merchant_id"`
	MerchantName string           `json:"merchant_name"`
	RawTexts     []string         `json:"raw_texts"`
	LLM          []*LLMSuggestion `json:"llm,omitempty"`
	LLMError     string           `json:"llm_error,omitempty"`
	Duplicate    bool             `json:"duplicate,omitempty"`
	FileName     string           `json:"file_name,omitempty"`
}

// ========== LLM ==========

// CategoryItem 传给 LLM 的分类条目
type CategoryItem struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // "income" | "expense"
}

// LLMSuggestion LLM 返回的记一笔建议
type LLMSuggestion struct {
	Type         string  `json:"type"`          // "expense" | "income"
	Amount       float64 `json:"amount"`
	MerchantName string  `json:"merchant_name"`
	Date         string  `json:"date"`          // "YYYY-MM-DD"
	CategoryID   uint64  `json:"category_id"`   // 匹配到的分类 ID（0 表示未匹配）
	CategoryHint string  `json:"category_hint"` // 中文分类提示，如"餐饮"
	Note         string  `json:"note"`
	SourceLines  []int   `json:"source_lines"`  // 对应 OCR 原始文字的行号（从 0 开始）
}

// TenantLLMConfigResp 返回给前端的租户 LLM 配置（APIKey 脱敏）
type TenantLLMConfigResp struct {
	ID              uint64 `json:"id"`
	TenantID        uint64 `json:"tenant_id"`
	Enabled         bool   `json:"enabled"`
	UsePlatform     bool   `json:"use_platform"`
	PlatformEnabled bool   `json:"platform_enabled"` // 平台是否已配置 LLM（供前端提示用）
	Provider        string `json:"provider"`
	BaseURL         string `json:"base_url"`
	APIKeyMask      string `json:"api_key_mask"` // 脱敏后，如 "sk-...abc"
	Model           string `json:"model"`
	Mode            string `json:"mode"`
}

// SaveTenantLLMConfigReq 保存租户 LLM 配置请求
type SaveTenantLLMConfigReq struct {
	Enabled     bool   `json:"enabled"`
	UsePlatform bool   `json:"use_platform"`
	Provider    string `json:"provider" binding:"omitempty,oneof=openai azure deepseek ollama"`
	BaseURL     string `json:"base_url"`
	APIKey      string `json:"api_key"` // 为空时保留原来的 key
	Model       string `json:"model"`
	Mode        string `json:"mode" binding:"omitempty,oneof=vision ocr_text"`
}

// LLMAnalyzeReq 单独调用 LLM 分析的请求（只接受 ocr_id，服务端从 ocr_records 取内容）
type LLMAnalyzeReq struct {
	OcrID uint64 `json:"ocr_id" binding:"required"`
}

// LLMAnalyzeResp 单独调用 LLM 分析的响应
type LLMAnalyzeResp struct {
	Suggestions []*LLMSuggestion `json:"suggestions"`
	Error       string           `json:"error,omitempty"`
}

// ========== 统计 ==========

type StatisticsResp struct {
	TotalIncome  float64 `json:"total_income"`
	TotalExpense float64 `json:"total_expense"`
	NetAmount    float64 `json:"net_amount"`
	StartDate    string  `json:"start_date,omitempty"`
	EndDate      string  `json:"end_date,omitempty"`
}

type MerchantStat struct {
	MerchantID   uint64  `json:"merchant_id"`
	MerchantName string  `json:"merchant_name"`
	Total        float64 `json:"total"`
	TxCount      int     `json:"tx_count"`
}

type MerchantStatsResp struct {
	TopMerchants []*MerchantStat `json:"top_merchants"`
	ComputedAt   string          `json:"computed_at,omitempty"`
	FromCache    bool            `json:"from_cache"`
}

// ========== RBAC ==========

type PermissionEntry struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

type CreateRoleReq struct {
	Name        string            `json:"name" binding:"required,min=1,max=50"`
	Permissions []PermissionEntry `json:"permissions" binding:"required,min=1"`
}

type UpdateRoleReq struct {
	Permissions []PermissionEntry `json:"permissions" binding:"required,min=1"`
}

type RoleResp struct {
	ID          uint64            `json:"id"`
	Name        string            `json:"name"`
	IsSystem    bool              `json:"is_system"`
	Permissions []PermissionEntry `json:"permissions"`
}

type DailyStatResp struct {
	Date         string  `json:"date"`
	TotalIncome  float64 `json:"total_income"`
	TotalExpense float64 `json:"total_expense"`
}

type CategoryStat struct {
	CategoryID   uint64  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	CategoryIcon string  `json:"category_icon"`
	Type         string  `json:"type"`
	Total        float64 `json:"total"`
}

// MonthlyStatResp 月统计响应
type MonthlyStatResp struct {
	Year         int              `json:"year"`
	Month        int              `json:"month"`
	Total        StatisticsResp   `json:"total"`
	Daily        []*DailyStatResp `json:"daily"`
	Categories   []*CategoryStat  `json:"categories"`
	TopMerchants []*MerchantStat  `json:"top_merchants"`
}

// MonthSummary 月度汇总（年视图用）
type MonthSummary struct {
	Month        int     `json:"month"`
	TotalIncome  float64 `json:"total_income"`
	TotalExpense float64 `json:"total_expense"`
}

// YearlyStatResp 年统计响应
type YearlyStatResp struct {
	Year         int              `json:"year"`
	Total        StatisticsResp   `json:"total"`
	Monthly      []*MonthSummary  `json:"monthly"`
	Categories   []*CategoryStat  `json:"categories"`
	TopMerchants []*MerchantStat  `json:"top_merchants"`
}

// RangeStatResp 日期范围统计响应
type RangeStatResp struct {
	Total        StatisticsResp   `json:"total"`
	Daily        []*DailyStatResp `json:"daily"`
	Categories   []*CategoryStat  `json:"categories"`
	TopMerchants []*MerchantStat  `json:"top_merchants"`
}

// ========== 平台管理员 ==========

type PlatformLoginReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type PlatformLoginResp struct {
	Token string `json:"token"`
	ID    uint64 `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type PlatformUserItem struct {
	ID        uint64 `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

type PlatformUserListResp struct {
	Total int64              `json:"total"`
	Page  int                `json:"page"`
	Size  int                `json:"size"`
	Items []*PlatformUserItem `json:"items"`
}

type PlatformUserDetailResp struct {
	UserID           uint64 `json:"user_id"`
	Username         string `json:"username"`
	Email            string `json:"email"`
	TenantCount      int64  `json:"tenant_count"`
	TransactionCount int64  `json:"transaction_count"`
	MediaCount       int64  `json:"media_count"`
}

type PlatformDashboardResp struct {
	TotalUsers        int64 `json:"total_users"`
	TotalTenants      int64 `json:"total_tenants"`
	TotalTransactions int64 `json:"total_transactions"`
}

type PlatformUserFilter struct {
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

func (f *PlatformUserFilter) Normalize() {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	if f.PageSize > 100 {
		f.PageSize = 100
	}
}

// ========== 平台配置 ==========

type PlatformConfigItem struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
	UpdatedAt   string `json:"updated_at"`
}

type PlatformConfigListResp struct {
	Items []*PlatformConfigItem `json:"items"`
}

type PlatformConfigUpdateReq struct {
	Value string `json:"value" binding:"required"`
}
