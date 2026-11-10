package xgjclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListProductsContract(t *testing.T) {
	var gotPath, gotSig, gotTS, gotNonce string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotSig = r.Header.Get("X-VP-Signature")
		gotTS = r.Header.Get("X-VP-Timestamp")
		gotNonce = r.Header.Get("X-VP-Nonce")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"products": []map[string]any{{
					"sku_code": "sample_goods_600", "name": "示例商品", "wx_product_id": "wp1",
					"price_fen": 600, "delivery_payload": `{"any":"opaque"}`, "status": "listed",
				}},
				"iap_enabled": true,
				"env":         0,
			},
		})
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, AppSlug: "hehuojizhangzhushou", Secret: "secret"})
	products, err := c.ListProducts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(products.Products) != 1 || products.Products[0].SKUCode != "sample_goods_600" || products.Products[0].PriceFen != 600 {
		t.Fatalf("products = %+v", products)
	}
	if products.Products[0].DeliveryPayload != `{"any":"opaque"}` {
		t.Fatalf("delivery payload changed: %s", products.Products[0].DeliveryPayload)
	}
	if !products.IAPEnabled {
		t.Fatal("iap_enabled not parsed")
	}
	if gotPath != "/api/hehuojizhangzhushou/virtualpay/products/s2s" {
		t.Fatalf("path = %s", gotPath)
	}
	if gotTS == "" || gotNonce == "" || gotSig == "" {
		t.Fatal("missing service signature headers")
	}
	if !strings.EqualFold(gotSig, Sign("secret", gotTS, gotNonce, `{}`)) {
		t.Fatal("signature mismatch")
	}
}

func TestPrepareOrderContract(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/app/virtualpay/orders/prepare-s2s" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"out_trade_no": "vp12345678", "mode": "short_series_goods",
				"signData": `{}`, "paySig": "p", "signature": "s", "env": 0,
			},
		})
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL, AppSlug: "app", Secret: "secret"})
	resp, err := c.PrepareOrder(context.Background(), "ip1", "code")
	if err != nil {
		t.Fatal(err)
	}
	if resp.OutTradeNo != "vp12345678" || resp.PaySig != "p" {
		t.Fatalf("resp = %+v", resp)
	}
}
