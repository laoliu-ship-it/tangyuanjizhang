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
	ErrPlatformAdminNotFound   = errors.New("平台管理员不存在")
	ErrPlatformInvalidPassword = errors.New("密码错误")
)

type PlatformAdminService interface {
	Login(ctx context.Context, req dto.PlatformLoginReq) (*dto.PlatformLoginResp, error)
	GetDashboard(ctx context.Context) (*dto.PlatformDashboardResp, error)
	ListUsers(ctx context.Context, filter dto.PlatformUserFilter) (*dto.PlatformUserListResp, error)
	GetUserDetail(ctx context.Context, userID uint64) (*dto.PlatformUserDetailResp, error)
	// 配置管理
	GetAllConfigs(ctx context.Context) (*dto.PlatformConfigListResp, error)
	UpdateConfig(ctx context.Context, key, value string) (*dto.PlatformConfigItem, error)
	// 平台 LLM 配置
	GetPlatformLLMConfig(ctx context.Context) (*dto.PlatformLLMConfigResp, error)
	UpdatePlatformLLMConfig(ctx context.Context, req dto.UpdatePlatformLLMConfigReq) (*dto.PlatformLLMConfigResp, error)
	// 积分管理
	GetTenantCredits(ctx context.Context, tenantID uint64) (*dto.TenantCreditsDetail, error)
	AdjustTenantCredits(ctx context.Context, tenantID uint64, req dto.CreditsAdjustReq) (*dto.TenantCreditsDetail, error)
	SetTenantUnlimited(ctx context.Context, tenantID uint64, unlimited bool) (*dto.TenantCreditsDetail, error)
	ListUserTenantCredits(ctx context.Context, userID uint64) (*dto.TenantCreditsListResp, error)
}

type platformAdminService struct {
	adminRepo  repo.PlatformAdminRepo
	statsRepo  repo.PlatformStatsRepo
	userRepo   repo.UserRepo
	configRepo repo.PlatformConfigRepo
	tenantRepo repo.TenantRepo
	creditsSvc CreditsService
	cfg        *config.Config
}

