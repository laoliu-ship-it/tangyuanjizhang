package handler

import (
	"context"
	"errors"
	"testing"

	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/model"
)

// ============================================================
// recordLLMTokenUsage：LLM 分析后落库 token 消耗记录，
// 用于成本核算——缓存命中(usage=nil)不应重复计费，写库失败也不能影响主业务流程。
// ============================================================

// fakeTokenUsageRepo 记录 Create 调用，可模拟写库失败
type fakeTokenUsageRepo struct {
	created []*model.LLMTokenUsage
	err     error
}

func (r *fakeTokenUsageRepo) Create(_ context.Context, u *model.LLMTokenUsage) error {
	if r.err != nil {
		return r.err
	}
	r.created = append(r.created, u)
	return nil
}

func TestRecordLLMTokenUsage_NilUsage_Skipped(t *testing.T) {
	// Scenario: 缓存命中时 usage 为 nil，代表本次没有产生新的 LLM 调用，不应写入 token 记录
	repo := &fakeTokenUsageRepo{}

	recordLLMTokenUsage(context.Background(), repo, 1, "ocr_record", 42, nil)

	if len(repo.created) != 0 {
		t.Errorf("expected no record created for nil usage, got %d", len(repo.created))
	}
}

func TestRecordLLMTokenUsage_NilRepo_NoPanic(t *testing.T) {
	// Scenario: repo 未注入（如测试/降级场景）时不应 panic
	usage := &dto.LLMUsage{TotalTokens: 10}
	recordLLMTokenUsage(context.Background(), nil, 1, "ocr_record", 42, usage)
}

func TestRecordLLMTokenUsage_ValidUsage_CreatesRecordWithCorrectFields(t *testing.T) {
	repo := &fakeTokenUsageRepo{}
	usage := &dto.LLMUsage{
		Provider:              "deepseek",
		Model:                 "deepseek-chat",
		PromptCacheHitTokens:  10,
		PromptCacheMissTokens: 90,
		CompletionTokens:      50,
		TotalTokens:           150,
	}

	recordLLMTokenUsage(context.Background(), repo, 7, "ocr_record", 99, usage)

	if len(repo.created) != 1 {
		t.Fatalf("expected 1 record created, got %d", len(repo.created))
	}
	got := repo.created[0]
	if got.TenantID != 7 || got.Action != llmActionOrc2Bill || got.RefType != "ocr_record" || got.RefID != 99 {
		t.Errorf("record identity fields incorrect: %+v", got)
	}
	if got.Provider != "deepseek" || got.Model != "deepseek-chat" {
		t.Errorf("record provider/model incorrect: %+v", got)
	}
	if got.PromptCacheHitTokens != 10 || got.PromptCacheMissTokens != 90 ||
		got.CompletionTokens != 50 || got.TotalTokens != 150 {
		t.Errorf("record token fields incorrect: %+v", got)
	}
}

func TestRecordLLMTokenUsage_RepoCreateFails_DoesNotPanic(t *testing.T) {
	// Scenario: 写库失败时仅记日志，不能让 LLM 分析主流程因此报错/崩溃
	repo := &fakeTokenUsageRepo{err: errors.New("db down")}
	usage := &dto.LLMUsage{TotalTokens: 1}

	recordLLMTokenUsage(context.Background(), repo, 1, "ocr_record", 1, usage)
	// 断言仅为“未 panic”；创建记录数为 0 属预期（因写库失败）
	if len(repo.created) != 0 {
		t.Errorf("expected 0 created records on failure, got %d", len(repo.created))
	}
}
