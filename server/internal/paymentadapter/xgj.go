// Package paymentadapter 把汤圆本地窄 PaymentGateway 映射到小管家 HTTP 契约。
// 核心 BillingService 不依赖本包；本包只在 main 装配。
package paymentadapter

import (
	"context"
	"encoding/json"

	"fandianjizhang/server/internal/service"
	"fandianjizhang/server/internal/xgjclient"
)

type XGJ struct {
	client *xgjclient.Client
}

func NewXGJ(client *xgjclient.Client) *XGJ { return &XGJ{client: client} }

func (a *XGJ) ListProducts(ctx context.Context) ([]service.PaymentProduct, service.PaymentCapabilities, error) {
	list, err := a.client.ListProducts(ctx)
	if err != nil {
		return nil, service.PaymentCapabilities{}, err
	}
	products := make([]service.PaymentProduct, 0, len(list.Products))
	for _, p := range list.Products {
		products = append(products, service.PaymentProduct{
			SKUCode: p.SKUCode, Name: p.Name, WxProductID: p.WxProductID, PriceFen: p.PriceFen,
			Status: p.Status, DeliveryPayload: json.RawMessage(p.DeliveryPayload), Sort: p.Sort,
		})
	}
	return products, service.PaymentCapabilities{IAPEnabled: list.IAPEnabled, Env: list.Env}, nil
}

func (a *XGJ) CreateIntent(ctx context.Context, req service.PaymentIntentRequest) (*service.PaymentIntentResult, error) {
	resp, err := a.client.CreateIntent(ctx, xgjclient.CreateIntentRequest{
		BusinessReference: req.BusinessReference, SKUCode: req.SKUCode, Quantity: req.Quantity,
		AmountFen: req.AmountFen, PayerOpenID: req.PayerOpenID,
	})
	if err != nil {
		return nil, err
	}
	return &service.PaymentIntentResult{IntentRef: resp.IntentRef, ExpiresAt: resp.ExpiresAt}, nil
}

func (a *XGJ) PrepareOrder(ctx context.Context, intentRef, code string) (*service.PaymentPrepareParams, error) {
	resp, err := a.client.PrepareOrder(ctx, intentRef, code)
	if err != nil {
		return nil, err
	}
	return &service.PaymentPrepareParams{
		OutTradeNo: resp.OutTradeNo, Mode: resp.Mode, SignData: resp.SignData,
		PaySig: resp.PaySig, Signature: resp.Signature, Env: resp.Env,
	}, nil
}
