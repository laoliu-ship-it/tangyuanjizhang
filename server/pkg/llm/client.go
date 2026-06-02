package llm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"fandianjizhang/server/config"
	"fandianjizhang/server/internal/dto"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
)

var jsonFenceRe = regexp.MustCompile("(?s)^```(?:json)?\\s*|\\s*```$")

// Client LLM 客户端，封装 langchaingo
type Client struct {
	model   llms.Model
	mode    string        // "vision" | "ocr_text"
	timeout time.Duration
}

// NewClient 根据 LLMConfig 创建客户端
func NewClient(cfg config.LLMConfig) (*Client, error) {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second

	var model llms.Model
	var err error

	switch cfg.Provider {
	case "ollama":
		opts := []ollama.Option{ollama.WithModel(cfg.Model)}
		if cfg.BaseURL != "" {
			opts = append(opts, ollama.WithServerURL(cfg.BaseURL))
		}
		model, err = ollama.New(opts...)
	default: // openai / deepseek / azure 均走 openai 兼容接口
		opts := []openai.Option{
			openai.WithToken(cfg.APIKey),
			openai.WithModel(cfg.Model),
		}
		if cfg.BaseURL != "" {
			opts = append(opts, openai.WithBaseURL(cfg.BaseURL))
		}
		model, err = openai.New(opts...)
	}
	if err != nil {
		return nil, fmt.Errorf("创建 LLM 客户端失败: %w", err)
	}

	mode := cfg.Mode
	if mode != "vision" {
		mode = "ocr_text"
	}

	return &Client{model: model, mode: mode, timeout: timeout}, nil
}

// Analyze 分析票据，返回记账建议列表（一张图可能含多笔交易）
func (c *Client) Analyze(ctx context.Context, imagePath string, rawTexts []string, categories []dto.CategoryItem) ([]*dto.LLMSuggestion, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	if c.mode == "vision" {
		return c.analyzeVision(ctx, imagePath, categories)
	}
	return c.analyzeOCRText(ctx, rawTexts, categories)
}

func (c *Client) analyzeOCRText(ctx context.Context, rawTexts []string, categories []dto.CategoryItem) ([]*dto.LLMSuggestion, error) {
	if len(rawTexts) == 0 {
		return nil, fmt.Errorf("OCR 文本为空")
	}

	messages := []llms.MessageContent{
		{
			Role:  llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{llms.TextPart(BuildSystemPrompt(categories))},
		},
		{
			Role:  llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{llms.TextPart(BuildOCRTextMessage(rawTexts))},
		},
	}

	resp, err := c.model.GenerateContent(ctx, messages, llms.WithTemperature(0.1))
	if err != nil {
		return nil, fmt.Errorf("LLM 调用失败: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("LLM 返回空响应")
	}
	return c.parseJSON(resp.Choices[0].Content)
}

func (c *Client) analyzeVision(ctx context.Context, imagePath string, categories []dto.CategoryItem) ([]*dto.LLMSuggestion, error) {
	dataURI, err := encodeImageAsDataURI(imagePath)
	if err != nil {
		return nil, fmt.Errorf("图片编码失败: %w", err)
	}

	messages := []llms.MessageContent{
		{
			Role:  llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{llms.TextPart(BuildSystemPrompt(categories))},
		},
		{
			Role: llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{
				llms.TextPart(BuildVisionMessage()),
				llms.ImageURLPart(dataURI),
			},
		},
	}

	resp, err := c.model.GenerateContent(ctx, messages, llms.WithTemperature(0.1))
	if err != nil {
		return nil, fmt.Errorf("LLM vision 调用失败: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("LLM 返回空响应")
	}
	return c.parseJSON(resp.Choices[0].Content)
}

func (c *Client) parseJSON(raw string) ([]*dto.LLMSuggestion, error) {
	clean := strings.TrimSpace(jsonFenceRe.ReplaceAllString(raw, ""))

	// 尝试解析为数组
	arrStart := strings.Index(clean, "[")
	arrEnd := strings.LastIndex(clean, "]")
	if arrStart >= 0 && arrEnd > arrStart {
		snippet := clean[arrStart : arrEnd+1]
		var list []*dto.LLMSuggestion
		if err := json.Unmarshal([]byte(snippet), &list); err == nil {
			return normalizeSuggestions(list), nil
		}
	}

	// 兼容旧的单对象格式
	objStart := strings.Index(clean, "{")
	objEnd := strings.LastIndex(clean, "}")
	if objStart < 0 || objEnd <= objStart {
		return nil, fmt.Errorf("LLM 返回内容不含有效 JSON: %q", raw)
	}
	var s dto.LLMSuggestion
	if err := json.Unmarshal([]byte(clean[objStart:objEnd+1]), &s); err != nil {
		return nil, fmt.Errorf("LLM JSON 解析失败: %w, raw: %q", err, raw)
	}
	return normalizeSuggestions([]*dto.LLMSuggestion{&s}), nil
}

func normalizeSuggestions(list []*dto.LLMSuggestion) []*dto.LLMSuggestion {
	out := make([]*dto.LLMSuggestion, 0, len(list))
	for _, s := range list {
		if s == nil {
			continue
		}
		if s.Type != "income" && s.Type != "expense" {
			s.Type = "expense"
		}
		out = append(out, s)
	}
	return out
}

func encodeImageAsDataURI(imagePath string) (string, error) {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return "", err
	}
	ext := strings.ToLower(imagePath[strings.LastIndex(imagePath, ".")+1:])
	mime := "image/jpeg"
	switch ext {
	case "png":
		mime = "image/png"
	case "webp":
		mime = "image/webp"
	case "bmp":
		mime = "image/bmp"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}
