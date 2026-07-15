package repo

import (
	"context"
	"errors"
	"time"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInsufficientCredits = errors.New("积分不足")

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

func (r *userRepo) CountByRegisterIP(ctx context.Context, ip string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("register_ip = ? AND deleted_at IS NULL", ip).Count(&count).Error
	return count, err
}

func (r *userRepo) UpdateLastLoginAt(ctx context.Context, userID uint64, t time.Time) error {
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).Update("last_login_at", t).Error
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

func (r *tenantRepo) ListAll(ctx context.Context) ([]*model.Tenant, error) {
	var tenants []*model.Tenant
	err := r.db.WithContext(ctx).Where("deleted_at IS NULL").Find(&tenants).Error
	return tenants, err
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

func (r *tenantMemberRepo) GetRoleWithID(ctx context.Context, tenantID, userID uint64) (*uint64, string, error) {
	var member model.TenantMember
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", nil
		}
		return nil, "", err
	}
	return member.RoleID, member.Role, nil
}

func (r *tenantMemberRepo) UpdateRole(ctx context.Context, tenantID, userID uint64, role string) error {
	return r.db.WithContext(ctx).
		Model(&model.TenantMember{}).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Update("role", role).Error
}

func (r *tenantMemberRepo) UpdateRoleID(ctx context.Context, tenantID, userID uint64, roleID uint64) error {
	return r.db.WithContext(ctx).
		Model(&model.TenantMember{}).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Update("role_id", roleID).Error
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

func (r *transactionRepo) BatchDelete(ctx context.Context, ids []uint64, tenantID uint64) (int64, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&model.Transaction{}).
		Where("id IN ? AND tenant_id = ? AND deleted_at IS NULL", ids, tenantID).
		Update("deleted_at", now)
	return result.RowsAffected, result.Error
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
	// 日期范围与排序共用同一时间字段：SortBy 已在 Normalize() 中白名单收敛为
	// "transaction_date" 或 "created_at"，此处直接复用，避免筛选字段与用户勾选的
	// 时间类型（交易时间/创建时间）不一致造成困惑
	if filter.StartDate != "" {
		query = query.Where("DATE("+filter.SortBy+") >= ?", filter.StartDate)
	}
	if filter.EndDate != "" {
		query = query.Where("DATE("+filter.SortBy+") <= ?", filter.EndDate)
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

// ========== 商户统计 ==========

type merchantStatRow struct {
	MerchantID   uint64  `gorm:"column:merchant_id"`
	MerchantName string  `gorm:"column:merchant_name"`
	Total        float64 `gorm:"column:total"`
	TxCount      int     `gorm:"column:tx_count"`
}

func merchantStatRows(rows []merchantStatRow) []*dto.MerchantStat {
	result := make([]*dto.MerchantStat, 0, len(rows))
	for _, r := range rows {
		result = append(result, &dto.MerchantStat{
			MerchantID:   r.MerchantID,
			MerchantName: r.MerchantName,
			Total:        r.Total,
			TxCount:      r.TxCount,
		})
	}
	return result
}

func (r *transactionRepo) MonthlyMerchantStats(ctx context.Context, tenantID uint64, year, month int) ([]*dto.MerchantStat, error) {
	var rows []merchantStatRow
	err := r.db.WithContext(ctx).
		Table("transactions t").
		Select(`t.merchant_id, m.name AS merchant_name, COALESCE(SUM(t.amount),0) AS total, COUNT(t.id) AS tx_count`).
		Joins("JOIN merchants m ON t.merchant_id = m.id AND m.deleted_at IS NULL").
		Where("t.tenant_id = ? AND t.type = 'expense' AND YEAR(t.transaction_date) = ? AND MONTH(t.transaction_date) = ? AND t.deleted_at IS NULL AND t.merchant_id > 0",
			tenantID, year, month).
		Group("t.merchant_id, m.name").
		Order("total DESC").
		Limit(10).
		Scan(&rows).Error
	return merchantStatRows(rows), err
}

func (r *transactionRepo) YearlyMerchantStats(ctx context.Context, tenantID uint64, year int) ([]*dto.MerchantStat, error) {
	var rows []merchantStatRow
	err := r.db.WithContext(ctx).
		Table("transactions t").
		Select(`t.merchant_id, m.name AS merchant_name, COALESCE(SUM(t.amount),0) AS total, COUNT(t.id) AS tx_count`).
		Joins("JOIN merchants m ON t.merchant_id = m.id AND m.deleted_at IS NULL").
		Where("t.tenant_id = ? AND t.type = 'expense' AND YEAR(t.transaction_date) = ? AND t.deleted_at IS NULL AND t.merchant_id > 0",
			tenantID, year).
		Group("t.merchant_id, m.name").
		Order("total DESC").
		Limit(10).
		Scan(&rows).Error
	return merchantStatRows(rows), err
}

func (r *transactionRepo) RangeMerchantStats(ctx context.Context, tenantID uint64, start, end string) ([]*dto.MerchantStat, error) {
	var rows []merchantStatRow
	err := r.db.WithContext(ctx).
		Table("transactions t").
		Select(`t.merchant_id, m.name AS merchant_name, COALESCE(SUM(t.amount),0) AS total, COUNT(t.id) AS tx_count`).
		Joins("JOIN merchants m ON t.merchant_id = m.id AND m.deleted_at IS NULL").
		Where("t.tenant_id = ? AND t.type = 'expense' AND DATE(t.transaction_date) >= ? AND DATE(t.transaction_date) <= ? AND t.deleted_at IS NULL AND t.merchant_id > 0",
			tenantID, start, end).
		Group("t.merchant_id, m.name").
		Order("total DESC").
		Limit(10).
		Scan(&rows).Error
	return merchantStatRows(rows), err
}

// ========== StatsCacheRepo ==========

type statsCacheRepo struct {
	db *gorm.DB
}

func NewStatsCacheRepo(db *gorm.DB) StatsCacheRepo {
	return &statsCacheRepo{db: db}
}

func (r *statsCacheRepo) Get(ctx context.Context, tenantID uint64, cacheType, periodKey string) (*model.StatsCache, error) {
	var cache model.StatsCache
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND cache_type = ? AND period_key = ?", tenantID, cacheType, periodKey).
		First(&cache).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cache, nil
}

func (r *statsCacheRepo) Upsert(ctx context.Context, cache *model.StatsCache) error {
	return r.db.WithContext(ctx).Exec(
		`INSERT INTO stats_cache (tenant_id, cache_type, period_key, data, computed_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE data = VALUES(data), computed_at = VALUES(computed_at)`,
		cache.TenantID, cache.CacheType, cache.PeriodKey, cache.Data, cache.ComputedAt,
	).Error
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

// ========== LLMTokenUsageRepo ==========

type llmTokenUsageRepo struct {
	db *gorm.DB
}

func NewLLMTokenUsageRepo(db *gorm.DB) LLMTokenUsageRepo {
	return &llmTokenUsageRepo{db: db}
}

// Create 按 (tenant_id, action, ref_type, ref_id) 幂等写入；已存在则静默跳过，
// 避免重复请求/竞态（如用户重复点击“AI整理”、并发请求同时未命中 LLM 缓存）产生重复 token 记录
func (r *llmTokenUsageRepo) Create(ctx context.Context, u *model.LLMTokenUsage) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(u).Error
}

// ========== TenantSettingsRepo ==========

type tenantSettingsRepo struct {
	db *gorm.DB
}

func NewTenantSettingsRepo(db *gorm.DB) TenantSettingsRepo {
	return &tenantSettingsRepo{db: db}
}

func (r *tenantSettingsRepo) GetByTenantID(ctx context.Context, tenantID uint64) (*model.TenantSettings, error) {
	var s model.TenantSettings
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *tenantSettingsRepo) Save(ctx context.Context, s *model.TenantSettings) error {
	return r.db.WithContext(ctx).Save(s).Error
}

// ========== MediaFileRepo ==========

type mediaFileRepo struct {
	db *gorm.DB
}

func NewMediaFileRepo(db *gorm.DB) MediaFileRepo {
	return &mediaFileRepo{db: db}
}

func (r *mediaFileRepo) GetByHash(ctx context.Context, tenantID uint64, originalHash string) (*model.MediaFile, error) {
	var mf model.MediaFile
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND original_hash = ?", tenantID, originalHash).
		First(&mf).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &mf, nil
}

func (r *mediaFileRepo) Create(ctx context.Context, mf *model.MediaFile) error {
	return r.db.WithContext(ctx).Create(mf).Error
}

// ========== PlatformAdminRepo ==========

type platformAdminRepo struct {
	db *gorm.DB
}

func NewPlatformAdminRepo(db *gorm.DB) PlatformAdminRepo {
	return &platformAdminRepo{db: db}
}

func (r *platformAdminRepo) Create(ctx context.Context, admin *model.PlatformAdmin) error {
	return r.db.WithContext(ctx).Create(admin).Error
}

func (r *platformAdminRepo) GetByEmail(ctx context.Context, email string) (*model.PlatformAdmin, error) {
	var admin model.PlatformAdmin
	err := r.db.WithContext(ctx).Where("email = ? AND deleted_at IS NULL", email).First(&admin).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &admin, nil
}

func (r *platformAdminRepo) GetByID(ctx context.Context, id uint64) (*model.PlatformAdmin, error) {
	var admin model.PlatformAdmin
	err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&admin).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &admin, nil
}

// ========== PlatformStatsRepo ==========

type platformStatsRepo struct {
	db *gorm.DB
}

func NewPlatformStatsRepo(db *gorm.DB) PlatformStatsRepo {
	return &platformStatsRepo{db: db}
}

func (r *platformStatsRepo) CountUsers(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Where("deleted_at IS NULL").Count(&count).Error
	return count, err
}

func (r *platformStatsRepo) CountTenants(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Tenant{}).Where("deleted_at IS NULL").Count(&count).Error
	return count, err
}

