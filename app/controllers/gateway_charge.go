package controllers

import (
	"errors"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/tertua/invoiceman/platform/gateway"
)

// ErrConversionUnsupported is returned when a charge cannot be expressed in
// the provider's currency with the configured manual rate.
var ErrConversionUnsupported = errors.New("currency conversion is not configured")

// chargeSpec is the provider-ready amount for one intent, plus the audit data
// needed to settle the source invoice later.
type chargeSpec struct {
	AmountMinor   int64
	AmountDecimal string
	Currency      string
	InvoiceAmount decimal.Decimal
	UsdToIdr      decimal.Decimal
}

// buildCharge converts an invoice-currency balance into the currency the
// routed provider charges, using the owner's manual IDR-per-USD rate. Midtrans
// is IDR-only; other providers accept the invoice currency as-is. No realtime
// rates are ever fetched.
func buildCharge(gw gateway.Gateway, invoiceCurrency string, balance, usdToIdr decimal.Decimal) (chargeSpec, error) {
	invoiceCurrency = strings.ToUpper(strings.TrimSpace(invoiceCurrency))
	target := invoiceCurrency
	if gw.Name() == "midtrans" {
		target = gateway.FiatIDR
	}
	converted, err := gateway.Convert(balance, invoiceCurrency, target, usdToIdr)
	if err != nil {
		return chargeSpec{}, ErrConversionUnsupported
	}
	spec := chargeSpec{Currency: target, InvoiceAmount: balance}
	if gateway.NeedsConversion(invoiceCurrency, target) {
		spec.UsdToIdr = usdToIdr
	}
	if target == gateway.FiatIDR {
		spec.AmountMinor = converted.Round(0).IntPart()
		spec.AmountDecimal = converted.String()
	} else {
		spec.AmountDecimal = converted.String()
	}
	return spec, nil
}

// request shapes the spec as a gateway creation request.
func (s chargeSpec) request(orderID, email, phone, method string) *gateway.CreateTxRequest {
	return &gateway.CreateTxRequest{
		OrderID:       orderID,
		AmountMinor:   s.AmountMinor,
		AmountDecimal: s.AmountDecimal,
		Currency:      s.Currency,
		Email:         email,
		Phone:         phone,
		PaymentMethod: method,
	}
}

// chargeAmount is the exact amount the provider charges, as a decimal string:
// IDR providers bill whole rupiah (AmountMinor), others the decimal amount.
func (s chargeSpec) chargeAmount() string {
	if s.Currency == gateway.FiatIDR {
		return decimal.NewFromInt(s.AmountMinor).String()
	}
	return s.AmountDecimal
}
