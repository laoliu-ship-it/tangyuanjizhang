package service

import (
	"context"
	"errors"
	"testing"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/model"
)

type fakeConfigRepo struct {
	values map[string]string
	err    error
}

func (f *fakeConfigRepo) GetByKey(ctx context.Context, key string) (*model.PlatformConfig, error) {
	if f.err != nil {
		return nil, f.err
	}
	v, ok := f.values[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return &model.PlatformConfig{ConfigKey: key, ConfigValue: v}, nil
}

func (f *fakeConfigRepo) GetAll(ctx context.Context) ([]*model.PlatformConfig, error) {
	return nil, nil
}

func (f *fakeConfigRepo) Upsert(ctx context.Context, cfg *model.PlatformConfig) error {
	return nil
}

func TestEstimateUsage_PartialZeroPrices(t *testing.T) {
	// upload 免费、OCR 收费时，应按 OCR 单独估算，不能把总成本当 0。
	got := estimateUsage(100, dto.BillingUnitPrices{Upload: 0, OCR: 10, AI: 0})
	if got.UploadOCR == nil || *got.UploadOCR != 10 {
		t.Fatalf("upload+OCR estimate = %+v", got.UploadOCR)
	}
	if got.AI != nil {
		t.Fatalf("AI free/unknown should be nil, got %v", *got.AI)
	}
	if got.FullChain == nil || *got.FullChain != 10 {
		t.Fatalf("full estimate = %+v", got.FullChain)
	}
}

func TestEstimateUsage_AllZeroOrInvalid(t *testing.T) {
	got := estimateUsage(100, dto.BillingUnitPrices{})
	if got.UploadOCR != nil || got.AI != nil || got.FullChain != nil {
		t.Fatalf("all zero should not fabricate estimates: %+v", got)
	}
	// 负单价按不可用处理。
	got = estimateUsage(100, dto.BillingUnitPrices{Upload: -1, OCR: 10, AI: 20})
	if got.UploadOCR != nil || got.FullChain != nil {
		t.Fatalf("negative price should not estimate: %+v", got)
	}
}

func TestEstimateUsage_Overflow(t *testing.T) {
	got := estimateUsage(100, dto.BillingUnitPrices{Upload: 1<<62 + 1, OCR: 1<<62 + 1, AI: 1})
	if got.UploadOCR != nil || got.FullChain != nil {
		t.Fatalf("overflow should not estimate: %+v", got)
	}
	if got.AI == nil || *got.AI != 100 {
		t.Fatalf("AI estimate = %+v", got.AI)
	}
}

func TestUnitPrices_SharedParsing(t *testing.T) {
	svc := &billingService{configRepo: &fakeConfigRepo{values: map[string]string{
		model.ConfigKeyCreditsPerUpload:   "0",
		model.ConfigKeyCreditsPerOCR:      "10",
		model.ConfigKeyCreditsPerOrc2bill: "12abc",
	}}}
	got := svc.unitPrices(context.Background())
	if got.Upload != 0 || got.OCR != 10 || got.AI != 0 {
		t.Fatalf("unit prices = %+v", got)
	}
	// 读取失败/缺配置与免费同样按 0，不展示虚构默认值。
	svc = &billingService{configRepo: &fakeConfigRepo{err: errors.New("db down")}}
	got = svc.unitPrices(context.Background())
	if got.Upload != 0 || got.OCR != 0 || got.AI != 0 {
		t.Fatalf("failed config should be zero: %+v", got)
	}
}
