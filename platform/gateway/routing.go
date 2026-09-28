package gateway

import (
	"errors"
	"sort"
	"strings"
)

var ErrUnsupportedPaymentMethod = errors.New("payment method is not supported")

// Route selects a registered provider for the requested method, honoring a preferred provider when it supports it and otherwise searching the registry in deterministic name order so routing never depends on Go map iteration.
// The optional allow predicate vetoes a provider for a method (e.g. a per-account method allowlist) before it is considered, so a vetoed provider is skipped and the next candidate is tried.
// Routing stays deterministic method -> provider with no automatic failover until the UI/UX for it exists; an opt-in fallback would live here behind an explicit caller flag.
func Route(preferredProvider, method string, allow func(provider, method string) bool) (Gateway, error) {
	method = strings.ToLower(strings.TrimSpace(method))
	preferredProvider = strings.ToLower(strings.TrimSpace(preferredProvider))
	permitted := func(name string) bool { return allow == nil || allow(name, method) }
	if preferredProvider != "" {
		g, err := Get(preferredProvider)
		if err != nil {
			return nil, err
		}
		if !permitted(preferredProvider) {
			return nil, ErrUnsupportedPaymentMethod
		}
		if method == "" || supports(g, method) {
			return g, nil
		}
		return nil, ErrUnsupportedPaymentMethod
	}
	for _, name := range sortedNames() {
		g, err := Get(name)
		if err == nil && ProviderReady(g) && permitted(name) && (method == "" || supports(g, method)) {
			return g, nil
		}
	}
	return nil, ErrUnsupportedPaymentMethod
}

// ProviderReady reports whether a provider declares itself usable. Providers
// without the optional ConfiguredProvider capability are assumed ready so
// test doubles and future gateways keep working.
func ProviderReady(g Gateway) bool {
	configured, ok := g.(ConfiguredProvider)
	return !ok || configured.Configured()
}

func supports(g Gateway, method string) bool {
	provider, ok := g.(PaymentMethodProvider)
	if !ok {
		return false
	}
	for _, supported := range provider.Methods() {
		if supported == method {
			return true
		}
	}
	return false
}

func sortedNames() []string {
	names := Names()
	sort.Strings(names)
	return names
}
