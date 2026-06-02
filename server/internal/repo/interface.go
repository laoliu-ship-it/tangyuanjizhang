package repo

import (
	"context"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/model"
)

// UserRepo 用户数据仓库接口
type UserRepo interface {
	Create(ctx context.Context, user *model.User) error
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByID(ctx context.Context, id uint64) (*model.User, error)
}

// TenantRepo 租户数据仓库接口
type TenantRepo interface {
	Create(ctx context.Context, tenant *model.Tenant) error
	GetByID(ctx context.Context, id uint64) (*model.Tenant, error)
	ListByUserID(ctx context.Context, userID uint64) ([]*model.Tenant, error)
	Update(ctx context.Context, tenant *model.Tenant) error
}

// TenantMemberRepo 租户成员数据仓库接口
type TenantMemberRepo interface {
	Add(ctx context.Context, member *model.TenantMember) error
	Remove(ctx context.Context, tenantID, userID uint64) error
	GetRole(ctx context.Context, tenantID, userID uint64) (string, error)
	UpdateRole(ctx context.Context, tenantID, userID uint64, role string) error
	ListMembers(ctx context.Context, tenantID uint64) ([]*model.TenantMember, error)
}

// CategoryRepo 分类数据仓库接口
type CategoryRepo interface {
	Create(ctx context.Context, cat *model.Category) error
	Update(ctx context.Context, cat *model.Category) error
	Delete(ctx context.Context, id, tenantID uint64) error
	GetByID(ctx context.Context, id, tenantID uint64) (*model.Category, error)
	List(ctx context.Context, tenantID uint64) ([]*model.Category, error)
}

// MerchantRepo 商户数据仓库接口
type MerchantRepo interface {
	Create(ctx context.Context, merchant *model.Merchant) error
	GetByID(ctx context.Context, id, tenantID uint64) (*model.Merchant, error)
	GetByName(ctx context.Context, name string, tenantID uint64) (*model.Merchant, error)
	List(ctx context.Context, tenantID uint64) ([]*model.Merchant, error)
	Update(ctx context.Context, merchant *model.Merchant) error
	Delete(ctx context.Context, id, tenantID uint64) error
}

// TransactionRepo 交易记录数据仓库接口
type TransactionRepo interface {
	Create(ctx context.Context, tx *model.Transaction) error
	BatchCreate(ctx context.Context, txs []*model.Transaction) error
	Update(ctx context.Context, tx *model.Transaction) error
	Delete(ctx context.Context, id, tenantID uint64) error
	GetByID(ctx context.Context, id, tenantID uint64) (*model.Transaction, error)
	List(ctx context.Context, tenantID uint64, filter dto.TransactionFilter) ([]*model.Transaction, int64, error)
	SaveImage(ctx context.Context, transactionID uint64, imagePath string, ocrAmount float64, ocrDate, ocrMerchant, ocrRawTexts string) error
	DailyStats(ctx context.Context, tenantID uint64, date string) (*dto.DailyStatResp, error)
	MonthlyStats(ctx context.Context, tenantID uint64, year, month int) ([]*dto.DailyStatResp, error)
	MonthlyCategoryStats(ctx context.Context, tenantID uint64, year, month int) ([]*dto.CategoryStat, error)
	RangeStats(ctx context.Context, tenantID uint64, start, end string) (*dto.StatisticsResp, error)
	RangeDailyStats(ctx context.Context, tenantID uint64, start, end string) ([]*dto.DailyStatResp, error)
	RangeCategoryStats(ctx context.Context, tenantID uint64, start, end string) ([]*dto.CategoryStat, error)
	YearlyStats(ctx context.Context, tenantID uint64, year int) ([]*dto.MonthSummary, error)
	YearlyCategoryStats(ctx context.Context, tenantID uint64, year int) ([]*dto.CategoryStat, error)
}

// TenantLLMConfigRepo 租户 LLM 配置数据仓库接口
type TenantLLMConfigRepo interface {
	// GetByTenantID 获取租户配置，不存在时返回 nil, nil
	GetByTenantID(ctx context.Context, tenantID uint64) (*model.TenantLLMConfig, error)
	// Save 创建或更新（upsert）租户配置
	Save(ctx context.Context, cfg *model.TenantLLMConfig) error
}
