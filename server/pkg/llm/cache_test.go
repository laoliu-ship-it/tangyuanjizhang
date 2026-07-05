package llm

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/model"
)

// ============================================================
// CachedLLMService.Analyze：缓存命中时必须返回 usage=nil（未产生新的 LLM 调用），
// 上层 handler 据此判断是否记 token 消耗——分类错误会导致缓存命中仍重复计费。
// ============================================================

// fakeServiceLLM 记录调用次数，返回可控的建议/用量/错误
type fakeServiceLLM struct {
	calls        int
	suggestions  []*dto.LLMSuggestion
	usage        *dto.LLMUsage
	err          error
	invalidCalls int
}

func (f *fakeServiceLLM) Analyze(_ context.Context, _ uint64, _ string, _ []string, _ []dto.CategoryItem) ([]*dto.LLMSuggestion, *dto.LLMUsage, error) {
	f.calls++
	return f.suggestions, f.usage, f.err
}
func (f *fakeServiceLLM) GetConfig(_ context.Context, _ uint64) (*dto.TenantLLMConfigResp, error) {
	return nil, nil
}
func (f *fakeServiceLLM) SaveConfig(_ context.Context, _ uint64, _ dto.SaveTenantLLMConfigReq) (*dto.TenantLLMConfigResp, error) {
	return nil, nil
}
func (f *fakeServiceLLM) GetPrompt(_ context.Context, _ uint64) (*dto.TenantPromptResp, error) {
	return nil, nil
}
func (f *fakeServiceLLM) SavePrompt(_ context.Context, _ uint64, _ dto.SaveTenantPromptReq) (*dto.TenantPromptResp, error) {
	return nil, nil
}
func (f *fakeServiceLLM) InvalidatePlatformClients() { f.invalidCalls++ }

// fakeConfigRepo 内存版平台配置仓库
type fakeConfigRepo struct {
	values map[string]string
}

func newFakeConfigRepo(values map[string]string) *fakeConfigRepo {
	return &fakeConfigRepo{values: values}
}

func (r *fakeConfigRepo) GetByKey(_ context.Context, key string) (*model.PlatformConfig, error) {
	v, ok := r.values[key]
	if !ok {
		return nil, nil
	}
	return &model.PlatformConfig{ConfigKey: key, ConfigValue: v}, nil
}
func (r *fakeConfigRepo) GetAll(_ context.Context) ([]*model.PlatformConfig, error) { return nil, nil }
func (r *fakeConfigRepo) Upsert(_ context.Context, _ *model.PlatformConfig) error   { return nil }

func textCacheConfig() *fakeConfigRepo {
	return newFakeConfigRepo(map[string]string{
		model.ConfigKeyCacheType:       "text",
		model.ConfigKeyCacheTTLMinutes: "30",
		model.ConfigKeyLLMCacheEnabled: "true",
	})
}

