package service

import (
	"context"
	"strconv"
	"time"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/repo"
)

// defaultCreditsValidityDays 积分默认有效期（天），platform_configs 未配置时兜底
const defaultCreditsValidityDays = 365

// ErrInsufficientCredits 积分不足错误（透传给 handler 返回 402）
var ErrInsufficientCredits = repo.ErrInsufficientCredits

type CreditsService interface {
	// GetBalance 查询租户积分余额及最近流水
	GetBalance(ctx context.Context, tenantID uint64) (*dto.CreditsResp, error)
	// CheckAndDeduct 按 action 读取单价并扣费；单价为0则跳过
	CheckAndDeduct(ctx context.Context, tenantID uint64, action, refType string, refID uint64, note string) error
	// Gift 发放积分（注册赠送 / 管理员充值均走此方法），生成按配置有效期到期的批次；负数为管理员扣减
	Gift(ctx context.Context, tenantID uint64, delta int64, action, note string) error
	// ExpireDueCredits 清理所有租户的到期积分并写 expire 流水，返回作废总数（定时任务用）
	ExpireDueCredits(ctx context.Context) (int64, error)
	// SetUnlimited 设置/取消无限制白名单
	SetUnlimited(ctx context.Context, tenantID uint64, unlimited bool) error
	// GetLogs 分页查询流水；amountType: "income"/"expense"/""
	GetLogs(ctx context.Context, tenantID uint64, page, size int, amountType string) (*dto.CreditLogsPageResp, error)
	// GetTenantCredits 查询某租户积分（平台管理员用）
	GetTenantCredits(ctx context.Context, tenantID uint64) (*dto.TenantCreditsDetail, error)
	// ListTenantCredits 批量查询多个租户积分（平台管理员用）
	ListTenantCredits(ctx context.Context, tenantIDs []uint64) ([]*dto.TenantCreditsDetail, error)
}

type creditsService struct {
	creditsRepo repo.TenantCreditsRepo
	configRepo  repo.PlatformConfigRepo
}

func NewCreditsService(creditsRepo repo.TenantCreditsRepo, configRepo repo.PlatformConfigRepo) CreditsService {
	return &creditsService{creditsRepo: creditsRepo, configRepo: configRepo}
}

