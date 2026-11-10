package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
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

	ErrWeChatNotConfigured = errors.New("微信登录暂未开通")
	ErrWeChatCodeInvalid   = errors.New("微信登录凭证无效或已过期")
	ErrWeChatBoundToOther  = errors.New("该微信已绑定其他账号")
	ErrAccountWeChatBound  = errors.New("该账号已绑定其他微信")
	ErrWeChatBindToken     = errors.New("微信登录会话已失效，请重新登录")
)

const (
	weChatBindScope       = "wechat_bind"
	weChatCode2SessionURL = "https://api.weixin.qq.com/sns/jscode2session"
	weChatBindTokenTTL    = 10 * time.Minute
)

type Claims struct {
	UserID       uint64 `json:"user_id"`
	Demo         bool   `json:"demo,omitempty"`
	DemoTenantID uint64 `json:"demo_tenant_id,omitempty"`
	jwt.RegisteredClaims
}

var ErrDemoDisabled = errors.New("体验账号未开启")

type PlatformClaims struct {
	AdminID uint64 `json:"admin_id"`
	jwt.RegisteredClaims
}

// WeChatBindClaims 首次微信登录时签发的短期凭证，仅用于关联/创建账号。
// 使用由 JWT Secret 派生的独立密钥签名，不能当作登录 token 使用。
type WeChatBindClaims struct {
	Scope   string `json:"scope"`
	OpenID  string `json:"openid"`
	UnionID string `json:"unionid,omitempty"`
	jwt.RegisteredClaims
}

type AuthService interface {
	Register(ctx context.Context, req dto.RegisterReq) (*dto.LoginResp, error)
	Login(ctx context.Context, req dto.LoginReq) (*dto.LoginResp, error)
	WeChatLogin(ctx context.Context, req dto.WeChatLoginReq) (*dto.WeChatLoginResp, error)
	WeChatBind(ctx context.Context, req dto.WeChatBindReq) (*dto.LoginResp, error)
	WeChatRegister(ctx context.Context, req dto.WeChatRegisterReq) (*dto.LoginResp, error)
	DemoLogin(ctx context.Context) (*dto.LoginResp, error)
	DemoEnabled() bool
	DemoTTLMinutes() int
}

type authService struct {
	userRepo         repo.UserRepo
	tenantRepo       repo.TenantRepo
	tenantMemberRepo repo.TenantMemberRepo
	categoryRepo     repo.CategoryRepo
	creditsSvc       CreditsService
	configRepo       repo.PlatformConfigRepo
	demoRepo         repo.DemoRepo
	cfg              *config.Config
	wechatClient     *http.Client
}

func NewAuthService(
	userRepo repo.UserRepo,
	tenantRepo repo.TenantRepo,
	tenantMemberRepo repo.TenantMemberRepo,
	categoryRepo repo.CategoryRepo,
	creditsSvc CreditsService,
	configRepo repo.PlatformConfigRepo,
	demoRepo repo.DemoRepo,
	cfg *config.Config,
) AuthService {
	return &authService{
		userRepo:         userRepo,
		tenantRepo:       tenantRepo,
		tenantMemberRepo: tenantMemberRepo,
		categoryRepo:     categoryRepo,
		creditsSvc:       creditsSvc,
		configRepo:       configRepo,
		demoRepo:         demoRepo,
		cfg:              cfg,
		wechatClient:     &http.Client{Timeout: 10 * time.Second},
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
		RegisterIP:   req.RegisterIP,
	}
	if err := s.createUserWithTenant(ctx, user, req.BookType, req.RegisterIP); err != nil {
		return nil, err
	}

	return s.buildLoginResp(ctx, user)
}

// createUserWithTenant 创建用户并初始化默认账本、管理员成员关系、默认分类与新用户积分。
func (s *authService) createUserWithTenant(ctx context.Context, user *model.User, bookType, registerIP string) error {
	if err := s.userRepo.Create(ctx, user); err != nil {
		return err
	}

	tenant := &model.Tenant{
		Name:     user.Username + "的记账本",
		OwnerID:  user.ID,
		BookType: bookType,
	}
	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		return err
	}

	member := &model.TenantMember{
		TenantID: tenant.ID,
		UserID:   user.ID,
		Role:     "admin",
		JoinedAt: time.Now(),
	}
	if err := s.tenantMemberRepo.Add(ctx, member); err != nil {
		return err
	}

	createDefaultCategories(ctx, s.categoryRepo, tenant.ID, bookType)

	// 新用户赠送积分（同 IP 超限则跳过）
	if s.shouldGiftCredits(ctx, registerIP) {
		s.giftNewUserCredits(ctx, tenant.ID)
	} else {
		log.Printf("[credits] register: IP %s 超过赠送限额，跳过赠送", registerIP)
	}
	return nil
}

