package controllers

import (
	"testing"

	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/platform/gateway"
)

// Midtrans is locked to QRIS (declared by platform/midtrans.DefaultMethods):
// any input normalizes to qris and the allowlist always admits exactly QRIS, so
// the stored per-owner value can never re-enable another Midtrans method.
func TestMidtransLockedToQRIS(t *testing.T) {
	for _, raw := range []string{"qris", "gopay", "bank_transfer", "qris,nonsense", "", "nonsense"} {
		if got := normalizeProviderMethods(gateway.DefaultProviderName, raw); got != gateway.MethodQRIS {
			t.Errorf("normalizeProviderMethods(%q) = %q, want %q", raw, got, gateway.MethodQRIS)
		}
	}

	// The stored provider_methods value is intentionally ignored: even a
	// settings row that claims another method resolves to QRIS-only.
	settings := models.Settings{}
	settings.ProviderMethods = "gopay,bank_transfer"
	allow := providerAllowlist(settings, gateway.DefaultProviderName)
	if !allow[gateway.MethodQRIS] {
		t.Fatalf("allowlist must admit QRIS, got %v", allow)
	}
	for _, other := range []string{gateway.MethodGopay, gateway.MethodBankTransfer, gateway.MethodCreditCard} {
		if allow[other] {
			t.Errorf("allowlist must reject %q, got %v", other, allow)
		}
	}
}

// A non-default provider has no owner allowlist today: it is unrestricted and
// its raw method value is preserved rather than normalized to QRIS.
func TestUnknownProviderIsUnrestricted(t *testing.T) {
	settings := models.Settings{}
	settings.ProviderMethods = "qris"
	if allow := providerAllowlist(settings, "someotherprovider"); allow != nil {
		t.Fatalf("unknown provider allowlist = %v, want nil", allow)
	}
	if got := normalizeProviderMethods("someotherprovider", "  crypto  "); got != "crypto" {
		t.Errorf("normalizeProviderMethods(someotherprovider) = %q, want %q", got, "crypto")
	}
}