func NewPlatformAdminService(
	adminRepo repo.PlatformAdminRepo,
	statsRepo repo.PlatformStatsRepo,
	userRepo repo.UserRepo,
	configRepo repo.PlatformConfigRepo,
	tenantRepo repo.TenantRepo,
	creditsSvc CreditsService,
	cfg *config.Config,
) PlatformAdminService {
	return &platformAdminService{
		adminRepo:  adminRepo,
		statsRepo:  statsRepo,
		userRepo:   userRepo,
		configRepo: configRepo,
		tenantRepo: tenantRepo,
		creditsSvc: creditsSvc,
		cfg:        cfg,
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

	userIDs := make([]uint64, 0, len(users))
	for _, u := range users {
		userIDs = append(userIDs, u.ID)
	}
	lastTxTimes, err := s.statsRepo.LastTransactionTimesByUserIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	items := make([]*dto.PlatformUserItem, 0, len(users))
	for _, u := range users {
		item := &dto.PlatformUserItem{
			ID:        u.ID,
			Username:  u.Username,
			Email:     u.Email,
			CreatedAt: u.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if u.LastLoginAt != nil {
			item.LastLoginAt = u.LastLoginAt.Format("2006-01-02 15:04:05")
		}
		if t, ok := lastTxTimes[u.ID]; ok {
			item.LastTransactionAt = t.Format("2006-01-02 15:04:05")
		}
		item.CreditBalance, item.CreditUnlimited, _ = s.sumUserCredits(ctx, u.ID)
		items = append(items, item)
	}
	return &dto.PlatformUserListResp{
		Total: total,
		Page:  filter.Page,
		Size:  filter.PageSize,
		Items: items,
	}, nil
}

// sumUserCredits 汇总某用户名下所有账本的剩余积分；只要有一个账本被设为无限积分白名单即视为无限
func (s *platformAdminService) sumUserCredits(ctx context.Context, userID uint64) (int64, bool, error) {
	tenants, err := s.tenantRepo.ListByUserID(ctx, userID)
	if err != nil {
		return 0, false, err
	}
	tenantIDs := make([]uint64, 0, len(tenants))
	for _, t := range tenants {
		tenantIDs = append(tenantIDs, t.ID)
	}
	creditsList, err := s.creditsSvc.ListTenantCredits(ctx, tenantIDs)
	if err != nil {
		return 0, false, err
	}
	var balance int64
	unlimited := false
	for _, c := range creditsList {
		balance += c.Balance
		if c.Unlimited {
			unlimited = true
		}
	}
	return balance, unlimited, nil
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

func (s *platformAdminService) GetAllConfigs(ctx context.Context) (*dto.PlatformConfigListResp, error) {
	configs, err := s.configRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]*dto.PlatformConfigItem, 0, len(configs))
	for _, c := range configs {
		items = append(items, &dto.PlatformConfigItem{
			Key:         c.ConfigKey,
			Value:       c.ConfigValue,
			Description: c.Description,
			UpdatedAt:   c.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &dto.PlatformConfigListResp{Items: items}, nil
}

func (s *platformAdminService) UpdateConfig(ctx context.Context, key, value string) (*dto.PlatformConfigItem, error) {
	cfg, err := s.configRepo.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, errors.New("配置项不存在")
	}

	cfg.ConfigValue = value
	if err := s.configRepo.Upsert(ctx, cfg); err != nil {
		return nil, err
	}

	return &dto.PlatformConfigItem{
		Key:         cfg.ConfigKey,
		Value:       cfg.ConfigValue,
		Description: cfg.Description,
		UpdatedAt:   cfg.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// ========== 平台 LLM 配置管理 ==========

func (s *platformAdminService) GetPlatformLLMConfig(ctx context.Context) (*dto.PlatformLLMConfigResp, error) {
	get := func(key string) string {
		if cfg, err := s.configRepo.GetByKey(ctx, key); err == nil && cfg != nil {
			return cfg.ConfigValue
		}
		return ""
	}
	return &dto.PlatformLLMConfigResp{
		Enabled:    get(model.ConfigKeyPlatformLLMEnabled) == "true",
		Provider:   get(model.ConfigKeyPlatformLLMProvider),
		BaseURL:    get(model.ConfigKeyPlatformLLMBaseURL),
		APIKeyMask: maskLLMAPIKey(get(model.ConfigKeyPlatformLLMAPIKey)),
		Model:      get(model.ConfigKeyPlatformLLMModel),
		Mode:       get(model.ConfigKeyPlatformLLMMode),
	}, nil
}

func (s *platformAdminService) UpdatePlatformLLMConfig(ctx context.Context, req dto.UpdatePlatformLLMConfigReq) (*dto.PlatformLLMConfigResp, error) {
	set := func(key, value string) error {
		cfg, err := s.configRepo.GetByKey(ctx, key)
		if err != nil {
			return err
		}
		if cfg == nil {
			cfg = &model.PlatformConfig{ConfigKey: key}
		}
		cfg.ConfigValue = value
		return s.configRepo.Upsert(ctx, cfg)
	}
	enabled := "false"
	if req.Enabled {
		enabled = "true"
	}
	if err := set(model.ConfigKeyPlatformLLMEnabled, enabled); err != nil {
		return nil, err
	}
	if req.Provider != "" {
		if err := set(model.ConfigKeyPlatformLLMProvider, req.Provider); err != nil {
			return nil, err
		}
	}
	if err := set(model.ConfigKeyPlatformLLMBaseURL, req.BaseURL); err != nil {
		return nil, err
	}
	// 空 APIKey 表示不修改
	if req.APIKey != "" {
		if err := set(model.ConfigKeyPlatformLLMAPIKey, req.APIKey); err != nil {
			return nil, err
		}
	}
	if req.Model != "" {
		if err := set(model.ConfigKeyPlatformLLMModel, req.Model); err != nil {
			return nil, err
		}
	}
	if req.Mode != "" {
		if err := set(model.ConfigKeyPlatformLLMMode, req.Mode); err != nil {
			return nil, err
		}
	}
	return s.GetPlatformLLMConfig(ctx)
}

func maskLLMAPIKey(key string) string {
	if len(key) == 0 {
		return ""
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

// ========== 积分管理 ==========

func (s *platformAdminService) GetTenantCredits(ctx context.Context, tenantID uint64) (*dto.TenantCreditsDetail, error) {
	detail, err := s.creditsSvc.GetTenantCredits(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if t, err := s.tenantRepo.GetByID(ctx, tenantID); err == nil && t != nil {
		detail.TenantName = t.Name
	}
	return detail, nil
}

func (s *platformAdminService) AdjustTenantCredits(ctx context.Context, tenantID uint64, req dto.CreditsAdjustReq) (*dto.TenantCreditsDetail, error) {
	note := req.Note
	if note == "" {
		note = "管理员调整"
	}
	if err := s.creditsSvc.Gift(ctx, tenantID, req.Delta, "admin_adjust", note); err != nil {
		return nil, err
	}
	return s.GetTenantCredits(ctx, tenantID)
}

func (s *platformAdminService) SetTenantUnlimited(ctx context.Context, tenantID uint64, unlimited bool) (*dto.TenantCreditsDetail, error) {
	if err := s.creditsSvc.SetUnlimited(ctx, tenantID, unlimited); err != nil {
		return nil, err
	}
	return s.GetTenantCredits(ctx, tenantID)
}

func (s *platformAdminService) ListUserTenantCredits(ctx context.Context, userID uint64) (*dto.TenantCreditsListResp, error) {
	tenants, err := s.tenantRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	tenantIDs := make([]uint64, 0, len(tenants))
	for _, t := range tenants {
		tenantIDs = append(tenantIDs, t.ID)
	}
	creditsList, err := s.creditsSvc.ListTenantCredits(ctx, tenantIDs)
	if err != nil {
		return nil, err
	}
	// 填充租户名
	nameByID := make(map[uint64]string, len(tenants))
	for _, t := range tenants {
		nameByID[t.ID] = t.Name
	}
	for _, c := range creditsList {
		c.TenantName = nameByID[c.TenantID]
	}
	return &dto.TenantCreditsListResp{Items: creditsList}, nil
}