func (r *platformStatsRepo) CountTransactions(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Transaction{}).Where("deleted_at IS NULL").Count(&count).Error
	return count, err
}

func (r *platformStatsRepo) ListUsers(ctx context.Context, keyword string, page, pageSize int) ([]*model.User, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.User{}).Where("deleted_at IS NULL")
	if keyword != "" {
		query = query.Where("username LIKE ? OR email LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []*model.User
	offset := (page - 1) * pageSize
	err := query.Order("id DESC").Limit(pageSize).Offset(offset).Find(&users).Error
	return users, total, err
}

func (r *platformStatsRepo) LastTransactionTimesByUserIDs(ctx context.Context, userIDs []uint64) (map[uint64]time.Time, error) {
	result := make(map[uint64]time.Time, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	var rows []struct {
		UserID uint64
		LastAt time.Time
	}
	err := r.db.WithContext(ctx).Model(&model.Transaction{}).
		Select("user_id, MAX(created_at) AS last_at").
		Where("user_id IN ? AND deleted_at IS NULL", userIDs).
		Group("user_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.UserID] = row.LastAt
	}
	return result, nil
}

func (r *platformStatsRepo) CountTransactionsByUserID(ctx context.Context, userID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Transaction{}).Where("user_id = ? AND deleted_at IS NULL", userID).Count(&count).Error
	return count, err
}

func (r *platformStatsRepo) CountMediaByUserID(ctx context.Context, userID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.MediaFile{}).
		Joins("JOIN tenant_members ON tenant_members.tenant_id = media_files.tenant_id").
		Where("tenant_members.user_id = ?", userID).
		Count(&count).Error
	return count, err
}

