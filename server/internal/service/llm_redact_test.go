package service

import (
	"strings"
	"testing"

	"fandianjizhang/server/internal/model"
)

func TestRedactConfigValue(t *testing.T) {
	const secret = "sk-1234567890abcdef"

	tests := []struct {
		name  string
		key   string
		value string
		want  string
	}{
		{name: "api key 已配置", key: model.ConfigKeyPlatformLLMAPIKey, value: secret, want: "<redacted>"},
		{name: "api key 为空", key: model.ConfigKeyPlatformLLMAPIKey, value: "", want: "<empty>"},
		{name: "api key 仅空白", key: model.ConfigKeyPlatformLLMAPIKey, value: "   ", want: "<empty>"},
		{name: "模型名正常输出", key: model.ConfigKeyPlatformLLMModel, value: "deepseek-v4-flash", want: `"deepseek-v4-flash"`},
		{name: "base url 正常输出", key: model.ConfigKeyPlatformLLMBaseURL, value: "https://api.deepseek.com", want: `"https://api.deepseek.com"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redactConfigValue(tt.key, tt.value)
			if got != tt.want {
				t.Fatalf("redactConfigValue(%q, %q) = %q, want %q", tt.key, tt.value, got, tt.want)
			}
			if strings.Contains(got, secret) {
				t.Fatalf("密钥原值不应出现在日志描述中: %q", got)
			}
		})
	}
}
