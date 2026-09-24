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
	"github.com/tertua/invoiceman/platform/gateway"
)

// payment is a NOWPayments payment object (IPN body and status fetch share
// the shape). Amounts decode as exact decimals so crypto precision is kept.
type payment struct {
	PaymentID     string
	OrderID       string
	Status        string
	PriceAmount   decimal.Decimal
	PriceCurrency string
	PayAmount     string
	PayCurrency   string
}

// MapStatus maps a NOWPayments payment_status to a shared relay status.
// partially_paid stays pending: funds are still due and must not settle.
func MapStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "finished":
		return gateway.StatusSuccess
	case "failed":
		return gateway.StatusFailed
	case "expired":
		return gateway.StatusExpired
	case "refunded":
		return gateway.StatusRefunded
	default: // waiting, confirming, confirmed, sending, partially_paid, unknown
		return gateway.StatusPending
	}
}

// VerifyNotification decodes an IPN body and verifies it with a live status
// fetch. The fetched payment_id and order_id must match the callback.
func VerifyNotification(ctx context.Context, cfg Config, raw []byte) (*gateway.NotificationResult, error) {
	if cfg.APIKey == "" {
		return nil, gateway.ErrNotConfigured
	}
	announced, err := decodePayment(raw)
	if err != nil {
		return nil, err
	}
	if announced.PaymentID == "" || announced.OrderID == "" {
		return nil, gateway.ErrInvalidPayload
	}

	confirmed, err := fetchPaymentStatus(ctx, cfg, announced.PaymentID)
	if err != nil {
		return nil, err
	}
	if confirmed.PaymentID != announced.PaymentID || confirmed.OrderID != announced.OrderID {
		return nil, gateway.ErrInvalidPayload
	}

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
	}, nil
}

// decodePayment parses a payment-shaped JSON document.
func decodePayment(raw []byte) (*payment, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var doc map[string]any
	if err := dec.Decode(&doc); err != nil {
		return nil, gateway.ErrInvalidPayload
	}
	price, _ := toDecimal(doc["price_amount"])
	return &payment{
		PaymentID:     stringValue(doc["payment_id"]),
		OrderID:       stringValue(doc["order_id"]),
		Status:        stringValue(doc["payment_status"]),
		PriceAmount:   price,
		PriceCurrency: strings.ToUpper(stringValue(doc["price_currency"])),
		PayAmount:     stringValue(doc["pay_amount"]),
		PayCurrency:   stringValue(doc["pay_currency"]),
	}, nil
}

// toDecimal converts numeric JSON values (json.Number, float64, numeric
// strings) to an exact decimal without panicking on missing or mistyped input.
func toDecimal(v any) (decimal.Decimal, error) {
	switch t := v.(type) {
	case nil:
		return decimal.Zero, nil
	case json.Number:
		return decimal.NewFromString(t.String())
	case float64:
		return decimal.NewFromFloat(t), nil
	case string:
		return decimal.NewFromString(strings.TrimSpace(t))
	default:
		return decimal.Zero, errors.New("not a number")
	}
}

// fetchPaymentStatus returns the live payment via GET /v1/payment/{id}.
func fetchPaymentStatus(ctx context.Context, cfg Config, paymentID string) (*payment, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.BaseURL()+"/payment/"+paymentID, http.NoBody)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("x-api-key", cfg.APIKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("nowpayments status: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("nowpayments status: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("nowpayments status: status %d: %s", resp.StatusCode, truncate(string(raw), 300))
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
