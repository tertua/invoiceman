package controllers

import (
	"testing"

	"github.com/tertua/tupay/platform/gateway"
)

// Midtrans is locked to QRIS (code-first source of truth): any input
// normalizes to qris and the allowlist always admits exactly QRIS, so the
// stored per-owner value can never re-enable another Midtrans method.
func TestMidtransLockedToQRIS(t *testing.T) {
	for _, raw := range []string{"qris", "gopay", "bank_transfer", "qris,nonsense", "", "nonsense"} {
		if got := normalizeMidtransMethods(raw); got != gateway.MethodQRIS {
			t.Errorf("normalizeMidtransMethods(%q) = %q, want %q", raw, got, gateway.MethodQRIS)
		}
	}

	allow := midtransMethodAllowlist()
	if !allow[gateway.MethodQRIS] {
		t.Fatalf("allowlist must admit QRIS, got %v", allow)
	}
	for _, other := range []string{gateway.MethodGopay, gateway.MethodBankTransfer, gateway.MethodCreditCard} {
		if allow[other] {
			t.Errorf("allowlist must reject %q, got %v", other, allow)
		}
	}

	enabled := enabledMidtransMethods()
	if len(enabled) != 1 || enabled[0] != gateway.MethodQRIS {
		t.Fatalf("enabledMidtransMethods = %v, want [qris]", enabled)
	}
}