// shouldGiftCredits 判断该 IP 是否还在赠送限额内，查询失败时放行（不因查询出错而阻止赠送）
func (s *authService) shouldGiftCredits(ctx context.Context, ip string) bool {
	if ip == "" {
		return true
	}
	limitCfg, err := s.configRepo.GetByKey(ctx, model.ConfigKeyMaxAccountsPerIP)
	if err != nil || limitCfg == nil {
		return true
	}
	limit, err := strconv.ParseInt(limitCfg.ConfigValue, 10, 64)
	if err != nil || limit <= 0 {
		return true
	}
	count, err := s.userRepo.CountByRegisterIP(ctx, ip)
	if err != nil {
		return true
	}
	return count <= limit
}

// giftNewUserCredits 从平台配置读取新用户赠送积分数并赠送，失败只记日志不影响注册
func (s *authService) giftNewUserCredits(ctx context.Context, tenantID uint64) {
	cfg, err := s.configRepo.GetByKey(ctx, model.ConfigKeyCreditsNewUserGift)
	if err != nil || cfg == nil {
		log.Printf("[credits] giftNewUser: config not found, skipping")
		return
	}
	amount, err := strconv.ParseInt(cfg.ConfigValue, 10, 64)
	if err != nil || amount <= 0 {
		return
	}
	if err := s.creditsSvc.Gift(ctx, tenantID, amount, "gift", "新用户注册赠送"); err != nil {
		log.Printf("[credits] giftNewUser: tenant=%d error: %v", tenantID, err)
	}
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

	return s.buildLoginResp(ctx, user)
}

