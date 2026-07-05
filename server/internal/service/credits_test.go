package service_test

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/repo"
	"fandianjizhang/server/internal/service"
)

// ============================================================
// Fake 实现（内存版，用于纯单元测试，无需数据库）
// ============================================================

type fakeCreditLog struct {
	tenantID     uint64
	amount       int64
	action       string
	balanceAfter int64
	refType      string
	refID        uint64
	note         string
}

// fakeCreditsRepo 内存版积分仓库（批次模型），支持并发测试
type fakeCreditsRepo struct {
	mu        sync.Mutex
	lots      map[uint64][]*model.CreditLot // tenantID -> 批次列表
	unlimited map[uint64]bool
	logs      []fakeCreditLog
	nextLotID uint64
}

func newFakeRepo() *fakeCreditsRepo {
	return &fakeCreditsRepo{
		lots:      make(map[uint64][]*model.CreditLot),
		unlimited: make(map[uint64]bool),
		nextLotID: 1,
	}
}

// balanceLocked 汇总未过期批次的剩余积分（调用方需持锁）
func (r *fakeCreditsRepo) balanceLocked(tenantID uint64, now time.Time) int64 {
	var sum int64
	for _, lot := range r.lots[tenantID] {
		if lot.ExpiresAt.After(now) {
			sum += lot.Remaining
		}
	}
	return sum
}

// sortedActiveLotsLocked 未过期且有剩余的批次，按 expires_at 升序（调用方需持锁）
func (r *fakeCreditsRepo) sortedActiveLotsLocked(tenantID uint64, now time.Time) []*model.CreditLot {
	var active []*model.CreditLot
	for _, lot := range r.lots[tenantID] {
		if lot.Remaining > 0 && lot.ExpiresAt.After(now) {
			active = append(active, lot)
		}
	}
	sort.Slice(active, func(i, j int) bool {
		if active[i].ExpiresAt.Equal(active[j].ExpiresAt) {
			return active[i].ID < active[j].ID
		}
		return active[i].ExpiresAt.Before(active[j].ExpiresAt)
	})
	return active
}

// consumeLocked 按最近过期优先扣减，不足返回 ErrInsufficientCredits（调用方需持锁）
func (r *fakeCreditsRepo) consumeLocked(tenantID uint64, cost int64, now time.Time) error {
	active := r.sortedActiveLotsLocked(tenantID, now)
	var available int64
	for _, lot := range active {
		available += lot.Remaining
	}
	if available < cost {
		return repo.ErrInsufficientCredits
	}
	remain := cost
	for _, lot := range active {
		if remain == 0 {
			break
		}
		take := min(lot.Remaining, remain)
		lot.Remaining -= take
		remain -= take
	}
	return nil
}

// setBalance 测试辅助：以一个远期批次的形式初始化余额
func (r *fakeCreditsRepo) setBalance(tenantID uint64, amount int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if amount <= 0 {
		return
	}
	r.lots[tenantID] = append(r.lots[tenantID], &model.CreditLot{
		ID: r.nextLotID, TenantID: tenantID, Amount: amount, Remaining: amount, Source: "test", ExpiresAt: time.Now().AddDate(1, 0, 0),
	})
	r.nextLotID++
}

// balance 测试辅助：读取当前可用余额
func (r *fakeCreditsRepo) balance(tenantID uint64) int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.balanceLocked(tenantID, time.Now())
}

func (r *fakeCreditsRepo) GetOrCreate(_ context.Context, tenantID uint64) (*model.TenantCredits, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return &model.TenantCredits{
		TenantID:  tenantID,
		Balance:   r.balanceLocked(tenantID, time.Now()),
		Unlimited: r.unlimited[tenantID],
	}, nil
}

func (r *fakeCreditsRepo) Deduct(_ context.Context, tenantID uint64, cost int64, action, refType string, refID uint64, note string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.unlimited[tenantID] {
		r.logs = append(r.logs, fakeCreditLog{tenantID: tenantID, amount: -cost, action: action, balanceAfter: -1, refType: refType, refID: refID, note: note})
		return nil
	}
	now := time.Now()
	if err := r.consumeLocked(tenantID, cost, now); err != nil {
		return err
	}
	r.logs = append(r.logs, fakeCreditLog{tenantID: tenantID, amount: -cost, action: action, balanceAfter: r.balanceLocked(tenantID, now), refType: refType, refID: refID, note: note})
	return nil
}

