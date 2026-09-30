package controllers

import (
	"strings"

	"github.com/shopspring/decimal"
	"github.com/tertua/tupay/platform/gateway"
)

// chargeSpec is the provider-ready amount for one intent, plus the audit data
// needed to settle the source invoice later.
type chargeSpec struct {
	AmountMinor   int64
	AmountDecimal string
	Currency      string
	InvoiceAmount decimal.Decimal
	UsdToIdr      decimal.Decimal
	// EnabledMethods narrows the provider's selectable methods (neutral ids).
	// Empty leaves the provider default untouched.
	EnabledMethods []string
}

// buildCharge converts an invoice-currency balance into the currency the
// routed provider charges, using the owner's manual IDR-per-USD rate. The
// target comes from the provider's ChargeCurrencyProvider capability: a
// provider that only accepts one fiat declares it, and providers without the
// capability charge in the invoice currency. No realtime rates are ever fetched.
// A per-invoice currency choice stays open for later, once the pay page grows a toggle.
func buildCharge(gw gateway.Gateway, invoiceCurrency string, balance, usdToIdr decimal.Decimal) (chargeSpec, error) {
	invoiceCurrency = strings.ToUpper(strings.TrimSpace(invoiceCurrency))
	target := invoiceCurrency
	if declared := gateway.ChargeCurrencyOf(gw); declared != "" {
		target = declared
	}
	converted, err := gateway.Convert(balance, invoiceCurrency, target, usdToIdr)
	if err != nil {
		return chargeSpec{}, err
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
		OrderID:        orderID,
		AmountMinor:    s.AmountMinor,
		AmountDecimal:  s.AmountDecimal,
		Currency:       s.Currency,
		Email:          email,
		Phone:          phone,
		PaymentMethod:  method,
		EnabledMethods: s.EnabledMethods,
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
