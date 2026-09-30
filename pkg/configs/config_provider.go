package configs

import (
	"os"
	"strconv"
	"strings"
)

// Provider returns one provider's raw environment keyed by registry name
// (e.g. "midtrans"), so adding a provider needs no code change here: its
// package reads <PROVIDER_UPPER_SNAKE>_<KEY> via Provider / ProviderString /
// ProviderBool / ProviderInt.
//
// Convention: MIDTRANS_SERVER_KEY, NOWPAYMENTS_IPN_SECRET, XENDIT_API_KEY —
// upper-snake of the registry name, an underscore, then the key. The map is
// built from the live environment on every call (configs are read through
// Get() in tests with t.Setenv), and never contains another provider's keys.
func (c Config) Provider(provider string) map[string]string {
	prefix := providerEnvPrefix(provider)
	if prefix == "" {
		return nil
	}
	out := map[string]string{}
	for _, kv := range os.Environ() {
		name, value, ok := strings.Cut(kv, "=")
		if !ok || !strings.HasPrefix(name, prefix) {
			continue
		}
		key := strings.ToLower(strings.TrimPrefix(name, prefix))
		if key == "" {
			continue
		}
		out[key] = strings.TrimSpace(value)
	}
	return out
}

// ProviderString returns the provider key's trimmed value or fallback when
// unset/blank.
func (c Config) ProviderString(provider, key, fallback string) string {
	if v := strings.TrimSpace(c.Provider(provider)[strings.ToLower(key)]); v != "" {
		return v
	}
	return fallback
}

// ProviderBool parses the provider key as a boolean (true/false spellings,
// case-insensitive); anything else, or a blank/unset value, falls back.
func (c Config) ProviderBool(provider, key string, fallback bool) bool {
	raw := strings.TrimSpace(c.Provider(provider)[strings.ToLower(key)])
	if raw == "" {
		return fallback
	}
	return strings.EqualFold(raw, "true")
}

// ProviderInt parses the provider key as an integer; anything else, or a
// blank/unset value, falls back.
func (c Config) ProviderInt(provider, key string, fallback int) int {
	raw := strings.TrimSpace(c.Provider(provider)[strings.ToLower(key)])
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

// providerEnvPrefix turns a registry name into the env prefix it owns:
// "midtrans" -> "MIDTRANS_". Underscores in the name are kept as-is so
// multi-word providers map predictably (there are none today).
func providerEnvPrefix(provider string) string {
	name := strings.ToUpper(strings.TrimSpace(provider))
	if name == "" {
		return ""
	}
	for _, r := range name {
		if (r < 'A' || r > 'Z') && r != '_' {
			return ""
		}
	}
	return name + "_"
}
