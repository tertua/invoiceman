package gateway

import (
	"errors"
	"sort"
	"strings"
)

var ErrUnsupportedPaymentMethod = errors.New("payment method is not supported")

// Route selects a registered provider for the requested method. A preferred
// provider (explicit request or project default) is honored when it supports
// the method; an empty method uses it for compatibility. With no preferred
// provider the registry is searched in deterministic name order so routing
// never depends on Go map iteration.
func Route(preferredProvider, method string) (Gateway, error) {
	method = strings.ToLower(strings.TrimSpace(method))
	preferredProvider = strings.ToLower(strings.TrimSpace(preferredProvider))
	if preferredProvider != "" {
		g, err := Get(preferredProvider)
		if err != nil {
			return nil, err
		}
		if method == "" || supports(g, method) {
			return g, nil
		}
		return nil, ErrUnsupportedPaymentMethod
	}
	for _, name := range sortedNames() {
		g, err := Get(name)
		if err == nil && ProviderReady(g) && (method == "" || supports(g, method)) {
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
