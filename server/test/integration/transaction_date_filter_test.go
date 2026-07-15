// Package integration 交易列表日期筛选的集成测试：连接本地 MySQL 验证真实 SQL 行为。
// MySQL 不可用时自动跳过（CI 或无数据库环境不阻塞 go test ./...）。
package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/repo"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const txFilterTestDBName = "fandian_tx_filter_it"

func setupTxFilterDB(t *testing.T) *gorm.DB {
	t.Helper()
	base := os.Getenv("TEST_DB_DSN_BASE")
	if base == "" {
		base = "root:allgood@tcp(localhost:3306)/"
	}
	admin, err := gorm.Open(mysql.Open(base+"?charset=utf8mb4&parseTime=True&loc=Local"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("本地 MySQL 不可用，跳过集成测试: %v", err)
	}
	if err := admin.Exec("CREATE DATABASE IF NOT EXISTS " + txFilterTestDBName + " CHARACTER SET utf8mb4").Error; err != nil {
		t.Skipf("创建测试库失败，跳过集成测试: %v", err)
	}

	db, err := gorm.Open(mysql.Open(base+txFilterTestDBName+"?charset=utf8mb4&parseTime=True&loc=Local"), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Skipf("连接测试库失败，跳过集成测试: %v", err)
	}
	if err := db.AutoMigrate(&model.Category{}, &model.Merchant{}, &model.Transaction{}, &model.TransactionImage{}); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
	db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	for _, table := range []string{"transaction_images", "transactions", "categories", "merchants"} {
		if err := db.Exec("TRUNCATE TABLE " + table).Error; err != nil {
			t.Fatalf("清空表 %s 失败: %v", table, err)
		}
	}
	db.Exec("SET FOREIGN_KEY_CHECKS = 1")
	return db
}

// TestTransactionList_DateFilterFollowsSortBy 验证日期范围筛选与 SortBy 使用同一时间字段：
// 交易发生日期与创建日期分属不同区间时，切换 sort_by 应筛出不同的结果集。
func TestTransactionList_DateFilterFollowsSortBy(t *testing.T) {
	db := setupTxFilterDB(t)
	r := repo.NewTransactionRepo(db)
	ctx := context.Background()
	const tenantID uint64 = 1

	category := &model.Category{TenantID: tenantID, Name: "餐饮", Type: "expense", Icon: "🍚"}
	if err := db.Create(category).Error; err != nil {
		t.Fatalf("创建分类失败: %v", err)
	}

	// 交易发生在 1 月，但录入（创建）在 3 月
	oldTx := &model.Transaction{
		TenantID:        tenantID,
		UserID:          1,
		Type:            "expense",
		Amount:          10,
		CategoryID:      category.ID,
		TransactionDate: "2026-01-15 12:00:00",
	}
	if err := r.Create(ctx, oldTx); err != nil {
		t.Fatalf("创建交易失败: %v", err)
	}
	if err := db.Model(&model.Transaction{}).Where("id = ?", oldTx.ID).
		Update("created_at", time.Date(2026, 3, 10, 9, 0, 0, 0, time.Local)).Error; err != nil {
		t.Fatalf("回填 created_at 失败: %v", err)
	}

	// 交易发生在 3 月，创建时间就是当天
	newTx := &model.Transaction{
		TenantID:        tenantID,
		UserID:          1,
		Type:            "expense",
		Amount:          20,
		CategoryID:      category.ID,
		TransactionDate: "2026-03-05 08:00:00",
	}
	if err := r.Create(ctx, newTx); err != nil {
		t.Fatalf("创建交易失败: %v", err)
	}
	if err := db.Model(&model.Transaction{}).Where("id = ?", newTx.ID).
		Update("created_at", time.Date(2026, 3, 5, 8, 0, 0, 0, time.Local)).Error; err != nil {
		t.Fatalf("回填 created_at 失败: %v", err)
	}

	// 按交易日期筛选 1 月：只应命中 oldTx
	byTxDate, total, err := r.List(ctx, tenantID, dto.TransactionFilter{
		StartDate: "2026-01-01",
		EndDate:   "2026-01-31",
		SortBy:    "transaction_date",
		Page:      1,
		PageSize:  20,
	})
	if err != nil {
		t.Fatalf("按交易日期查询失败: %v", err)
	}
	if total != 1 || len(byTxDate) != 1 || byTxDate[0].ID != oldTx.ID {
		t.Fatalf("按交易日期筛选 1 月应只命中 oldTx，实际 total=%d, items=%v", total, byTxDate)
	}

	// 按创建时间筛选 1 月：oldTx 是 3 月创建的，此时应命中 0 条
	byCreatedAtJan, total, err := r.List(ctx, tenantID, dto.TransactionFilter{
		StartDate: "2026-01-01",
		EndDate:   "2026-01-31",
		SortBy:    "created_at",
		Page:      1,
		PageSize:  20,
	})
	if err != nil {
		t.Fatalf("按创建时间查询失败: %v", err)
	}
	if total != 0 || len(byCreatedAtJan) != 0 {
		t.Fatalf("按创建时间筛选 1 月应命中 0 条，实际 total=%d, items=%v", total, byCreatedAtJan)
	}

	// 按创建时间筛选 3 月：应同时命中 oldTx（3 月创建）与 newTx（3 月创建）
	byCreatedAtMar, total, err := r.List(ctx, tenantID, dto.TransactionFilter{
		StartDate: "2026-03-01",
		EndDate:   "2026-03-31",
		SortBy:    "created_at",
		Page:      1,
		PageSize:  20,
	})
	if err != nil {
		t.Fatalf("按创建时间查询失败: %v", err)
	}
	if total != 2 || len(byCreatedAtMar) != 2 {
		t.Fatalf("按创建时间筛选 3 月应命中 2 条，实际 total=%d, items=%v", total, byCreatedAtMar)
	}
}
