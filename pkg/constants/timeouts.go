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
// Format: "InvoiceMan/VERSION (+https://github.com/tertua/invoiceman)"
func UserAgent() string {
	return fmt.Sprintf("InvoiceMan/%s (+https://github.com/tertua/invoiceman)", Version)
}

// RelayUserAgent returns the User-Agent for webhook forwarding to downstream projects.
// Format: "InvoiceMan-Relay/VERSION"
func RelayUserAgent() string {
	return fmt.Sprintf("InvoiceMan-Relay/%s", Version)
}

// HTTP client timeouts for external API calls.
const (
	// GatewayAPITimeout is the default timeout for gateway API operations
	// (charge creation, status checks).
	GatewayAPITimeout = 15 * time.Second

	// GatewayPaymentTimeout is the timeout for payment creation operations
	// that may involve additional processing (crypto payments, invoice creation).
	GatewayPaymentTimeout = 25 * time.Second

	// QRFetchTimeout is the timeout for fetching QR code images.
	QRFetchTimeout = 10 * time.Second

	// WebhookForwardTimeout is the timeout for forwarding webhooks to
	// downstream services.
	WebhookForwardTimeout = 10 * time.Second
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
// for gateway API calls. Use this instead of http.DefaultClient.
var DefaultHTTPClient = &http.Client{
	Timeout: GatewayAPITimeout,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	},
}

// PaymentHTTPClient is configured with a longer timeout for payment operations
// that may take more time (crypto payments, invoice generation).
var PaymentHTTPClient = &http.Client{
	Timeout: GatewayPaymentTimeout,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	},
}