func (r *fakeCreditsRepo) Gift(_ context.Context, tenantID uint64, delta int64, expiresAt time.Time, action, note string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	switch {
	case delta > 0:
		r.lots[tenantID] = append(r.lots[tenantID], &model.CreditLot{
			ID: r.nextLotID, TenantID: tenantID, Amount: delta, Remaining: delta, Source: action, ExpiresAt: expiresAt,
		})
		r.nextLotID++
	case delta < 0:
		if err := r.consumeLocked(tenantID, -delta, now); err != nil {
			return err
		}
	}
	r.logs = append(r.logs, fakeCreditLog{tenantID: tenantID, amount: delta, action: action, balanceAfter: r.balanceLocked(tenantID, now), note: note})
	return nil
}

func (r *fakeCreditsRepo) ListActiveLots(_ context.Context, tenantID uint64) ([]*model.CreditLot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sortedActiveLotsLocked(tenantID, time.Now()), nil
}

func (r *fakeCreditsRepo) ExpireDueLots(_ context.Context) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	var total int64
	for tenantID, lots := range r.lots {
		for _, lot := range lots {
			if lot.Remaining > 0 && !lot.ExpiresAt.After(now) {
				total += lot.Remaining
				r.logs = append(r.logs, fakeCreditLog{tenantID: tenantID, amount: -lot.Remaining, action: "expire", refType: "credit_lot", refID: lot.ID, note: "积分到期作废"})
				lot.Remaining = 0
			}
		}
	}
	return total, nil
}

func (r *fakeCreditsRepo) SetUnlimited(_ context.Context, tenantID uint64, unlimited bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.unlimited[tenantID] = unlimited
	return nil
}

func (r *fakeCreditsRepo) ListLogs(_ context.Context, tenantID uint64, limit int) ([]*model.CreditLog, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*model.CreditLog
	for i := len(r.logs) - 1; i >= 0 && len(result) < limit; i-- {
		l := r.logs[i]
		if l.tenantID == tenantID {
			result = append(result, &model.CreditLog{
				TenantID:     l.tenantID,
				Amount:       l.amount,
				Action:       l.action,
				BalanceAfter: l.balanceAfter,
				RefType:      l.refType,
				RefID:        l.refID,
				Note:         l.note,
			})
		}
	}
	return result, nil
}

func (r *fakeCreditsRepo) ListLogsPaged(_ context.Context, tenantID uint64, page, size int, amountType string) ([]*model.CreditLog, int64, error) {
	all, _ := r.ListLogs(context.Background(), tenantID, 10000)
	var filtered []*model.CreditLog
	for _, l := range all {
		if amountType == "income" && l.Amount <= 0 {
			continue
		}
		if amountType == "expense" && l.Amount >= 0 {
			continue
		}
		filtered = append(filtered, l)
	}
	total := int64(len(filtered))
	start := (page - 1) * size
	if start >= len(filtered) {
		return nil, total, nil
	}
	end := start + size
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[start:end], total, nil
}

func (r *fakeCreditsRepo) ListByTenantIDs(_ context.Context, tenantIDs []uint64) ([]*model.TenantCredits, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*model.TenantCredits
	for _, tid := range tenantIDs {
		result = append(result, &model.TenantCredits{
			TenantID:  tid,
			Balance:   r.balanceLocked(tid, time.Now()),
			Unlimited: r.unlimited[tid],
		})
	}
	return result, nil
}

// fakeConfigRepo 返回固定积分单价
type fakeConfigRepo struct {
	prices map[string]string
}

func newFakeConfigRepo(prices map[string]string) *fakeConfigRepo {
	return &fakeConfigRepo{prices: prices}
}

func (r *fakeConfigRepo) GetByKey(_ context.Context, key string) (*model.PlatformConfig, error) {
	v, ok := r.prices[key]
	if !ok {
		return nil, nil
	}
	return &model.PlatformConfig{ConfigKey: key, ConfigValue: v}, nil
}

func (r *fakeConfigRepo) GetAll(_ context.Context) ([]*model.PlatformConfig, error) { return nil, nil }
func (r *fakeConfigRepo) Upsert(_ context.Context, _ *model.PlatformConfig) error   { return nil }

// 默认积分单价
var defaultPrices = map[string]string{
	model.ConfigKeyCreditsPerUpload:   "5",
	model.ConfigKeyCreditsPerOCR:      "10",
	model.ConfigKeyCreditsPerOrc2bill: "20",
}

func newSvc(repo *fakeCreditsRepo) service.CreditsService {
	return service.NewCreditsService(repo, newFakeConfigRepo(defaultPrices))
}

