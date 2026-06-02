package repo

import (
	"context"
	"errors"
	"time"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/model"
	"gorm.io/gorm"
)

// ========== UserRepo ==========

type userRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) UserRepo {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("email = ? AND deleted_at IS NULL", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("username = ? AND deleted_at IS NULL", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) GetByID(ctx context.Context, id uint64) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// ========== TenantRepo ==========

type tenantRepo struct {
	db *gorm.DB
}

func NewTenantRepo(db *gorm.DB) TenantRepo {
	return &tenantRepo{db: db}
}

func (r *tenantRepo) Create(ctx context.Context, tenant *model.Tenant) error {
	return r.db.WithContext(ctx).Create(tenant).Error
}

func (r *tenantRepo) GetByID(ctx context.Context, id uint64) (*model.Tenant, error) {
	var tenant model.Tenant
	err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&tenant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tenant, nil
}

func (r *tenantRepo) ListByUserID(ctx context.Context, userID uint64) ([]*model.Tenant, error) {
	var tenants []*model.Tenant
	err := r.db.WithContext(ctx).
		Joins("JOIN tenant_members ON tenant_members.tenant_id = tenants.id").
		Where("tenant_members.user_id = ? AND tenants.deleted_at IS NULL", userID).
		Find(&tenants).Error
	if err != nil {
		return nil, err
	}
	return tenants, nil
}

func (r *tenantRepo) Update(ctx context.Context, tenant *model.Tenant) error {
	return r.db.WithContext(ctx).Save(tenant).Error
}

// ========== TenantMemberRepo ==========

type tenantMemberRepo struct {
	db *gorm.DB
}

func NewTenantMemberRepo(db *gorm.DB) TenantMemberRepo {
	return &tenantMemberRepo{db: db}
}

func (r *tenantMemberRepo) Add(ctx context.Context, member *model.TenantMember) error {
	return r.db.WithContext(ctx).Create(member).Error
}

func (r *tenantMemberRepo) Remove(ctx context.Context, tenantID, userID uint64) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Delete(&model.TenantMember{}).Error
}

func (r *tenantMemberRepo) GetRole(ctx context.Context, tenantID, userID uint64) (string, error) {
	var member model.TenantMember
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return member.Role, nil
}

func (r *tenantMemberRepo) UpdateRole(ctx context.Context, tenantID, userID uint64, role string) error {
	return r.db.WithContext(ctx).
		Model(&model.TenantMember{}).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Update("role", role).Error
}

func (r *tenantMemberRepo) ListMembers(ctx context.Context, tenantID uint64) ([]*model.TenantMember, error) {
	var members []*model.TenantMember
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("tenant_id = ?", tenantID).
		Find(&members).Error
	if err != nil {
		return nil, err
	}
	return members, nil
}

// ========== CategoryRepo ==========

type categoryRepo struct {
	db *gorm.DB
}

func NewCategoryRepo(db *gorm.DB) CategoryRepo {
	return &categoryRepo{db: db}
}

func (r *categoryRepo) Create(ctx context.Context, cat *model.Category) error {
	return r.db.WithContext(ctx).Create(cat).Error
}

func (r *categoryRepo) Update(ctx context.Context, cat *model.Category) error {
	return r.db.WithContext(ctx).Save(cat).Error
}

func (r *categoryRepo) Delete(ctx context.Context, id, tenantID uint64) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.Category{}).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		Update("deleted_at", now).Error
}

func (r *categoryRepo) GetByID(ctx context.Context, id, tenantID uint64) (*model.Category, error) {
	var cat model.Category
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		First(&cat).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cat, nil
}

func (r *categoryRepo) List(ctx context.Context, tenantID uint64) ([]*model.Category, error) {
	var cats []*model.Category
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Order("type, name").
		Find(&cats).Error
	if err != nil {
		return nil, err
	}
	return cats, nil
}

// ========== MerchantRepo ==========

type merchantRepo struct {
	db *gorm.DB
}

