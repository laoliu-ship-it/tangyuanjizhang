package handler

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fandianjizhang/server/config"
	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/service"
	"fandianjizhang/server/internal/xgjclient"
	"github.com/gin-gonic/gin"
)

type fakeBillingService struct {
	service.BillingService
	delivers int
	refunds  int
}

func (f *fakeBillingService) ProcessDelivery(context.Context, *model.BillingDeliveryEvent) (bool, error) {
	f.delivers++
	return false, nil
}

func (f *fakeBillingService) ProcessRefund(context.Context, *model.BillingDeliveryEvent) (bool, error) {
	f.refunds++
	return false, nil
}

func webhookRequest(t *testing.T, f *fakeBillingService, handlerSecret, eventType string, signed bool) *httptest.ResponseRecorder {
	return webhookRequestSignedWith(t, f, handlerSecret, handlerSecret, eventType, signed)
}

func webhookRequestSignedWith(t *testing.T, f *fakeBillingService, handlerSecret, signSecret, eventType string, signed bool) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	body := []byte(fmt.Sprintf(`{"event_key":"e1","event_type":%q,"business_reference":"b1","delivery_payload":{"credit_amount":600,"validity_days":365}}`, eventType))
	req := httptest.NewRequest(http.MethodPost, "/api/internal/billing/delivery", bytes.NewReader(body))
	ts := fmt.Sprintf("%d", time.Now().Unix())
	nonce := "n1"
	if signed {
		req.Header.Set("X-VP-Timestamp", ts)
		req.Header.Set("X-VP-Nonce", nonce)
		req.Header.Set("X-VP-Signature", xgjclient.Sign(signSecret, ts, nonce, string(body)))
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	h := NewBillingHandler(f, config.BillingConfig{XGJSecret: handlerSecret, ClockSkewSeconds: 300})
	h.Webhook(c)
	return w
}

func TestWebhookUnknownEventDoesNotDeliver(t *testing.T) {
	f := &fakeBillingService{}
	w := webhookRequest(t, f, "secret", "something_unknown", true)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if f.delivers != 0 || f.refunds != 0 {
		t.Fatalf("unknown event must not process: deliver=%d refund=%d", f.delivers, f.refunds)
	}
}

func TestWebhookDeliverRequiresSignature(t *testing.T) {
	f := &fakeBillingService{}
	w := webhookRequest(t, f, "secret", "billing_deliver", false)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if f.delivers != 0 {
		t.Fatal("unsigned webhook must not deliver")
	}
}

func TestWebhookBadSignatureDoesNotDeliver(t *testing.T) {
	f := &fakeBillingService{}
	w := webhookRequestSignedWith(t, f, "secret", "other-secret", "billing_deliver", true)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if f.delivers != 0 {
		t.Fatal("bad signature must not deliver")
	}
}

func TestWebhookRoutesKnownEvents(t *testing.T) {
	f := &fakeBillingService{}
	w := webhookRequest(t, f, "secret", "billing_deliver", true)
	if w.Code != http.StatusOK || f.delivers != 1 || f.refunds != 0 {
		t.Fatalf("deliver route failed: status=%d deliver=%d refund=%d", w.Code, f.delivers, f.refunds)
	}
	f2 := &fakeBillingService{}
	w = webhookRequest(t, f2, "secret", "billing_refund", true)
	if w.Code != http.StatusOK || f2.refunds != 1 || f2.delivers != 0 {
		t.Fatalf("refund route failed: status=%d deliver=%d refund=%d", w.Code, f2.delivers, f2.refunds)
	}
}
