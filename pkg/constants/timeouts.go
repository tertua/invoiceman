package constants

import (
	"fmt"
	"net/http"
	"time"
)

// Version is the application version, injected at build time via ldflags.
// Falls back to "dev" if not set.
var Version = "dev"

// UserAgent returns the HTTP User-Agent header value for external API calls.
// Format: "Tupay/VERSION (+https://github.com/tertua/tupay)"
func UserAgent() string {
	return fmt.Sprintf("Tupay/%s (+https://github.com/tertua/tupay)", Version)
}

// RelayUserAgent returns the User-Agent for webhook forwarding to downstream projects.
// Format: "Tupay-Relay/VERSION"
func RelayUserAgent() string {
	return fmt.Sprintf("Tupay-Relay/%s", Version)
}

// HTTP client timeouts for external API calls. The per-request context
// timeout is now driven by config (see pkg/configs GatewayConfig); these
// client-level values are only a coarse ceiling that the context deadline
// (always tighter) enforces in practice.
const (
	// DefaultClientTimeout is the ceiling for the shared gateway client.
	DefaultClientTimeout = 15 * time.Second

	// PaymentClientTimeout is the ceiling for the slower payment client.
	PaymentClientTimeout = 25 * time.Second
)

// Response size limits for external API responses.
const (
	// MaxAPIResponseSize is the maximum size for JSON API responses (1MB).
	MaxAPIResponseSize = 1 << 20

	// MaxQRImageSize is the maximum size for QR code images (256KB).
	MaxQRImageSize = 256 << 10

	// MaxErrorBodyLog is the maximum length of error response bodies to log.
	MaxErrorBodyLog = 300
)

// DefaultHTTPClient is a pre-configured HTTP client with reasonable defaults
// for gateway API calls. Use this instead of http.DefaultClient. The Timeout
// is only a ceiling; callers set a tighter per-request context deadline from
// config (GATEWAY_API_TIMEOUT_SECONDS and friends).
var DefaultHTTPClient = &http.Client{
	Timeout: DefaultClientTimeout,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	},
}

// PaymentHTTPClient is configured with a longer timeout for payment operations
// that may take more time (crypto payments, invoice generation). Timeout is a
// ceiling; per-request config (GATEWAY_PAYMENT_TIMEOUT_SECONDS) is tighter.
var PaymentHTTPClient = &http.Client{
	Timeout: PaymentClientTimeout,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	},
}
