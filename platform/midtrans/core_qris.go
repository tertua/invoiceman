package midtrans

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

	"github.com/tertua/invoiceman/pkg/constants"
	"github.com/tertua/invoiceman/platform/gateway"
)

// coreChargeAction is one entry in the Core API charge response's actions
// list (e.g. the generate-qr-code URL the payer scans).
type coreChargeAction struct {
	Name   string `json:"name"`
	Method string `json:"method"`
	URL    string `json:"url"`
}

// coreChargeResponse is the subset of a Core API POST /v2/charge body used
// for an on-page QRIS charge.
type coreChargeResponse struct {
	OrderID           string             `json:"order_id"`
	TransactionID     string             `json:"transaction_id"`
	TransactionStatus string             `json:"transaction_status"`
	PaymentType       string             `json:"payment_type"`
	ExpiryTime        string             `json:"expiry_time"`
	QRString          string             `json:"qr_string"`
	Actions           []coreChargeAction `json:"actions"`
}

// CoreURL returns the Core API charge endpoint.
func (c Config) CoreURL() string {
	if c.CoreBase != "" {
		return strings.TrimRight(c.CoreBase, "/")
	}
	if c.IsProd {
		return "https://api.midtrans.com/v2/charge"
	}
	return "https://api.sandbox.midtrans.com/v2/charge"
}

// qrisCorePayload builds the Core API charge body for QRIS settled by the
// Gopay acquirer. Snap cannot express qris.acquirer, which is why QRIS-on-page
// takes the Core API path instead of Snap.
func qrisCorePayload(orderID string, amountIDR int64) ([]byte, error) {
	return json.Marshal(map[string]any{
		"payment_type": "qris",
		"transaction_details": map[string]any{
			"order_id":     orderID,
			"gross_amount": amountIDR,
		},
		"qris": map[string]any{"acquirer": "gopay"},
	})
}

// CreateQRISCharge opens a QRIS charge through the Midtrans Core API with the
// Gopay acquirer and returns the QR image URL plus the QRIS expiry, so the
// payer scans it in the on-page widget. Token is intentionally empty: an
// on-page charge must never be mistaken for a Snap token by the client.
func CreateQRISCharge(ctx context.Context, cfg Config, orderID string, amountIDR int64) (*gateway.CreateTxResponse, error) {
	if cfg.ServerKey == "" {
		return nil, ErrNotConfigured
	}
	if orderID == "" || amountIDR <= 0 {
		return nil, errors.New("invalid order or amount")
	}
	body, err := qrisCorePayload(orderID, amountIDR)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, constants.GatewayAPITimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.CoreURL(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", constants.UserAgent())
	req.SetBasicAuth(cfg.ServerKey, "")

	resp, err := constants.DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("midtrans qris: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, constants.MaxAPIResponseSize))
	if err != nil {
		return nil, fmt.Errorf("midtrans qris: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("midtrans qris: %w", gateway.NewProviderError("midtrans", resp.StatusCode, truncate(string(raw), constants.MaxErrorBodyLog)))
	}
	out := &coreChargeResponse{}
	if err := json.Unmarshal(raw, out); err != nil {
		return nil, fmt.Errorf("midtrans qris decode: %w", err)
	}
	qrURL := qrActionURL(out.Actions)
	if qrURL == "" {
		return nil, errors.New("midtrans qris: no qr action in response")
	}
	return &gateway.CreateTxResponse{
		PaymentURL:    qrURL,
		QRString:      strings.TrimSpace(out.QRString),
		ExpiresAt:     normalizeExpiry(out.ExpiryTime),
		PaymentMethod: gateway.MethodQRIS,
	}, nil
}

// qrActionURL picks the scannable QR image from the charge actions.
func qrActionURL(actions []coreChargeAction) string {
	for _, a := range actions {
		if a.URL != "" && (a.Name == "generate-qr-code" || strings.Contains(a.Name, "qr")) {
			return strings.TrimSpace(a.URL)
		}
	}
	return ""
}

// normalizeExpiry converts Midtrans' WIB "2006-01-02 15:04:05" expiry_time
// into RFC3339 so the browser can build an exact countdown regardless of the
// viewer's timezone. Unknown shapes are passed through unchanged.
func normalizeExpiry(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.FixedZone("WIB", 7*60*60)); err == nil {
		return t.Format(time.RFC3339)
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Format(time.RFC3339)
	}
	return s
}
