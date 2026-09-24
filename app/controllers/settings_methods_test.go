package controllers

import (
	"testing"

	"github.com/tertua/invoiceman/app/models"
)

func settingsWithMethods(csv string) models.Settings {
	return models.Settings{SettingsGatewayMethods: models.SettingsGatewayMethods{MidtransMethods: csv}}
}

// normalizeMidtransMethods keeps the first supported id and falls back to
// gopay for empty, unknown, or legacy multi-method values.
func TestNormalizeMidtransMethods(t *testing.T) {
	cases := []struct{ in, want string }{
		{"qris,nonsense,gopay,qris", "qris"},
		{"", "gopay"},
		{"nonsense", "gopay"},
		{" QRIS , Gopay ", "qris"},
	}
	for _, tc := range cases {
		if got := normalizeMidtransMethods(tc.in); got != tc.want {
			t.Errorf("normalizeMidtransMethods(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// An empty CSV must read as "no restriction" (nil), while a populated one
// becomes an allowlist the router can consult.
func TestMidtransMethodAllowlist(t *testing.T) {
	if midtransMethodAllowlist(settingsWithMethods("")) != nil {
		t.Fatal("empty CSV should mean no restriction (nil)")
	}
	allow := midtransMethodAllowlist(settingsWithMethods("qris,gopay"))
	if !allow["qris"] || !allow["gopay"] || allow["crypto"] {
		t.Fatalf("unexpected allowlist: %v", allow)
	}
}
