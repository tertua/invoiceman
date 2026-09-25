package nowpayments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tertua/invoiceman/platform/gateway"
)

type DirectPayment struct {
	PaymentID     string `json:"payment_id"`
	PayAddress    string `json:"pay_address"`
	PayAmount     string `json:"pay_amount"`
	PayCurrency   string `json:"pay_currency"`
	PriceAmount   string `json:"price_amount"`
	PriceCurrency string `json:"price_currency"`
	OrderID       string `json:"order_id"`
	ExpiresAt     string `json:"expires_at,omitempty"`
}

func CreateDirectPayment(ctx context.Context, cfg Config, req *DirectPaymentRequest) (*DirectPayment, error) {
	if cfg.APIKey == "" {
		return nil, ErrNotConfigured
	}
	body, err := json.Marshal(map[string]interface{}{
		"price_amount":      req.PriceAmount,
		"price_currency":    req.PriceCurrency,
		"pay_currency":      req.PayCurrency,
		"order_id":          req.OrderID,
		"order_description": "Payment " + req.OrderID,
		"ipn_callback_url":  cfg.CallbackURL(),
	})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.BaseURL()+"/payment", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("x-api-key", cfg.APIKey)
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("nowpayments payment: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("nowpayments payment: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("nowpayments payment: status %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("nowpayments payment decode: %w", err)
	}
	pid := stringValue(decoded["payment_id"])
	addr := stringValue(decoded["pay_address"])
	amt := stringValue(decoded["pay_amount"])
	pcur := stringValue(decoded["pay_currency"])
	prAmt := stringValue(decoded["price_amount"])
	prCur := stringValue(decoded["price_currency"])
	oid := stringValue(decoded["order_id"])
	if pid == "" || addr == "" || amt == "" {
		return nil, errors.New("nowpayments payment: missing required fields")
	}
	out := &DirectPayment{PaymentID: pid, PayAddress: addr, PayAmount: amt, PayCurrency: pcur, PriceAmount: prAmt, PriceCurrency: prCur, OrderID: oid}
	if exp, ok := decoded["expires_at"].(string); ok && exp != "" {
		out.ExpiresAt = exp
	}
	return out, nil
}

type DirectPaymentRequest struct {
	OrderID       string
	PriceAmount   string
	PriceCurrency string
	PayCurrency   string
}

// CreateTransaction creates a NOWPayments payment. Relayed intents reached
// without a pay currency use the hosted crypto invoice; public-pay direct USDT
// (PayCurrency set) returns an on-page deposit address instead of a redirect.
func (Gateway) CreateTransaction(ctx context.Context, req *gateway.CreateTxRequest) (*gateway.CreateTxResponse, error) {
	cfg := FromEnv()
	if payCcy := strings.ToUpper(strings.TrimSpace(req.PayCurrency)); payCcy != "" {
		if !strings.HasPrefix(payCcy, "USDT") {
			return nil, errors.New("nowpayments: unsupported pay currency")
		}
		dp, err := CreateDirectPayment(ctx, cfg, &DirectPaymentRequest{
			OrderID:       req.OrderID,
			PriceAmount:   req.AmountDecimal,
			PriceCurrency: strings.ToUpper(strings.TrimSpace(req.Currency)),
			PayCurrency:   payCcy,
		})
		if err != nil {
			if errors.Is(err, ErrNotConfigured) {
				return nil, gateway.ErrNotConfigured
			}
			return nil, err
		}
		return &gateway.CreateTxResponse{Token: dp.PaymentID, Address: dp.PayAddress, PaymentMethod: gateway.MethodCrypto, RawPayload: dp.PayAmount, ExpiresAt: dp.ExpiresAt}, nil
	}
	inv, err := CreateInvoice(ctx, cfg, req)
	if err != nil {
		if errors.Is(err, ErrNotConfigured) {
			return nil, gateway.ErrNotConfigured
		}
		return nil, err
	}
	return nowPaymentsResponse(inv), nil
}
