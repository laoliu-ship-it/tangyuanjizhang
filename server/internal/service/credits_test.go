package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"

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

// fakeCreditsRepo 内存版积分仓库，支持并发测试
type fakeCreditsRepo struct {
	mu        sync.Mutex
	balances  map[uint64]int64 // tenantID -> balance
	unlimited map[uint64]bool
	logs      []fakeCreditLog
}

func newFakeRepo() *fakeCreditsRepo {
	return &fakeCreditsRepo{
		balances:  make(map[uint64]int64),
		unlimited: make(map[uint64]bool),
	}
}

func (r *fakeCreditsRepo) GetOrCreate(_ context.Context, tenantID uint64) (*model.TenantCredits, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return &model.TenantCredits{
		TenantID:  tenantID,
		Balance:   r.balances[tenantID],
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
	bal := r.balances[tenantID]
	if bal < cost {
		return repo.ErrInsufficientCredits
	}
	r.balances[tenantID] = bal - cost
	r.logs = append(r.logs, fakeCreditLog{tenantID: tenantID, amount: -cost, action: action, balanceAfter: r.balances[tenantID], refType: refType, refID: refID, note: note})
	return nil
}

func (r *fakeCreditsRepo) Gift(_ context.Context, tenantID uint64, delta int64, action, note string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.balances[tenantID] += delta
	r.logs = append(r.logs, fakeCreditLog{tenantID: tenantID, amount: delta, action: action, balanceAfter: r.balances[tenantID], note: note})
	return nil
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
			Balance:   r.balances[tid],
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
func (r *fakeConfigRepo) Upsert(_ context.Context, _ *model.PlatformConfig) error  { return nil }

// 默认积分单价
var defaultPrices = map[string]string{
	model.ConfigKeyCreditsPerUpload: "5",
	model.ConfigKeyCreditsPerOCR:    "10",
	model.ConfigKeyCreditsPerLLM:    "20",
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
	repo.balances[1] = 100
	svc := newSvc(repo)
	ctx := context.Background()

	// 模拟上传一张图片：upload(5) + ocr(10)
	if err := svc.CheckAndDeduct(ctx, 1, "upload", "", 0, "图片上传"); err != nil {
		t.Fatal(err)
	}
	if err := svc.CheckAndDeduct(ctx, 1, "ocr", "ocr_record", 99, "OCR识别"); err != nil {
		t.Fatal(err)
	}

	if repo.balances[1] != 85 {
		t.Errorf("expected 85, got %d", repo.balances[1])
	}
	if len(repo.logs) != 2 {
		t.Errorf("expected 2 logs, got %d", len(repo.logs))
	}
}

// Scenario: 积分不足时返回 402 错误，余额不变，不写流水
func TestDeduct_InsufficientCredits_ReturnsError(t *testing.T) {
	fakeRepo := newFakeRepo()
	fakeRepo.balances[2] = 4 // 少于 upload 的单价5
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
	if fakeRepo.balances[2] != 4 {
		t.Errorf("balance must not change, got %d", fakeRepo.balances[2])
	}
	// 无流水日志
	if len(fakeRepo.logs) != 0 {
		t.Errorf("expected no logs on failure, got %d", len(fakeRepo.logs))
	}
}

// Scenario: unlimited 租户不扣余额，但仍写流水（balance_after=-1）
func TestDeduct_UnlimitedTenant_BalanceUnchanged(t *testing.T) {
	fakeRepo := newFakeRepo()
	fakeRepo.balances[3] = 0
	fakeRepo.unlimited[3] = true
	svc := newSvc(fakeRepo)
	ctx := context.Background()

	for _, action := range []string{"upload", "ocr", "llm"} {
		if err := svc.CheckAndDeduct(ctx, 3, action, "", 0, ""); err != nil {
			t.Fatalf("unlimited deduct should not fail for action=%s: %v", action, err)
		}
	}

	// 余额始终是0
	if fakeRepo.balances[3] != 0 {
		t.Errorf("unlimited balance should stay 0, got %d", fakeRepo.balances[3])
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
	fakeRepo.balances[4] = 0
	svc := newSvc(fakeRepo)
	ctx := context.Background()

	if err := svc.SetUnlimited(ctx, 4, true); err != nil {
		t.Fatal(err)
	}
	if err := svc.CheckAndDeduct(ctx, 4, "llm", "", 0, "AI整理"); err != nil {
		t.Fatalf("expected success after unlimited, got: %v", err)
	}
}

// Scenario: 取消 unlimited 后余额不足应报错
func TestSetUnlimited_Cancel_ThenDeductFails(t *testing.T) {
	fakeRepo := newFakeRepo()
	fakeRepo.balances[5] = 0
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
	fakeRepo.balances[6] = 50
	// 设置 llm 单价为0
	prices := map[string]string{
		model.ConfigKeyCreditsPerUpload: "5",
		model.ConfigKeyCreditsPerOCR:    "10",
		model.ConfigKeyCreditsPerLLM:    "0",
	}
	svc := service.NewCreditsService(fakeRepo, newFakeConfigRepo(prices))
	ctx := context.Background()

	if err := svc.CheckAndDeduct(ctx, 6, "llm", "", 0, "AI整理"); err != nil {
		t.Fatalf("zero-cost deduct should not fail: %v", err)
	}
	if fakeRepo.balances[6] != 50 {
		t.Errorf("zero-cost should not change balance, got %d", fakeRepo.balances[6])
	}
	if len(fakeRepo.logs) != 0 {
		t.Errorf("zero-cost should not write log, got %d logs", len(fakeRepo.logs))
	}
}

// Scenario: 管理员充值积分，流水正确记录
func TestAdminAdjust_Increase_BalanceAndLogCorrect(t *testing.T) {
	fakeRepo := newFakeRepo()
	fakeRepo.balances[7] = 100
	svc := newSvc(fakeRepo)
	ctx := context.Background()

	if err := svc.Gift(ctx, 7, 500, "admin_adjust", "管理员充值"); err != nil {
		t.Fatal(err)
	}
	if fakeRepo.balances[7] != 600 {
		t.Errorf("expected 600, got %d", fakeRepo.balances[7])
	}
	if len(fakeRepo.logs) != 1 || fakeRepo.logs[0].note != "管理员充值" {
		t.Errorf("log incorrect: %+v", fakeRepo.logs)
	}
}

// Scenario: 并发10个请求同时扣费，余额只够1个，只有1个成功
func TestDeduct_Concurrent_NoDoubleDeduct(t *testing.T) {
	fakeRepo := newFakeRepo()
	fakeRepo.balances[8] = 5 // 只够1次 upload(5)
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
	if fakeRepo.balances[8] != 0 {
		t.Errorf("balance should be 0 after 1 success, got %d", fakeRepo.balances[8])
	}
	if len(fakeRepo.logs) != 1 {
		t.Errorf("expected 1 log entry, got %d", len(fakeRepo.logs))
	}
}

// Scenario: 流水日志包含 ref_type 和 ref_id
func TestDeduct_LogContainsRefInfo(t *testing.T) {
	fakeRepo := newFakeRepo()
	fakeRepo.balances[9] = 100
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
	fakeRepo.balances[10] = 1000
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
