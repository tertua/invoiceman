package gateway

import (
	"errors"
	"strings"

	"github.com/shopspring/decimal"
)

// ErrUnsupportedCurrency is returned when a conversion needs a rate that the
// manual USD/IDR setting cannot provide.
var ErrUnsupportedCurrency = errors.New("currency conversion is not supported")

// Manual conversion only covers the two fiat currencies the relay settles in.
const (
	FiatIDR = "IDR"
	FiatUSD = "USD"
)

// Convert exchanges amount between fiat currencies using a manual
// IDR-per-USD rate (no realtime feed). Same-currency is a no-op; only
// USD<->IDR is supported. A non-positive rate means conversion is not
// configured, matching the zero-value Settings field.
func Convert(amount decimal.Decimal, from, to string, idrPerUSD decimal.Decimal) (decimal.Decimal, error) {
	from = strings.ToUpper(strings.TrimSpace(from))
	to = strings.ToUpper(strings.TrimSpace(to))
	if from == to {
		return amount, nil
	}
	if !idrPerUSD.GreaterThan(decimal.Zero) {
		return decimal.Zero, ErrUnsupportedCurrency
	}
	switch {
	case from == FiatUSD && to == FiatIDR:
		return amount.Mul(idrPerUSD), nil
	case from == FiatIDR && to == FiatUSD:
		return amount.Div(idrPerUSD), nil
	default:
		return decimal.Zero, ErrUnsupportedCurrency
	}
}

// NeedsConversion reports whether paying amount in from via to requires a rate.
func NeedsConversion(from, to string) bool {
	return !strings.EqualFold(strings.TrimSpace(from), strings.TrimSpace(to))
}

// SettleAmount converts a provider charge back into the invoice's settlement
// currency. A blank or unconvertible target falls back to the raw charge so
// legacy rows (and same-currency flows) keep settling unchanged.
func SettleAmount(charge decimal.Decimal, chargeCurrency, invoiceCurrency string, idrPerUSD decimal.Decimal) decimal.Decimal {
	if strings.TrimSpace(invoiceCurrency) == "" {
		invoiceCurrency = chargeCurrency
	}
	converted, err := Convert(charge, chargeCurrency, invoiceCurrency, idrPerUSD)
	if err != nil {
		return charge
	}
	return converted
}