func (r *platformStatsRepo) CountTenantsByUserID(ctx context.Context, userID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.TenantMember{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

// ========== PlatformConfigRepo ==========

type platformConfigRepo struct {
	db *gorm.DB
}

func NewPlatformConfigRepo(db *gorm.DB) PlatformConfigRepo {
	return &platformConfigRepo{db: db}
}

func (r *platformConfigRepo) GetByKey(ctx context.Context, key string) (*model.PlatformConfig, error) {
	var cfg model.PlatformConfig
	err := r.db.WithContext(ctx).Where("config_key = ?", key).First(&cfg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cfg, nil
}

func (r *platformConfigRepo) GetAll(ctx context.Context) ([]*model.PlatformConfig, error) {
	var configs []*model.PlatformConfig
	err := r.db.WithContext(ctx).Order("config_key ASC").Find(&configs).Error
	return configs, err
}

func (r *platformConfigRepo) Upsert(ctx context.Context, cfg *model.PlatformConfig) error {
	cfg.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.PlatformConfig
		err := tx.Where("config_key = ?", cfg.ConfigKey).First(&existing).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return tx.Create(cfg).Error
			}
			return err
		}
		return tx.Model(&existing).Updates(map[string]interface{}{
			"config_value": cfg.ConfigValue,
			"description":  cfg.Description,
			"updated_at":   cfg.UpdatedAt,
		}).Error
	})
}

