package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"fandianjizhang/server/config"
	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/repo"
	pkgllm "fandianjizhang/server/pkg/llm"
)

// LLMService 提供 LLM 分析能力，支持租户级配置热加载
type LLMService interface {
	// Analyze 分析票据，返回记账建议列表（一张图可能含多笔交易）及本次调用的 token 消耗
	Analyze(ctx context.Context, tenantID uint64, imagePath string, rawTexts []string, categories []dto.CategoryItem) ([]*dto.LLMSuggestion, *dto.LLMUsage, error)
	// GetConfig 获取当前租户的 LLM 配置（用于前端展示）
	GetConfig(ctx context.Context, tenantID uint64) (*dto.TenantLLMConfigResp, error)
	// SaveConfig 保存租户 LLM 配置，自动使新配置生效
	SaveConfig(ctx context.Context, tenantID uint64, req dto.SaveTenantLLMConfigReq) (*dto.TenantLLMConfigResp, error)
	// GetPrompt 获取租户识别提示词配置（含平台默认模板供前端编辑）
	GetPrompt(ctx context.Context, tenantID uint64) (*dto.TenantPromptResp, error)
	// SavePrompt 保存租户自定义识别提示词，传空串恢复平台默认
	SavePrompt(ctx context.Context, tenantID uint64, req dto.SaveTenantPromptReq) (*dto.TenantPromptResp, error)
	// InvalidatePlatformClients 平台 LLM 配置变更后清除所有使用平台配置的租户缓存
	InvalidatePlatformClients()
}

type llmService struct {
	platformCfg     config.LLMConfig
	repo            repo.TenantLLMConfigRepo
	platformCfgRepo repo.PlatformConfigRepo

	// 每租户缓存的 LLM 客户端，避免每次请求重建连接
	mu      sync.RWMutex
	clients map[uint64]*pkgllm.Client // nil 表示不可用
}

func NewLLMService(platformCfg config.LLMConfig, r repo.TenantLLMConfigRepo, platformCfgRepo repo.PlatformConfigRepo) LLMService {
	return &llmService{
		platformCfg:     platformCfg,
		repo:            r,
		platformCfgRepo: platformCfgRepo,
		clients:         make(map[uint64]*pkgllm.Client),
	}
}

// InvalidatePlatformClients 清除所有租户客户端缓存，平台配置变更时调用
func (s *llmService) InvalidatePlatformClients() {
	s.mu.Lock()
	s.clients = make(map[uint64]*pkgllm.Client)
	s.mu.Unlock()
}

// loadPlatformLLMConfig 从 DB 读平台 LLM 配置，不存在则回退到静态配置
func (s *llmService) loadPlatformLLMConfig(ctx context.Context) config.LLMConfig {
	if s.platformCfgRepo == nil {
		log.Printf("[LLM] loadPlatformLLMConfig: platformCfgRepo is nil, using static config (enabled=%v apiKey=%v)", s.platformCfg.Enabled, s.platformCfg.APIKey != "")
		return s.platformCfg
	}
	get := func(key string) string {
		cfg, err := s.platformCfgRepo.GetByKey(ctx, key)
		if err != nil {
			log.Printf("[LLM] loadPlatformLLMConfig: GetByKey(%s) error: %v", key, err)
			return ""
		}
		if cfg == nil {
			log.Printf("[LLM] loadPlatformLLMConfig: GetByKey(%s) = nil (key not found)", key)
			return ""
		}
		log.Printf("[LLM] loadPlatformLLMConfig: GetByKey(%s) = %q", key, cfg.ConfigValue)
		return cfg.ConfigValue
	}
	enabled := get(model.ConfigKeyPlatformLLMEnabled) == "true"
	apiKey := get(model.ConfigKeyPlatformLLMAPIKey)
	if apiKey == "" {
		apiKey = s.platformCfg.APIKey
	}
	provider := get(model.ConfigKeyPlatformLLMProvider)
	if provider == "" {
		provider = s.platformCfg.Provider
	}
	baseURL := get(model.ConfigKeyPlatformLLMBaseURL)
	if baseURL == "" {
		baseURL = s.platformCfg.BaseURL
	}
	modelName := get(model.ConfigKeyPlatformLLMModel)
	if modelName == "" {
		modelName = s.platformCfg.Model
	}
	mode := get(model.ConfigKeyPlatformLLMMode)
	if mode == "" {
		mode = s.platformCfg.Mode
	}
	log.Printf("[LLM] loadPlatformLLMConfig: result enabled=%v provider=%s model=%s apiKey=%v baseURL=%q mode=%s",
		enabled, provider, modelName, apiKey != "", baseURL, mode)
	return config.LLMConfig{
		Enabled:        enabled,
		Provider:       provider,
		BaseURL:        baseURL,
		APIKey:         apiKey,
		Model:          modelName,
		Mode:           mode,
		TimeoutSeconds: s.platformCfg.TimeoutSeconds,
	}
}

