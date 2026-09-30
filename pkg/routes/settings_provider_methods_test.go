package routes

import "testing"

// TestSettingsProviderMethodsDualKey proves the settings method key is
// non-breaking: the canonical provider_methods input is honored, the deprecated
// midtrans_methods alias still binds, and every response carries BOTH keys with
// the same value during the transition.
func TestSettingsProviderMethodsDualKey(t *testing.T) {
	t.Setenv("MIDTRANS_SERVER_KEY", "dual-key-server")
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Dual Key","email":"dual-key@example.com","password":"secret123"}`, nil)
	if resp.StatusCode != 201 {
		t.Fatalf("register: got %d", resp.StatusCode)
	}
	cookies := resp.Cookies()
	resp.Body.Close()

	// Canonical key in, both keys out.
	resp = doRequest(t, app, "PATCH", "/api/settings",
		`{"company_name":"Dual Key","currency":"IDR","tax_rate":0,"invoice_prefix":"INV-","provider_methods":"qris"}`, cookies)
	if resp.StatusCode != 200 {
		t.Fatalf("settings patch (provider_methods): got %d", resp.StatusCode)
	}
	settings := decodeBody(t, resp)["settings"].(map[string]interface{})
	if settings["provider_methods"] != "qris" {
		t.Errorf("expected provider_methods=qris, got %v", settings["provider_methods"])
	}
	if settings["midtrans_methods"] != "qris" {
		t.Errorf("expected deprecated midtrans_methods alias=qris, got %v", settings["midtrans_methods"])
	}
	resp.Body.Close()

	// Deprecated alias in, both keys out.
	resp = doRequest(t, app, "PATCH", "/api/settings",
		`{"company_name":"Dual Key","currency":"IDR","tax_rate":0,"invoice_prefix":"INV-","midtrans_methods":"qris"}`, cookies)
	if resp.StatusCode != 200 {
		t.Fatalf("settings patch (midtrans_methods alias): got %d", resp.StatusCode)
	}
	settings = decodeBody(t, resp)["settings"].(map[string]interface{})
	if settings["provider_methods"] != "qris" || settings["midtrans_methods"] != "qris" {
		t.Errorf("expected both method keys=qris after alias patch, got %v / %v",
			settings["provider_methods"], settings["midtrans_methods"])
	}
	resp.Body.Close()

	// The read path emits both keys too.
	resp = doRequest(t, app, "GET", "/api/settings", "", cookies)
	if resp.StatusCode != 200 {
		t.Fatalf("settings get: got %d", resp.StatusCode)
	}
	settings = decodeBody(t, resp)["settings"].(map[string]interface{})
	if _, ok := settings["provider_methods"]; !ok {
		t.Error("settings response missing provider_methods")
	}
	if _, ok := settings["midtrans_methods"]; !ok {
		t.Error("settings response missing deprecated midtrans_methods alias")
	}
	resp.Body.Close()
}
