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

	"github.com/shopspring/decimal"
	"github.com/tertua/invoiceman/pkg/constants"
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
	amount, err := decimal.NewFromString(strings.TrimSpace(req.PriceAmount))
	if err != nil {
		return nil, fmt.Errorf("nowpayments payment: invalid amount %q: %w", req.PriceAmount, err)
	}
	body, err := json.Marshal(map[string]interface{}{
		"price_amount":      json.Number(amount.String()),
		"price_currency":    req.PriceCurrency,
		"pay_currency":      req.PayCurrency,
		"order_id":          req.OrderID,
		"order_description": "Payment " + req.OrderID,
		"ipn_callback_url":  cfg.CallbackURL(),
	})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, constants.GatewayPaymentTimeout)
	defer cancel()
	raw, status, err := postWithRetry(ctx, cfg, "/payment", body)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		if status == http.StatusTooManyRequests {
			return nil, fmt.Errorf("nowpayments payment: %w", gateway.ErrRateLimited)
		}
		if isAmountMinimalError(raw) {
			return nil, fmt.Errorf("nowpayments payment: %w: %s", gateway.ErrAmountBelowMinimum, truncate(string(raw), constants.MaxErrorBodyLog))
		}
		return nil, fmt.Errorf("nowpayments payment: status %d: %s", status, truncate(string(raw), constants.MaxErrorBodyLog))
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

// isAmountMinimalError reports whether the provider rejected the charge only
// because it is below NOWPayments' minimum for the requested pay currency.
func isAmountMinimalError(raw []byte) bool {
	var body struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return false
	}
	return body.Code == "AMOUNT_MINIMAL_ERROR"
}

// postWithRetry posts one JSON body to a NOWPayments endpoint. A 429 answer
// is retried honoring the Retry-After header (capped) with jittered backoff
// fallback, so a transient rate limit does not fail the caller; the final
// status is returned once the limit persists.
func postWithRetry(ctx context.Context, cfg Config, path string, body []byte) (raw []byte, status int, err error) {
	for attempt := 1; ; attempt++ {
		var header http.Header
		raw, status, header, err = postOnce(ctx, cfg, path, body)
		if err != nil {
			return nil, 0, err
		}
		if status != http.StatusTooManyRequests || attempt == constants.NowpaymentsMaxAttempts {
			return raw, status, nil
		}
		select {
		case <-ctx.Done():
			return nil, 0, fmt.Errorf("nowpayments payment: %w", ctx.Err())
		case <-time.After(constants.RetryDelayWithHeader(header, attempt)):
		}
	}
}

func postOnce(ctx context.Context, cfg Config, path string, body []byte) (raw []byte, status int, header http.Header, err error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.BaseURL()+path, bytes.NewReader(body))
	if err != nil {
		return nil, 0, nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", constants.UserAgent())
	httpReq.Header.Set("x-api-key", cfg.APIKey)
	resp, err := constants.PaymentHTTPClient.Do(httpReq)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("nowpayments payment: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err = io.ReadAll(io.LimitReader(resp.Body, constants.MaxAPIResponseSize))
	if err != nil {
		return nil, 0, nil, fmt.Errorf("nowpayments payment: %w", err)
	}
	return raw, resp.StatusCode, resp.Header, nil
}

// CreateTransaction creates a NOWPayments payment. Relayed intents reached
// without a pay currency use the hosted crypto invoice; public-pay direct USDT
// (PayCurrency set) returns an on-page deposit address instead of a redirect.
//
// Dev path — NOWPayments pay currency (scaffold):
//
//	[done]  one pay currency on purpose: USDT TRC20 via direct payment
//	[later] replace the USDT prefix check with a table of supported networks
//	        (usdterc20, usdtbep20, ...) and pass the network through
//	        CreateTxRequest
//
// End dev path
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
		return &gateway.CreateTxResponse{Token: dp.PaymentID, Address: dp.PayAddress, PaymentMethod: gateway.MethodCrypto, RawPayload: dp.PayAmount, PayCurrency: dp.PayCurrency, ExpiresAt: dp.ExpiresAt}, nil
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
