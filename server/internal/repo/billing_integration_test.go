package repo

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"fandianjizhang/server/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func billingTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("FANDIAN_TEST_DSN_REPO")
	if dsn == "" {
		dsn = "root:allgood@tcp(127.0.0.1:3306)/fandian_vp_repo_it?charset=utf8mb4&parseTime=True&loc=Local"
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Skipf("test MySQL 不可用（设置 FANDIAN_TEST_DSN_REPO）: %v", err)
	}
	models := []any{
		&model.TenantCredits{}, &model.CreditLot{}, &model.CreditLog{}, &model.CreditLogDetail{},
		&model.BillingPackage{}, &model.BillingPurchaseIntent{}, &model.BillingProcessedEvent{},
		&model.BillingRefund{}, &model.BillingCapability{},
	}
	_ = db.Exec("SET FOREIGN_KEY_CHECKS=0").Error
	for _, m := range models {
		_ = db.Migrator().DropTable(m)
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func seedIntent(t *testing.T, db *gorm.DB, tenantID uint64, ref string, amount, credits int64) {
	t.Helper()
	intent := &model.BillingPurchaseIntent{
		TenantID: tenantID, UserID: 1, AppSlug: "app", BusinessRef: ref,
		SKUCode: "sku", AmountFen: amount, CreditAmount: credits, ValidityDays: 365,
		Status: model.BillingIntentCreated, ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := db.Create(intent).Error; err != nil {
		t.Fatal(err)
	}
}

func deliveryEvent(ref string, amount, credits int64, key string) *model.BillingDeliveryEvent {
	return &model.BillingDeliveryEvent{
		EventKey: key, EventType: "billing_deliver", BusinessReference: ref,
		SKUCode: "sku", AmountFen: amount, CreditAmount: credits, ValidityDays: 365,
	}
}

func TestProcessDeliveryGrantsCreditsOnce(t *testing.T) {
	db := billingTestDB(t)
	r := NewBillingRepo(db)
	ctx := context.Background()
	seedIntent(t, db, 100, "br1", 600, 600)

	already, err := r.ProcessBillingDelivery(ctx, deliveryEvent("br1", 600, 600, "e1"))
	if err != nil || already {
		t.Fatalf("deliver: already=%v err=%v", already, err)
	}
	var credits model.TenantCredits
	db.Where("tenant_id = ?", 100).First(&credits)
	if credits.Balance != 600 {
		t.Fatalf("balance = %d", credits.Balance)
	}
	var lots []model.CreditLot
	db.Where("tenant_id = ?", 100).Find(&lots)
	if len(lots) != 1 || lots[0].Source != "purchase" || lots[0].OrderRef != "br1" || lots[0].Remaining != 600 {
		t.Fatalf("lots = %+v", lots)
	}
	var logs int64
	db.Model(&model.CreditLog{}).Where("tenant_id = ? AND action = ?", 100, "purchase").Count(&logs)
	if logs != 1 {
		t.Fatalf("purchase logs = %d", logs)
	}

	// 重复/并发同一个 event_key 只发一次。
	already, err = r.ProcessBillingDelivery(ctx, deliveryEvent("br1", 600, 600, "e1"))
	if err != nil || !already {
		t.Fatalf("duplicate deliver: already=%v err=%v", already, err)
	}
	db.Where("tenant_id = ?", 100).First(&credits)
	if credits.Balance != 600 {
		t.Fatalf("duplicate changed balance: %d", credits.Balance)
	}
}

func TestRefundOnlyOriginalPurchaseLot(t *testing.T) {
	db := billingTestDB(t)
	r := NewBillingRepo(db)
	creditsRepo := NewTenantCreditsRepo(db)
	ctx := context.Background()
	seedIntent(t, db, 101, "br2", 600, 600)
	if _, err := r.ProcessBillingDelivery(ctx, deliveryEvent("br2", 600, 600, "e2")); err != nil {
		t.Fatal(err)
	}
	// 购买批次更早到期，确保扣费从购买批次扣；赠送积分不能被退款回收。
	if err := db.Model(&model.CreditLot{}).Where("tenant_id = ? AND source = ?", 101, "purchase").
		Update("expires_at", time.Now().Add(time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.CreditLot{TenantID: 101, Amount: 100, Remaining: 100, Source: "gift", ExpiresAt: time.Now().Add(365 * 24 * time.Hour)}).Error; err != nil {
		t.Fatal(err)
	}
	// 购买积分消费 200，应进入已消费审计，而不是被当成未使用。
	if err := creditsRepo.Deduct(ctx, 101, 200, "ocr", "ocr_record", 1, "test consume"); err != nil {
		t.Fatal(err)
	}

	ev := &model.BillingDeliveryEvent{
		EventKey: "refund2", EventType: "billing_refund", BusinessReference: "br2",
		MchRefundID: "MR2", RefundFeeFen: 600,
	}
	already, err := r.ProcessBillingRefund(ctx, ev)
	if err != nil || already {
		t.Fatalf("refund: already=%v err=%v", already, err)
	}
	var purchase model.CreditLot
	db.Where("tenant_id = ? AND source = ?", 101, "purchase").First(&purchase)
	if purchase.Remaining != 0 {
		t.Fatalf("purchase remaining = %d", purchase.Remaining)
	}
	var gift model.CreditLot
	db.Where("tenant_id = ? AND source = ?", 101, "gift").First(&gift)
	if gift.Remaining != 100 {
		t.Fatalf("gift touched: %d", gift.Remaining)
	}
	var credits model.TenantCredits
	db.Where("tenant_id = ?", 101).First(&credits)
	if credits.Balance != 100 {
		t.Fatalf("balance = %d, want 100", credits.Balance)
	}
	var refund model.BillingRefund
	db.Where("refund_key = ?", "refund2").First(&refund)
	if refund.ReclaimedCredits != 400 || refund.ConsumedCredits != 200 {
		t.Fatalf("refund accounting = %+v", refund)
	}
	var details int64
	db.Model(&model.CreditLogDetail{}).Where("lot_id = ? AND amount = ?", purchase.ID, 400).Count(&details)
	if details == 0 {
		t.Fatal("refund must write positive credit_log_details for reclaimed lot")
	}
	assertNegativeLogsMatchDetails(t, db, 101)
}

// 既有对账约定：扣减/退款日志的 abs(amount) 必须等于明细合计。
func assertNegativeLogsMatchDetails(t *testing.T, db *gorm.DB, tenantID uint64) {
	t.Helper()
	var logs []model.CreditLog
	if err := db.Where("tenant_id = ? AND amount < 0", tenantID).Find(&logs).Error; err != nil {
		t.Fatal(err)
	}
	for _, log := range logs {
		var sum int64
		if err := db.Model(&model.CreditLogDetail{}).Where("log_id = ?", log.ID).
			Select("COALESCE(SUM(amount),0)").Scan(&sum).Error; err != nil {
			t.Fatal(err)
		}
		want := -log.Amount
		if sum != want {
			t.Fatalf("log %d action=%s abs(amount)=%d details=%d", log.ID, log.Action, want, sum)
		}
	}
}

func TestPartialRefundDoesNotReclaim(t *testing.T) {
	db := billingTestDB(t)
	r := NewBillingRepo(db)
	ctx := context.Background()
	seedIntent(t, db, 102, "br3", 600, 600)
	if _, err := r.ProcessBillingDelivery(ctx, deliveryEvent("br3", 600, 600, "e3")); err != nil {
		t.Fatal(err)
	}
	ev := &model.BillingDeliveryEvent{
		EventKey: "refund3", EventType: "billing_refund", BusinessReference: "br3",
		MchRefundID: "MR3", RefundFeeFen: 100,
	}
	if _, err := r.ProcessBillingRefund(ctx, ev); err != nil {
		t.Fatal(err)
	}
	var credits model.TenantCredits
	db.Where("tenant_id = ?", 102).First(&credits)
	if credits.Balance != 600 {
		t.Fatalf("partial refund changed balance: %d", credits.Balance)
	}
	var lot model.CreditLot
	db.Where("tenant_id = ? AND source = ?", 102, "purchase").First(&lot)
	if lot.Remaining != 600 {
		t.Fatalf("partial refund reclaimed lot: %d", lot.Remaining)
	}
	var refund model.BillingRefund
	db.Where("refund_key = ?", "refund3").First(&refund)
	if refund.Status != "human_review" || refund.ReclaimedCredits != 0 {
		t.Fatalf("refund = %+v", refund)
	}
}

func TestExpiredRemainingNotTreatedAsConsumed(t *testing.T) {
	db := billingTestDB(t)
	r := NewBillingRepo(db)
	ctx := context.Background()
	// 直接构造已过期购买批次 + 意图，退款时先惰性清理，剩余不应算消费。
	seedIntent(t, db, 103, "br4", 600, 600)
	intent := &model.BillingPurchaseIntent{}
	db.Where("business_ref = ?", "br4").First(intent)
	intent.Status = model.BillingIntentDelivered
	db.Save(intent)
	db.Create(&model.CreditLot{TenantID: 103, Amount: 600, Remaining: 600, Source: "purchase", OrderRef: "br4", ExpiresAt: time.Now().Add(-time.Hour)})

	ev := &model.BillingDeliveryEvent{
		EventKey: "refund4", EventType: "billing_refund", BusinessReference: "br4",
		MchRefundID: "MR4", RefundFeeFen: 600,
	}
	if _, err := r.ProcessBillingRefund(ctx, ev); err != nil {
		t.Fatal(err)
	}
	var refund model.BillingRefund
	db.Where("refund_key = ?", "refund4").First(&refund)
	if refund.ConsumedCredits != 0 || refund.ReclaimedCredits != 0 {
		t.Fatalf("expired remaining misclassified: %+v", refund)
	}
	var credits model.TenantCredits
	db.Where("tenant_id = ?", 103).First(&credits)
	if credits.Balance != 0 {
		t.Fatalf("expired balance = %d", credits.Balance)
	}
}

func TestDeliveryAndRefundConcurrentWithDeduct(t *testing.T) {
	db := billingTestDB(t)
	r := NewBillingRepo(db)
	creditsRepo := NewTenantCreditsRepo(db)
	ctx := context.Background()
	seedIntent(t, db, 104, "br5", 600, 600)
	if _, err := r.ProcessBillingDelivery(ctx, deliveryEvent("br5", 600, 600, "e5")); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = creditsRepo.Deduct(ctx, 104, 10, "ocr", "ocr_record", uint64(n+1), "concurrent")
		}(i)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		ev := &model.BillingDeliveryEvent{
			EventKey: "refund5", EventType: "billing_refund", BusinessReference: "br5",
			MchRefundID: "MR5", RefundFeeFen: 600,
		}
		if _, err := r.ProcessBillingRefund(ctx, ev); err != nil {
			t.Errorf("refund: %v", err)
		}
	}()
	wg.Wait()

	var credits model.TenantCredits
	db.Where("tenant_id = ?", 104).First(&credits)
	var sum int64
	db.Model(&model.CreditLot{}).Where("tenant_id = ?", 104).Select("COALESCE(SUM(remaining),0)").Scan(&sum)
	if credits.Balance != sum || credits.Balance < 0 {
		t.Fatalf("balance cache %d != lots %d", credits.Balance, sum)
	}
}