// Analyze 先查租户配置，决定用平台 key 还是自己的 key，再调用 LLM
func (s *llmService) Analyze(ctx context.Context, tenantID uint64, imagePath string, rawTexts []string, categories []dto.CategoryItem) ([]*dto.LLMSuggestion, *dto.LLMUsage, error) {
	client, err := s.getOrBuildClient(ctx, tenantID)
	if err != nil {
		return nil, nil, err
	}
	if client == nil {
		return nil, nil, fmt.Errorf("LLM 未配置或未启用")
	}
	return client.Analyze(ctx, imagePath, rawTexts, categories)
}

func (s *llmService) GetConfig(ctx context.Context, tenantID uint64) (*dto.TenantLLMConfigResp, error) {
	cfg, err := s.repo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	platformCfg := s.loadPlatformLLMConfig(ctx)
	platformEnabled := platformCfg.Enabled && platformCfg.APIKey != ""
	if cfg == nil {
		return &dto.TenantLLMConfigResp{
			TenantID:        tenantID,
			Enabled:         true,
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

	platformCfg := s.loadPlatformLLMConfig(ctx)
	platformEnabled := platformCfg.Enabled && platformCfg.APIKey != ""
	return toConfigResp(&cfg, platformEnabled), nil
}

// 自定义提示词长度上限（字符数），防止把超长文本塞进每次请求
const maxCustomPromptLen = 20000

func (s *llmService) GetPrompt(ctx context.Context, tenantID uint64) (*dto.TenantPromptResp, error) {
	cfg, err := s.repo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	resp := &dto.TenantPromptResp{
		DefaultPrompt: pkgllm.DefaultPromptTemplate,
		Placeholder:   pkgllm.CategoriesPlaceholder,
	}
	if cfg != nil {
		resp.CustomPrompt = cfg.CustomPrompt
	}
	return resp, nil
}

func (s *llmService) SavePrompt(ctx context.Context, tenantID uint64, req dto.SaveTenantPromptReq) (*dto.TenantPromptResp, error) {
	prompt := strings.TrimSpace(req.CustomPrompt)
	if len([]rune(prompt)) > maxCustomPromptLen {
		return nil, fmt.Errorf("提示词过长，最多 %d 字", maxCustomPromptLen)
	}
	// 内容与平台默认一致时按"未自定义"处理，后续平台升级默认提示词可自动跟随
	if prompt == strings.TrimSpace(pkgllm.DefaultPromptTemplate) {
		prompt = ""
	}

	existing, err := s.repo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	var cfg model.TenantLLMConfig
	if existing != nil {
		cfg = *existing
	} else {
		cfg.TenantID = tenantID
		cfg.UsePlatform = true
		cfg.Provider = "openai"
		cfg.Model = "gpt-4o"
		cfg.Mode = "ocr_text"
	}
	cfg.CustomPrompt = prompt

	if err := s.repo.Save(ctx, &cfg); err != nil {
		return nil, err
	}

	// 提示词变更后删除客户端缓存，下次 Analyze 时带新提示词重建
	s.mu.Lock()
	delete(s.clients, tenantID)
	s.mu.Unlock()

	return &dto.TenantPromptResp{
		CustomPrompt:  cfg.CustomPrompt,
		DefaultPrompt: pkgllm.DefaultPromptTemplate,
		Placeholder:   pkgllm.CategoriesPlaceholder,
	}, nil
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

	// nil 配置视为"已启用 + 使用平台"（新租户默认行为）
	if tenantCfg == nil {
		log.Printf("[LLM] tenant=%d 无自定义配置，使用平台默认", tenantID)
		tenantCfg = &model.TenantLLMConfig{
			TenantID:    tenantID,
			Enabled:     true,
			UsePlatform: true,
		}
	} else {
		log.Printf("[LLM] tenant=%d 配置: enabled=%v use_platform=%v apiKey=%v", tenantID, tenantCfg.Enabled, tenantCfg.UsePlatform, tenantCfg.APIKey != "")
	}

	platformCfg := s.loadPlatformLLMConfig(ctx)
	var client *pkgllm.Client

	if tenantCfg.Enabled {
		if tenantCfg.UsePlatform {
			log.Printf("[LLM] tenant=%d 使用平台配置: platform.enabled=%v platform.apiKey=%v", tenantID, platformCfg.Enabled, platformCfg.APIKey != "")
			// 平台可用时优先用平台；平台不可用但租户有自己的 key 时自动降级
			if platformCfg.Enabled && platformCfg.APIKey != "" {
				client, err = pkgllm.NewClientFromConfig(platformCfg)
			} else if tenantCfg.APIKey != "" {
				client, err = pkgllm.NewClient(config.LLMConfig{
					Enabled:        true,
					Provider:       tenantCfg.Provider,
					BaseURL:        tenantCfg.BaseURL,
					APIKey:         tenantCfg.APIKey,
					Model:          tenantCfg.Model,
					Mode:           tenantCfg.Mode,
					TimeoutSeconds: platformCfg.TimeoutSeconds,
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
				TimeoutSeconds: platformCfg.TimeoutSeconds,
			})
		}
		if err != nil {
			return nil, fmt.Errorf("初始化 LLM 客户端失败: %w", err)
		}
		// 无论用平台还是自定义 key，租户自定义提示词均生效
		client.SetPromptTemplate(tenantCfg.CustomPrompt)
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
