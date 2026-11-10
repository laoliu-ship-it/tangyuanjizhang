package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"fandianjizhang/server/config"
	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/repo"
)

var (
	ErrBillingDisabled    = errors.New("积分购买功能未开启")
	ErrBillingForbidden   = errors.New("仅账本管理员可购买")
	ErrBillingPackageGone = errors.New("商品不存在或未上架")
	ErrBillingIntentGone  = errors.New("购买意图不存在或已过期")
	ErrBillingIntentOwner = errors.New("购买意图不属于当前用户")
	ErrPriceChanged       = errors.New("商品价格或积分权益已更新，请刷新后重试")
	ErrIOSIAPDisabled     = errors.New("iOS 支付能力暂未开通")
	ErrInvalidDelivery    = errors.New("无效的交付快照")
)

// PaymentProduct / PaymentGateway 是汤圆本地窄接口；核心 BillingService 不依赖小管家 wire DTO/HTTP。
type PaymentProduct struct {
	SKUCode         string
	Name            string
	WxProductID     string
	PriceFen        int64
	Status          string
	DeliveryPayload json.RawMessage
	Sort            int
}

type PaymentCapabilities struct {
	IAPEnabled bool
	Env        int
}

type PaymentIntentRequest struct {
	BusinessReference string
	SKUCode           string
	Quantity          int
	AmountFen         int64
	PayerOpenID       string
}

type PaymentIntentResult struct {
	IntentRef string
	ExpiresAt time.Time
}

type PaymentPrepareParams struct {
	OutTradeNo string
	Mode       string
	SignData   string
	PaySig     string
	Signature  string
	Env        int
}

type PaymentGateway interface {
	ListProducts(ctx context.Context) ([]PaymentProduct, PaymentCapabilities, error)
	CreateIntent(ctx context.Context, req PaymentIntentRequest) (*PaymentIntentResult, error)
	PrepareOrder(ctx context.Context, intentRef, code string) (*PaymentPrepareParams, error)
}

// DeliveryPayload 是汤圆自己的交付 schema；小管家只把它当不透明 JSON 原样保存/投递。
type DeliveryPayload struct {
	CreditAmount int64 `json:"credit_amount"`
	ValidityDays int   `json:"validity_days"`
}

func ParseDeliveryPayload(raw json.RawMessage) (*DeliveryPayload, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("%w: 缺少 delivery_payload", ErrInvalidDelivery)
	}
	var p DeliveryPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidDelivery, err)
	}
	if p.CreditAmount <= 0 || p.CreditAmount > 1_000_000_000 {
		return nil, fmt.Errorf("%w: credit_amount 必须为 1..1000000000", ErrInvalidDelivery)
	}
	if p.ValidityDays <= 0 {
		p.ValidityDays = 365
	}
	if p.ValidityDays > 3650 {
		return nil, fmt.Errorf("%w: validity_days 过大", ErrInvalidDelivery)
	}
	return &p, nil
}

type BillingService interface {
	Enabled() bool
	Summary(ctx context.Context, tenantID uint64, refresh bool) (*dto.BillingSummaryResp, error)
	SyncPackages(ctx context.Context) (int, error)
	CreatePurchase(ctx context.Context, tenantID, userID uint64, sku string, expectedPriceFen int64, platform string) (*dto.BillingIntentResp, error)
	PreparePayment(ctx context.Context, tenantID, userID uint64, ref, code string) (*dto.BillingPrepareResp, error)
	GetIntent(ctx context.Context, tenantID uint64, ref string) (*dto.BillingIntentResp, error)
	ListIntents(ctx context.Context, tenantID uint64, page, size int) (*dto.BillingIntentPageResp, error)
	ProcessDelivery(ctx context.Context, event *model.BillingDeliveryEvent) (bool, error)
	ProcessRefund(ctx context.Context, event *model.BillingDeliveryEvent) (bool, error)
	SeedPackages(ctx context.Context, packages []*model.BillingPackage) error
}

type billingService struct {
	repo        repo.BillingRepo
	creditsRepo repo.TenantCreditsRepo
	configRepo  repo.PlatformConfigRepo
	gateway     PaymentGateway
	cfg         config.BillingConfig
}

func NewBillingService(billingRepo repo.BillingRepo, creditsRepo repo.TenantCreditsRepo, configRepo repo.PlatformConfigRepo, gateway PaymentGateway, cfg config.BillingConfig) BillingService {
	return &billingService{repo: billingRepo, creditsRepo: creditsRepo, configRepo: configRepo, gateway: gateway, cfg: cfg}
}

