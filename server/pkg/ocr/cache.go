package ocr

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"sync"
	"time"

	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/repo"
)

// cacheEntry 缓存条目
type cacheEntry struct {
	result    *Result
	expiresAt time.Time
}

// CachedEngine wraps an Engine with configurable cache
type CachedEngine struct {
	inner      Engine
	configRepo repo.PlatformConfigRepo
	mu         sync.Mutex
	cache      map[string]*cacheEntry
	// 配置缓存
	lastConfigCheck time.Time
	cacheType       string // "file" or "text"
	cacheTTL        time.Duration
	enabled         bool
}

func NewCachedEngine(inner Engine, configRepo repo.PlatformConfigRepo) *CachedEngine {
	return &CachedEngine{
		inner:      inner,
		configRepo: configRepo,
		cache:      make(map[string]*cacheEntry),
		cacheType:  "file",
		cacheTTL:   30 * time.Minute,
		enabled:    true,
	}
}

// loadConfig 从数据库加载配置（每分钟最多检查一次）
func (c *CachedEngine) loadConfig(ctx context.Context) {
	if time.Since(c.lastConfigCheck) < time.Minute {
		return
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

	cfg, err = c.configRepo.GetByKey(ctx, model.ConfigKeyOCRCacheEnabled)
	if err == nil && cfg != nil {
		c.enabled = cfg.ConfigValue == "true"
	}

	c.lastConfigCheck = time.Now()
}

func (c *CachedEngine) Recognize(ctx context.Context, imagePath string) (*Result, error) {
	// 加载配置
	c.loadConfig(ctx)

	// 缓存未启用，直接调用
	if !c.enabled {
		return c.inner.Recognize(ctx, imagePath)
	}

	// 文件缓存模式：用文件 SHA256 做 key
	hash, err := fileHash(imagePath)
	if err != nil {
		return c.inner.Recognize(ctx, imagePath)
	}

	c.mu.Lock()
	entry, ok := c.cache[hash]
	if ok && time.Now().Before(entry.expiresAt) {
		c.mu.Unlock()
		return entry.result, nil
	}
	c.mu.Unlock()

	result, err := c.inner.Recognize(ctx, imagePath)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.cache[hash] = &cacheEntry{result: result, expiresAt: time.Now().Add(c.cacheTTL)}
	c.mu.Unlock()

	return result, nil
}

// RecognizeWithTextCache 文本缓存模式（用于 LLM 缓存）
// 返回 OCR 结果的同时，提供可用于文本缓存的 hash
func (c *CachedEngine) RecognizeWithTextCache(ctx context.Context, imagePath string) (*Result, string, error) {
	// 加载配置
	c.loadConfig(ctx)

	// 缓存未启用，直接调用
	if !c.enabled {
		result, err := c.inner.Recognize(ctx, imagePath)
		if err != nil {
			return nil, "", err
		}
		// 返回文本 hash 供 LLM 缓存使用
		textHash := textHashFromResult(result)
		return result, textHash, nil
	}

	// 根据缓存类型生成 key
	var key string
	if c.cacheType == "text" {
		// 文本缓存需要先识别，无法预判
		result, err := c.inner.Recognize(ctx, imagePath)
		if err != nil {
			return nil, "", err
		}
		key = textHashFromResult(result)

		c.mu.Lock()
		c.cache[key] = &cacheEntry{result: result, expiresAt: time.Now().Add(c.cacheTTL)}
		c.mu.Unlock()

		return result, key, nil
	}

	// 文件缓存
	hash, err := fileHash(imagePath)
	if err != nil {
		result, err := c.inner.Recognize(ctx, imagePath)
		if err != nil {
			return nil, "", err
		}
		return result, textHashFromResult(result), nil
	}

	c.mu.Lock()
	entry, ok := c.cache[hash]
	if ok && time.Now().Before(entry.expiresAt) {
		textHash := textHashFromResult(entry.result)
		c.mu.Unlock()
		return entry.result, textHash, nil
	}
	c.mu.Unlock()

	result, err := c.inner.Recognize(ctx, imagePath)
	if err != nil {
		return nil, "", err
	}

	c.mu.Lock()
	c.cache[hash] = &cacheEntry{result: result, expiresAt: time.Now().Add(c.cacheTTL)}
	c.mu.Unlock()

	return result, textHashFromResult(result), nil
}

// textHashFromResult 从 OCR 结果生成文本 hash
func textHashFromResult(result *Result) string {
	if result == nil || len(result.RawTexts) == 0 {
		return ""
	}
	h := sha256.New()
	for _, t := range result.RawTexts {
		h.Write([]byte(t.Text))
		h.Write([]byte("\n"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

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

// ClearCache 清空缓存
func (c *CachedEngine) ClearCache() {
	c.mu.Lock()
	c.cache = make(map[string]*cacheEntry)
	c.mu.Unlock()
}