// buildLoginResp 生成登录态：JWT、最近登录时间与账本列表。
func (s *authService) buildLoginResp(ctx context.Context, user *model.User) (*dto.LoginResp, error) {
	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, err
	}

	// 最近登录时间仅用于活跃度统计，失败不阻断登录
	if err := s.userRepo.UpdateLastLoginAt(ctx, user.ID, time.Now()); err != nil {
		log.Printf("[auth] UpdateLastLoginAt: user=%d error: %v", user.ID, err)
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

// WeChatLogin 小程序微信登录：已绑定则直接登录，未绑定返回短期 bind_token 供前端选择关联或注册。
func (s *authService) WeChatLogin(ctx context.Context, req dto.WeChatLoginReq) (*dto.WeChatLoginResp, error) {
	session, err := s.code2Session(ctx, req.Code)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByWeChatOpenID(ctx, session.OpenID)
	if err != nil {
		return nil, err
	}
	if user != nil {
		auth, err := s.buildLoginResp(ctx, user)
		if err != nil {
			return nil, err
		}
		return &dto.WeChatLoginResp{Bound: true, Auth: auth}, nil
	}

	bindToken, err := s.generateWeChatBindToken(session.OpenID, session.UnionID)
	if err != nil {
		return nil, err
	}
	return &dto.WeChatLoginResp{Bound: false, BindToken: bindToken}, nil
}

// WeChatBind 将当前微信关联到已有邮箱账号（校验邮箱密码后建立绑定）。
func (s *authService) WeChatBind(ctx context.Context, req dto.WeChatBindReq) (*dto.LoginResp, error) {
	openID, unionID, err := s.parseWeChatBindToken(req.BindToken)
	if err != nil {
		return nil, err
	}

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

	if user.WeChatOpenID != nil && *user.WeChatOpenID != "" && *user.WeChatOpenID != openID {
		return nil, ErrAccountWeChatBound
	}

	boundUser, err := s.userRepo.GetByWeChatOpenID(ctx, openID)
	if err != nil {
		return nil, err
	}
	if boundUser != nil && boundUser.ID != user.ID {
		return nil, ErrWeChatBoundToOther
	}

	if user.WeChatOpenID == nil || *user.WeChatOpenID != openID {
		if err := s.userRepo.BindWeChat(ctx, user.ID, openID, unionID); err != nil {
			return nil, err
		}
	}

	return s.buildLoginResp(ctx, user)
}

// WeChatRegister 首次微信登录时直接创建新账号（邮箱为占位地址，密码随机不可登录）。
func (s *authService) WeChatRegister(ctx context.Context, req dto.WeChatRegisterReq) (*dto.LoginResp, error) {
	openID, unionID, err := s.parseWeChatBindToken(req.BindToken)
	if err != nil {
		return nil, err
	}

	// 幂等：重复提交/并发时若已绑定，直接返回登录态
	if user, err := s.userRepo.GetByWeChatOpenID(ctx, openID); err != nil {
		return nil, err
	} else if user != nil {
		return s.buildLoginResp(ctx, user)
	}

	existingUser, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrUsernameExists
	}

	randomPwd, err := randomHex(32)
	if err != nil {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(randomPwd), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	wechatEmail := fmt.Sprintf("wx_%s@weixin.local", openID)
	user := &model.User{
		Username:     req.Username,
		Email:        wechatEmail,
		PasswordHash: string(hash),
		RegisterIP:   req.RegisterIP,
		WeChatOpenID: &openID,
	}
	if unionID != "" {
		user.WeChatUnionID = &unionID
	}

	if err := s.createUserWithTenant(ctx, user, req.BookType, req.RegisterIP); err != nil {
		return nil, err
	}

	return s.buildLoginResp(ctx, user)
}

type weChatSessionResp struct {
	OpenID     string `json:"openid"`
	UnionID    string `json:"unionid"`
	SessionKey string `json:"session_key"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

// code2Session 调用微信 jscode2session 换取 openid/unionid。
func (s *authService) code2Session(ctx context.Context, code string) (*weChatSessionResp, error) {
	if s.cfg.WeChat.AppID == "" || s.cfg.WeChat.Secret == "" {
		return nil, ErrWeChatNotConfigured
	}

	query := url.Values{}
	query.Set("appid", s.cfg.WeChat.AppID)
	query.Set("secret", s.cfg.WeChat.Secret)
	query.Set("js_code", code)
	query.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, weChatCode2SessionURL+"?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.wechatClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求微信接口失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("微信接口返回 HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("读取微信接口响应失败: %w", err)
	}

	var out weChatSessionResp
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("解析微信接口响应失败: %w", err)
	}

	if out.ErrCode != 0 {
		log.Printf("[wechat] jscode2session error: code=%d msg=%s", out.ErrCode, out.ErrMsg)
		switch out.ErrCode {
		case 40029, 40163, 45011: // code 无效 / 已使用 / 频率限制
			return nil, ErrWeChatCodeInvalid
		default:
			return nil, fmt.Errorf("微信登录失败: %s", out.ErrMsg)
		}
	}
	if out.OpenID == "" {
		return nil, ErrWeChatCodeInvalid
	}
	return &out, nil
}

func (s *authService) generateWeChatBindToken(openID, unionID string) (string, error) {
	claims := WeChatBindClaims{
		Scope:   weChatBindScope,
		OpenID:  openID,
		UnionID: unionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(weChatBindTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.wechatBindSecret())
}

func (s *authService) parseWeChatBindToken(tokenStr string) (string, string, error) {
	claims := &WeChatBindClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return s.wechatBindSecret(), nil
	})
	if err != nil || !token.Valid || claims.Scope != weChatBindScope || claims.OpenID == "" {
		return "", "", ErrWeChatBindToken
	}
	return claims.OpenID, claims.UnionID, nil
}

// wechatBindSecret 由 JWT Secret 派生独立密钥，确保 bind token 不能用于访问业务接口。
func (s *authService) wechatBindSecret() []byte {
	mac := hmac.New(sha256.New, []byte(s.cfg.JWT.Secret))
	_, _ = mac.Write([]byte(weChatBindScope))
	return mac.Sum(nil)
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (s *authService) generateToken(userID uint64) (string, error) {
	return s.signToken(userID, false, 0, 7*24*time.Hour)
}

func (s *authService) signToken(userID uint64, demo bool, demoTenantID uint64, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID:       userID,
		Demo:         demo,
		DemoTenantID: demoTenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWT.Secret))
}

func (s *authService) DemoEnabled() bool { return s.cfg.Billing.DemoLoginEnabled }

func (s *authService) DemoTTLMinutes() int {
	if s.cfg.Billing.DemoTTLMinutes <= 0 {
		return 30
	}
	return s.cfg.Billing.DemoTTLMinutes
}

// DemoLogin 返回固定演示账号的短期 token；不创建新用户、不赠送积分。
func (s *authService) DemoLogin(ctx context.Context) (*dto.LoginResp, error) {
	if !s.cfg.Billing.DemoLoginEnabled {
		return nil, ErrDemoDisabled
	}
	// 只信任系统演示关联记录，绝不按 username 查找/复用普通用户。
	if s.demoRepo == nil {
		return nil, errors.New("演示账号尚未初始化")
	}
	account, err := s.demoRepo.GetAccount(ctx)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, errors.New("演示账号尚未初始化")
	}
	user, err := s.userRepo.GetByID(ctx, account.UserID)
	if err != nil {
		return nil, err
	}
	tenant, err := s.tenantRepo.GetByID(ctx, account.TenantID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.IsDemo || tenant == nil || !tenant.IsDemo || tenant.OwnerID != user.ID {
		return nil, errors.New("演示关联记录异常，拒绝登录")
	}
	ttl := time.Duration(s.cfg.Billing.DemoTTLMinutes) * time.Minute
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	token, err := s.signToken(user.ID, true, tenant.ID, ttl)
	if err != nil {
		return nil, err
	}
	// 只返回唯一演示账本，即使演示用户被错误加入其他账本也绝不暴露。
	return &dto.LoginResp{
		Token: token, UserID: user.ID, Username: user.Username, Email: user.Email,
		Tenants: []dto.TenantSummary{{ID: tenant.ID, Name: tenant.Name}}, Demo: true,
	}, nil
}
