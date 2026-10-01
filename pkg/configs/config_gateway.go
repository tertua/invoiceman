package configs

import "time"

// GatewayConfig holds provider-call timeouts, retry policy and polling cadence.
// Each field has an accessor that converts to time.Duration so call sites stay
// import-light (they read configs, not time).
type GatewayConfig struct {
	ReconcileMinutes         int // GATEWAY_RECONCILE_MINUTES
	APITimeoutSec            int // GATEWAY_API_TIMEOUT_SECONDS
	PaymentTimeoutSec        int // GATEWAY_PAYMENT_TIMEOUT_SECONDS
	QRFetchTimeoutSec        int // QR_FETCH_TIMEOUT_SECONDS
	WebhookForwardTimeoutSec int // WEBHOOK_FORWARD_TIMEOUT_SECONDS
	MaxAttempts              int // GATEWAY_MAX_ATTEMPTS
	RetryBaseSec             int // GATEWAY_RETRY_BASE_SECONDS
	RetryMaxSec              int // GATEWAY_RETRY_MAX_SECONDS
	WebhookRetryMinutes      int // WEBHOOK_RETRY_MINUTES
}

// APITimeout is the per-request timeout for gateway API operations.
func (g GatewayConfig) APITimeout() time.Duration {
	return time.Duration(g.APITimeoutSec) * time.Second
}

// PaymentTimeout bounds payment-creation calls that may take longer.
func (g GatewayConfig) PaymentTimeout() time.Duration {
	return time.Duration(g.PaymentTimeoutSec) * time.Second
}

// QRFetchTimeout bounds QR image fetches.
func (g GatewayConfig) QRFetchTimeout() time.Duration {
	return time.Duration(g.QRFetchTimeoutSec) * time.Second
}

// WebhookForwardTimeout bounds forwarding webhooks downstream.
func (g GatewayConfig) WebhookForwardTimeout() time.Duration {
	return time.Duration(g.WebhookForwardTimeoutSec) * time.Second
}

// RetryBase is the linear backoff base before jitter.
func (g GatewayConfig) RetryBase() time.Duration {
	return time.Duration(g.RetryBaseSec) * time.Second
}

// RetryMax is the backoff ceiling and the Retry-After cap (one knob).
func (g GatewayConfig) RetryMax() time.Duration {
	return time.Duration(g.RetryMaxSec) * time.Second
}

// ReconcileAge spaces provider status polls per stale transaction.
func (g GatewayConfig) ReconcileAge() time.Duration {
	return time.Duration(g.ReconcileMinutes) * time.Minute
}

// WebhookRetry is the manual retry delay for relay deliveries.
func (g GatewayConfig) WebhookRetry() time.Duration {
	return time.Duration(g.WebhookRetryMinutes) * time.Minute
}
