package midtrans

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/tertua/invoiceman/pkg/configs"
	"github.com/tertua/invoiceman/pkg/constants"
)

// ErrNotConfigured is returned when the Midtrans server key is missing.
var ErrNotConfigured = errors.New("payment gateway is not configured")

// Config holds Midtrans credentials. All values come from env;
// no domain or key is hardcoded.
type Config struct {
	ServerKey string
	ClientKey string
	IsProd    bool
	// BaseURL overrides the Snap endpoint (used by tests).
	BaseURL string
	// CoreBase overrides the Core API base (used by tests).
	CoreBase string
}

// FromEnv reads Midtrans config from the central config.
func FromEnv() Config {
	cfg := configs.Get().Midtrans
	return Config{
		ServerKey: cfg.ServerKey,
		ClientKey: cfg.ClientKey,
		IsProd:    cfg.IsProd,
		BaseURL:   cfg.SnapBase,
		// Test-only override, mirrors SnapBase. Read straight from the
		// environment so the frozen configs file does not grow for it.
		CoreBase: strings.TrimSpace(os.Getenv("MIDTRANS_CORE_BASE_URL")),
	}
}

// SnapURL returns the Snap transaction endpoint.
func (c Config) SnapURL() string {
	if c.BaseURL != "" {
		return strings.TrimRight(c.BaseURL, "/")
	}
	if c.IsProd {
		return "https://app.midtrans.com/snap/v1/transactions"
	}
	return "https://app.sandbox.midtrans.com/snap/v1/transactions"
}

// SnapResponse is the subset of the Snap response used by the relay.
type SnapResponse struct {
	Token       string `json:"token"`
	RedirectURL string `json:"redirect_url"`
}

// CreateSnapTransaction creates a Snap transaction for orderID/amountIDR.
// Customer details are best-effort; empty email/phone are omitted.
func CreateSnapTransaction(ctx context.Context, cfg Config, orderID string, amountIDR int64, email, phone string, methods []string) (*SnapResponse, error) {
	if cfg.ServerKey == "" {
		return nil, ErrNotConfigured
	}
	if orderID == "" || amountIDR <= 0 {
		return nil, errors.New("invalid order or amount")
	}

	customer := map[string]string{}
	if strings.TrimSpace(email) != "" {
		customer["email"] = strings.TrimSpace(email)
	}
	if strings.TrimSpace(phone) != "" {
		customer["phone"] = strings.TrimSpace(phone)
	}

	body, err := snapPayload(orderID, amountIDR, customer, methods)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, constants.GatewayAPITimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.SnapURL(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", constants.UserAgent())
	req.SetBasicAuth(cfg.ServerKey, "")

	resp, err := constants.DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("midtrans snap: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, constants.MaxAPIResponseSize))
	if err != nil {
		return nil, fmt.Errorf("midtrans snap: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("midtrans snap: status %d: %s", resp.StatusCode, truncate(string(raw), constants.MaxErrorBodyLog))
	}
	out := &SnapResponse{}
	if err := json.Unmarshal(raw, out); err != nil {
		return nil, fmt.Errorf("midtrans snap decode: %w", err)
	}
	if out.Token == "" && out.RedirectURL == "" {
		return nil, errors.New("midtrans snap: empty response")
	}
	return out, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
