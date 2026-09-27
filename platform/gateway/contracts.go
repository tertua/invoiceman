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
