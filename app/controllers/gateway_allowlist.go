package controllers

import (
	"strings"

	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/platform/gateway"
)

// providerAllowlist resolves the owner's allowed methods for one provider.
// nil means "unrestricted" (no veto). The legacy midtrans_methods column is
// read as the allowlist for provider "midtrans"; every other provider is
// unrestricted today because no other provider needs owner narrowing yet. A
// future provider gets its own column only when it needs one (no speculative
// schema).
func providerAllowlist(settings models.Settings, provider string) map[string]bool {
	if provider != gateway.DefaultProviderName {
		return nil
	}
	// The stored midtrans_methods value is intentionally ignored: the account
	// only offers QRIS, so the allowlist stays pinned to QRIS regardless of
	// what the owner submitted or what an old row holds.
	return map[string]bool{gateway.MethodQRIS: true}
}

// methodAllowedFor reports whether a provider may offer a neutral method under
// the owner's allowlist. A nil allowlist means the owner never restricted
// methods, so everything is allowed; an empty method (the provider default) is
// always allowed.
func methodAllowedFor(provider string, allow map[string]bool, method string) bool {
	if allow == nil || method == "" {
		return true
	}
	return allow[method]
}

// applyEnabledMethods narrows the charge's selectable methods to the ones the
// provider declares via DefaultMethodsProvider. Providers without the
// capability keep their own defaults untouched.
func applyEnabledMethods(spec *chargeSpec, gw gateway.Gateway) {
	provider, ok := gw.(gateway.DefaultMethodsProvider)
	if !ok {
		return
	}
	spec.EnabledMethods = provider.DefaultMethods()
}

// normalizeProviderMethods keeps the settings PATCH write consistent with the
// routing above. The default provider is locked to QRIS, so any input
// normalizes to qris; providers with no allowlist today keep the raw trimmed
// value untouched.
func normalizeProviderMethods(provider, raw string) string {
	if provider != gateway.DefaultProviderName {
		return strings.TrimSpace(raw)
	}
	return gateway.MethodQRIS
}