// ============================================================
// 测试场景
// ============================================================

// Scenario: 新用户赠送积分
func TestGift_NewUserReceivesCredits(t *testing.T) {
	repo := newFakeRepo()
	svc := newSvc(repo)
	ctx := context.Background()

	if err := svc.Gift(ctx, 1, 10000, "gift", "新用户注册赠送"); err != nil {
		t.Fatalf("Gift failed: %v", err)
	}

	resp, err := svc.GetBalance(ctx, 1)
	if err != nil {
		t.Fatalf("GetBalance failed: %v", err)
	}
	if resp.Balance != 10000 {
		t.Errorf("expected balance=10000, got %d", resp.Balance)
	}
	if len(resp.Logs) != 1 || resp.Logs[0].Action != "gift" {
		t.Errorf("expected 1 gift log, got %+v", resp.Logs)
	}
}

// Scenario: 上传扣费后余额正确递减
func TestDeduct_UploadAndOCR_BalanceDecreases(t *testing.T) {
	repo := newFakeRepo()
	repo.setBalance(1, 100)
	svc := newSvc(repo)
	ctx := context.Background()

	// 模拟上传一张图片：upload(5) + ocr(10)
	if err := svc.CheckAndDeduct(ctx, 1, "upload", "", 0, "图片上传"); err != nil {
		t.Fatal(err)
	}
	if err := svc.CheckAndDeduct(ctx, 1, "ocr", "ocr_record", 99, "OCR识别"); err != nil {
		t.Fatal(err)
	}

	if repo.balance(1) != 85 {
		t.Errorf("expected 85, got %d", repo.balance(1))
	}
	if len(repo.logs) != 2 {
		t.Errorf("expected 2 logs, got %d", len(repo.logs))
	}
}

// Scenario: 积分不足时返回 402 错误，余额不变，不写流水
func TestDeduct_InsufficientCredits_ReturnsError(t *testing.T) {
	fakeRepo := newFakeRepo()
	fakeRepo.setBalance(2, 4) // 少于 upload 的单价5
	svc := newSvc(fakeRepo)
	ctx := context.Background()

	err := svc.CheckAndDeduct(ctx, 2, "upload", "", 0, "图片上传")
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !errors.Is(err, service.ErrInsufficientCredits) {
		t.Errorf("expected ErrInsufficientCredits, got %v", err)
	}
	// 余额不变
	if fakeRepo.balance(2) != 4 {
		t.Errorf("balance must not change, got %d", fakeRepo.balance(2))
	}
	// 无流水日志
	if len(fakeRepo.logs) != 0 {
		t.Errorf("expected no logs on failure, got %d", len(fakeRepo.logs))
	}
}

// Scenario: unlimited 租户不扣余额，但仍写流水（balance_after=-1）
func TestDeduct_UnlimitedTenant_BalanceUnchanged(t *testing.T) {
	fakeRepo := newFakeRepo()
	fakeRepo.setBalance(3, 0)
	fakeRepo.unlimited[3] = true
	svc := newSvc(fakeRepo)
	ctx := context.Background()

	for _, action := range []string{"upload", "ocr", "orc2bill"} {
		if err := svc.CheckAndDeduct(ctx, 3, action, "", 0, ""); err != nil {
			t.Fatalf("unlimited deduct should not fail for action=%s: %v", action, err)
		}
	}

	// 余额始终是0
	if fakeRepo.balance(3) != 0 {
		t.Errorf("unlimited balance should stay 0, got %d", fakeRepo.balance(3))
	}
	// 流水中 balance_after 均为 -1
	for _, l := range fakeRepo.logs {
		if l.tenantID == 3 && l.balanceAfter != -1 {
			t.Errorf("unlimited log should have balance_after=-1, got %d for action=%s", l.balanceAfter, l.action)
		}
	}
}

// Scenario: 管理员设置为 unlimited 后再扣费不报错
func TestSetUnlimited_ThenDeduct_Succeeds(t *testing.T) {
	fakeRepo := newFakeRepo()
	fakeRepo.setBalance(4, 0)
	svc := newSvc(fakeRepo)
	ctx := context.Background()

	if err := svc.SetUnlimited(ctx, 4, true); err != nil {
		t.Fatal(err)
	}
	if err := svc.CheckAndDeduct(ctx, 4, "orc2bill", "", 0, "AI整理"); err != nil {
		t.Fatalf("expected success after unlimited, got: %v", err)
	}
}

