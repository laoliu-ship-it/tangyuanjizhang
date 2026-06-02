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
)

// llmCacheEntry 缓存条目
type llmCacheEntry struct {
	suggestions []*dto.LLMSuggestion
	expiresAt   time.Time
}

// CachedLLMService 包装 LLM 服务实现，提供基于文件 SHA256 的内存缓存
// 实现了 service.LLMService 接口，可直接替换
type CachedLLMService struct {
	inner   serviceLLM
	mu      sync.Mutex
	cache   map[string]*llmCacheEntry
	ttl     time.Duration
}

// serviceLLM 包装器的内部接口，等同于 service.LLMService
type serviceLLM interface {
	Analyze(ctx context.Context, tenantID uint64, imagePath string, rawTexts []string, categories []dto.CategoryItem) ([]*dto.LLMSuggestion, error)
	GetConfig(ctx context.Context, tenantID uint64) (*dto.TenantLLMConfigResp, error)
	SaveConfig(ctx context.Context, tenantID uint64, req dto.SaveTenantLLMConfigReq) (*dto.TenantLLMConfigResp, error)
}

// NewCachedLLMService 创建带缓存的 LLM 服务，ttl 为缓存有效时间
func NewCachedLLMService(inner serviceLLM, ttl time.Duration) *CachedLLMService {
	return &CachedLLMService{
		inner: inner,
		cache: make(map[string]*llmCacheEntry),
		ttl:   ttl,
	}
}

// Analyze 先查缓存，未命中再调用 LLM
func (c *CachedLLMService) Analyze(ctx context.Context, tenantID uint64, imagePath string, rawTexts []string, categories []dto.CategoryItem) ([]*dto.LLMSuggestion, error) {
	hash, err := fileHash(imagePath)
	if err != nil {
		return c.inner.Analyze(ctx, tenantID, imagePath, rawTexts, categories)
	}

	// 缓存 key: SHA256(文件内容)
	key := hash

	c.mu.Lock()
	entry, ok := c.cache[key]
	if ok && time.Now().Before(entry.expiresAt) {
		c.mu.Unlock()
		return entry.suggestions, nil
	}
	c.mu.Unlock()

	suggestions, err := c.inner.Analyze(ctx, tenantID, imagePath, rawTexts, categories)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.cache[key] = &llmCacheEntry{suggestions: suggestions, expiresAt: time.Now().Add(c.ttl)}
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
