package gateway

import (
	"context"

	"github.com/shopspring/decimal"
)

type CreateTxRequest struct {
	OrderID        string
	AmountMinor    int64
	AmountDecimal  string
	Currency       string
	Email          string
	Phone          string
	PaymentMethod  string
	EnabledMethods []string
	PayCurrency    string
	// DirectQRIS asks the provider to return an on-page QRIS payload (Core
	// API) instead of a hosted/Snap checkout when the method is qris.
	DirectQRIS bool
}

type CreateTxResponse struct {
	Token         string
	RedirectURL   string
	PaymentURL    string
	QRString      string // raw EMVCo QRIS payload the provider returns with its QR image, for client-side rendering
	Address       string
	ExpiresAt     string
	PaymentMethod string
	RawPayload    string
	PayCurrency   string
}

type NotificationResult struct {
	OrderID       string
	TransactionID string
	Status        string
	// PaymentType is the provider's raw payment type (e.g. Midtrans "bni",
	// NOWPayments "btc"); kept for downstream compatibility.
	PaymentType string
	// PaymentMethod is the provider-neutral method (one of the Method*
	// constants) derived from the raw type.
	PaymentMethod string
	GrossMinor    int64
	GrossDecimal  string
	Currency      string
}

type PaymentMethodProvider interface {
	Methods() []string
}

// ConfiguredProvider is an optional capability: providers that can tell
// whether their credentials are present report it so endpoints can hide
// unusable methods.
type ConfiguredProvider interface {
	Configured() bool
}

// SandboxProvider is an optional capability: providers that can tell whether they are running against a test environment report it so status endpoints never hardcode provider names.
type SandboxProvider interface {
	Sandbox() bool
}

// ChargeCurrencyProvider is an optional capability: providers that only
// accept one fiat report it so callers never switch on the provider name. An
// empty string means the provider charges in the invoice currency.
type ChargeCurrencyProvider interface {
	ChargeCurrency() string
}

// DecimalAmountProvider is an optional capability: providers that bill with a
// decimal amount (crypto, sub-unit fiat) require the caller to send one, so
// callers never switch on the provider name.
type DecimalAmountProvider interface {
	RequiresDecimalAmount() bool
}

// BrowserSDKProvider is an optional capability: providers whose stored token
// is meant to be consumed by an embedded browser checkout SDK report it, so
// public payloads expose the token as "snap_token" only when a widget can use
// it. A hosted payment-page id kept in the same column must never leak out as
// a widget token (the pay panel would hand it to the wrong SDK).
type BrowserSDKProvider interface {
	BrowserSDK() bool
}

// PayerConfigProvider is an optional capability: providers whose browser
// checkout needs public configuration report it so the pay page and the
// /gateway/config endpoint never hardcode a provider. Only public values
// (client key, environment flags) may be returned — never server secrets.
type PayerConfigProvider interface {
	PayerConfig() map[string]any
}

// MinAmountChecker is an optional capability: providers with live
// per-currency minimums report them so endpoints can hide a method whose
// charge could never succeed, instead of failing after the payer commits.
// currencyFrom is the charge currency (e.g. USD); payCurrency is the payer's
// chosen crypto asset (e.g. usdtbsc) so the minimum belongs to that asset —
// empty means no asset has been picked yet and the provider falls back to
// its canonical pay currency.
type MinAmountChecker interface {
	MinAmount(ctx context.Context, currencyFrom, payCurrency string) (decimal.Decimal, error)
}
