package service

import (
	"context"
	"errors"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/repo"
)

var ErrCategoryNotFound = errors.New("分类不存在")

type CategoryService interface {
	Create(ctx context.Context, tenantID uint64, req dto.CreateCategoryReq) (*model.Category, error)
	Update(ctx context.Context, id, tenantID uint64, req dto.UpdateCategoryReq) (*model.Category, error)
	Delete(ctx context.Context, id, tenantID uint64) error
	List(ctx context.Context, tenantID uint64) ([]*model.Category, error)
}

type categoryService struct {
	categoryRepo repo.CategoryRepo
}

func NewCategoryService(categoryRepo repo.CategoryRepo) CategoryService {
	return &categoryService{categoryRepo: categoryRepo}
}

func (s *categoryService) Create(ctx context.Context, tenantID uint64, req dto.CreateCategoryReq) (*model.Category, error) {
	cat := &model.Category{
		TenantID: tenantID,
		Name:     req.Name,
		Type:     req.Type,
		Icon:     req.Icon,
	}
	if err := s.categoryRepo.Create(ctx, cat); err != nil {
		return nil, err
	}
	return cat, nil
}

func (s *categoryService) Update(ctx context.Context, id, tenantID uint64, req dto.UpdateCategoryReq) (*model.Category, error) {
	cat, err := s.categoryRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	if cat == nil {
		return nil, ErrCategoryNotFound
	}
	cat.Name = req.Name
	cat.Type = req.Type
	cat.Icon = req.Icon
	if err := s.categoryRepo.Update(ctx, cat); err != nil {
		return nil, err
	}
	return cat, nil
}

func (s *categoryService) Delete(ctx context.Context, id, tenantID uint64) error {
	cat, err := s.categoryRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return err
	}
	if cat == nil {
		return ErrCategoryNotFound
	}
	return s.categoryRepo.Delete(ctx, id, tenantID)
}

func (s *categoryService) List(ctx context.Context, tenantID uint64) ([]*model.Category, error) {
	return s.categoryRepo.List(ctx, tenantID)
}