func NewMerchantRepo(db *gorm.DB) MerchantRepo {
	return &merchantRepo{db: db}
}

func (r *merchantRepo) Create(ctx context.Context, merchant *model.Merchant) error {
	return r.db.WithContext(ctx).Create(merchant).Error
}

func (r *merchantRepo) GetByID(ctx context.Context, id, tenantID uint64) (*model.Merchant, error) {
	var m model.Merchant
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *merchantRepo) GetByName(ctx context.Context, name string, tenantID uint64) (*model.Merchant, error) {
	var m model.Merchant
	err := r.db.WithContext(ctx).
		Where("name = ? AND tenant_id = ? AND deleted_at IS NULL", name, tenantID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *merchantRepo) List(ctx context.Context, tenantID uint64) ([]*model.Merchant, error) {
	var merchants []*model.Merchant
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Order("name").
		Find(&merchants).Error
	if err != nil {
		return nil, err
	}
	return merchants, nil
}

func (r *merchantRepo) Update(ctx context.Context, merchant *model.Merchant) error {
	return r.db.WithContext(ctx).Save(merchant).Error
}

func (r *merchantRepo) Delete(ctx context.Context, id, tenantID uint64) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.Merchant{}).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		Update("deleted_at", now).Error
}

// ========== TransactionRepo ==========

type transactionRepo struct {
	db *gorm.DB
}

func NewTransactionRepo(db *gorm.DB) TransactionRepo {
	return &transactionRepo{db: db}
}

func (r *transactionRepo) Create(ctx context.Context, tx *model.Transaction) error {
	return r.db.WithContext(ctx).Create(tx).Error
}

func (r *transactionRepo) BatchCreate(ctx context.Context, txs []*model.Transaction) error {
	return r.db.WithContext(ctx).CreateInBatches(txs, 100).Error
}

func (r *transactionRepo) Update(ctx context.Context, tx *model.Transaction) error {
	return r.db.WithContext(ctx).Save(tx).Error
}

func (r *transactionRepo) SaveImage(ctx context.Context, transactionID uint64, imagePath string, ocrAmount float64, ocrDate, ocrMerchant, ocrRawTexts string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("transaction_id = ?", transactionID).Delete(&model.TransactionImage{}).Error; err != nil {
			return err
		}
		if imagePath == "" {
			return nil
		}
		return tx.Create(&model.TransactionImage{
			TransactionID: transactionID,
			ImagePath:     imagePath,
			OCRAmount:     ocrAmount,
			OCRDate:       ocrDate,
			OCRMerchant:   ocrMerchant,
			OCRRawTexts:   ocrRawTexts,
		}).Error
	})
}

func (r *transactionRepo) Delete(ctx context.Context, id, tenantID uint64) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.Transaction{}).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		Update("deleted_at", now).Error
}

