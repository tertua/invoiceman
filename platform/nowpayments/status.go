package nowpayments

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/tertua/invoiceman/pkg/constants"
	"github.com/tertua/invoiceman/platform/gateway"
)

// StatusNotification fetches the live payment by id and renders it as the
// same NotificationResult an IPN produces, so background reconcile settles
// with exactly the webhook's rules.
func StatusNotification(ctx context.Context, cfg Config, paymentID string) (*gateway.NotificationResult, error) {
	if cfg.APIKey == "" {
		return nil, gateway.ErrNotConfigured
	}
	if strings.TrimSpace(paymentID) == "" {
		return nil, errors.New("nowpayments status: empty payment id")
	}
	confirmed, err := fetchPaymentStatus(ctx, cfg, paymentID)
	if err != nil {
		return nil, err
	}
	return paymentNotification(confirmed), nil
}

// paymentNotification renders a confirmed payment into the shared relay result.
func paymentNotification(confirmed *payment) *gateway.NotificationResult {
	paymentType := strings.ToLower(strings.TrimSpace(confirmed.PayCurrency))
	if paymentType == "" {
		paymentType = gateway.MethodCrypto
	}
	return &gateway.NotificationResult{
		OrderID:       confirmed.OrderID,
		TransactionID: confirmed.PaymentID,
		Status:        MapStatus(confirmed.Status),
		PaymentType:   paymentType,
		PaymentMethod: gateway.MethodCrypto,
		GrossMinor:    priceMinor(confirmed.PriceAmount, confirmed.PriceCurrency),
		GrossDecimal:  confirmed.PayAmount,
		Currency:      strings.ToUpper(strings.TrimSpace(confirmed.PayCurrency)),
	}
}

// fetchPaymentStatus returns the live payment via GET /v1/payment/{id}.
func fetchPaymentStatus(ctx context.Context, cfg Config, paymentID string) (*payment, error) {
	ctx, cancel := context.WithTimeout(ctx, constants.GatewayAPITimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.BaseURL()+"/payment/"+paymentID, http.NoBody)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", constants.UserAgent())
	httpReq.Header.Set("x-api-key", cfg.APIKey)

	resp, err := constants.DefaultHTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("nowpayments status: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, constants.MaxAPIResponseSize))
	if err != nil {
		return nil, fmt.Errorf("nowpayments status: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("nowpayments status: %w", gateway.NewProviderError("nowpayments", resp.StatusCode, truncate(string(raw), constants.MaxErrorBodyLog)))
	}
	p, err := decodePayment(raw)
	if err != nil {
		return nil, fmt.Errorf("nowpayments status decode: %w", err)
	}
	if p.PaymentID == "" {
		return nil, errors.New("nowpayments status: empty response")
	}
	return p, nil
}

// priceMinor converts a fiat price to minor units for local settlement.
// Only IDR is scaled 1:1 here; other fiats leave GrossMinor at zero and
// settlement credits the full balance on 'finished' (which guarantees the
// full expected amount arrived).
func priceMinor(amount decimal.Decimal, currency string) int64 {
	if strings.ToUpper(strings.TrimSpace(currency)) != "IDR" {
		return 0
	}
	return amount.Round(0).IntPart()
}
