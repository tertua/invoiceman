package gateway

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// ErrNotConfigured is returned when a gateway has no credentials.
var ErrNotConfigured = errors.New("payment gateway is not configured")

// ErrUnknownGateway is returned for unregistered gateway names.
var ErrUnknownGateway = errors.New("unknown payment gateway")

// ErrInvalidSignature is returned when a webhook authenticity check fails.
var ErrInvalidSignature = errors.New("invalid signature")

// ErrInvalidPayload is returned when a webhook body cannot be decoded.
var ErrInvalidPayload = errors.New("invalid notification")

// Transaction statuses shared by all gateways and the relay payload.
// Gateways map their provider-specific states to these.
const (
	StatusPending         = "pending"
	StatusSuccess         = "success"
	StatusFailed          = "failed"
	StatusExpired         = "expired"
	StatusRefunded        = "refunded"
	StatusPartialRefunded = "partially_refunded"
)

// Gateway is implemented once per provider (midtrans, xendit, coinpayments...).
// Adding a gateway = new package under platform/ implementing this +
// one Register call. Controllers and relay never touch provider SDKs.
type Gateway interface {
	Name() string
	CreateTransaction(ctx context.Context, req *CreateTxRequest) (*CreateTxResponse, error)
	// ParseAndVerify decodes the raw webhook body and verifies its
	// authenticity (signature, HMAC, IP allowlist, or API fetch).
	ParseAndVerify(raw []byte) (*NotificationResult, error)
}

var (
	mu       sync.RWMutex
	registry = map[string]Gateway{}
)

// Register adds a gateway implementation. Called once per provider,
// typically from platform wiring at startup.
func Register(g Gateway) {
	mu.Lock()
	defer mu.Unlock()
	registry[g.Name()] = g
}

// Get returns the gateway by name (e.g. "midtrans").
func Get(name string) (Gateway, error) {
	mu.RLock()
	defer mu.RUnlock()
	g, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("payment gateway %q is not registered", name)
	}
	return g, nil
}

// Names lists registered gateway names for admin/status endpoints.
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(registry))
	for name := range registry {
		out = append(out, name)
	}
	return out
}
