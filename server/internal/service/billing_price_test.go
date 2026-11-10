package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"fandianjizhang/server/config"
	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/repo"
	"fandianjizhang/server/internal/service"
)

type fakeGateway struct {
	products   []service.PaymentProduct
	iapEnabled bool
	creates    int
	prepares   int
}

func (f *fakeGateway) CreateIntent(ctx context.Context, req service.PaymentIntentRequest) (*service.PaymentIntentResult, error) {
	f.creates++
	return &service.PaymentIntentResult{IntentRef: "ip-fake"}, nil
}

func (f *fakeGateway) PrepareOrder(ctx context.Context, intentRef, code string) (*service.PaymentPrepareParams, error) {
	f.prepares++
	return &service.PaymentPrepareParams{OutTradeNo: "vp12345678", Mode: "short_series_goods", SignData: "{}", PaySig: "p", Signature: "s"}, nil
}

func (f *fakeGateway) ListProducts(ctx context.Context) ([]service.PaymentProduct, service.PaymentCapabilities, error) {
	return f.products, service.PaymentCapabilities{IAPEnabled: f.iapEnabled}, nil
}

func creditProduct(sku string, price, credits int64, status string) service.PaymentProduct {
	payload, _ := json.Marshal(service.DeliveryPayload{CreditAmount: credits, ValidityDays: 365})
	return service.PaymentProduct{
		SKUCode: sku, Name: "示例商品", WxProductID: "wp1", PriceFen: price,
		Status: status, DeliveryPayload: payload,
	}
}

func TestSummarySyncsAuthoritativeProducts(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	r := repo.NewBillingRepo(db)
	creditsRepo := repo.NewTenantCreditsRepo(db)
	cfg := config.BillingConfig{Enabled: true, AppSlug: "hehuojizhangzhushou"}
	gw := &fakeGateway{products: []service.PaymentProduct{creditProduct("sample_goods_600", 600, 600, "listed")}}
	svc := service.NewBillingService(r, creditsRepo, nil, gw, cfg)
	summary, err := svc.Summary(ctx, 300, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.Packages) != 1 || summary.Packages[0].PriceFen != 600 || summary.Packages[0].CreditAmount != 600 {
		t.Fatalf("summary = %+v", summary)
	}
	if summary.SyncedAt == nil {
		t.Fatal("synced_at missing")
	}
}

func TestCreatePurchaseRejectsSilentPriceChange(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	r := repo.NewBillingRepo(db)
	cfg := config.BillingConfig{Enabled: true, AppSlug: "hehuojizhangzhushou"}
	gw := &fakeGateway{products: []service.PaymentProduct{creditProduct("sample_goods_600", 700, 700, "listed")}}
	svc := service.NewBillingService(r, repo.NewTenantCreditsRepo(db), nil, gw, cfg)
	if _, err := svc.CreatePurchase(ctx, 301, 1, "sample_goods_600", 600, "android"); !errors.Is(err, service.ErrPriceChanged) {
		t.Fatalf("expected ErrPriceChanged, got %v", err)
	}
	if gw.creates != 0 {
		t.Fatal("must not create intent when price changed")
	}
}

func TestCreatePurchaseRejectsInvalidDeliveryPayload(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	r := repo.NewBillingRepo(db)
	cfg := config.BillingConfig{Enabled: true, AppSlug: "hehuojizhangzhushou"}
	gw := &fakeGateway{products: []service.PaymentProduct{{
		SKUCode: "sample_goods_600", Name: "示例商品", WxProductID: "wp1", PriceFen: 600,
		Status: "listed", DeliveryPayload: json.RawMessage(`{"credit_amount":0}`),
	}}}
	svc := service.NewBillingService(r, repo.NewTenantCreditsRepo(db), nil, gw, cfg)
	if _, err := svc.CreatePurchase(ctx, 302, 1, "sample_goods_600", 600, "android"); !errors.Is(err, service.ErrInvalidDelivery) {
		t.Fatalf("expected ErrInvalidDelivery, got %v", err)
	}
}

func TestCreatePurchaseRejectsUnlisted(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	r := repo.NewBillingRepo(db)
	cfg := config.BillingConfig{Enabled: true, AppSlug: "hehuojizhangzhushou"}
	gw := &fakeGateway{products: []service.PaymentProduct{creditProduct("sample_goods_600", 600, 600, "unlisted")}}
	svc := service.NewBillingService(r, repo.NewTenantCreditsRepo(db), nil, gw, cfg)
	if _, err := svc.CreatePurchase(ctx, 303, 1, "sample_goods_600", 600, "android"); !errors.Is(err, service.ErrBillingPackageGone) {
		t.Fatalf("expected unlisted error, got %v", err)
	}
}

func TestPrepareRechecksPriceAndCredits(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	r := repo.NewBillingRepo(db)
	cfg := config.BillingConfig{Enabled: true, AppSlug: "hehuojizhangzhushou"}
	gw := &fakeGateway{products: []service.PaymentProduct{creditProduct("sample_goods_600", 600, 600, "listed")}}
	svc := service.NewBillingService(r, repo.NewTenantCreditsRepo(db), nil, gw, cfg)
	intent, err := svc.CreatePurchase(ctx, 304, 1, "sample_goods_600", 600, "android")
	if err != nil {
		t.Fatal(err)
	}
	gw.products[0] = creditProduct("sample_goods_600", 600, 700, "listed")
	if _, err := svc.PreparePayment(ctx, 304, 1, intent.BusinessRef, "code"); !errors.Is(err, service.ErrPriceChanged) {
		t.Fatalf("expected ErrPriceChanged on prepare, got %v", err)
	}
	if gw.prepares != 0 {
		t.Fatal("must not call prepare after credit change")
	}
}

func TestSyncMarksMissingPackagesUnlisted(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	r := repo.NewBillingRepo(db)
	cfg := config.BillingConfig{Enabled: true, AppSlug: "hehuojizhangzhushou"}
	if err := r.UpsertBillingPackages(ctx, []*model.BillingPackage{{
		AppSlug: cfg.AppSlug, SKUCode: "old_sku", Name: "旧商品", PriceFen: 600,
		CreditAmount: 600, ValidityDays: 365, Status: "listed",
	}}); err != nil {
		t.Fatal(err)
	}
	gw := &fakeGateway{products: []service.PaymentProduct{creditProduct("sample_goods_600", 600, 600, "listed")}}
	svc := service.NewBillingService(r, repo.NewTenantCreditsRepo(db), nil, gw, cfg)
	if _, err := svc.SyncPackages(ctx); err != nil {
		t.Fatal(err)
	}
	pkgs, err := r.ListBillingPackages(ctx, cfg.AppSlug, false)
	if err != nil {
		t.Fatal(err)
	}
	status := map[string]string{}
	for _, p := range pkgs {
		status[p.SKUCode] = p.Status
	}
	if status["old_sku"] != "unlisted" {
		t.Fatalf("missing remote package not unlisted: %+v", status)
	}
	if status["sample_goods_600"] != "listed" {
		t.Fatalf("new package not synced: %+v", status)
	}
}
