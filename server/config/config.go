package config

import (
	"os"
	"strconv"
)

type Config struct {
	DB      DBConfig
	JWT     JWTConfig
	OCR     OCRConfig
	LLM     LLMConfig
	Upload  UploadConfig
	Server  ServerConfig
	WeChat  WeChatConfig
	Billing BillingConfig
}

// BillingConfig 次数包/虚拟支付。默认关闭；XGJ 服务间 secret 只从环境变量读取，不写入日志。
type BillingConfig struct {
	Enabled             bool
	DemoLoginEnabled    bool
	DemoTTLMinutes      int
	AppSlug             string
	XGJBaseURL          string
	XGJSecret           string
	ClockSkewSeconds    int
	DefaultValidityDays int
}

// WeChatConfig 微信小程序登录配置（AppID / AppSecret 在微信公众平台获取）
type WeChatConfig struct {
	AppID  string
	Secret string
}

// LLMConfig 平台级 LLM 配置，从环境变量读取，作为租户未配置时的兜底
type LLMConfig struct {
	Enabled        bool
	Provider       string // "openai" | "azure" | "deepseek" | "ollama"
	BaseURL        string
	APIKey         string
	Model          string
	Mode           string // "vision" | "ocr_text"
	TimeoutSeconds int
}

type DBConfig struct {
	DSN string
}

type JWTConfig struct {
	Secret string
}

type OCRConfig struct {
	Engine             string
	RapidOCRURL        string
	AIMode             bool
	LLMCacheTTLSeconds int // LLM 结果缓存时长（秒），0=不缓存
}

type UploadConfig struct {
	Path    string
	MaxSize int64 // 上传最大文件大小（字节），默认 1MB
}

type ServerConfig struct {
	Port string
}

func Load() *Config {
	return &Config{
		DB: DBConfig{
			DSN: getEnv("DB_DSN", "root:password@tcp(localhost:3306)/fandian?charset=utf8mb4&parseTime=True&loc=Local"),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", "default_jwt_secret_change_in_production"),
		},
		OCR: OCRConfig{
			Engine:             getEnv("OCR_ENGINE", "rapidocr"),
			RapidOCRURL:        getEnv("OCR_RAPIDOCR_URL", "http://localhost:8001/ocr"),
			AIMode:             getEnv("OCR_AI_MODE", "false") == "true",
			LLMCacheTTLSeconds: parseInt(getEnv("OCR_LLM_CACHE_TTL", "3600")),
		},
		LLM: LLMConfig{
			Enabled:        getEnv("LLM_ENABLED", "false") == "true",
			Provider:       getEnv("LLM_PROVIDER", "openai"),
			BaseURL:        getEnv("LLM_BASE_URL", ""),
			APIKey:         getEnv("LLM_API_KEY", ""),
			Model:          getEnv("LLM_MODEL", "gpt-4o"),
			Mode:           getEnv("LLM_MODE", "ocr_text"),
			TimeoutSeconds: parseInt(getEnv("LLM_TIMEOUT_SECONDS", "30")),
		},
		Upload: UploadConfig{
			Path:    getEnv("UPLOAD_PATH", "./uploads"),
			MaxSize: parseInt64(getEnv("UPLOAD_MAX_SIZE", "1048576")),
		},
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "7072"),
		},
		WeChat: WeChatConfig{
			AppID:  getEnv("WECHAT_MINI_APPID", ""),
			Secret: getEnv("WECHAT_MINI_SECRET", ""),
		},
		Billing: BillingConfig{
			Enabled:             getEnv("BILLING_ENABLED", "false") == "true",
			DemoLoginEnabled:    getEnv("BILLING_DEMO_LOGIN_ENABLED", "false") == "true",
			DemoTTLMinutes:      parseInt(getEnv("BILLING_DEMO_TTL_MINUTES", "30")),
			AppSlug:             getEnv("BILLING_XGJ_APP_SLUG", "hehuojizhangzhushou"),
			XGJBaseURL:          getEnv("BILLING_XGJ_BASE_URL", ""),
			XGJSecret:           getEnv("BILLING_XGJ_SECRET", ""),
			ClockSkewSeconds:    parseInt(getEnv("BILLING_CLOCK_SKEW_SECONDS", "300")),
			DefaultValidityDays: parseInt(getEnv("BILLING_DEFAULT_VALIDITY_DAYS", "365")),
		},
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func parseInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 30
	}
	return n
}

func parseInt64(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 {
		return 1048576
	}
	return n
}