func TestCachedAnalyze_FirstCall_HitsInnerAndReturnsUsage(t *testing.T) {
	inner := &fakeServiceLLM{
		suggestions: []*dto.LLMSuggestion{{Type: "expense", Amount: 10}},
		usage:       &dto.LLMUsage{TotalTokens: 123},
	}
	c := NewCachedLLMService(inner, textCacheConfig())
	ctx := context.Background()

	suggestions, usage, err := c.Analyze(ctx, 1, "", []string{"发票内容"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inner.calls != 1 {
		t.Fatalf("expected inner called once, got %d", inner.calls)
	}
	if usage == nil || usage.TotalTokens != 123 {
		t.Errorf("expected usage propagated from inner call, got %+v", usage)
	}
	if len(suggestions) != 1 {
		t.Errorf("expected 1 suggestion, got %d", len(suggestions))
	}
}

func TestCachedAnalyze_SecondCall_CacheHit_UsageNil_NoInnerCall(t *testing.T) {
	// Scenario: 相同租户+相同 OCR 文本第二次分析，命中缓存，不应重复调用 LLM，
	// 也不应返回 usage（否则上层会对缓存命中重复记 token/扣积分）
	inner := &fakeServiceLLM{
		suggestions: []*dto.LLMSuggestion{{Type: "expense", Amount: 10}},
		usage:       &dto.LLMUsage{TotalTokens: 123},
	}
	c := NewCachedLLMService(inner, textCacheConfig())
	ctx := context.Background()

	if _, _, err := c.Analyze(ctx, 1, "", []string{"发票内容"}, nil); err != nil {
		t.Fatal(err)
	}
	suggestions, usage, err := c.Analyze(ctx, 1, "", []string{"发票内容"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inner.calls != 1 {
		t.Errorf("expected inner NOT called again on cache hit, got %d calls", inner.calls)
	}
	if usage != nil {
		t.Errorf("expected usage=nil on cache hit, got %+v", usage)
	}
	if len(suggestions) != 1 {
		t.Errorf("expected cached suggestion returned, got %d", len(suggestions))
	}
}

func TestCachedAnalyze_DifferentTenant_SameText_NotSharedCache(t *testing.T) {
	// Scenario: 不同租户的分类/提示词不同，即便 OCR 文本相同也不能共享缓存结果
	inner := &fakeServiceLLM{
		suggestions: []*dto.LLMSuggestion{{Type: "expense", Amount: 10}},
		usage:       &dto.LLMUsage{TotalTokens: 123},
	}
	c := NewCachedLLMService(inner, textCacheConfig())
	ctx := context.Background()

	if _, _, err := c.Analyze(ctx, 1, "", []string{"发票内容"}, nil); err != nil {
		t.Fatal(err)
	}
	if _, usage, err := c.Analyze(ctx, 2, "", []string{"发票内容"}, nil); err != nil {
		t.Fatal(err)
	} else if usage == nil {
		t.Error("expected different tenant to trigger a fresh inner call (usage != nil)")
	}
	if inner.calls != 2 {
		t.Errorf("expected inner called once per tenant, got %d", inner.calls)
	}
}

func TestCachedAnalyze_InnerError_NotCached(t *testing.T) {
	// Scenario: LLM 调用失败时不写入缓存，且错误必须透传给上层（用于展示友好错误提示/避免误判成功）
	inner := &fakeServiceLLM{err: errors.New("LLM 调用失败")}
	c := NewCachedLLMService(inner, textCacheConfig())
	ctx := context.Background()

	_, _, err := c.Analyze(ctx, 1, "", []string{"发票内容"}, nil)
	if err == nil {
		t.Fatal("expected error propagated from inner")
	}

	// 第二次调用应再次触发 inner（未被缓存）
	_, _, err = c.Analyze(ctx, 1, "", []string{"发票内容"}, nil)
	if err == nil {
		t.Fatal("expected error again since failed calls are not cached")
	}
	if inner.calls != 2 {
		t.Errorf("expected inner called twice (no caching of errors), got %d", inner.calls)
	}
}

func TestCachedAnalyze_CacheDisabled_AlwaysCallsInner(t *testing.T) {
	inner := &fakeServiceLLM{
		suggestions: []*dto.LLMSuggestion{{Type: "expense", Amount: 10}},
		usage:       &dto.LLMUsage{TotalTokens: 5},
	}
	cfg := newFakeConfigRepo(map[string]string{
		model.ConfigKeyLLMCacheEnabled: "false",
	})
	c := NewCachedLLMService(inner, cfg)
	ctx := context.Background()

	for i := range 3 {
		if _, usage, err := c.Analyze(ctx, 1, "", []string{"同样的文本"}, nil); err != nil {
			t.Fatal(err)
		} else if usage == nil {
			t.Errorf("call %d: expected usage returned when cache disabled", i)
		}
	}
	if inner.calls != 3 {
		t.Errorf("expected inner called every time when cache disabled, got %d", inner.calls)
	}
}

func TestCachedAnalyze_FileCache_SameFileContent_Hits(t *testing.T) {
	// Scenario: 默认文件缓存模式下，相同文件内容（即便路径不同）应命中同一缓存 key
	dir := t.TempDir()
	pathA := filepath.Join(dir, "a.jpg")
	pathB := filepath.Join(dir, "b.jpg")
	content := []byte("identical-image-bytes")
	if err := os.WriteFile(pathA, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pathB, content, 0o644); err != nil {
		t.Fatal(err)
	}

	inner := &fakeServiceLLM{
		suggestions: []*dto.LLMSuggestion{{Type: "expense", Amount: 1}},
		usage:       &dto.LLMUsage{TotalTokens: 1},
	}
	cfg := newFakeConfigRepo(map[string]string{
		model.ConfigKeyCacheType:       "file",
		model.ConfigKeyCacheTTLMinutes: "30",
		model.ConfigKeyLLMCacheEnabled: "true",
	})
	c := NewCachedLLMService(inner, cfg)
	ctx := context.Background()

	if _, _, err := c.Analyze(ctx, 1, pathA, nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, usage, err := c.Analyze(ctx, 1, pathB, nil, nil); err != nil {
		t.Fatal(err)
	} else if usage != nil {
		t.Errorf("expected cache hit (usage=nil) for identical file content, got %+v", usage)
	}
	if inner.calls != 1 {
		t.Errorf("expected inner called once for identical file content, got %d", inner.calls)
	}
}

func TestSavePrompt_Success_ClearsCache(t *testing.T) {
	// Scenario: 提示词变更后旧的分析结果不再适用，SavePrompt 成功后必须清空缓存
	inner := &fakeServiceLLM{
		suggestions: []*dto.LLMSuggestion{{Type: "expense", Amount: 1}},
		usage:       &dto.LLMUsage{TotalTokens: 1},
	}
	c := NewCachedLLMService(inner, textCacheConfig())
	ctx := context.Background()

	if _, _, err := c.Analyze(ctx, 1, "", []string{"文本"}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := c.SavePrompt(ctx, 1, dto.SaveTenantPromptReq{}); err != nil {
		t.Fatal(err)
	}

	if _, usage, err := c.Analyze(ctx, 1, "", []string{"文本"}, nil); err != nil {
		t.Fatal(err)
	} else if usage == nil {
		t.Error("expected cache cleared after SavePrompt, so a fresh call is made (usage != nil)")
	}
	if inner.calls != 2 {
		t.Errorf("expected inner called again after prompt change, got %d", inner.calls)
	}
}
