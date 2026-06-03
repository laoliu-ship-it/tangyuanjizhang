package service

import (
	"context"
	"errors"
	"time"

	"fandianjizhang/server/config"
	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/repo"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrPlatformAdminNotFound   = errors.New("平台管理员不存在")
	ErrPlatformInvalidPassword = errors.New("密码错误")
)

type PlatformAdminService interface {
	Login(ctx context.Context, req dto.PlatformLoginReq) (*dto.PlatformLoginResp, error)
	GetDashboard(ctx context.Context) (*dto.PlatformDashboardResp, error)
	ListUsers(ctx context.Context, filter dto.PlatformUserFilter) (*dto.PlatformUserListResp, error)
	GetUserDetail(ctx context.Context, userID uint64) (*dto.PlatformUserDetailResp, error)
}

type platformAdminService struct {
	adminRepo repo.PlatformAdminRepo
	statsRepo repo.PlatformStatsRepo
	userRepo  repo.UserRepo
	cfg       *config.Config
}

func NewPlatformAdminService(
	adminRepo repo.PlatformAdminRepo,
	statsRepo repo.PlatformStatsRepo,
	userRepo repo.UserRepo,
	cfg *config.Config,
) PlatformAdminService {
	return &platformAdminService{
		adminRepo: adminRepo,
		statsRepo: statsRepo,
		userRepo:  userRepo,
		cfg:       cfg,
	}
}

func (s *platformAdminService) Login(ctx context.Context, req dto.PlatformLoginReq) (*dto.PlatformLoginResp, error) {
	admin, err := s.adminRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if admin == nil {
		return nil, ErrPlatformAdminNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrPlatformInvalidPassword
	}

	token, err := s.generateToken(admin.ID)
	if err != nil {
		return nil, err
	}

	return &dto.PlatformLoginResp{
		Token: token,
		ID:    admin.ID,
		Name:  admin.Name,
		Email: admin.Email,
	}, nil
}

func (s *platformAdminService) generateToken(adminID uint64) (string, error) {
	claims := PlatformClaims{
		AdminID: adminID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWT.Secret))
}

func (s *platformAdminService) GetDashboard(ctx context.Context) (*dto.PlatformDashboardResp, error) {
	users, _ := s.statsRepo.CountUsers(ctx)
	tenants, _ := s.statsRepo.CountTenants(ctx)
	transactions, _ := s.statsRepo.CountTransactions(ctx)
	return &dto.PlatformDashboardResp{
		TotalUsers:        users,
		TotalTenants:      tenants,
		TotalTransactions: transactions,
	}, nil
}

func (s *platformAdminService) ListUsers(ctx context.Context, filter dto.PlatformUserFilter) (*dto.PlatformUserListResp, error) {
	filter.Normalize()
	users, total, err := s.statsRepo.ListUsers(ctx, filter.Keyword, filter.Page, filter.PageSize)
	if err != nil {
		return nil, err
	}

	items := make([]*dto.PlatformUserItem, 0, len(users))
	for _, u := range users {
		items = append(items, &dto.PlatformUserItem{
			ID:        u.ID,
			Username:  u.Username,
			Email:     u.Email,
			CreatedAt: u.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &dto.PlatformUserListResp{
		Total: total,
		Page:  filter.Page,
		Size:  filter.PageSize,
		Items: items,
	}, nil
}

func (s *platformAdminService) GetUserDetail(ctx context.Context, userID uint64) (*dto.PlatformUserDetailResp, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	tenantCount, _ := s.statsRepo.CountTenantsByUserID(ctx, userID)
	txCount, _ := s.statsRepo.CountTransactionsByUserID(ctx, userID)
	mediaCount, _ := s.statsRepo.CountMediaByUserID(ctx, userID)

	return &dto.PlatformUserDetailResp{
		UserID:           user.ID,
		Username:         user.Username,
		Email:            user.Email,
		TenantCount:      tenantCount,
		TransactionCount: txCount,
		MediaCount:       mediaCount,
	}, nil
}
