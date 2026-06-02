package service

import (
	"context"
	"errors"
	"time"

	"fandianjizhang/server/config"
	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/repo"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound       = errors.New("用户不存在")
	ErrInvalidPassword    = errors.New("密码错误")
	ErrEmailAlreadyExists = errors.New("邮箱已被注册")
	ErrUsernameExists     = errors.New("用户名已被使用")
)

type Claims struct {
	UserID uint64 `json:"user_id"`
	jwt.RegisteredClaims
}

type AuthService interface {
	Register(ctx context.Context, req dto.RegisterReq) (*dto.LoginResp, error)
	Login(ctx context.Context, req dto.LoginReq) (*dto.LoginResp, error)
}

type authService struct {
	userRepo         repo.UserRepo
	tenantRepo       repo.TenantRepo
	tenantMemberRepo repo.TenantMemberRepo
	categoryRepo     repo.CategoryRepo
	cfg              *config.Config
}

func NewAuthService(
	userRepo repo.UserRepo,
	tenantRepo repo.TenantRepo,
	tenantMemberRepo repo.TenantMemberRepo,
	categoryRepo repo.CategoryRepo,
	cfg *config.Config,
) AuthService {
	return &authService{
		userRepo:         userRepo,
		tenantRepo:       tenantRepo,
		tenantMemberRepo: tenantMemberRepo,
		categoryRepo:     categoryRepo,
		cfg:              cfg,
	}
}

func (s *authService) Register(ctx context.Context, req dto.RegisterReq) (*dto.LoginResp, error) {
	// 检查邮箱是否已存在
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrEmailAlreadyExists
	}

	// 检查用户名是否已存在
	existingUser, err = s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrUsernameExists
	}

	// 加密密码
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// 注册时自动创建默认租户
	tenant := &model.Tenant{
		Name:    req.Username + "的记账本",
		OwnerID: user.ID,
	}
	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, err
	}

	// 将用户加入租户为 admin
	member := &model.TenantMember{
		TenantID: tenant.ID,
		UserID:   user.ID,
		Role:     "admin",
		JoinedAt: time.Now(),
	}
	if err := s.tenantMemberRepo.Add(ctx, member); err != nil {
		return nil, err
	}

	// 创建默认分类
	createDefaultCategories(ctx, s.categoryRepo, tenant.ID)

	// 生成 JWT
	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &dto.LoginResp{
		Token:    token,
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Tenants:  []dto.TenantSummary{{ID: tenant.ID, Name: tenant.Name}},
	}, nil
}

func (s *authService) Login(ctx context.Context, req dto.LoginReq) (*dto.LoginResp, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidPassword
	}

	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, err
	}

	tenants, err := s.tenantRepo.ListByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	tenantSummaries := make([]dto.TenantSummary, 0, len(tenants))
	for _, t := range tenants {
		tenantSummaries = append(tenantSummaries, dto.TenantSummary{ID: t.ID, Name: t.Name})
	}

	return &dto.LoginResp{
		Token:    token,
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Tenants:  tenantSummaries,
	}, nil
}

func (s *authService) generateToken(userID uint64) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWT.Secret))
}
