package llm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"sync"
	"time"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/repo"
)

// llmCacheEntry 缓存条目
type llmCacheEntry struct {
	suggestions []*dto.LLMSuggestion
	expiresAt   time.Time
}

// CachedLLMService 包装 LLM 服务实现，提供基于配置的缓存
type CachedLLMService struct {
	inner      serviceLLM
	configRepo repo.PlatformConfigRepo
	mu         sync.Mutex
	cache      map[string]*llmCacheEntry
	// 配置缓存（避免每次请求都查数据库）
	lastConfigCheck time.Time
	cacheType       string // "file" or "text"
	cacheTTL        time.Duration
	enabled         bool
}

// serviceLLM 包装器的内部接口，等同于 service.LLMService
type serviceLLM interface {
	Analyze(ctx context.Context, tenantID uint64, imagePath string, rawTexts []string, categories []dto.CategoryItem) ([]*dto.LLMSuggestion, error)
	GetConfig(ctx context.Context, tenantID uint64) (*dto.TenantLLMConfigResp, error)
	SaveConfig(ctx context.Context, tenantID uint64, req dto.SaveTenantLLMConfigReq) (*dto.TenantLLMConfigResp, error)
}

// NewCachedLLMService 创建带缓存的 LLM 服务
func NewCachedLLMService(inner serviceLLM, configRepo repo.PlatformConfigRepo) *CachedLLMService {
	return &CachedLLMService{
		inner:      inner,
		configRepo: configRepo,
		cache:      make(map[string]*llmCacheEntry),
		cacheType:  "file", // 默认文件缓存
		cacheTTL:   30 * time.Minute, // 默认 30 分钟
		enabled:    true,
	}
}

// loadConfig 从数据库加载配置（每分钟最多检查一次）
func (c *CachedLLMService) loadConfig(ctx context.Context) {
	if time.Since(c.lastConfigCheck) < time.Minute {
		return // 1 分钟内不重复检查
	}

	cfg, err := c.configRepo.GetByKey(ctx, model.ConfigKeyCacheType)
	if err == nil && cfg != nil {
		c.cacheType = cfg.ConfigValue
	}

	cfg, err = c.configRepo.GetByKey(ctx, model.ConfigKeyCacheTTLMinutes)
	if err == nil && cfg != nil {
		minutes := 30
		if mins, err := time.ParseDuration(cfg.ConfigValue + "m"); err == nil && mins > 0 {
			minutes = int(mins.Minutes())
		}
		c.cacheTTL = time.Duration(minutes) * time.Minute
	}

	cfg, err = c.configRepo.GetByKey(ctx, model.ConfigKeyLLMCacheEnabled)
	if err == nil && cfg != nil {
		c.enabled = cfg.ConfigValue == "true"
	}

	c.lastConfigCheck = time.Now()
}

// Analyze 先查缓存，未命中再调用 LLM
func (c *CachedLLMService) Analyze(ctx context.Context, tenantID uint64, imagePath string, rawTexts []string, categories []dto.CategoryItem) ([]*dto.LLMSuggestion, error) {
	// 加载配置
	c.loadConfig(ctx)

	// 缓存未启用，直接调用
	if !c.enabled {
		return c.inner.Analyze(ctx, tenantID, imagePath, rawTexts, categories)
	}

	// 根据缓存类型生成 key
	var key string
	if c.cacheType == "text" && len(rawTexts) > 0 {
		// 文本缓存：用 OCR 文本内容做 key
		key = textHash(rawTexts)
	} else {
		// 文件缓存：用文件 SHA256 做 key
		hash, err := fileHash(imagePath)
		if err != nil {
			return c.inner.Analyze(ctx, tenantID, imagePath, rawTexts, categories)
		}
		key = hash
	}

	// 查缓存
	c.mu.Lock()
	entry, ok := c.cache[key]
	if ok && time.Now().Before(entry.expiresAt) {
		c.mu.Unlock()
		return entry.suggestions, nil
	}
	c.mu.Unlock()

	// 调用 LLM
	suggestions, err := c.inner.Analyze(ctx, tenantID, imagePath, rawTexts, categories)
	if err != nil {
		return nil, err
	}

	// 存缓存
	c.mu.Lock()
	c.cache[key] = &llmCacheEntry{suggestions: suggestions, expiresAt: time.Now().Add(c.cacheTTL)}
	c.mu.Unlock()

	return suggestions, nil
}

// GetConfig 透传
func (c *CachedLLMService) GetConfig(ctx context.Context, tenantID uint64) (*dto.TenantLLMConfigResp, error) {
	return c.inner.GetConfig(ctx, tenantID)
}

// SaveConfig 透传
func (c *CachedLLMService) SaveConfig(ctx context.Context, tenantID uint64, req dto.SaveTenantLLMConfigReq) (*dto.TenantLLMConfigResp, error) {
	return c.inner.SaveConfig(ctx, tenantID, req)
}

// fileHash 计算文件内容的 SHA256
func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// textHash 计算文本内容的 SHA256
func textHash(texts []string) string {
	h := sha256.New()
	for _, t := range texts {
		h.Write([]byte(t))
		h.Write([]byte("\n"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// ClearCache 清空缓存（配置变更时可调用）
func (c *CachedLLMService) ClearCache() {
	c.mu.Lock()
	c.cache = make(map[string]*llmCacheEntry)
	c.mu.Unlock()
}