// Package integration 积分批次系统的集成测试：连接本地 MySQL 验证真实 SQL 行为。
// MySQL 不可用时自动跳过（CI 或无数据库环境不阻塞 go test ./...）。
package integration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/repo"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const testDBName = "fandian_credits_it"

// setupDB 连接本地 MySQL 并准备干净的测试库；连不上则跳过测试
func setupDB(t *testing.T) *gorm.DB {
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
	if err := admin.Exec("CREATE DATABASE IF NOT EXISTS " + testDBName + " CHARACTER SET utf8mb4").Error; err != nil {
		t.Skipf("创建测试库失败，跳过集成测试: %v", err)
	}

	db, err := gorm.Open(mysql.Open(base+testDBName+"?charset=utf8mb4&parseTime=True&loc=Local"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("连接测试库失败，跳过集成测试: %v", err)
	}
	if err := db.AutoMigrate(&model.TenantCredits{}, &model.CreditLog{}, &model.CreditLot{}, &model.CreditLogDetail{}); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
	for _, table := range []string{"tenant_credits", "credit_logs", "credit_lots", "credit_log_details"} {
		if err := db.Exec("TRUNCATE TABLE " + table).Error; err != nil {
			t.Fatalf("清空表 %s 失败: %v", table, err)
		}
	}
	return db
}

// addLot 直接插入一个批次并同步余额缓存
func addLot(t *testing.T, db *gorm.DB, tenantID uint64, amount int64, expiresAt time.Time) uint64 {
	t.Helper()
	lot := &model.CreditLot{TenantID: tenantID, Amount: amount, Remaining: amount, Source: "test", ExpiresAt: expiresAt}
	if err := db.Create(lot).Error; err != nil {
		t.Fatalf("插入批次失败: %v", err)
	}
	var c model.TenantCredits
	if err := db.Where(model.TenantCredits{TenantID: tenantID}).FirstOrCreate(&c).Error; err != nil {
		t.Fatalf("初始化余额失败: %v", err)
	}
	if err := db.Model(&model.TenantCredits{}).Where("tenant_id = ?", tenantID).
		Update("balance", gorm.Expr("balance + ?", amount)).Error; err != nil {
		t.Fatalf("同步余额失败: %v", err)
	}
	return lot.ID
}

func lotRemaining(t *testing.T, db *gorm.DB, lotID uint64) int64 {
	t.Helper()
	var lot model.CreditLot
	if err := db.First(&lot, lotID).Error; err != nil {
		t.Fatalf("查询批次失败: %v", err)
	}
	return lot.Remaining
}

func cachedBalance(t *testing.T, db *gorm.DB, tenantID uint64) int64 {
	t.Helper()
	var c model.TenantCredits
	if err := db.Where("tenant_id = ?", tenantID).First(&c).Error; err != nil {
		t.Fatalf("查询余额失败: %v", err)
	}
	return c.Balance
}

// Scenario: 扣费优先消耗最近将要过期的批次，且写批次明细
func TestIT_Deduct_SoonestExpiringFirst(t *testing.T) {
	db := setupDB(t)
	r := repo.NewTenantCreditsRepo(db)
	ctx := context.Background()
	now := time.Now()

	lateLot := addLot(t, db, 1, 100, now.AddDate(1, 0, 0)) // 先充值，一年后过期
	soonLot := addLot(t, db, 1, 100, now.AddDate(0, 1, 0)) // 后充值，一个月后过期

	if err := r.Deduct(ctx, 1, 30, "ocr", "ocr_record", 1, "测试"); err != nil {
		t.Fatal(err)
	}

	if got := lotRemaining(t, db, soonLot); got != 70 {
		t.Errorf("最近过期批次应剩 70，实际 %d", got)
	}
	if got := lotRemaining(t, db, lateLot); got != 100 {
		t.Errorf("远期批次不应被扣，实际 %d", got)
	}
	if got := cachedBalance(t, db, 1); got != 170 {
		t.Errorf("余额缓存应为 170，实际 %d", got)
	}

	// 批次明细可追溯
	var details []model.CreditLogDetail
	if err := db.Find(&details).Error; err != nil {
		t.Fatal(err)
	}
	if len(details) != 1 || details[0].LotID != soonLot || details[0].Amount != 30 {
		t.Errorf("批次明细不正确: %+v", details)
	}
}

// Scenario: 一个批次不够时跨批次扣减，明细逐批记录
func TestIT_Deduct_SpansLots(t *testing.T) {
	db := setupDB(t)
	r := repo.NewTenantCreditsRepo(db)
	now := time.Now()

	soonLot := addLot(t, db, 2, 6, now.AddDate(0, 0, 7))
	lateLot := addLot(t, db, 2, 100, now.AddDate(1, 0, 0))

	if err := r.Deduct(context.Background(), 2, 10, "ocr", "ocr_record", 2, ""); err != nil {
		t.Fatal(err)
	}
	if got := lotRemaining(t, db, soonLot); got != 0 {
		t.Errorf("近期批次应扣光，实际 %d", got)
	}
	if got := lotRemaining(t, db, lateLot); got != 96 {
		t.Errorf("远期批次应剩 96，实际 %d", got)
	}
	var details []model.CreditLogDetail
	db.Order("lot_id").Find(&details)
	if len(details) != 2 {
		t.Fatalf("应有 2 条明细，实际 %d", len(details))
	}
}

// Scenario: 到期批次在扣费时被惰性作废（写 expire 流水），不可用于扣费
func TestIT_Deduct_LazyExpiry(t *testing.T) {
	db := setupDB(t)
	r := repo.NewTenantCreditsRepo(db)
	ctx := context.Background()
	now := time.Now()

	expiredLot := addLot(t, db, 3, 50, now.Add(-time.Hour)) // 已过期
	validLot := addLot(t, db, 3, 30, now.AddDate(0, 1, 0))

	if err := r.Deduct(ctx, 3, 20, "ocr", "ocr_record", 3, ""); err != nil {
		t.Fatal(err)
	}

	if got := lotRemaining(t, db, expiredLot); got != 0 {
		t.Errorf("过期批次应被清零，实际 %d", got)
	}
	if got := lotRemaining(t, db, validLot); got != 10 {
		t.Errorf("有效批次应剩 10，实际 %d", got)
	}
	if got := cachedBalance(t, db, 3); got != 10 {
		t.Errorf("余额缓存应为 10（过期作废后），实际 %d", got)
	}

	var expireLogs []model.CreditLog
	db.Where("tenant_id = ? AND action = ?", 3, "expire").Find(&expireLogs)
	if len(expireLogs) != 1 || expireLogs[0].Amount != -50 || expireLogs[0].RefID != expiredLot {
		t.Errorf("expire 流水不正确: %+v", expireLogs)
	}
}

// Scenario: 只剩过期积分时扣费报积分不足
func TestIT_Deduct_OnlyExpired_Insufficient(t *testing.T) {
	db := setupDB(t)
	r := repo.NewTenantCreditsRepo(db)

	addLot(t, db, 4, 100, time.Now().Add(-time.Minute))

	err := r.Deduct(context.Background(), 4, 10, "upload", "", 0, "")
	if !errors.Is(err, repo.ErrInsufficientCredits) {
		t.Errorf("期望 ErrInsufficientCredits，实际 %v", err)
	}
}

// Scenario: 同一资源同一动作重复扣费幂等（只扣一次）
func TestIT_Deduct_Idempotent(t *testing.T) {
	db := setupDB(t)
	r := repo.NewTenantCreditsRepo(db)
	ctx := context.Background()

	addLot(t, db, 5, 100, time.Now().AddDate(1, 0, 0))

	for range 3 {
		if err := r.Deduct(ctx, 5, 10, "ocr", "ocr_record", 55, ""); err != nil {
			t.Fatal(err)
		}
	}
	if got := cachedBalance(t, db, 5); got != 90 {
		t.Errorf("幂等扣费应只扣一次，余额应为 90，实际 %d", got)
	}
}

// Scenario: 并发扣费不超扣——20 个并发，余额只够 10 次
func TestIT_Deduct_Concurrent_NoOverdraft(t *testing.T) {
	db := setupDB(t)
	r := repo.NewTenantCreditsRepo(db)
	ctx := context.Background()

	addLot(t, db, 6, 50, time.Now().AddDate(1, 0, 0)) // 只够 10 次 × 5 分

	const n = 20
	errs := make([]error, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			errs[i] = r.Deduct(ctx, 6, 5, "upload", "", 0, fmt.Sprintf("并发%d", i))
		}(i)
	}
	wg.Wait()

	success := 0
	for _, err := range errs {
		if err == nil {
			success++
		} else if !errors.Is(err, repo.ErrInsufficientCredits) {
			t.Errorf("意外错误: %v", err)
		}
	}
	if success != 10 {
		t.Errorf("应恰好成功 10 次，实际 %d", success)
	}
	if got := cachedBalance(t, db, 6); got != 0 {
		t.Errorf("最终余额应为 0，实际 %d", got)
	}
	var negative int64
	db.Model(&model.CreditLot{}).Where("remaining < 0").Count(&negative)
	if negative != 0 {
		t.Errorf("不允许出现负剩余批次，实际 %d 条", negative)
	}
}

