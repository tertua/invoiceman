package nowpayments

import (
	"strings"

	"github.com/tertua/tupay/pkg/configs"
)

// Config holds NOWPayments credentials. All values come from env;
// no domain or key is hardcoded.
type Config struct {
	// APIKey authenticates invoice creation and status fetches.
	APIKey string
	// IPNSecret is reserved for future header-signature verification.
	// Verification today uses an API status fetch instead.
	IPNSecret string
	// Sandbox selects the NOWPayments sandbox API (testing, no funds move).
	Sandbox bool
	// Endpoint overrides the API base URL (used by tests).
	Endpoint string
	// CallbackBase is the public API origin used to build the per-invoice
	// IPN callback URL (TUPAY_PUBLIC_URL, fallback INVOICEMAN_PUBLIC_URL).
	CallbackBase string
}

// FromEnv reads NOWPayments config from the central config.
func FromEnv() Config {
	cfg := configs.Get()
	return Config{
		APIKey:       cfg.NOWPayments.APIKey,
		IPNSecret:    cfg.NOWPayments.IPNSecret,
		Sandbox:      cfg.NOWPayments.Sandbox,
		Endpoint:     cfg.NOWPayments.BaseURL,
		CallbackBase: cfg.Relay.PublicURL,
	}
}

// BaseURL returns the NOWPayments API endpoint.
func (c Config) BaseURL() string {
	if c.Endpoint != "" {
		return strings.TrimRight(c.Endpoint, "/")
	}
	if c.Sandbox {
		return "https://api-sandbox.nowpayments.io/v1"
	}
	return "https://api.nowpayments.io/v1"
}

// CallbackURL returns the absolute IPN endpoint for created invoices.
// NOWPayments cannot reach localhost without a public address; CallbackBase
// must be publicly reachable in production.
func (c Config) CallbackURL() string {
	base := c.CallbackBase
	if base == "" {
		base = "http://localhost:5000"
	}
	return strings.TrimRight(base, "/") + "/api/webhooks/nowpayments"
}
