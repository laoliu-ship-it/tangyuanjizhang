package service

import (
	"context"
	"errors"
	"time"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/repo"
)

var ErrMerchantNotFound = errors.New("商户不存在")

type MerchantService interface {
	Create(ctx context.Context, tenantID uint64, name string) (*model.Merchant, error)
	GetOrCreateByName(ctx context.Context, tenantID uint64, name string) (*model.Merchant, error)
	GetByID(ctx context.Context, id, tenantID uint64) (*model.Merchant, error)
	List(ctx context.Context, tenantID uint64) ([]*dto.MerchantResp, error)
	Update(ctx context.Context, id, tenantID uint64, name string) (*model.Merchant, error)
	Delete(ctx context.Context, id, tenantID uint64) error
}

type merchantService struct {
	merchantRepo repo.MerchantRepo
}

func NewMerchantService(merchantRepo repo.MerchantRepo) MerchantService {
	return &merchantService{merchantRepo: merchantRepo}
}

func (s *merchantService) Create(ctx context.Context, tenantID uint64, name string) (*model.Merchant, error) {
	m := &model.Merchant{
		TenantID: tenantID,
		Name:     name,
	}
	if err := s.merchantRepo.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *merchantService) GetOrCreateByName(ctx context.Context, tenantID uint64, name string) (*model.Merchant, error) {
	// 先查找
	m, err := s.merchantRepo.GetByName(ctx, name, tenantID)
	if err != nil {
		return nil, err
	}
	if m != nil {
		return m, nil
	}
	// 不存在则创建
	m = &model.Merchant{
		TenantID: tenantID,
		Name:     name,
	}
	if err := s.merchantRepo.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *merchantService) GetByID(ctx context.Context, id, tenantID uint64) (*model.Merchant, error) {
	m, err := s.merchantRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrMerchantNotFound
	}
	return m, nil
}

func (s *merchantService) List(ctx context.Context, tenantID uint64) ([]*dto.MerchantResp, error) {
	merchants, err := s.merchantRepo.List(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	items := make([]*dto.MerchantResp, 0, len(merchants))
	for _, m := range merchants {
		items = append(items, &dto.MerchantResp{
			ID:        m.ID,
			Name:      m.Name,
			CreatedAt: m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return items, nil
}

func (s *merchantService) Update(ctx context.Context, id, tenantID uint64, name string) (*model.Merchant, error) {
	m, err := s.merchantRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrMerchantNotFound
	}
	m.Name = name
	m.UpdatedAt = time.Now()
	if err := s.merchantRepo.Update(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *merchantService) Delete(ctx context.Context, id, tenantID uint64) error {
	m, err := s.merchantRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return err
	}
	if m == nil {
		return ErrMerchantNotFound
	}
	return s.merchantRepo.Delete(ctx, id, tenantID)
}