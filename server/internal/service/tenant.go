package service

import (
	"context"
	"errors"
	"time"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/repo"
)

var (
	ErrTenantNotFound    = errors.New("租户不存在")
	ErrMemberNotFound    = errors.New("成员不存在")
	ErrNotTenantMember   = errors.New("不是租户成员")
	ErrCannotRemoveOwner = errors.New("不能移除租户所有者")
	ErrPermissionDenied  = errors.New("权限不足")
)

type TenantService interface {
	Create(ctx context.Context, userID uint64, req dto.CreateTenantReq) (*model.Tenant, error)
	Update(ctx context.Context, tenantID, userID uint64, req dto.UpdateTenantReq) (*model.Tenant, error)
	List(ctx context.Context, userID uint64) ([]*model.Tenant, error)
	InviteMember(ctx context.Context, tenantID uint64, req dto.InviteMemberReq) error
	RemoveMember(ctx context.Context, tenantID, targetUserID uint64) error
	UpdateMemberRole(ctx context.Context, tenantID, targetUserID uint64, role string) error
	ListMembers(ctx context.Context, tenantID uint64) ([]*dto.MemberResp, error)
	GetSettings(ctx context.Context, tenantID uint64) (*dto.TenantSettingsResp, error)
	UpdateSettings(ctx context.Context, tenantID uint64, req dto.UpdateTenantSettingsReq) (*dto.TenantSettingsResp, error)
}

type tenantService struct {
	tenantRepo       repo.TenantRepo
	tenantMemberRepo repo.TenantMemberRepo
	userRepo         repo.UserRepo
	categoryRepo     repo.CategoryRepo
	settingsRepo     repo.TenantSettingsRepo
}

func NewTenantService(
	tenantRepo repo.TenantRepo,
	tenantMemberRepo repo.TenantMemberRepo,
	userRepo repo.UserRepo,
	categoryRepo repo.CategoryRepo,
	settingsRepo repo.TenantSettingsRepo,
) TenantService {
	return &tenantService{
		tenantRepo:       tenantRepo,
		tenantMemberRepo: tenantMemberRepo,
		userRepo:         userRepo,
		categoryRepo:     categoryRepo,
		settingsRepo:     settingsRepo,
	}
}

func (s *tenantService) Create(ctx context.Context, userID uint64, req dto.CreateTenantReq) (*model.Tenant, error) {
	tenant := &model.Tenant{
		Name:    req.Name,
		OwnerID: userID,
	}
	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, err
	}

	member := &model.TenantMember{
		TenantID: tenant.ID,
		UserID:   userID,
		Role:     "admin",
		JoinedAt: time.Now(),
	}
	if err := s.tenantMemberRepo.Add(ctx, member); err != nil {
		return nil, err
	}

	// 创建默认分类
	createDefaultCategories(ctx, s.categoryRepo, tenant.ID)

	return tenant, nil
}

func (s *tenantService) Update(ctx context.Context, tenantID, userID uint64, req dto.UpdateTenantReq) (*model.Tenant, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFound
	}
	if tenant.OwnerID != userID {
		return nil, ErrPermissionDenied
	}
	tenant.Name = req.Name
	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, err
	}
	return tenant, nil
}

func (s *tenantService) List(ctx context.Context, userID uint64) ([]*model.Tenant, error) {
	return s.tenantRepo.ListByUserID(ctx, userID)
}

func (s *tenantService) InviteMember(ctx context.Context, tenantID uint64, req dto.InviteMemberReq) error {
	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	// 检查是否已经是成员
	role, err := s.tenantMemberRepo.GetRole(ctx, tenantID, user.ID)
	if err != nil {
		return err
	}
	if role != "" {
		return errors.New("用户已是租户成员")
	}

	member := &model.TenantMember{
		TenantID: tenantID,
		UserID:   user.ID,
		Role:     req.Role,
		JoinedAt: time.Now(),
	}
	return s.tenantMemberRepo.Add(ctx, member)
}

func (s *tenantService) RemoveMember(ctx context.Context, tenantID, targetUserID uint64) error {
	// 检查租户是否存在
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return err
	}
	if tenant == nil {
		return ErrTenantNotFound
	}

	// 不能移除所有者
	if tenant.OwnerID == targetUserID {
		return ErrCannotRemoveOwner
	}

	return s.tenantMemberRepo.Remove(ctx, tenantID, targetUserID)
}

func (s *tenantService) UpdateMemberRole(ctx context.Context, tenantID, targetUserID uint64, role string) error {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return err
	}
	if tenant == nil {
		return ErrTenantNotFound
	}
	if tenant.OwnerID == targetUserID {
		return errors.New("不能修改所有者的角色")
	}

	existing, err := s.tenantMemberRepo.GetRole(ctx, tenantID, targetUserID)
	if err != nil {
		return err
	}
	if existing == "" {
		return ErrMemberNotFound
	}

	return s.tenantMemberRepo.UpdateRole(ctx, tenantID, targetUserID, role)
}

func (s *tenantService) ListMembers(ctx context.Context, tenantID uint64) ([]*dto.MemberResp, error) {
	members, err := s.tenantMemberRepo.ListMembers(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	result := make([]*dto.MemberResp, 0, len(members))
	for _, m := range members {
		resp := &dto.MemberResp{
			UserID:   m.UserID,
			Role:     m.Role,
			JoinedAt: m.JoinedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		if m.User != nil {
			resp.Username = m.User.Username
			resp.Email = m.User.Email
		}
		result = append(result, resp)
	}
	return result, nil
}

func (s *tenantService) GetSettings(ctx context.Context, tenantID uint64) (*dto.TenantSettingsResp, error) {
	existing, err := s.settingsRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		// 默认值：支出必须上传图片
		return &dto.TenantSettingsResp{RequireExpenseImage: true}, nil
	}
	return &dto.TenantSettingsResp{RequireExpenseImage: existing.RequireExpenseImage}, nil
}

func (s *tenantService) UpdateSettings(ctx context.Context, tenantID uint64, req dto.UpdateTenantSettingsReq) (*dto.TenantSettingsResp, error) {
	existing, err := s.settingsRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		existing = &model.TenantSettings{TenantID: tenantID}
	}
	existing.RequireExpenseImage = req.RequireExpenseImage
	if err := s.settingsRepo.Save(ctx, existing); err != nil {
		return nil, err
	}
	return &dto.TenantSettingsResp{RequireExpenseImage: existing.RequireExpenseImage}, nil
}
