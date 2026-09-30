package controllers

import (
	"context"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/platform/gateway"
)

// publicChargeMethod is one selectable method on the public pay page. The
// amount is what the provider would actually charge (already converted from
// the invoice currency with the owner's manual rate).
type publicChargeMethod struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
	Amount   string `json:"amount"`
}

// availableChargeMethods lists provider-neutral methods that can settle the
// invoice balance, one entry per method, sorted by id. Providers whose
// currency cannot be reached with the configured rate are skipped, so the
// payer never sees a method that would fail at intent creation. The owner's
// per-provider allowlist (gateway_allowlist.go) narrows the default provider's
// methods; a nil allowlist allows them all.
// Providers reporting a live minimum (gateway.MinAmountChecker, cached
// hourly) also hide a method whose charge sits below it — but only for the
// payer's chosen crypto asset: the minimum is per asset, so an asset-agnostic
// list (payCurrency empty) offers crypto and lets the asset's own minimum gate
// it once picked. Any minimum-fetch failure keeps the method visible (fail-open)
// and lets intent creation return the authoritative error.
// Later: group and order this list per audience (IDR methods first for domestic payers) while keeping publicChargeMethod unchanged.
func availableChargeMethods(ctx context.Context, invoiceCurrency string, balance, usdToIdr decimal.Decimal, settings models.Settings, payCurrency string) []publicChargeMethod {
	out := make([]publicChargeMethod, 0)
	for _, ref := range gateway.OfferedMethods() {
		if !methodAllowedFor(ref.Provider.Name(), providerAllowlist(settings, ref.Provider.Name()), ref.ID) {
			continue
		}
		spec, err := buildCharge(ref.Provider, invoiceCurrency, balance, usdToIdr)
		if err != nil {
			continue
		}
		if belowLiveMinimum(ctx, ref.Provider, ref.ID, spec, payCurrency) {
			continue
		}
		out = append(out, publicChargeMethod{
			ID:       ref.ID,
			Name:     gateway.MethodName(ref.ID),
			Currency: spec.Currency,
			Amount:   spec.chargeAmount(),
		})
	}
	return out
}

// belowLiveMinimum reports whether a charge sits below the provider's live
// minimum for the given neutral method. Only crypto is checked (fiat methods
// have no provider minimum); providers without a checker never filter. An
// empty payCurrency means the payer has not picked an asset yet — nothing is
// filtered, because no asset's minimum applies. Any fetch error keeps the
// method visible — intent creation stays authoritative.
func belowLiveMinimum(ctx context.Context, gw gateway.Gateway, method string, spec chargeSpec, payCurrency string) bool {
	if method != gateway.MethodCrypto || strings.TrimSpace(payCurrency) == "" {
		return false
	}
	checker, ok := gw.(gateway.MinAmountChecker)
	if !ok {
		return false
	}
	charge, err := decimal.NewFromString(spec.AmountDecimal)
	if err != nil {
		return false
	}
	limit, err := checker.MinAmount(ctx, spec.Currency, payCurrency)
	if err != nil {
		return false
	}
	return charge.LessThan(limit)
}
