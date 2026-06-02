package ocr

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"sync"
	"time"
)

const cacheTTL = time.Minute

type cacheEntry struct {
	result    *Result
	expiresAt time.Time
}

// CachedEngine wraps an Engine with an in-memory SHA256-keyed cache (TTL: 1 min).
// Same image content → skip the OCR service call.
type CachedEngine struct {
	inner Engine
	mu    sync.Mutex
	cache map[string]*cacheEntry
}

func NewCachedEngine(inner Engine) *CachedEngine {
	return &CachedEngine{
		inner: inner,
		cache: make(map[string]*cacheEntry),
	}
}

func (c *CachedEngine) Recognize(ctx context.Context, imagePath string) (*Result, error) {
	hash, err := fileHash(imagePath)
	if err != nil {
		// 哈希失败不阻断流程，直接透传
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
	c.cache[hash] = &cacheEntry{result: result, expiresAt: time.Now().Add(cacheTTL)}
	c.mu.Unlock()

	return result, nil
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