// Scenario: 充值生成批次；管理员负数调整按最近过期优先扣，不足报错
func TestIT_Gift_And_NegativeAdjust(t *testing.T) {
	db := setupDB(t)
	r := repo.NewTenantCreditsRepo(db)
	ctx := context.Background()
	now := time.Now()

	if err := r.Gift(ctx, 7, 500, now.AddDate(1, 0, 0), "gift", "充值"); err != nil {
		t.Fatal(err)
	}
	var lots []model.CreditLot
	db.Where("tenant_id = ?", 7).Find(&lots)
	if len(lots) != 1 || lots[0].Remaining != 500 || lots[0].Source != "gift" {
		t.Fatalf("充值应生成批次: %+v", lots)
	}

	if err := r.Gift(ctx, 7, -200, time.Time{}, "admin_adjust", "扣减"); err != nil {
		t.Fatal(err)
	}
	if got := cachedBalance(t, db, 7); got != 300 {
		t.Errorf("余额应为 300，实际 %d", got)
	}

	err := r.Gift(ctx, 7, -400, time.Time{}, "admin_adjust", "超额扣减")
	if !errors.Is(err, repo.ErrInsufficientCredits) {
		t.Errorf("超额扣减应报积分不足，实际 %v", err)
	}
}

// Scenario: 定时清理任务作废所有租户的到期批次并同步余额
func TestIT_ExpireDueLots_Sweep(t *testing.T) {
	db := setupDB(t)
	r := repo.NewTenantCreditsRepo(db)
	now := time.Now()

	addLot(t, db, 8, 40, now.Add(-time.Minute))
	addLot(t, db, 8, 60, now.AddDate(0, 1, 0))
	addLot(t, db, 9, 25, now.Add(-time.Hour))

	total, err := r.ExpireDueLots(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if total != 65 {
		t.Errorf("应作废 65 积分，实际 %d", total)
	}
	if got := cachedBalance(t, db, 8); got != 60 {
		t.Errorf("租户8余额应为 60，实际 %d", got)
	}
	if got := cachedBalance(t, db, 9); got != 0 {
		t.Errorf("租户9余额应为 0，实际 %d", got)
	}
	var expireLogs int64
	db.Model(&model.CreditLog{}).Where("action = ?", "expire").Count(&expireLogs)
	if expireLogs != 2 {
		t.Errorf("应有 2 条 expire 流水，实际 %d", expireLogs)
	}
}

// Scenario: 流水与批次可对账——SUM(流水) == SUM(批次剩余) == 余额缓存
func TestIT_Auditability_LogsReconcileWithLots(t *testing.T) {
	db := setupDB(t)
	r := repo.NewTenantCreditsRepo(db)
	ctx := context.Background()
	now := time.Now()

	if err := r.Gift(ctx, 10, 300, now.AddDate(0, 0, 30), "gift", "充值1"); err != nil {
		t.Fatal(err)
	}
	if err := r.Gift(ctx, 10, 200, now.AddDate(1, 0, 0), "gift", "充值2"); err != nil {
		t.Fatal(err)
	}
	for i := uint64(1); i <= 5; i++ {
		if err := r.Deduct(ctx, 10, 20, "ocr", "ocr_record", i, ""); err != nil {
			t.Fatal(err)
		}
	}

	var logSum, lotSum int64
	db.Model(&model.CreditLog{}).Where("tenant_id = ?", 10).Select("COALESCE(SUM(amount),0)").Scan(&logSum)
	db.Model(&model.CreditLot{}).Where("tenant_id = ?", 10).Select("COALESCE(SUM(remaining),0)").Scan(&lotSum)
	balance := cachedBalance(t, db, 10)

	if logSum != 400 || lotSum != 400 || balance != 400 {
		t.Errorf("对账失败：流水和=%d 批次剩余和=%d 余额缓存=%d，应均为 400", logSum, lotSum, balance)
	}

	// 每条扣费流水都有明细，且明细之和等于扣费额
	var logs []model.CreditLog
	db.Where("tenant_id = ? AND amount < 0", 10).Find(&logs)
	for _, l := range logs {
		var detailSum int64
		db.Model(&model.CreditLogDetail{}).Where("log_id = ?", l.ID).Select("COALESCE(SUM(amount),0)").Scan(&detailSum)
		if detailSum != -l.Amount {
			t.Errorf("流水 %d 明细之和 %d 与扣费额 %d 不符", l.ID, detailSum, -l.Amount)
		}
	}
}
