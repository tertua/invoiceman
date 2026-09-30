package gateway

import (
	"os"
	"testing"
)

type currencyGateway struct {
	stubGateway
	currency string
}

func (g currencyGateway) ChargeCurrency() string { return g.currency }

func TestChargeCurrencyOf(t *testing.T) {
	tests := []struct {
		name string
		g    Gateway
		want string
	}{
		{"capability present", currencyGateway{stubGateway{name: "idrpay"}, FiatIDR}, FiatIDR},
		{"capability present but blank", currencyGateway{stubGateway{name: "any"}, "  "}, ""},
		{"capability absent", stubGateway{name: "plain"}, ""},
		{"nil gateway", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ChargeCurrencyOf(tt.g); got != tt.want {
				t.Fatalf("ChargeCurrencyOf = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDefaultProvider(t *testing.T) {
	// Genuinely unset the env, restoring whatever it was afterwards.
	original, had := os.LookupEnv(DefaultProviderEnv)
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(DefaultProviderEnv, original)
		} else {
			_ = os.Unsetenv(DefaultProviderEnv)
		}
	})
	_ = os.Unsetenv(DefaultProviderEnv)

	tests := []struct {
		name string
		env  string
		want string
	}{
		{"env unset uses constant", "", DefaultProviderName},
		{"env set is used", "xendit", "xendit"},
		{"env whitespace is trimmed", "  nowpayments  ", "nowpayments"},
		{"blank env falls back to constant", "   ", DefaultProviderName},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(DefaultProviderEnv, tt.env)
			if got := DefaultProvider(); got != tt.want {
				t.Fatalf("DefaultProvider = %q, want %q", got, tt.want)
			}
		})
	}
}
