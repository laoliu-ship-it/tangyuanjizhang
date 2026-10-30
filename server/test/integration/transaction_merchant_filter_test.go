// Package integration 交易列表按商户筛选的集成测试：连接本地 MySQL 验证真实 SQL 行为。
// MySQL 不可用时自动跳过（CI 或无数据库环境不阻塞 go test ./...）。
package integration

import (
	"context"
	"testing"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/repo"
)

// TestTransactionList_FilterByMerchantID 验证 merchant_id 过滤只命中该商户名下的交易，
// 用于「统计页商户支出 Top10 点击查看该商户交易记录」功能。
func TestTransactionList_FilterByMerchantID(t *testing.T) {
	db := setupTxFilterDB(t)
	r := repo.NewTransactionRepo(db)
	ctx := context.Background()
	const tenantID uint64 = 1

	category := &model.Category{TenantID: tenantID, Name: "餐饮", Type: "expense", Icon: "🍚"}
	if err := db.Create(category).Error; err != nil {
		t.Fatalf("创建分类失败: %v", err)
	}

	merchantA := &model.Merchant{TenantID: tenantID, Name: "肯德基"}
	merchantB := &model.Merchant{TenantID: tenantID, Name: "麦当劳"}
	if err := db.Create(merchantA).Error; err != nil {
		t.Fatalf("创建商户A失败: %v", err)
	}
	if err := db.Create(merchantB).Error; err != nil {
		t.Fatalf("创建商户B失败: %v", err)
	}

	txA1 := &model.Transaction{
		TenantID: tenantID, UserID: 1, Type: "expense", Amount: 30,
		CategoryID: category.ID, MerchantID: merchantA.ID, TransactionDate: "2026-03-01 12:00:00",
	}
	txA2 := &model.Transaction{
		TenantID: tenantID, UserID: 1, Type: "expense", Amount: 25,
		CategoryID: category.ID, MerchantID: merchantA.ID, TransactionDate: "2026-03-02 12:00:00",
	}
	txB1 := &model.Transaction{
		TenantID: tenantID, UserID: 1, Type: "expense", Amount: 40,
		CategoryID: category.ID, MerchantID: merchantB.ID, TransactionDate: "2026-03-03 12:00:00",
	}
	for _, tx := range []*model.Transaction{txA1, txA2, txB1} {
		if err := r.Create(ctx, tx); err != nil {
			t.Fatalf("创建交易失败: %v", err)
		}
	}

	items, total, err := r.List(ctx, tenantID, dto.TransactionFilter{
		MerchantID: merchantA.ID,
		SortBy:     "transaction_date",
		Page:       1,
		PageSize:   20,
	})
	if err != nil {
		t.Fatalf("按商户查询失败: %v", err)
	}
	if total != 2 || len(items) != 2 {
		t.Fatalf("按商户A筛选应命中2条，实际 total=%d, items=%v", total, items)
	}
	for _, item := range items {
		if item.MerchantID != merchantA.ID {
			t.Fatalf("命中记录商户ID不符，期望=%d 实际=%d", merchantA.ID, item.MerchantID)
		}
	}

	// merchant_id + 日期范围叠加筛选：只取商户A在 03-02 之后的记录
	itemsWithDate, total, err := r.List(ctx, tenantID, dto.TransactionFilter{
		MerchantID: merchantA.ID,
		StartDate:  "2026-03-02",
		EndDate:    "2026-03-31",
		SortBy:     "transaction_date",
		Page:       1,
		PageSize:   20,
	})
	if err != nil {
		t.Fatalf("按商户+日期查询失败: %v", err)
	}
	if total != 1 || len(itemsWithDate) != 1 || itemsWithDate[0].ID != txA2.ID {
		t.Fatalf("按商户A+日期筛选应只命中 txA2，实际 total=%d, items=%v", total, itemsWithDate)
	}
}
