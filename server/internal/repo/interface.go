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
	ListAll(ctx context.Context) ([]*model.Tenant, error)
	Update(ctx context.Context, tenant *model.Tenant) error
}

// TenantMemberRepo 租户成员数据仓库接口
type TenantMemberRepo interface {
	Add(ctx context.Context, member *model.TenantMember) error
	Remove(ctx context.Context, tenantID, userID uint64) error
	GetRole(ctx context.Context, tenantID, userID uint64) (string, error)
	GetRoleWithID(ctx context.Context, tenantID, userID uint64) (*uint64, string, error)
	UpdateRole(ctx context.Context, tenantID, userID uint64, role string) error
	UpdateRoleID(ctx context.Context, tenantID, userID uint64, roleID uint64) error
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
	MonthlyMerchantStats(ctx context.Context, tenantID uint64, year, month int) ([]*dto.MerchantStat, error)
	YearlyMerchantStats(ctx context.Context, tenantID uint64, year int) ([]*dto.MerchantStat, error)
	RangeMerchantStats(ctx context.Context, tenantID uint64, start, end string) ([]*dto.MerchantStat, error)
}

// StatsCacheRepo 统计缓存数据仓库接口
type StatsCacheRepo interface {
	Get(ctx context.Context, tenantID uint64, cacheType, periodKey string) (*model.StatsCache, error)
	Upsert(ctx context.Context, cache *model.StatsCache) error
}

// OcrRecordRepo OCR记录数据仓库接口
type OcrRecordRepo interface {
	Create(ctx context.Context, r *model.OcrRecord) error
	// GetByID 同时校验 tenantID，防止跨租户访问
	GetByID(ctx context.Context, id, tenantID uint64) (*model.OcrRecord, error)
}

// TenantLLMConfigRepo 租户 LLM 配置数据仓库接口
type TenantLLMConfigRepo interface {
	// GetByTenantID 获取租户配置，不存在时返回 nil, nil
	GetByTenantID(ctx context.Context, tenantID uint64) (*model.TenantLLMConfig, error)
	// Save 创建或更新（upsert）租户配置
	Save(ctx context.Context, cfg *model.TenantLLMConfig) error
}

// TenantRoleRepo 租户角色数据仓库接口
type TenantRoleRepo interface {
	Create(ctx context.Context, role *model.TenantRole) error
	GetByID(ctx context.Context, id uint64) (*model.TenantRole, error)
	GetByName(ctx context.Context, tenantID uint64, name string) (*model.TenantRole, error)
	ListByTenant(ctx context.Context, tenantID uint64) ([]*model.TenantRole, error)
	Update(ctx context.Context, role *model.TenantRole) error
	Delete(ctx context.Context, id uint64) error
}

// RolePermissionRepo 角色权限数据仓库接口
type RolePermissionRepo interface {
	Create(ctx context.Context, perm *model.RolePermission) error
	BatchCreate(ctx context.Context, perms []*model.RolePermission) error
	DeleteByRoleID(ctx context.Context, roleID uint64) error
	ListByRoleID(ctx context.Context, roleID uint64) ([]*model.RolePermission, error)
	ListByTenantID(ctx context.Context, tenantID uint64) ([]*model.RolePermission, error)
}

// MediaFileRepo 媒体文件数据仓库接口（用于去重和资源统计）
type MediaFileRepo interface {
	GetByHash(ctx context.Context, tenantID uint64, originalHash string) (*model.MediaFile, error)
	Create(ctx context.Context, mf *model.MediaFile) error
}

// TenantSettingsRepo 租户通用设置数据仓库接口
type TenantSettingsRepo interface {
	GetByTenantID(ctx context.Context, tenantID uint64) (*model.TenantSettings, error)
	Save(ctx context.Context, s *model.TenantSettings) error
}

// PlatformAdminRepo 平台管理员数据仓库接口
type PlatformAdminRepo interface {
	Create(ctx context.Context, admin *model.PlatformAdmin) error
	GetByEmail(ctx context.Context, email string) (*model.PlatformAdmin, error)
	GetByID(ctx context.Context, id uint64) (*model.PlatformAdmin, error)
}

// PlatformStatsRepo 平台统计查询接口（跨租户）
type PlatformStatsRepo interface {
	CountUsers(ctx context.Context) (int64, error)
	CountTenants(ctx context.Context) (int64, error)
	CountTransactions(ctx context.Context) (int64, error)
	ListUsers(ctx context.Context, keyword string, page, pageSize int) ([]*model.User, int64, error)
	CountTransactionsByUserID(ctx context.Context, userID uint64) (int64, error)
	CountMediaByUserID(ctx context.Context, userID uint64) (int64, error)
	CountTenantsByUserID(ctx context.Context, userID uint64) (int64, error)
}

// PlatformConfigRepo 平台配置数据仓库接口
type PlatformConfigRepo interface {
	GetByKey(ctx context.Context, key string) (*model.PlatformConfig, error)
	GetAll(ctx context.Context) ([]*model.PlatformConfig, error)
	Upsert(ctx context.Context, cfg *model.PlatformConfig) error
}
