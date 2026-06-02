package llm

import "fandianjizhang/server/config"

// NewClientFromConfig 根据平台级配置创建客户端，LLM_ENABLED=false 时返回 nil
func NewClientFromConfig(cfg config.LLMConfig) (*Client, error) {
	if !cfg.Enabled || cfg.APIKey == "" {
		return nil, nil
	}
	return NewClient(cfg)
}
