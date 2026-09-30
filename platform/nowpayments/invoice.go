package nowpayments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/tertua/tupay/pkg/configs"
	"github.com/tertua/tupay/pkg/constants"
	"github.com/tertua/tupay/platform/gateway"
)

// Invoice is the subset of a NOWPayments invoice used by the relay.
type Invoice struct {
	ID         string
	InvoiceURL string
}

// CreateInvoice creates a NOWPayments hosted invoice: AmountDecimal wins, else AmountMinor.
func CreateInvoice(ctx context.Context, cfg Config, req *gateway.CreateTxRequest) (*Invoice, error) {
	if cfg.APIKey == "" {
		return nil, gateway.ErrNotConfigured
	}
	if req == nil || strings.TrimSpace(req.OrderID) == "" {
		return nil, errors.New("nowpayments: order id is required")
	}
	priceAmount, priceCurrency, err := priceFromRequest(req)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(map[string]any{
		"price_amount":      json.Number(priceAmount.String()),
		"price_currency":    priceCurrency,
		"order_id":          strings.TrimSpace(req.OrderID),
		"order_description": "Payment " + strings.TrimSpace(req.OrderID),
		"ipn_callback_url":  cfg.CallbackURL(),
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, configs.Get().Gateway.APITimeout())
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.BaseURL()+"/invoice", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", constants.UserAgent())
	httpReq.Header.Set("x-api-key", cfg.APIKey)

	resp, err := constants.DefaultHTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("nowpayments invoice: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, constants.MaxAPIResponseSize))
	if err != nil {
		return nil, fmt.Errorf("nowpayments invoice: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("nowpayments invoice: %w", gateway.NewProviderError("nowpayments", resp.StatusCode, truncate(string(raw), constants.MaxErrorBodyLog)))
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("nowpayments invoice decode: %w", err)
	}
	id := stringValue(decoded["id"])
	url, _ := decoded["invoice_url"].(string)
	if id == "" || strings.TrimSpace(url) == "" {
		return nil, errors.New("nowpayments invoice: empty response")
	}
	return &Invoice{ID: id, InvoiceURL: strings.TrimSpace(url)}, nil
}

// priceFromRequest resolves the fiat price for an invoice request.
func priceFromRequest(req *gateway.CreateTxRequest) (amount decimal.Decimal, currency string, err error) {
	currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		currency = "IDR"
	}
	if raw := strings.TrimSpace(req.AmountDecimal); raw != "" {
		value, parseErr := decimal.NewFromString(raw)
		if parseErr != nil || !value.GreaterThan(decimal.Zero) {
			return decimal.Zero, "", errors.New("nowpayments: invalid amount_decimal")
		}
		return value, currency, nil
	}
	if req.AmountMinor <= 0 {
		return decimal.Zero, "", errors.New("nowpayments: amount_decimal or amount_minor is required")
	}
	return decimal.NewFromInt(req.AmountMinor), currency, nil
}

// stringValue converts string or numeric JSON values to string.
func stringValue(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case json.Number:
		return strings.TrimSpace(t.String())
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
