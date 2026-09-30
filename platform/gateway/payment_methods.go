package gateway

import (
	"sort"
	"strings"
)

const (
	MethodBankTransfer = "bank_transfer"
	MethodQRIS         = "qris"
	MethodGopay        = "gopay"
	MethodCreditCard   = "credit_card"
	MethodCrypto       = "crypto"
	MethodOther        = "other"
)

// IDs lists every routable provider-neutral method id, oldest first; MethodOther is excluded because it only labels unmapped provider notifications.
func IDs() []string {
	return []string{
		MethodBankTransfer, MethodQRIS, MethodGopay,
		MethodCreditCard, MethodCrypto,
	}
}

// MethodName maps a stable method ID to its display label.
func MethodName(method string) string {
	labels := map[string]string{MethodBankTransfer: "Bank Transfer", MethodQRIS: "QRIS", MethodGopay: "GoPay", MethodCreditCard: "Credit Card", MethodCrypto: "Cryptocurrency", MethodOther: "Other"}
	if label, ok := labels[method]; ok {
		return label
	}
	return method
}

// DisplayName maps a registry name to its human-facing provider label; an empty name means no explicit provider and resolves to the default one.
func DisplayName(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "midtrans":
		return "Midtrans"
	case "nowpayments":
		return "NOWPayments"
	default:
		return strings.TrimSpace(name)
	}
}

// ChargeCurrencyOf returns the provider's preferred charge currency, or ""
// when the provider charges in the invoice currency (or has no opinion).
// Nil-safe: a nil gateway returns "".
func ChargeCurrencyOf(g Gateway) string {
	provider, ok := g.(ChargeCurrencyProvider)
	if !ok {
		return ""
	}
	return strings.TrimSpace(provider.ChargeCurrency())
}

// ProviderLegacyLabels maps a provider name to the neutral method label used
// for historical payments rows written before the transactions table carried
// the method id. It labels old data only; live labels come from the
// transaction's method id. Keyed by registry name.
func ProviderLegacyLabels() map[string]string {
	return map[string]string{
		"midtrans":    "QRIS",
		"nowpayments": "Crypto",
	}
}

// MethodRef pairs a provider-neutral method id with the ready provider that offers it.
type MethodRef struct {
	Provider Gateway
	ID       string
}

// OfferedMethods lists every method of every ready provider, deduplicated by id and sorted by id, visiting providers in name order so a shared method id always resolves to the same owner.
func OfferedMethods() []MethodRef {
	seen := make(map[string]bool)
	out := make([]MethodRef, 0)
	for _, name := range sortedNames() {
		g, err := Get(name)
		if err != nil || !ProviderReady(g) {
			continue
		}
		provider, ok := g.(PaymentMethodProvider)
		if !ok {
			continue
		}
		for _, method := range provider.Methods() {
			if seen[method] {
				continue
			}
			seen[method] = true
			out = append(out, MethodRef{Provider: g, ID: method})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ExposesBrowserToken reports whether a stored provider token may be sent to
// browsers under "provider_token" (and its deprecated "snap_token" alias). Only
// providers declaring the BrowserSDKProvider capability qualify; unresolvable
// gateways and empty tokens expose nothing. This keeps hosted payment-page ids
// (stored in the same column) from being fed to an embedded checkout SDK.
func ExposesBrowserToken(g Gateway, token string) bool {
	if token == "" {
		return false
	}
	sdk, ok := g.(BrowserSDKProvider)
	return ok && sdk.BrowserSDK()
}