// ========== OcrRecordRepo ==========

type ocrRecordRepo struct {
	db *gorm.DB
}

func NewOcrRecordRepo(db *gorm.DB) OcrRecordRepo {
	return &ocrRecordRepo{db: db}
}

func (r *ocrRecordRepo) Create(ctx context.Context, rec *model.OcrRecord) error {
	return r.db.WithContext(ctx).Create(rec).Error
}

func (r *ocrRecordRepo) GetByID(ctx context.Context, id, tenantID uint64) (*model.OcrRecord, error) {
	var rec model.OcrRecord
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		First(&rec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}

func (r *ocrRecordRepo) GetByHash(ctx context.Context, tenantID uint64, imageHash string) (*model.OcrRecord, error) {
	if imageHash == "" {
		return nil, nil
	}
	var rec model.OcrRecord
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND image_hash = ?", tenantID, imageHash).
		Order("id DESC").
		First(&rec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}

// ========== TenantCreditsRepo ==========

type tenantCreditsRepo struct {
	db *gorm.DB
}

func NewTenantCreditsRepo(db *gorm.DB) TenantCreditsRepo {
	return &tenantCreditsRepo{db: db}
}

func (r *tenantCreditsRepo) GetOrCreate(ctx context.Context, tenantID uint64) (*model.TenantCredits, error) {
	var c model.TenantCredits
	err := r.db.WithContext(ctx).
		Where(model.TenantCredits{TenantID: tenantID}).
		FirstOrCreate(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// lockCreditsRowTx 确保 tenant_credits 记录存在并加行锁（充当该租户积分操作的互斥锁），返回最新记录
func lockCreditsRowTx(tx *gorm.DB, tenantID uint64) (*model.TenantCredits, error) {
	var c model.TenantCredits
	// 先确保记录存在，再加行锁
	if err := tx.Where(model.TenantCredits{TenantID: tenantID}).FirstOrCreate(&c).Error; err != nil {
		return nil, err
	}
	// 重新加锁读取最新值
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("tenant_id = ?", tenantID).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// sumLotsRemainingTx 汇总该租户所有批次的剩余积分（不区分是否过期；过期批次由惰性清理保证 remaining=0）
func sumLotsRemainingTx(tx *gorm.DB, tenantID uint64) (int64, error) {
	var sum int64
	err := tx.Model(&model.CreditLot{}).
		Where("tenant_id = ?", tenantID).
		Select("COALESCE(SUM(remaining), 0)").Scan(&sum).Error
	return sum, err
}

// expireDueLotsTx 清零该租户已到期批次的剩余积分，逐批写 action=expire 的流水，返回作废总数。
// 调用方必须已持有该租户 tenant_credits 行锁；不更新余额缓存，由调用方最终统一同步。
func expireDueLotsTx(tx *gorm.DB, tenantID uint64, now time.Time) (int64, error) {
	var due []*model.CreditLot
	if err := tx.Where("tenant_id = ? AND remaining > 0 AND expires_at <= ?", tenantID, now).
		Order("expires_at ASC, id ASC").Find(&due).Error; err != nil {
		return 0, err
	}
	if len(due) == 0 {
		return 0, nil
	}
	balance, err := sumLotsRemainingTx(tx, tenantID)
	if err != nil {
		return 0, err
	}
	var expired int64
	for _, lot := range due {
		res := tx.Model(&model.CreditLot{}).
			Where("id = ? AND remaining = ?", lot.ID, lot.Remaining).
			Update("remaining", 0)
		if res.Error != nil {
			return 0, res.Error
		}
		if res.RowsAffected == 0 {
			// 行锁保护下不应发生；出现说明有并发写绕过了锁，回滚
			return 0, errors.New("积分批次并发修改冲突")
		}
		balance -= lot.Remaining
		expired += lot.Remaining
		log := &model.CreditLog{
			TenantID:     tenantID,
			Amount:       -lot.Remaining,
			Action:       "expire",
			BalanceAfter: balance,
			RefType:      "credit_lot",
			RefID:        lot.ID,
			Note:         "积分到期作废",
		}
		if err := tx.Create(log).Error; err != nil {
			return 0, err
		}
	}
	return expired, nil
}

// consumeLotsTx 按 expires_at 升序（最近过期优先）逐批扣减 cost 积分，返回各批次扣减明细（LogID 由调用方回填）。
// 余额不足返回 ErrInsufficientCredits；调用方必须已持有行锁并完成惰性过期。
func consumeLotsTx(tx *gorm.DB, tenantID uint64, cost int64, now time.Time) ([]*model.CreditLogDetail, error) {
	var lots []*model.CreditLot
	if err := tx.Where("tenant_id = ? AND remaining > 0 AND expires_at > ?", tenantID, now).
		Order("expires_at ASC, id ASC").Find(&lots).Error; err != nil {
		return nil, err
	}
	var available int64
	for _, lot := range lots {
		available += lot.Remaining
	}
	if available < cost {
		return nil, ErrInsufficientCredits
	}
	details := make([]*model.CreditLogDetail, 0, 2)
	remain := cost
	for _, lot := range lots {
		if remain == 0 {
			break
		}
		take := min(lot.Remaining, remain)
		// 条件更新是防超扣的第二道保险：并发绕过行锁时 remaining 不够会更新 0 行
		res := tx.Model(&model.CreditLot{}).
			Where("id = ? AND remaining >= ?", lot.ID, take).
			UpdateColumn("remaining", gorm.Expr("remaining - ?", take))
		if res.Error != nil {
			return nil, res.Error
		}
		if res.RowsAffected == 0 {
			return nil, ErrInsufficientCredits
		}
		details = append(details, &model.CreditLogDetail{LotID: lot.ID, Amount: take})
		remain -= take
	}
	return details, nil
}

// createLogWithDetailsTx 写流水并回填批次明细的 LogID
func createLogWithDetailsTx(tx *gorm.DB, log *model.CreditLog, details []*model.CreditLogDetail) error {
	if err := tx.Create(log).Error; err != nil {
		return err
	}
	if len(details) == 0 {
		return nil
	}
	for _, d := range details {
		d.LogID = log.ID
	}
	return tx.Create(&details).Error
}

// Deduct 原子扣费：锁租户行 → 幂等检查 → 惰性过期 → 按最近过期优先逐批扣减 → 同步余额缓存 → 写流水与明细
func (r *tenantCreditsRepo) Deduct(ctx context.Context, tenantID uint64, cost int64, action, refType string, refID uint64, note string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		c, err := lockCreditsRowTx(tx, tenantID)
		if err != nil {
			return err
		}

		// 幂等检查：同一资源同一动作已扣过则直接跳过
		if refType != "" && refID != 0 {
			var count int64
			if err := tx.Model(&model.CreditLog{}).
				Where("tenant_id = ? AND action = ? AND ref_type = ? AND ref_id = ?",
					tenantID, action, refType, refID).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return nil
			}
		}

		balanceAfter := int64(-1)
		var details []*model.CreditLogDetail
		if !c.Unlimited {
			now := time.Now()
			if _, err := expireDueLotsTx(tx, tenantID, now); err != nil {
				return err
			}
			details, err = consumeLotsTx(tx, tenantID, cost, now)
			if err != nil {
				return err
			}
			balanceAfter, err = sumLotsRemainingTx(tx, tenantID)
			if err != nil {
				return err
			}
			if err := tx.Model(c).Update("balance", balanceAfter).Error; err != nil {
				return err
			}
		}

		log := &model.CreditLog{
			TenantID:     tenantID,
			Amount:       -cost,
			Action:       action,
			BalanceAfter: balanceAfter,
			RefType:      refType,
			RefID:        refID,
			Note:         note,
		}
		return createLogWithDetailsTx(tx, log, details)
	})
}

// Gift 发放/扣减积分：delta>0 生成一个到期时间为 expiresAt 的新批次；
// delta<0（管理员扣减）按最近过期优先消耗批次，不足返回 ErrInsufficientCredits
func (r *tenantCreditsRepo) Gift(ctx context.Context, tenantID uint64, delta int64, expiresAt time.Time, action, note string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		c, err := lockCreditsRowTx(tx, tenantID)
		if err != nil {
			return err
		}

		now := time.Now()
		if _, err := expireDueLotsTx(tx, tenantID, now); err != nil {
			return err
		}

		var details []*model.CreditLogDetail
		refType, refID := "", uint64(0)
		switch {
		case delta > 0:
			lot := &model.CreditLot{
				TenantID:  tenantID,
				Amount:    delta,
				Remaining: delta,
				Source:    action,
				ExpiresAt: expiresAt,
			}
			if err := tx.Create(lot).Error; err != nil {
				return err
			}
			refType, refID = "credit_lot", lot.ID
		case delta < 0:
			details, err = consumeLotsTx(tx, tenantID, -delta, now)
			if err != nil {
				return err
			}
		default:
			return nil
		}

		balance, err := sumLotsRemainingTx(tx, tenantID)
		if err != nil {
			return err
		}
		if err := tx.Model(c).Update("balance", balance).Error; err != nil {
			return err
		}

		log := &model.CreditLog{
			TenantID:     tenantID,
			Amount:       delta,
			Action:       action,
			BalanceAfter: balance,
			RefType:      refType,
			RefID:        refID,
			Note:         note,
		}
		return createLogWithDetailsTx(tx, log, details)
	})
}

