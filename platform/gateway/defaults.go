package gateway

import (
	"os"
	"strings"
)

// DefaultProviderName is the single built-in fallback provider. It is used
// when nothing else (request, project default, env) names a provider, so
// controllers never hardcode a provider name.
const DefaultProviderName = "midtrans"

// DefaultProviderEnv overrides DefaultProviderName at runtime.
const DefaultProviderEnv = "GATEWAY_DEFAULT_PROVIDER"

// DefaultProvider resolves the fallback provider name: the
// GATEWAY_DEFAULT_PROVIDER env value when set, otherwise DefaultProviderName.
// It reads the environment directly so pkg/configs needs no provider field.
func DefaultProvider() string {
	if name := strings.TrimSpace(os.Getenv(DefaultProviderEnv)); name != "" {
		return name
	}
	return DefaultProviderName
}
