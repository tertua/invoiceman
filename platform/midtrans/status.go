package midtrans

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/tertua/invoiceman/pkg/constants"
)

// TxStatus is the subset of the Core API transaction status used to
// reconcile a stored gateway transaction. Status uses the same relay
// vocabulary as webhook notifications (see MapStatus).
type TxStatus struct {
	OrderID       string
	Status        string
	TransactionID string
	PaymentType   string
	GrossAmount   decimal.Decimal
}

// statusResponse decodes the Core API GET /v2/{order_id}/status body.
type statusResponse struct {
	OrderID           string `json:"order_id"`
	TransactionStatus string `json:"transaction_status"`
	TransactionID     string `json:"transaction_id"`
	PaymentType       string `json:"payment_type"`
	GrossAmount       string `json:"gross_amount"`
}

// FetchStatus polls the Midtrans Core API for one order. A non-2xx response
// (unknown order, auth failure, rate limit) is an error: the caller keeps
// the stored status and retries on a later tick.
func FetchStatus(ctx context.Context, cfg Config, orderID string) (*TxStatus, error) {
	if cfg.ServerKey == "" {
		return nil, ErrNotConfigured
	}
	if strings.TrimSpace(orderID) == "" {
		return nil, errors.New("midtrans status: empty order id")
	}
	base := "https://api.sandbox.midtrans.com"
	if cfg.IsProd {
		base = "https://api.midtrans.com"
	}
	ctx, cancel := context.WithTimeout(ctx, constants.GatewayAPITimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/v2/"+orderID+"/status", http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", constants.UserAgent())
	req.SetBasicAuth(cfg.ServerKey, "")

	resp, err := constants.DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("midtrans status: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, constants.MaxAPIResponseSize))
	if err != nil {
		return nil, fmt.Errorf("midtrans status: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("midtrans status: status %d: %s", resp.StatusCode, truncate(string(raw), constants.MaxErrorBodyLog))
	}
	out := &statusResponse{}
	if err := json.Unmarshal(raw, out); err != nil {
		return nil, fmt.Errorf("midtrans status decode: %w", err)
	}
	gross, err := decimal.NewFromString(strings.TrimSpace(out.GrossAmount))
	if err != nil {
		return nil, fmt.Errorf("midtrans status decode: %w", err)
	}
	return &TxStatus{
		OrderID:       out.OrderID,
		Status:        MapStatus(out.TransactionStatus),
		TransactionID: out.TransactionID,
		PaymentType:   out.PaymentType,
		GrossAmount:   gross,
	}, nil
}