// ListActiveLots 查询未过期且有剩余的批次，按 expires_at 升序（最先过期的在前）
func (r *tenantCreditsRepo) ListActiveLots(ctx context.Context, tenantID uint64) ([]*model.CreditLot, error) {
	var lots []*model.CreditLot
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND remaining > 0 AND expires_at > ?", tenantID, time.Now()).
		Order("expires_at ASC, id ASC").
		Find(&lots).Error
	return lots, err
}

// ExpireDueLots 全量清理所有租户的到期批次（定时任务兜底），逐租户独立事务，返回作废积分总数
func (r *tenantCreditsRepo) ExpireDueLots(ctx context.Context) (int64, error) {
	var tenantIDs []uint64
	if err := r.db.WithContext(ctx).Model(&model.CreditLot{}).
		Where("remaining > 0 AND expires_at <= ?", time.Now()).
		Distinct().Pluck("tenant_id", &tenantIDs).Error; err != nil {
		return 0, err
	}
	var total int64
	for _, tid := range tenantIDs {
		err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			c, err := lockCreditsRowTx(tx, tid)
			if err != nil {
				return err
			}
			expired, err := expireDueLotsTx(tx, tid, time.Now())
			if err != nil {
				return err
			}
			if expired == 0 {
				return nil
			}
			balance, err := sumLotsRemainingTx(tx, tid)
			if err != nil {
				return err
			}
			total += expired
			return tx.Model(c).Update("balance", balance).Error
		})
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func (r *tenantCreditsRepo) SetUnlimited(ctx context.Context, tenantID uint64, unlimited bool) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var c model.TenantCredits
		if err := tx.Where(model.TenantCredits{TenantID: tenantID}).FirstOrCreate(&c).Error; err != nil {
			return err
		}
		return tx.Model(&c).Updates(map[string]interface{}{
			"unlimited":  unlimited,
			"updated_at": time.Now(),
		}).Error
	})
}

func (r *tenantCreditsRepo) ListLogs(ctx context.Context, tenantID uint64, limit int) ([]*model.CreditLog, error) {
	var logs []*model.CreditLog
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

func (r *tenantCreditsRepo) ListLogsPaged(ctx context.Context, tenantID uint64, page, size int, amountType string) ([]*model.CreditLog, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.CreditLog{}).Where("tenant_id = ?", tenantID)
	switch amountType {
	case "income":
		q = q.Where("amount > 0")
	case "expense":
		q = q.Where("amount < 0")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var logs []*model.CreditLog
	err := q.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&logs).Error
	return logs, total, err
}

func (r *tenantCreditsRepo) ListByTenantIDs(ctx context.Context, tenantIDs []uint64) ([]*model.TenantCredits, error) {
	if len(tenantIDs) == 0 {
		return nil, nil
	}
	var list []*model.TenantCredits
	err := r.db.WithContext(ctx).Where("tenant_id IN ?", tenantIDs).Find(&list).Error
	return list, err
}
