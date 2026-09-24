package controllers

import (
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
func availableChargeMethods(invoiceCurrency string, balance, usdToIdr decimal.Decimal, midtransAllow map[string]bool) []publicChargeMethod {
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

// methodAllowed reports whether method passes the Midtrans allowlist. A nil
// allowlist means the owner never restricted methods, so everything is
// allowed; an empty method (default Snap) is always allowed.
func methodAllowed(allow map[string]bool, method string) bool {
	if allow == nil || method == "" {
		return true
	}
	return allow[method]
}