func (s *billingService) Enabled() bool { return s.cfg.Enabled }

func (s *billingService) Summary(ctx context.Context, tenantID uint64, refresh bool) (*dto.BillingSummaryResp, error) {
	packages, err := s.repo.ListBillingPackages(ctx, s.cfg.AppSlug, true)
	if err != nil {
		return nil, err
	}
	stale := false
	if s.cfg.Enabled && (refresh || len(packages) == 0) {
		if _, serr := s.SyncPackages(ctx); serr != nil {
			if len(packages) == 0 {
				return nil, fmt.Errorf("读取商品失败: %w", serr)
			}
			stale = true
		} else {
			packages, err = s.repo.ListBillingPackages(ctx, s.cfg.AppSlug, true)
			if err != nil {
				return nil, err
			}
		}
	}
	credits, err := s.creditsRepo.GetOrCreate(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	capability, err := s.repo.GetBillingCapability(ctx, s.cfg.AppSlug)
	if err != nil {
		return nil, err
	}
	unit := s.unitPrices(ctx)
	items := make([]dto.BillingPackageItem, 0, len(packages))
	var syncedAt *time.Time
	for _, p := range packages {
		items = append(items, dto.BillingPackageItem{
			SKUCode: p.SKUCode, Name: p.Name, WxProductID: p.WxProductID, PriceFen: p.PriceFen,
			CreditAmount: p.CreditAmount, ValidityDays: p.ValidityDays, Status: p.Status,
			SyncedAt: p.SyncedAt, Estimates: estimateUsage(p.CreditAmount, unit),
		})
		if p.SyncedAt != nil && (syncedAt == nil || p.SyncedAt.After(*syncedAt)) {
			syncedAt = p.SyncedAt
		}
	}
	return &dto.BillingSummaryResp{
		Enabled: s.cfg.Enabled, Packages: items,
		CreditBalance: credits.Balance, Unlimited: credits.Unlimited,
		UnitPrices: unit, Estimates: estimateUsage(credits.Balance, unit),
		SyncedAt: syncedAt, Stale: stale, IAPEnabled: capability.IAPEnabled,
	}, nil
}

func (s *billingService) SyncPackages(ctx context.Context) (int, error) {
	if s.gateway == nil {
		return 0, ErrBillingDisabled
	}
	products, capability, err := s.gateway.ListProducts(ctx)
	if err != nil {
		return 0, err
	}
	now := time.Now()
	if err := s.repo.SaveBillingCapability(ctx, &model.BillingCapability{
		AppSlug: s.cfg.AppSlug, IAPEnabled: capability.IAPEnabled, Env: capability.Env, SyncedAt: &now,
	}); err != nil {
		return 0, err
	}
	local, err := s.repo.ListBillingPackages(ctx, s.cfg.AppSlug, false)
	if err != nil {
		return 0, err
	}
	bySKU := make(map[string]bool, len(products))
	upserts := make([]*model.BillingPackage, 0, len(products)+len(local))
	for _, p := range products {
		pkg, perr := s.remoteToPackage(p, now)
		if perr != nil {
			// 无效的汤圆积分 schema 不能上架，也不让通用模块解释它。
			// 不加入 bySKU，从而把本地同 SKU 标记下架，避免继续售卖旧权益。
			continue
		}
		bySKU[p.SKUCode] = true
		upserts = append(upserts, pkg)
	}
	for _, p := range local {
		if bySKU[p.SKUCode] {
			continue
		}
		p.Status = "unlisted"
		p.SyncedAt = &now
		upserts = append(upserts, p)
	}
	if err := s.repo.UpsertBillingPackages(ctx, upserts); err != nil {
		return 0, err
	}
	return len(upserts), nil
}

func (s *billingService) remoteToPackage(p PaymentProduct, now time.Time) (*model.BillingPackage, error) {
	payload, err := ParseDeliveryPayload(p.DeliveryPayload)
	if err != nil {
		return nil, err
	}
	return &model.BillingPackage{
		AppSlug: s.cfg.AppSlug, SKUCode: p.SKUCode, Name: p.Name, WxProductID: p.WxProductID,
		PriceFen: p.PriceFen, CreditAmount: payload.CreditAmount, ValidityDays: payload.ValidityDays,
		Status: p.Status, Sort: p.Sort, SyncedAt: &now,
	}, nil
}

func (s *billingService) authoritativeProduct(ctx context.Context, sku string) (*PaymentProduct, error) {
	if s.gateway == nil {
		return nil, ErrBillingDisabled
	}
	products, _, err := s.gateway.ListProducts(ctx)
	if err != nil {
		return nil, err
	}
	for i := range products {
		if products[i].SKUCode == sku {
			return &products[i], nil
		}
	}
	return nil, ErrBillingPackageGone
}

func (s *billingService) CreatePurchase(ctx context.Context, tenantID, userID uint64, sku string, expectedPriceFen int64, platform string) (*dto.BillingIntentResp, error) {
	if !s.cfg.Enabled {
		return nil, ErrBillingDisabled
	}
	if platform == "ios" {
		capability, err := s.repo.GetBillingCapability(ctx, s.cfg.AppSlug)
		if err != nil {
			return nil, err
		}
		if !capability.IAPEnabled {
			return nil, ErrIOSIAPDisabled
		}
	}
	remote, err := s.authoritativeProduct(ctx, sku)
	if err != nil {
		return nil, err
	}
	if remote.Status != "listed" || remote.WxProductID == "" || remote.PriceFen <= 0 {
		return nil, ErrBillingPackageGone
	}
	payload, err := ParseDeliveryPayload(remote.DeliveryPayload)
	if err != nil {
		return nil, err
	}
	if expectedPriceFen > 0 && expectedPriceFen != remote.PriceFen {
		_, _ = s.SyncPackages(ctx)
		return nil, fmt.Errorf("%w（当前 %d 分）", ErrPriceChanged, remote.PriceFen)
	}
	local, err := s.remoteToPackage(*remote, time.Now())
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpsertBillingPackages(ctx, []*model.BillingPackage{local}); err != nil {
		return nil, err
	}
	ref, err := newBusinessRef()
	if err != nil {
		return nil, err
	}
	intent := &model.BillingPurchaseIntent{
		TenantID: tenantID, UserID: userID, AppSlug: s.cfg.AppSlug, BusinessRef: ref,
		SKUCode: remote.SKUCode, AmountFen: remote.PriceFen,
		CreditAmount: payload.CreditAmount, ValidityDays: payload.ValidityDays,
		Status: model.BillingIntentCreated, ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	if err := s.repo.CreateBillingIntent(ctx, intent); err != nil {
		return nil, err
	}
	resp, err := s.gateway.CreateIntent(ctx, PaymentIntentRequest{
		BusinessReference: ref, SKUCode: remote.SKUCode, Quantity: 1, AmountFen: remote.PriceFen,
	})
	if err != nil {
		intent.Status = model.BillingIntentCancelled
		_ = s.repo.UpdateBillingIntent(ctx, intent)
		return nil, err
	}
	intent.XgjIntentRef = resp.IntentRef
	if !resp.ExpiresAt.IsZero() {
		intent.ExpiresAt = resp.ExpiresAt
	}
	if err := s.repo.UpdateBillingIntent(ctx, intent); err != nil {
		return nil, err
	}
	return &dto.BillingIntentResp{
		BusinessRef: ref, SKUCode: remote.SKUCode, Name: remote.Name, AmountFen: remote.PriceFen,
		CreditAmount: payload.CreditAmount, ValidityDays: payload.ValidityDays,
		ExpiresAt: intent.ExpiresAt, Status: intent.Status,
	}, nil
}

func (s *billingService) PreparePayment(ctx context.Context, tenantID, userID uint64, ref, code string) (*dto.BillingPrepareResp, error) {
	if !s.cfg.Enabled {
		return nil, ErrBillingDisabled
	}
	intent, err := s.repo.GetBillingIntentByRef(ctx, tenantID, ref)
	if err != nil || intent == nil {
		return nil, ErrBillingIntentGone
	}
	if intent.UserID != userID {
		return nil, ErrBillingIntentOwner
	}
	if intent.Status != model.BillingIntentCreated || time.Now().After(intent.ExpiresAt) {
		return nil, ErrBillingIntentGone
	}
	if intent.XgjIntentRef == "" {
		return nil, fmt.Errorf("购买意图未完成服务间注册")
	}
	remote, err := s.authoritativeProduct(ctx, intent.SKUCode)
	if err != nil {
		return nil, err
	}
	payload, err := ParseDeliveryPayload(remote.DeliveryPayload)
	if err != nil {
		return nil, err
	}
	if remote.Status != "listed" || remote.PriceFen != intent.AmountFen ||
		payload.CreditAmount != intent.CreditAmount || payload.ValidityDays != intent.ValidityDays {
		_, _ = s.SyncPackages(ctx)
		return nil, fmt.Errorf("%w（当前价格 %d 分，积分 %d）", ErrPriceChanged, remote.PriceFen, payload.CreditAmount)
	}
	resp, err := s.gateway.PrepareOrder(ctx, intent.XgjIntentRef, code)
	if err != nil {
		return nil, err
	}
	return &dto.BillingPrepareResp{
		OutTradeNo: resp.OutTradeNo, Mode: resp.Mode, SignData: resp.SignData,
		PaySig: resp.PaySig, Signature: resp.Signature, Env: resp.Env,
	}, nil
}

func (s *billingService) GetIntent(ctx context.Context, tenantID uint64, ref string) (*dto.BillingIntentResp, error) {
	intent, err := s.repo.GetBillingIntentByRef(ctx, tenantID, ref)
	if err != nil || intent == nil {
		return nil, ErrBillingIntentGone
	}
	return intentResp(intent, ""), nil
}

func (s *billingService) ListIntents(ctx context.Context, tenantID uint64, page, size int) (*dto.BillingIntentPageResp, error) {
	items, total, err := s.repo.ListBillingIntents(ctx, tenantID, page, size)
	if err != nil {
		return nil, err
	}
	out := make([]dto.BillingIntentResp, 0, len(items))
	for _, i := range items {
		out = append(out, *intentResp(i, ""))
	}
	return &dto.BillingIntentPageResp{Total: total, Page: page, Size: size, Items: out}, nil
}

func intentResp(i *model.BillingPurchaseIntent, name string) *dto.BillingIntentResp {
	return &dto.BillingIntentResp{
		BusinessRef: i.BusinessRef, SKUCode: i.SKUCode, Name: name, AmountFen: i.AmountFen,
		CreditAmount: i.CreditAmount, ValidityDays: i.ValidityDays,
		ExpiresAt: i.ExpiresAt, Status: i.Status,
	}
}

// ProcessDelivery 把不透明 delivery_payload 解析成汤圆本地积分 schema 后再落库。
func (s *billingService) ProcessDelivery(ctx context.Context, event *model.BillingDeliveryEvent) (bool, error) {
	if event == nil {
		return false, ErrInvalidDelivery
	}
	payload, err := ParseDeliveryPayload(event.DeliveryPayload)
	if err != nil {
		return false, err
	}
	event.CreditAmount = payload.CreditAmount
	event.ValidityDays = payload.ValidityDays
	return s.repo.ProcessBillingDelivery(ctx, event)
}

func (s *billingService) ProcessRefund(ctx context.Context, event *model.BillingDeliveryEvent) (bool, error) {
	if event == nil {
		return false, ErrInvalidDelivery
	}
	return s.repo.ProcessBillingRefund(ctx, event)
}

func (s *billingService) SeedPackages(ctx context.Context, packages []*model.BillingPackage) error {
	return s.repo.UpsertBillingPackages(ctx, packages)
}

// ── 积分估算（只展示，不是可兑现的独立配额）──

func (s *billingService) unitPrices(ctx context.Context) dto.BillingUnitPrices {
	// 与 CheckAndDeduct 使用同一解析函数：缺失/读取失败/非法/负数一律 0，不虚构默认单价。
	return dto.BillingUnitPrices{
		Upload: getPlatformActionCost(ctx, s.configRepo, "upload"),
		OCR:    getPlatformActionCost(ctx, s.configRepo, "ocr"),
		AI:     getPlatformActionCost(ctx, s.configRepo, "orc2bill"),
	}
}

func estimateUsage(credits int64, unit dto.BillingUnitPrices) dto.BillingEstimates {
	divide := func(credits, cost int64) *int64 {
		if credits < 0 || cost <= 0 {
			return nil
		}
		v := credits / cost
		return &v
	}
	add := func(vals ...int64) (int64, bool) {
		var total int64
		for _, v := range vals {
			if v < 0 || v > math.MaxInt64-total {
				return 0, false
			}
			total += v
		}
		return total, true
	}
	uploadOCR, ok1 := add(unit.Upload, unit.OCR)
	full, ok2 := add(unit.Upload, unit.OCR, unit.AI)
	if !ok1 {
		uploadOCR = 0
	}
	if !ok2 {
		full = 0
	}
	return dto.BillingEstimates{
		UploadOCR: divide(credits, uploadOCR),
		AI:        divide(credits, unit.AI),
		FullChain: divide(credits, full),
	}
}

func newBusinessRef() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "br" + fmt.Sprintf("%d", time.Now().Unix()) + hex.EncodeToString(b[:]), nil
}