func (r *transactionRepo) GetByID(ctx context.Context, id, tenantID uint64) (*model.Transaction, error) {
	var tx model.Transaction
	err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("Merchant").
		Preload("Images").
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		First(&tx).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

func (r *transactionRepo) List(ctx context.Context, tenantID uint64, filter dto.TransactionFilter) ([]*model.Transaction, int64, error) {
	filter.Normalize()

	query := r.db.WithContext(ctx).
		Model(&model.Transaction{}).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID)

	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.CategoryID > 0 {
		query = query.Where("category_id = ?", filter.CategoryID)
	}
	if filter.StartDate != "" {
		query = query.Where("DATE(transaction_date) >= ?", filter.StartDate)
	}
	if filter.EndDate != "" {
		query = query.Where("DATE(transaction_date) <= ?", filter.EndDate)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var txs []*model.Transaction
	offset := (filter.Page - 1) * filter.PageSize
	err := query.
		Preload("Category").
		Preload("Merchant").
		Preload("Images").
		Order(filter.SortBy + " DESC, id DESC").
		Limit(filter.PageSize).
		Offset(offset).
		Find(&txs).Error
	if err != nil {
		return nil, 0, err
	}
	return txs, total, nil
}

type dailyStatRow struct {
	Date         string  `gorm:"column:date"`
	TotalIncome  float64 `gorm:"column:total_income"`
	TotalExpense float64 `gorm:"column:total_expense"`
}

func (r *transactionRepo) DailyStats(ctx context.Context, tenantID uint64, date string) (*dto.DailyStatResp, error) {
	var row dailyStatRow
	err := r.db.WithContext(ctx).
		Model(&model.Transaction{}).
		Select(`
			DATE(transaction_date) AS date,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) AS total_income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) AS total_expense
		`).
		Where("tenant_id = ? AND DATE(transaction_date) = ? AND deleted_at IS NULL", tenantID, date).
		Group("DATE(transaction_date)").
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	return &dto.DailyStatResp{
		Date:         date,
		TotalIncome:  row.TotalIncome,
		TotalExpense: row.TotalExpense,
	}, nil
}

func (r *transactionRepo) MonthlyStats(ctx context.Context, tenantID uint64, year, month int) ([]*dto.DailyStatResp, error) {
	var rows []dailyStatRow
	err := r.db.WithContext(ctx).
		Model(&model.Transaction{}).
		Select(`
			DATE(transaction_date) AS date,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) AS total_income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) AS total_expense
		`).
		Where("tenant_id = ? AND YEAR(transaction_date) = ? AND MONTH(transaction_date) = ? AND deleted_at IS NULL",
			tenantID, year, month).
		Group("DATE(transaction_date)").
		Order("DATE(transaction_date) ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]*dto.DailyStatResp, 0, len(rows))
	for _, row := range rows {
		result = append(result, &dto.DailyStatResp{
			Date:         row.Date,
			TotalIncome:  row.TotalIncome,
			TotalExpense: row.TotalExpense,
		})
	}
	return result, nil
}

type rangeStatRow struct {
	TotalIncome  float64 `gorm:"column:total_income"`
	TotalExpense float64 `gorm:"column:total_expense"`
}

func (r *transactionRepo) RangeStats(ctx context.Context, tenantID uint64, start, end string) (*dto.StatisticsResp, error) {
	var row rangeStatRow
	err := r.db.WithContext(ctx).
		Model(&model.Transaction{}).
		Select(`
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) AS total_income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) AS total_expense
		`).
		Where("tenant_id = ? AND DATE(transaction_date) >= ? AND DATE(transaction_date) <= ? AND deleted_at IS NULL",
			tenantID, start, end).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	return &dto.StatisticsResp{
		TotalIncome:  row.TotalIncome,
		TotalExpense: row.TotalExpense,
		NetAmount:    row.TotalIncome - row.TotalExpense,
		StartDate:    start,
		EndDate:      end,
	}, nil
}

// MonthlyCategoryStats 查询某月的分类统计（JOIN categories 表获取名称和图标）
func (r *transactionRepo) MonthlyCategoryStats(ctx context.Context, tenantID uint64, year, month int) ([]*dto.CategoryStat, error) {
	var rows []*dto.CategoryStat
	err := r.db.WithContext(ctx).
		Table("transactions t").
		Select(`
			t.category_id,
			c.name AS category_name,
			c.icon AS category_icon,
			t.type,
			COALESCE(SUM(t.amount), 0) AS total
		`).
		Joins("JOIN categories c ON t.category_id = c.id AND c.deleted_at IS NULL").
		Where("t.tenant_id = ? AND YEAR(t.transaction_date) = ? AND MONTH(t.transaction_date) = ? AND t.deleted_at IS NULL",
			tenantID, year, month).
		Group("t.category_id, c.name, c.icon, t.type").
		Order("total DESC").
		Scan(&rows).Error
	return rows, err
}

// RangeDailyStats 查询日期范围内的每日统计
func (r *transactionRepo) RangeDailyStats(ctx context.Context, tenantID uint64, start, end string) ([]*dto.DailyStatResp, error) {
	var rows []dailyStatRow
	err := r.db.WithContext(ctx).
		Model(&model.Transaction{}).
		Select(`
			DATE(transaction_date) AS date,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) AS total_income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) AS total_expense
		`).
		Where("tenant_id = ? AND DATE(transaction_date) >= ? AND DATE(transaction_date) <= ? AND deleted_at IS NULL",
			tenantID, start, end).
		Group("DATE(transaction_date)").
		Order("DATE(transaction_date) ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]*dto.DailyStatResp, 0, len(rows))
	for _, row := range rows {
		result = append(result, &dto.DailyStatResp{
			Date:         row.Date,
			TotalIncome:  row.TotalIncome,
			TotalExpense: row.TotalExpense,
		})
	}
	return result, nil
}

// RangeCategoryStats 查询日期范围内的分类统计
func (r *transactionRepo) RangeCategoryStats(ctx context.Context, tenantID uint64, start, end string) ([]*dto.CategoryStat, error) {
	var rows []*dto.CategoryStat
	err := r.db.WithContext(ctx).
		Table("transactions t").
		Select(`
			t.category_id,
			c.name AS category_name,
			c.icon AS category_icon,
			t.type,
			COALESCE(SUM(t.amount), 0) AS total
		`).
		Joins("JOIN categories c ON t.category_id = c.id AND c.deleted_at IS NULL").
		Where("t.tenant_id = ? AND DATE(t.transaction_date) >= ? AND DATE(t.transaction_date) <= ? AND t.deleted_at IS NULL",
			tenantID, start, end).
		Group("t.category_id, c.name, c.icon, t.type").
		Order("total DESC").
		Scan(&rows).Error
	return rows, err
}

// YearlyStats 查询某年的月度汇总
func (r *transactionRepo) YearlyStats(ctx context.Context, tenantID uint64, year int) ([]*dto.MonthSummary, error) {
	var rows []*dto.MonthSummary
	err := r.db.WithContext(ctx).
		Model(&model.Transaction{}).
		Select(`
			MONTH(transaction_date) AS month,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) AS total_income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) AS total_expense
		`).
		Where("tenant_id = ? AND YEAR(transaction_date) = ? AND deleted_at IS NULL",
			tenantID, year).
		Group("MONTH(transaction_date)").
		Order("MONTH(transaction_date) ASC").
		Scan(&rows).Error
	return rows, err
}

// YearlyCategoryStats 查询某年的分类统计
func (r *transactionRepo) YearlyCategoryStats(ctx context.Context, tenantID uint64, year int) ([]*dto.CategoryStat, error) {
	var rows []*dto.CategoryStat
	err := r.db.WithContext(ctx).
		Table("transactions t").
		Select(`
			t.category_id,
			c.name AS category_name,
			c.icon AS category_icon,
			t.type,
			COALESCE(SUM(t.amount), 0) AS total
		`).
		Joins("JOIN categories c ON t.category_id = c.id AND c.deleted_at IS NULL").
		Where("t.tenant_id = ? AND YEAR(t.transaction_date) = ? AND t.deleted_at IS NULL",
			tenantID, year).
		Group("t.category_id, c.name, c.icon, t.type").
		Order("total DESC").
		Scan(&rows).Error
	return rows, err
}

// ========== TenantLLMConfigRepo ==========

type tenantLLMConfigRepo struct {
	db *gorm.DB
}

func NewTenantLLMConfigRepo(db *gorm.DB) TenantLLMConfigRepo {
	return &tenantLLMConfigRepo{db: db}
}

func (r *tenantLLMConfigRepo) GetByTenantID(ctx context.Context, tenantID uint64) (*model.TenantLLMConfig, error) {
	var cfg model.TenantLLMConfig
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).First(&cfg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cfg, nil
}

func (r *tenantLLMConfigRepo) Save(ctx context.Context, cfg *model.TenantLLMConfig) error {
	return r.db.WithContext(ctx).Save(cfg).Error
}