// Scenario: 取消 unlimited 后余额不足应报错
func TestSetUnlimited_Cancel_ThenDeductFails(t *testing.T) {
	fakeRepo := newFakeRepo()
	fakeRepo.setBalance(5, 0)
	fakeRepo.unlimited[5] = true
	svc := newSvc(fakeRepo)
	ctx := context.Background()

	if err := svc.SetUnlimited(ctx, 5, false); err != nil {
		t.Fatal(err)
	}
	err := svc.CheckAndDeduct(ctx, 5, "upload", "", 0, "")
	if !errors.Is(err, service.ErrInsufficientCredits) {
		t.Errorf("expected ErrInsufficientCredits after cancel unlimited, got %v", err)
	}
}

// Scenario: 单价为0时跳过扣费（不写流水）
func TestDeduct_ZeroCost_NoLog(t *testing.T) {
	fakeRepo := newFakeRepo()
	fakeRepo.setBalance(6, 50)
	// 设置 orc2bill 单价为0
	prices := map[string]string{
		model.ConfigKeyCreditsPerUpload:   "5",
		model.ConfigKeyCreditsPerOCR:      "10",
		model.ConfigKeyCreditsPerOrc2bill: "0",
	}
	svc := service.NewCreditsService(fakeRepo, newFakeConfigRepo(prices))
	ctx := context.Background()

	if err := svc.CheckAndDeduct(ctx, 6, "orc2bill", "", 0, "AI整理"); err != nil {
		t.Fatalf("zero-cost deduct should not fail: %v", err)
	}
	if fakeRepo.balance(6) != 50 {
		t.Errorf("zero-cost should not change balance, got %d", fakeRepo.balance(6))
	}
	if len(fakeRepo.logs) != 0 {
		t.Errorf("zero-cost should not write log, got %d logs", len(fakeRepo.logs))
	}
}

// Scenario: 管理员充值积分，流水正确记录
func TestAdminAdjust_Increase_BalanceAndLogCorrect(t *testing.T) {
	fakeRepo := newFakeRepo()
	fakeRepo.setBalance(7, 100)
	svc := newSvc(fakeRepo)
	ctx := context.Background()

	if err := svc.Gift(ctx, 7, 500, "admin_adjust", "管理员充值"); err != nil {
		t.Fatal(err)
	}
	if fakeRepo.balance(7) != 600 {
		t.Errorf("expected 600, got %d", fakeRepo.balance(7))
	}
	if len(fakeRepo.logs) != 1 || fakeRepo.logs[0].note != "管理员充值" {
		t.Errorf("log incorrect: %+v", fakeRepo.logs)
	}
}

// Scenario: 并发10个请求同时扣费，余额只够1个，只有1个成功
func TestDeduct_Concurrent_NoDoubleDeduct(t *testing.T) {
	fakeRepo := newFakeRepo()
	fakeRepo.setBalance(8, 5) // 只够1次 upload(5)
	svc := newSvc(fakeRepo)
	ctx := context.Background()

	const n = 10
	errs := make([]error, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			errs[i] = svc.CheckAndDeduct(ctx, 8, "upload", "", 0, "并发测试")
		}()
	}
	wg.Wait()

	successCount := 0
	for _, err := range errs {
		if err == nil {
			successCount++
		} else if !errors.Is(err, service.ErrInsufficientCredits) {
			t.Errorf("unexpected error: %v", err)
		}
	}
	if successCount != 1 {
		t.Errorf("expected exactly 1 success, got %d", successCount)
	}
	if fakeRepo.balance(8) != 0 {
		t.Errorf("balance should be 0 after 1 success, got %d", fakeRepo.balance(8))
	}
	if len(fakeRepo.logs) != 1 {
		t.Errorf("expected 1 log entry, got %d", len(fakeRepo.logs))
	}
}

// Scenario: 流水日志包含 ref_type 和 ref_id
func TestDeduct_LogContainsRefInfo(t *testing.T) {
	fakeRepo := newFakeRepo()
	fakeRepo.setBalance(9, 100)
	svc := newSvc(fakeRepo)
	ctx := context.Background()

	if err := svc.CheckAndDeduct(ctx, 9, "ocr", "ocr_record", 42, "OCR识别"); err != nil {
		t.Fatal(err)
	}
	if len(fakeRepo.logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(fakeRepo.logs))
	}
	l := fakeRepo.logs[0]
	if l.refType != "ocr_record" || l.refID != 42 {
		t.Errorf("log ref mismatch: refType=%s refID=%d", l.refType, l.refID)
	}
	if l.amount != -10 {
		t.Errorf("expected -10, got %d", l.amount)
	}
	if l.balanceAfter != 90 {
		t.Errorf("expected balanceAfter=90, got %d", l.balanceAfter)
	}
}

