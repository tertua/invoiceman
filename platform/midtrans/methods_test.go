package midtrans

import (
	"testing"

	"github.com/tertua/tupay/platform/gateway"
)

func TestStandardizePaymentType(t *testing.T) {
	for _, test := range []struct {
		raw    string
		method string
	}{
		{"qris", gateway.MethodQRIS},
		{"bni", gateway.MethodBankTransfer},
		{"credit_card", gateway.MethodCreditCard},
		{"unknown_provider_type", gateway.MethodOther},
	} {
		if got := StandardizePaymentType(test.raw); got != test.method {
			t.Errorf("StandardizePaymentType(%q) = %q, want %q", test.raw, got, test.method)
		}
	}
}

// TestPayerConfigNeverExposesServerKey guards the browser config contract:
// only the public client key and the environment flag may be returned.
func TestPayerConfigNeverExposesServerKey(t *testing.T) {
	t.Setenv("MIDTRANS_SERVER_KEY", "sk-secret")
	t.Setenv("MIDTRANS_CLIENT_KEY", "ck-public")
	t.Setenv("MIDTRANS_IS_PROD", "true")
	conf := (Gateway{}).PayerConfig()
	if conf["client_key"] != "ck-public" || conf["is_production"] != true {
		t.Fatalf("unexpected payer config: %v", conf)
	}
	for key := range conf {
		if key == "server_key" {
			t.Fatalf("PayerConfig must never expose server_key, got %v", conf)
		}
	}
}
