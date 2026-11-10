// Package xgjclient 是饭店记账到小管家虚拟支付的服务间客户端。
// 使用独立 HMAC 凭据、时间窗口与 nonce 防重放；不记录/回显 secret。
package xgjclient

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	BaseURL string
	AppSlug string
	Secret  string
	Timeout time.Duration
}

type Client struct {
	cfg  Config
	http *http.Client
}

func New(cfg Config) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &Client{cfg: cfg, http: &http.Client{Timeout: cfg.Timeout}}
}

type CreateIntentRequest struct {
	BusinessReference string `json:"business_reference"`
	SKUCode           string `json:"sku_code"`
	Quantity          int    `json:"quantity"`
	AmountFen         int64  `json:"amount_fen"`
	PayerOpenID       string `json:"payer_open_id,omitempty"`
	ClientNonce       string `json:"client_nonce,omitempty"`
}

type CreateIntentResponse struct {
	IntentRef string    `json:"intent_ref"`
	ExpiresAt time.Time `json:"expires_at"`
	SKUCode   string    `json:"sku_code"`
	Quantity  int       `json:"quantity"`
	AmountFen int64     `json:"amount_fen"`
}

type PrepareResponse struct {
	OutTradeNo string `json:"out_trade_no"`
	Mode       string `json:"mode"`
	SignData   string `json:"signData"`
	PaySig     string `json:"paySig"`
	Signature  string `json:"signature"`
	Env        int    `json:"env"`
}

type Product struct {
	SKUCode         string `json:"sku_code"`
	Name            string `json:"name"`
	WxProductID     string `json:"wx_product_id"`
	PriceFen        int64  `json:"price_fen"`
	DeliveryPayload string `json:"delivery_payload"` // 不透明 JSON 字符串，由接入方解释
	Status          string `json:"status"`
	Sort            int    `json:"sort"`
}

type ProductList struct {
	Products   []Product `json:"products"`
	IAPEnabled bool      `json:"iap_enabled"`
	Env        int       `json:"env"`
}

// ListProducts 读取小管家权威商品列表（含上下架/价格/权益与 iOS IAP 能力）。
func (c *Client) ListProducts(ctx context.Context) (*ProductList, error) {
	var out ProductList
	if err := c.post(ctx, "/products/s2s", map[string]any{}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error"`
}

// CreateIntent 注册购买意图；到账账本由饭店记账后端决定，客户端无法指定。
func (c *Client) CreateIntent(ctx context.Context, req CreateIntentRequest) (*CreateIntentResponse, error) {
	var out CreateIntentResponse
	if err := c.post(ctx, "/intents", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PrepareOrder 代理小程序下单（code 由小程序本次 wx.login 获取）。
func (c *Client) PrepareOrder(ctx context.Context, intentRef, code string) (*PrepareResponse, error) {
	var out PrepareResponse
	body := map[string]string{"intent_ref": intentRef, "code": code}
	if err := c.post(ctx, "/orders/prepare-s2s", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) post(ctx context.Context, path string, payload, out any) error {
	if strings.TrimSpace(c.cfg.BaseURL) == "" || strings.TrimSpace(c.cfg.AppSlug) == "" || strings.TrimSpace(c.cfg.Secret) == "" {
		return fmt.Errorf("虚拟支付服务间配置不完整")
	}
	u, err := url.Parse(c.cfg.BaseURL)
	if err != nil {
		return fmt.Errorf("虚拟支付服务地址非法: %w", err)
	}
	if u.Scheme != "https" && !isLoopback(u.Hostname()) {
		return fmt.Errorf("虚拟支付服务地址必须为 https")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	ts := fmt.Sprintf("%d", time.Now().Unix())
	nonce, err := randomNonce()
	if err != nil {
		return err
	}
	sig := Sign(c.cfg.Secret, ts, nonce, string(raw))
	endpoint := strings.TrimRight(c.cfg.BaseURL, "/") + "/api/" + url.PathEscape(c.cfg.AppSlug) + "/virtualpay" + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-VP-Timestamp", ts)
	req.Header.Set("X-VP-Nonce", nonce)
	req.Header.Set("X-VP-Signature", sig)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("请求小管家失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	var env envelope
	if err := json.Unmarshal(respBody, &env); err != nil {
		return fmt.Errorf("小管家响应解析失败 (http %d)", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || !env.Success {
		msg := env.Error
		if msg == "" {
			msg = fmt.Sprintf("http %d", resp.StatusCode)
		}
		return fmt.Errorf("小管家返回错误: %s", msg)
	}
	if len(env.Data) == 0 {
		return fmt.Errorf("小管家响应缺少 data")
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return fmt.Errorf("小管家 data 解析失败: %w", err)
	}
	return nil
}

// Sign 与 xiaoguanjia 服务间验签一致：hex(hmac_sha256(secret, ts+"\n"+nonce+"\n"+body))。
func Sign(secret, timestamp, nonce, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte{'\n'})
	mac.Write([]byte(nonce))
	mac.Write([]byte{'\n'})
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

func randomNonce() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func isLoopback(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