func (s *creditsService) GetBalance(ctx context.Context, tenantID uint64) (*dto.CreditsResp, error) {
	c, err := s.creditsRepo.GetOrCreate(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	logs, err := s.creditsRepo.ListLogs(ctx, tenantID, 20)
	if err != nil {
		return nil, err
	}
	items := make([]*dto.CreditLogItem, 0, len(logs))
	for _, l := range logs {
		items = append(items, creditLogToDTO(l))
	}
	lots, err := s.creditsRepo.ListActiveLots(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	lotItems := make([]*dto.CreditLotItem, 0, len(lots))
	for _, lot := range lots {
		lotItems = append(lotItems, &dto.CreditLotItem{
			Remaining: lot.Remaining,
			ExpiresAt: lot.ExpiresAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &dto.CreditsResp{
		TenantID:  tenantID,
		Balance:   c.Balance,
		Unlimited: c.Unlimited,
		Lots:      lotItems,
		Logs:      items,
	}, nil
}

func (s *creditsService) GetLogs(ctx context.Context, tenantID uint64, page, size int, amountType string) (*dto.CreditLogsPageResp, error) {
	logs, total, err := s.creditsRepo.ListLogsPaged(ctx, tenantID, page, size, amountType)
	if err != nil {
		return nil, err
	}
	items := make([]*dto.CreditLogItem, 0, len(logs))
	for _, l := range logs {
		items = append(items, creditLogToDTO(l))
	}
	return &dto.CreditLogsPageResp{Total: total, Page: page, Size: size, Logs: items}, nil
}

func (s *creditsService) CheckAndDeduct(ctx context.Context, tenantID uint64, action, refType string, refID uint64, note string) error {
	cost := s.getActionCost(ctx, action)
	if cost == 0 {
		return nil
	}
	return s.creditsRepo.Deduct(ctx, tenantID, cost, action, refType, refID, note)
}

func (s *creditsService) Gift(ctx context.Context, tenantID uint64, delta int64, action, note string) error {
	expiresAt := time.Now().AddDate(0, 0, s.getValidityDays(ctx))
	return s.creditsRepo.Gift(ctx, tenantID, delta, expiresAt, action, note)
}

func (s *creditsService) ExpireDueCredits(ctx context.Context) (int64, error) {
	return s.creditsRepo.ExpireDueLots(ctx)
}

// getValidityDays 从 platform_configs 读取积分有效期（天），未配置或非法时返回默认值
func (s *creditsService) getValidityDays(ctx context.Context) int {
	cfg, err := s.configRepo.GetByKey(ctx, model.ConfigKeyCreditsValidityDays)
	if err != nil || cfg == nil {
		return defaultCreditsValidityDays
	}
	v, err := strconv.Atoi(cfg.ConfigValue)
	if err != nil || v <= 0 {
		return defaultCreditsValidityDays
	}
	return v
}

func (s *creditsService) SetUnlimited(ctx context.Context, tenantID uint64, unlimited bool) error {
	return s.creditsRepo.SetUnlimited(ctx, tenantID, unlimited)
}

func (s *creditsService) GetTenantCredits(ctx context.Context, tenantID uint64) (*dto.TenantCreditsDetail, error) {
	c, err := s.creditsRepo.GetOrCreate(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	logs, err := s.creditsRepo.ListLogs(ctx, tenantID, 50)
	if err != nil {
		return nil, err
	}
	items := make([]*dto.CreditLogItem, 0, len(logs))
	for _, l := range logs {
		items = append(items, creditLogToDTO(l))
	}
	return &dto.TenantCreditsDetail{
		TenantID:  tenantID,
		Balance:   c.Balance,
		Unlimited: c.Unlimited,
		Logs:      items,
	}, nil
}

func (s *creditsService) ListTenantCredits(ctx context.Context, tenantIDs []uint64) ([]*dto.TenantCreditsDetail, error) {
	list, err := s.creditsRepo.ListByTenantIDs(ctx, tenantIDs)
	if err != nil {
		return nil, err
	}
	// index by tenantID for O(1) lookup
	byID := make(map[uint64]*model.TenantCredits, len(list))
	for _, c := range list {
		byID[c.TenantID] = c
	}
	result := make([]*dto.TenantCreditsDetail, 0, len(tenantIDs))
	for _, tid := range tenantIDs {
		if c, ok := byID[tid]; ok {
			result = append(result, &dto.TenantCreditsDetail{
				TenantID:  tid,
				Balance:   c.Balance,
				Unlimited: c.Unlimited,
			})
		} else {
			result = append(result, &dto.TenantCreditsDetail{TenantID: tid})
		}
	}
	return result, nil
}

// getActionCost 从 platform_configs 读取 action 对应单价，读取失败或未配置返回 0
func (s *creditsService) getActionCost(ctx context.Context, action string) int64 {
	key := actionToConfigKey(action)
	if key == "" {
		return 0
	}
	cfg, err := s.configRepo.GetByKey(ctx, key)
	if err != nil || cfg == nil {
		return 0
	}
	v, err := strconv.ParseInt(cfg.ConfigValue, 10, 64)
	if err != nil || v < 0 {
		return 0
	}
	return v
}

func actionToConfigKey(action string) string {
	switch action {
	case "upload":
		return model.ConfigKeyCreditsPerUpload
	case "ocr":
		return model.ConfigKeyCreditsPerOCR
	case "orc2bill":
		return model.ConfigKeyCreditsPerOrc2bill
	case "storage":
		return model.ConfigKeyCreditsPerGBPerDay
	default:
		return ""
	}
}

func creditLogToDTO(l *model.CreditLog) *dto.CreditLogItem {
	return &dto.CreditLogItem{
		ID:           l.ID,
		Amount:       l.Amount,
		Action:       l.Action,
		BalanceAfter: l.BalanceAfter,
		RefType:      l.RefType,
		RefID:        l.RefID,
		Note:         l.Note,
		CreatedAt:    l.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}
