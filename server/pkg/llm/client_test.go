package llm

import (
	"errors"
	"testing"

	"github.com/tmc/langchaingo/llms"
)

// ============================================================
// ClassifyError：将任意 provider 的原始错误归类为与 provider 无关的 ErrKind，
// 直接决定 friendlyLLMError 展示给用户的提示文案，分类错误会导致用户看到误导性提示。
// ============================================================

func TestClassifyError_Nil_ReturnsUnknown(t *testing.T) {
	if got := ClassifyError(nil); got != ErrKindUnknown {
		t.Errorf("expected ErrKindUnknown, got %v", got)
	}
}

func TestClassifyError_FallbackStringMatching(t *testing.T) {
	// Scenario: langchaingo 未识别的 provider 原始错误，按关键字兜底分类
	tests := []struct {
		name string
		err  error
		want ErrKind
	}{
		{"429 状态码", errors.New("request failed: 429 too many requests"), ErrKindRateLimit},
		{"rate_limit 关键字", errors.New("error code: rate_limit_exceeded"), ErrKindRateLimit},
		{"并发超限", errors.New("concurrent requests limit reached"), ErrKindRateLimit},
		{"401 未授权", errors.New("401 Unauthorized"), ErrKindAuth},
		{"api key 无效", errors.New("invalid API Key provided"), ErrKindAuth},
		{"超时", errors.New("context deadline exceeded"), ErrKindTimeout},
		{"503 不可用", errors.New("503 Service Unavailable"), ErrKindUnavailable},
		{"overloaded", errors.New("the model is overloaded, please retry"), ErrKindUnavailable},
		{"400 参数错误", errors.New("400 invalid request: unknown variant image_url"), ErrKindBadRequest},
		{"完全无法识别", errors.New("something totally unexpected happened"), ErrKindUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyError(tt.err); got != tt.want {
				t.Errorf("ClassifyError(%q) = %v, want %v", tt.err.Error(), got, tt.want)
			}
		})
	}
}

// ============================================================
// Client.parseJSON：解析 LLM 返回文本为记账建议列表，
// 这是 OCR→记账的核心转换逻辑，解析错误会直接导致用户看不到任何建议或数据错位。
// ============================================================

func TestParseJSON_StandardArray_Success(t *testing.T) {
	c := &Client{}
	raw := `[{"type":"expense","amount":88.5,"merchant_name":"沃尔玛","date":"2026-07-01"}]`

	got, err := c.parseJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(got))
	}
	if got[0].Amount != 88.5 || got[0].MerchantName != "沃尔玛" {
		t.Errorf("unexpected suggestion: %+v", got[0])
	}
}

func TestParseJSON_MarkdownFencedArray_Success(t *testing.T) {
	// Scenario: LLM 习惯性用 ```json ... ``` 包裹返回内容
	c := &Client{}
	raw := "```json\n[{\"type\":\"income\",\"amount\":100}]\n```"

	got, err := c.parseJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Type != "income" {
		t.Errorf("expected income suggestion, got %+v", got)
	}
}

func TestParseJSON_LegacySingleObject_Success(t *testing.T) {
	// Scenario: 兼容旧版单对象格式（非数组）
	c := &Client{}
	raw := `{"type":"expense","amount":30,"merchant_name":"星巴克"}`

	got, err := c.parseJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].MerchantName != "星巴克" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestParseJSON_InvalidType_DefaultsToExpense(t *testing.T) {
	// Scenario: type 字段既不是 income 也不是 expense（或缺失）时，兜底为 expense，
	// 保证前端记账列表不会出现无法识别的交易类型
	c := &Client{}
	raw := `[{"type":"refund","amount":10},{"amount":20}]`

	got, err := c.parseJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 suggestions, got %d", len(got))
	}
	for _, s := range got {
		if s.Type != "expense" {
			t.Errorf("expected type=expense fallback, got %q", s.Type)
		}
	}
}

func TestParseJSON_NullElementsInArray_Filtered(t *testing.T) {
	// Scenario: 数组中混入 null 元素（LLM 输出瑕疵）不应导致 panic 或产生空白记录
	c := &Client{}
	raw := `[{"type":"expense","amount":5},null,{"type":"income","amount":6}]`

	got, err := c.parseJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected null entries filtered out, got %d suggestions", len(got))
	}
}

