package service

import (
	"context"
	"fmt"
	"sync"

	"fandianjizhang/server/config"
	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/repo"
	pkgllm "fandianjizhang/server/pkg/llm"
)

// LLMService 提供 LLM 分析能力，支持租户级配置热加载
type LLMService interface {
	// Analyze 分析票据，返回记账建议列表（一张图可能含多笔交易）
	Analyze(ctx context.Context, tenantID uint64, imagePath string, rawTexts []string, categories []dto.CategoryItem) ([]*dto.LLMSuggestion, error)
	// GetConfig 获取当前租户的 LLM 配置（用于前端展示）
	GetConfig(ctx context.Context, tenantID uint64) (*dto.TenantLLMConfigResp, error)
	// SaveConfig 保存租户 LLM 配置，自动使新配置生效
	SaveConfig(ctx context.Context, tenantID uint64, req dto.SaveTenantLLMConfigReq) (*dto.TenantLLMConfigResp, error)
}

type llmService struct {
	platformCfg config.LLMConfig
	repo        repo.TenantLLMConfigRepo

	// 每租户缓存的 LLM 客户端，避免每次请求重建连接
	mu      sync.RWMutex
	clients map[uint64]*pkgllm.Client // nil 表示不可用
}

func NewLLMService(platformCfg config.LLMConfig, r repo.TenantLLMConfigRepo) LLMService {
	return &llmService{
		platformCfg: platformCfg,
		repo:        r,
		clients:     make(map[uint64]*pkgllm.Client),
	}
}

// Analyze 先查租户配置，决定用平台 key 还是自己的 key，再调用 LLM
func (s *llmService) Analyze(ctx context.Context, tenantID uint64, imagePath string, rawTexts []string, categories []dto.CategoryItem) ([]*dto.LLMSuggestion, error) {
	client, err := s.getOrBuildClient(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, fmt.Errorf("LLM 未配置或未启用")
	}
	return client.Analyze(ctx, imagePath, rawTexts, categories)
}

func (s *llmService) GetConfig(ctx context.Context, tenantID uint64) (*dto.TenantLLMConfigResp, error) {
	cfg, err := s.repo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	platformEnabled := s.platformCfg.Enabled && s.platformCfg.APIKey != ""
	if cfg == nil {
		return &dto.TenantLLMConfigResp{
			TenantID:        tenantID,
			Enabled:         false,
			UsePlatform:     true,
			PlatformEnabled: platformEnabled,
			Provider:        "openai",
			Model:           "gpt-4o",
			Mode:            "ocr_text",
		}, nil
	}
	return toConfigResp(cfg, platformEnabled), nil
}

func (s *llmService) SaveConfig(ctx context.Context, tenantID uint64, req dto.SaveTenantLLMConfigReq) (*dto.TenantLLMConfigResp, error) {
	existing, err := s.repo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var cfg model.TenantLLMConfig
	if existing != nil {
		cfg = *existing
	} else {
		cfg.TenantID = tenantID
	}

	cfg.Enabled = req.Enabled
	cfg.UsePlatform = req.UsePlatform
	if req.Provider != "" {
		cfg.Provider = req.Provider
	}
	if cfg.Provider == "" {
		cfg.Provider = "openai"
	}
	cfg.BaseURL = req.BaseURL
	if req.APIKey != "" {
		cfg.APIKey = req.APIKey
	}
	if req.Model != "" {
		cfg.Model = req.Model
	}
	if cfg.Model == "" {
		cfg.Model = "gpt-4o"
	}
	if req.Mode != "" {
		cfg.Mode = req.Mode
	}
	if cfg.Mode == "" {
		cfg.Mode = "ocr_text"
	}

	if err := s.repo.Save(ctx, &cfg); err != nil {
		return nil, err
	}

	// 配置变更后删除缓存，下次 Analyze 时重建
	s.mu.Lock()
	delete(s.clients, tenantID)
	s.mu.Unlock()

	platformEnabled := s.platformCfg.Enabled && s.platformCfg.APIKey != ""
	return toConfigResp(&cfg, platformEnabled), nil
}

// getOrBuildClient 获取或构建租户的 LLM 客户端（带缓存）
func (s *llmService) getOrBuildClient(ctx context.Context, tenantID uint64) (*pkgllm.Client, error) {
	s.mu.RLock()
	if client, ok := s.clients[tenantID]; ok {
		s.mu.RUnlock()
		return client, nil
	}
	s.mu.RUnlock()

	// 从数据库加载租户配置
	tenantCfg, err := s.repo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var client *pkgllm.Client

	if tenantCfg != nil && tenantCfg.Enabled {
		if tenantCfg.UsePlatform {
			// 平台可用时优先用平台；平台不可用但租户有自己的 key 时自动降级
			if s.platformCfg.Enabled && s.platformCfg.APIKey != "" {
				client, err = pkgllm.NewClientFromConfig(s.platformCfg)
			} else if tenantCfg.APIKey != "" {
				client, err = pkgllm.NewClient(config.LLMConfig{
					Enabled:        true,
					Provider:       tenantCfg.Provider,
					BaseURL:        tenantCfg.BaseURL,
					APIKey:         tenantCfg.APIKey,
					Model:          tenantCfg.Model,
					Mode:           tenantCfg.Mode,
					TimeoutSeconds: s.platformCfg.TimeoutSeconds,
				})
			} else {
				return nil, fmt.Errorf("平台暂未开放 AI 分析功能，请联系管理员或使用自定义 API Key")
			}
		} else {
			// 使用租户自己的配置
			if tenantCfg.APIKey == "" {
				return nil, fmt.Errorf("租户未配置 API Key")
			}
			client, err = pkgllm.NewClient(config.LLMConfig{
				Enabled:        true,
				Provider:       tenantCfg.Provider,
				BaseURL:        tenantCfg.BaseURL,
				APIKey:         tenantCfg.APIKey,
				Model:          tenantCfg.Model,
				Mode:           tenantCfg.Mode,
				TimeoutSeconds: s.platformCfg.TimeoutSeconds,
			})
		}
		if err != nil {
			return nil, fmt.Errorf("初始化 LLM 客户端失败: %w", err)
		}
	}
	// 缓存结果（nil 也缓存，避免重复查库）
	s.mu.Lock()
	s.clients[tenantID] = client
	s.mu.Unlock()

	return client, nil
}

func toConfigResp(cfg *model.TenantLLMConfig, platformEnabled bool) *dto.TenantLLMConfigResp {
	return &dto.TenantLLMConfigResp{
		ID:              cfg.ID,
		TenantID:        cfg.TenantID,
		Enabled:         cfg.Enabled,
		UsePlatform:     cfg.UsePlatform,
		PlatformEnabled: platformEnabled,
		Provider:        cfg.Provider,
		BaseURL:         cfg.BaseURL,
		APIKeyMask:      maskAPIKey(cfg.APIKey),
		Model:           cfg.Model,
		Mode:            cfg.Mode,
	}
}

func maskAPIKey(key string) string {
	if len(key) == 0 {
		return ""
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}