// Scenario: GetBalance 返回最近20条流水且顺序倒序
func TestGetBalance_ReturnsLogsInDescOrder(t *testing.T) {
	fakeRepo := newFakeRepo()
	fakeRepo.setBalance(10, 1000)
	svc := newSvc(fakeRepo)
	ctx := context.Background()

	// 写25条日志，每次扣1积分（自定义价格为1）
	prices := map[string]string{model.ConfigKeyCreditsPerUpload: "1"}
	svc1 := service.NewCreditsService(fakeRepo, newFakeConfigRepo(prices))
	for i := 0; i < 25; i++ {
		_ = svc1.CheckAndDeduct(ctx, 10, "upload", "", 0, "")
	}

	resp, err := svc.GetBalance(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Logs) != 20 {
		t.Errorf("expected 20 logs (last 20), got %d", len(resp.Logs))
	}
}

// ============================================================
// 批次（lot）语义测试：最近过期优先 / 到期作废 / 有效期配置
// ============================================================

// addLot 测试辅助：直接注入一个指定过期时间的批次
func (r *fakeCreditsRepo) addLot(tenantID uint64, amount int64, expiresAt time.Time) uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := r.nextLotID
	r.lots[tenantID] = append(r.lots[tenantID], &model.CreditLot{
		ID: id, TenantID: tenantID, Amount: amount, Remaining: amount, Source: "test", ExpiresAt: expiresAt,
	})
	r.nextLotID++
	return id
}

// Scenario: 扣费优先消耗最近将要过期的批次
func TestDeduct_ConsumesSoonestExpiringLotFirst(t *testing.T) {
	fakeRepo := newFakeRepo()
	now := time.Now()
	lateLot := fakeRepo.addLot(11, 100, now.AddDate(1, 0, 0))  // 一年后过期（先充值）
	soonLot := fakeRepo.addLot(11, 100, now.AddDate(0, 1, 0))  // 一个月后过期（后充值）
	svc := newSvc(fakeRepo)
	ctx := context.Background()

	// 扣 10（upload 5 + ocr 挂在 soonLot 上应先扣）
	if err := svc.CheckAndDeduct(ctx, 11, "ocr", "ocr_record", 1, ""); err != nil {
		t.Fatal(err)
	}

	fakeRepo.mu.Lock()
	defer fakeRepo.mu.Unlock()
	for _, lot := range fakeRepo.lots[11] {
		switch lot.ID {
		case soonLot:
			if lot.Remaining != 90 {
				t.Errorf("最近过期批次应先被扣减到 90，实际 %d", lot.Remaining)
			}
		case lateLot:
			if lot.Remaining != 100 {
				t.Errorf("远期批次不应被扣减，实际 %d", lot.Remaining)
			}
		}
	}
}

// Scenario: 一个批次不够时跨批次扣减，仍按过期时间升序
func TestDeduct_SpansLotsInExpiryOrder(t *testing.T) {
	fakeRepo := newFakeRepo()
	now := time.Now()
	soonLot := fakeRepo.addLot(12, 6, now.AddDate(0, 0, 7))
	lateLot := fakeRepo.addLot(12, 100, now.AddDate(1, 0, 0))
	svc := newSvc(fakeRepo)

	// ocr 扣 10：soonLot 全部 6 分 + lateLot 4 分
	if err := svc.CheckAndDeduct(context.Background(), 12, "ocr", "ocr_record", 2, ""); err != nil {
		t.Fatal(err)
	}

	fakeRepo.mu.Lock()
	defer fakeRepo.mu.Unlock()
	for _, lot := range fakeRepo.lots[12] {
		switch lot.ID {
		case soonLot:
			if lot.Remaining != 0 {
				t.Errorf("近期批次应被扣光，实际 %d", lot.Remaining)
			}
		case lateLot:
			if lot.Remaining != 96 {
				t.Errorf("远期批次应剩 96，实际 %d", lot.Remaining)
			}
		}
	}
}

