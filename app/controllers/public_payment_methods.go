package controllers

import (
	"context"
	"sort"

	"github.com/shopspring/decimal"
	"github.com/tertua/invoiceman/platform/gateway"
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
// Midtrans allowlist narrows the Midtrans methods; nil allows them all.
// Providers reporting a live minimum (gateway.MinAmountChecker, cached
// hourly) also hide a method whose charge sits below it — e.g. a small IDR
// invoice can never clear the crypto minimum, so crypto is not offered. Any
// minimum-fetch failure keeps the method visible (fail-open) and lets intent
// creation return the authoritative error.
//
// Dev path — public method list (scaffold):
//
//	[done]  flat provider-neutral list, sorted by id
//	[done]  live-minimum filter for checker providers (crypto)
//	[next]  group/order per audience (IDR methods first for domestic payers)
//	        while keeping publicChargeMethod unchanged
//
// End dev path
func availableChargeMethods(ctx context.Context, invoiceCurrency string, balance, usdToIdr decimal.Decimal, midtransAllow map[string]bool) []publicChargeMethod {
	seen := make(map[string]bool)
	out := make([]publicChargeMethod, 0)
	for _, name := range gateway.Names() {
		gw, err := gateway.Get(name)
		if err != nil || !gateway.ProviderReady(gw) {
			continue
		}
		provider, ok := gw.(gateway.PaymentMethodProvider)
		if !ok {
			continue
		}
		spec, err := buildCharge(gw, invoiceCurrency, balance, usdToIdr)
		if err != nil {
			continue
		}
		for _, method := range provider.Methods() {
			if seen[method] {
				continue
			}
			if name == "midtrans" && !methodAllowed(midtransAllow, method) {
				continue
			}
			if belowLiveMinimum(ctx, gw, method, spec) {
				continue
			}
			seen[method] = true
			out = append(out, publicChargeMethod{
				ID:       method,
				Name:     gateway.MethodName(method),
				Currency: spec.Currency,
				Amount:   spec.chargeAmount(),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// belowLiveMinimum reports whether a charge sits below the provider's live
// minimum for the given neutral method. Only crypto is checked (fiat methods
// have no provider minimum); providers without a checker never filter. Any
// fetch error keeps the method visible — intent creation stays authoritative.
func belowLiveMinimum(ctx context.Context, gw gateway.Gateway, method string, spec chargeSpec) bool {
	if method != gateway.MethodCrypto {
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
	limit, err := checker.MinAmount(ctx, spec.Currency)
	if err != nil {
		return false
	}
	return charge.LessThan(limit)
}

// methodAllowed reports whether method passes the Midtrans allowlist. A nil
// allowlist means the owner never restricted methods, so everything is
// allowed; an empty method (default Snap) is always allowed.
func methodAllowed(allow map[string]bool, method string) bool {
	if allow == nil || method == "" {
		return true
	}
	return allow[method]
}
