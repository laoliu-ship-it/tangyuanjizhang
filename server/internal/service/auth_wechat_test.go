package service

import (
	"context"
	"errors"
	"testing"

	"fandianjizhang/server/config"
	"fandianjizhang/server/internal/dto"

	"github.com/golang-jwt/jwt/v5"
)

func newWeChatTestService() *authService {
	return &authService{
		cfg: &config.Config{
			JWT:    config.JWTConfig{Secret: "test-jwt-secret"},
			WeChat: config.WeChatConfig{AppID: "", Secret: ""},
		},
	}
}

func TestWeChatBindTokenRoundTrip(t *testing.T) {
	svc := newWeChatTestService()

	token, err := svc.generateWeChatBindToken("openid-1", "unionid-1")
	if err != nil {
		t.Fatalf("generate bind token: %v", err)
	}

	openID, unionID, err := svc.parseWeChatBindToken(token)
	if err != nil {
		t.Fatalf("parse bind token: %v", err)
	}
	if openID != "openid-1" || unionID != "unionid-1" {
		t.Fatalf("unexpected claims: openID=%q unionID=%q", openID, unionID)
	}
}

// bind token 使用派生密钥签名，不能当作业务登录 token 使用
func TestWeChatBindTokenIsNotLoginToken(t *testing.T) {
	svc := newWeChatTestService()

	tokenStr, err := svc.generateWeChatBindToken("openid-1", "")
	if err != nil {
		t.Fatalf("generate bind token: %v", err)
	}

	claims := &Claims{}
	_, err = jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(svc.cfg.JWT.Secret), nil
	})
	if err == nil {
		t.Fatal("bind token should not be accepted with the JWT secret")
	}
}

func TestWeChatBindTokenRejectsWrongScope(t *testing.T) {
	svc := newWeChatTestService()

	claims := WeChatBindClaims{
		Scope:  "login",
		OpenID: "openid-1",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(svc.wechatBindSecret())
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	if _, _, err := svc.parseWeChatBindToken(tokenStr); !errors.Is(err, ErrWeChatBindToken) {
		t.Fatalf("expected ErrWeChatBindToken, got %v", err)
	}
}

func TestWeChatLoginRequiresConfiguration(t *testing.T) {
	svc := newWeChatTestService()

	if _, err := svc.WeChatLogin(context.Background(), dto.WeChatLoginReq{Code: "code"}); !errors.Is(err, ErrWeChatNotConfigured) {
		t.Fatalf("expected ErrWeChatNotConfigured, got %v", err)
	}
}