// Scenario: 已过期批次不可用于扣费，余额也不计入
func TestDeduct_ExpiredLotUnusable(t *testing.T) {
	fakeRepo := newFakeRepo()
	fakeRepo.addLot(13, 100, time.Now().Add(-time.Hour)) // 已过期
	svc := newSvc(fakeRepo)
	ctx := context.Background()

	resp, err := svc.GetBalance(ctx, 13)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Balance != 0 {
		t.Errorf("过期积分不应计入余额，实际 %d", resp.Balance)
	}
	err = svc.CheckAndDeduct(ctx, 13, "upload", "", 0, "")
	if !errors.Is(err, service.ErrInsufficientCredits) {
		t.Errorf("过期积分不可扣费，期望 ErrInsufficientCredits，实际 %v", err)
	}
}

// Scenario: ExpireDueCredits 清理到期批次并写 expire 流水
func TestExpireDueCredits_WritesExpireLog(t *testing.T) {
	fakeRepo := newFakeRepo()
	fakeRepo.addLot(14, 30, time.Now().Add(-time.Minute)) // 已到期
	fakeRepo.addLot(14, 50, time.Now().AddDate(0, 1, 0))  // 未到期
	svc := newSvc(fakeRepo)

	total, err := svc.ExpireDueCredits(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if total != 30 {
		t.Errorf("应作废 30 积分，实际 %d", total)
	}
	found := false
	for _, l := range fakeRepo.logs {
		if l.tenantID == 14 && l.action == "expire" && l.amount == -30 {
			found = true
		}
	}
	if !found {
		t.Error("应有 action=expire 的作废流水")
	}
}

// Scenario: Gift 生成的批次过期时间遵循 credits_validity_days 配置
func TestGift_LotExpiryFollowsValidityConfig(t *testing.T) {
	fakeRepo := newFakeRepo()
	prices := map[string]string{model.ConfigKeyCreditsValidityDays: "30"}
	svc := service.NewCreditsService(fakeRepo, newFakeConfigRepo(prices))
	ctx := context.Background()

	if err := svc.Gift(ctx, 15, 500, "gift", "充值"); err != nil {
		t.Fatal(err)
	}
	lots, err := fakeRepo.ListActiveLots(ctx, 15)
	if err != nil || len(lots) != 1 {
		t.Fatalf("应有 1 个批次，err=%v n=%d", err, len(lots))
	}
	want := time.Now().AddDate(0, 0, 30)
	if diff := lots[0].ExpiresAt.Sub(want); diff < -time.Minute || diff > time.Minute {
		t.Errorf("批次过期时间应约为 30 天后，实际 %v", lots[0].ExpiresAt)
	}
}

// Scenario: 管理员负数调整按最近过期优先扣，不足时报错
func TestAdminAdjust_NegativeDelta_ConsumesLots(t *testing.T) {
	fakeRepo := newFakeRepo()
	now := time.Now()
	soonLot := fakeRepo.addLot(16, 40, now.AddDate(0, 0, 3))
	fakeRepo.addLot(16, 60, now.AddDate(1, 0, 0))
	svc := newSvc(fakeRepo)
	ctx := context.Background()

	if err := svc.Gift(ctx, 16, -50, "admin_adjust", "管理员扣减"); err != nil {
		t.Fatal(err)
	}
	if got := fakeRepo.balance(16); got != 50 {
		t.Errorf("余额应为 50，实际 %d", got)
	}
	fakeRepo.mu.Lock()
	for _, lot := range fakeRepo.lots[16] {
		if lot.ID == soonLot && lot.Remaining != 0 {
			t.Errorf("近期批次应先被扣光，实际剩 %d", lot.Remaining)
		}
	}
	fakeRepo.mu.Unlock()

	// 超过余额的扣减应失败
	err := svc.Gift(ctx, 16, -100, "admin_adjust", "超额扣减")
	if !errors.Is(err, service.ErrInsufficientCredits) {
		t.Errorf("超额扣减应报积分不足，实际 %v", err)
	}
}

// Scenario: GetBalance 返回按过期时间升序的批次列表
func TestGetBalance_ReturnsLotsSortedByExpiry(t *testing.T) {
	fakeRepo := newFakeRepo()
	now := time.Now()
	fakeRepo.addLot(17, 100, now.AddDate(1, 0, 0))
	fakeRepo.addLot(17, 50, now.AddDate(0, 0, 10))
	svc := newSvc(fakeRepo)

	resp, err := svc.GetBalance(context.Background(), 17)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Lots) != 2 {
		t.Fatalf("应返回 2 个批次，实际 %d", len(resp.Lots))
	}
	if resp.Lots[0].Remaining != 50 {
		t.Errorf("第一个批次应是最先过期的（50 分），实际 %+v", resp.Lots[0])
	}
}
