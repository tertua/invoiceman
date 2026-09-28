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