func TestParseJSON_NoJSONContent_ReturnsError(t *testing.T) {
	c := &Client{}
	_, err := c.parseJSON("抱歉，我无法识别这张图片")
	if err == nil {
		t.Fatal("expected error for non-JSON content, got nil")
	}
}

func TestParseJSON_MalformedObject_ReturnsError(t *testing.T) {
	c := &Client{}
	_, err := c.parseJSON(`{"type": "expense", "amount": }`)
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

// ============================================================
// Client.extractUsage：从 provider 返回的 GenerationInfo 中提取 token 用量，
// 用于按次计费/统计成本，字段缺失或类型不一致时绝不能 panic，也不能产生负数。
// ============================================================

func TestExtractUsage_FullFields_ComputesCacheMiss(t *testing.T) {
	c := &Client{provider: "deepseek", modelName: "deepseek-chat"}
	choice := &llms.ContentChoice{GenerationInfo: map[string]any{
		"PromptTokens":       100,
		"PromptCachedTokens": 30,
		"CompletionTokens":   50,
		"TotalTokens":        150,
	}}

	usage := c.extractUsage(choice)

	if usage.Provider != "deepseek" || usage.Model != "deepseek-chat" {
		t.Errorf("provider/model not propagated: %+v", usage)
	}
	if usage.PromptCacheHitTokens != 30 {
		t.Errorf("expected cache hit=30, got %d", usage.PromptCacheHitTokens)
	}
	if usage.PromptCacheMissTokens != 70 {
		t.Errorf("expected cache miss=100-30=70, got %d", usage.PromptCacheMissTokens)
	}
	if usage.CompletionTokens != 50 || usage.TotalTokens != 150 {
		t.Errorf("unexpected completion/total: %+v", usage)
	}
}

func TestExtractUsage_MissingGenerationInfo_AllZero(t *testing.T) {
	// Scenario: provider 未返回 GenerationInfo（如某些 ollama 模型），不应 panic
	c := &Client{provider: "ollama", modelName: "llava"}
	choice := &llms.ContentChoice{}

	usage := c.extractUsage(choice)

	if usage.PromptCacheHitTokens != 0 || usage.PromptCacheMissTokens != 0 ||
		usage.CompletionTokens != 0 || usage.TotalTokens != 0 {
		t.Errorf("expected all-zero usage, got %+v", usage)
	}
}

func TestExtractUsage_CachedExceedsPrompt_MissClampedToZero(t *testing.T) {
	// Scenario: provider 返回的缓存 token 数异常地大于总 prompt token 数，
	// miss 不能算出负数（负数会污染统计报表）
	c := &Client{}
	choice := &llms.ContentChoice{GenerationInfo: map[string]any{
		"PromptTokens":       10,
		"PromptCachedTokens": 999,
	}}

	usage := c.extractUsage(choice)

	if usage.PromptCacheMissTokens != 0 {
		t.Errorf("expected miss clamped to 0, got %d", usage.PromptCacheMissTokens)
	}
}

func TestExtractUsage_Float64Values_ParsedCorrectly(t *testing.T) {
	// Scenario: 部分 provider（走 JSON 反序列化）的 GenerationInfo 数值类型是 float64 而非 int
	c := &Client{}
	choice := &llms.ContentChoice{GenerationInfo: map[string]any{
		"PromptTokens":     float64(200),
		"CompletionTokens": float64(80),
		"TotalTokens":      float64(280),
	}}

	usage := c.extractUsage(choice)

	if usage.PromptCacheMissTokens != 200 || usage.CompletionTokens != 80 || usage.TotalTokens != 280 {
		t.Errorf("float64 values not parsed correctly: %+v", usage)
	}
}

func TestExtractUsage_UnrecognizedValueType_TreatedAsZero(t *testing.T) {
	// Scenario: 未知类型（如字符串）不应 panic，按 0 处理
	c := &Client{}
	choice := &llms.ContentChoice{GenerationInfo: map[string]any{
		"PromptTokens": "not-a-number",
	}}

	usage := c.extractUsage(choice)

	if usage.PromptCacheMissTokens != 0 {
		t.Errorf("expected 0 for unrecognized type, got %d", usage.PromptCacheMissTokens)
	}
}
