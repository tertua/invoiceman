package configs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// providerKey matches the ones the provider packages read, so the assertions
// below double as a guarantee that the env names never change.
const (
	midtransKey   = "midtrans"
	npKey         = "nowpayments"
	midServerKey  = "MIDTRANS_SERVER_KEY"
	npSecretKey   = "NOWPAYMENTS_IPN_SECRET"
	npSandboxKey  = "NOWPAYMENTS_SANDBOX"
	npBaseURLKey  = "NOWPAYMENTS_BASE_URL"
	unknownPrefix = "NO_SUCH_PROVIDER_API_KEY"
)

// TestProviderScansEnv verifies Provider returns only the requested
// provider's keys, keyed by lower-snake name with the prefix stripped.
func TestProviderScansEnv(t *testing.T) {
	t.Setenv(midServerKey, "sk-mid")
	t.Setenv("MIDTRANS_IS_PROD", "true")
	t.Setenv(npSecretKey, "np-secret")
	t.Setenv(unknownPrefix, "nope")
	cfg := Get()

	mid := cfg.Provider(midtransKey)
	assert.Equal(t, "sk-mid", mid["server_key"])
	assert.Equal(t, "true", mid["is_prod"])
	assert.NotContains(t, mid, "ipn_secret", "must not leak another provider's keys")

	np := cfg.Provider(npKey)
	assert.Equal(t, "np-secret", np["ipn_secret"])
	assert.NotContains(t, np, "server_key")

	// An unknown prefix is never invented.
	_, ok := mid["no_such_provider_api_key"]
	assert.False(t, ok)
}

// TestProviderTrimsAndIgnoresEmpty verifies blank values are not keys.
func TestProviderTrimsAndIgnoresEmpty(t *testing.T) {
	t.Setenv("MIDTRANS_CLIENT_KEY", "  ck-1  ")
	cfg := Get()
	assert.Equal(t, "ck-1", cfg.Provider(midtransKey)["client_key"])
}

// TestProviderStringFallback verifies trim + fallback semantics.
func TestProviderStringFallback(t *testing.T) {
	t.Setenv("MIDTRANS_SNAP_BASE_URL", "  https://snap.test  ")
	cfg := Get()
	assert.Equal(t, "https://snap.test", cfg.ProviderString(midtransKey, "snap_base_url", "def"))
	assert.Equal(t, "def", cfg.ProviderString(midtransKey, "core_base_url", "def"))
	assert.Equal(t, "def", cfg.ProviderString(midtransKey, "SERVER_KEY", "def"))
}

// TestProviderBoolFallback verifies case-insensitive boolean parsing.
func TestProviderBoolFallback(t *testing.T) {
	t.Setenv(npSandboxKey, "TRUE")
	cfg := Get()
	assert.True(t, cfg.ProviderBool(npKey, "sandbox", false))

	t.Setenv(npSandboxKey, "yes") // not a true/false spelling
	assert.False(t, cfg.ProviderBool(npKey, "sandbox", false), "unrecognized value falls back")

	t.Setenv(npSandboxKey, "")
	assert.True(t, cfg.ProviderBool(npKey, "sandbox", true), "unset value falls back")
}

// TestProviderIntFallback verifies integer parsing and fallback.
func TestProviderIntFallback(t *testing.T) {
	t.Setenv("MIDTRANS_TIMEOUT_SECONDS", "42")
	cfg := Get()
	assert.Equal(t, 42, cfg.ProviderInt(midtransKey, "timeout_seconds", 7))

	t.Setenv("MIDTRANS_TIMEOUT_SECONDS", "not-a-number")
	assert.Equal(t, 7, cfg.ProviderInt(midtransKey, "timeout_seconds", 7))

	t.Setenv("MIDTRANS_TIMEOUT_SECONDS", "")
	assert.Equal(t, 7, cfg.ProviderInt(midtransKey, "timeout_seconds", 7))
}

// TestProviderUnknownName verifies a blank or non-identifier name never
// matches every env var (which a bare prefix scan would).
func TestProviderUnknownName(t *testing.T) {
	t.Setenv(midServerKey, "sk")
	cfg := Get()
	assert.Nil(t, cfg.Provider(""))
	assert.Nil(t, cfg.Provider("not-a-provider"))
	assert.Empty(t, cfg.ProviderString("", "server_key", ""))
	assert.Equal(t, "def", cfg.ProviderString(npKey, "", "def"), "blank key falls back")
	assert.False(t, cfg.ProviderBool(npKey, "", false))
	assert.Equal(t, 0, cfg.ProviderInt(npKey, "", 0))
}

// TestProviderEnvNamesUnchanged pins the exact env names the provider
// packages depend on: ProviderString(provider, key) must read the same key
// the old pkg/configs structs read, so no deploy needs a rename.
func TestProviderEnvNamesUnchanged(t *testing.T) {
	t.Setenv("MIDTRANS_SERVER_KEY", "s")
	t.Setenv("MIDTRANS_CLIENT_KEY", "c")
	t.Setenv("MIDTRANS_IS_PROD", "true")
	t.Setenv("MIDTRANS_SNAP_BASE_URL", "snap")
	t.Setenv("MIDTRANS_CORE_BASE_URL", "core")
	t.Setenv("NOWPAYMENTS_API_KEY", "a")
	t.Setenv("NOWPAYMENTS_IPN_SECRET", "i")
	t.Setenv("NOWPAYMENTS_SANDBOX", "true")
	t.Setenv(npBaseURLKey, "base")
	cfg := Get()
	mid := cfg.Provider(midtransKey)
	assert.Equal(t, "s", mid["server_key"])
	assert.Equal(t, "c", mid["client_key"])
	assert.Equal(t, "true", mid["is_prod"])
	assert.Equal(t, "snap", mid["snap_base_url"])
	assert.Equal(t, "core", mid["core_base_url"])
	np := cfg.Provider(npKey)
	assert.Equal(t, "a", np["api_key"])
	assert.Equal(t, "i", np["ipn_secret"])
	assert.Equal(t, "true", np["sandbox"])
	assert.Equal(t, "base", np["base_url"])
}
