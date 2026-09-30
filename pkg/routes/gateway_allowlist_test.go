package routes

import (
	"context"
	"testing"

	"github.com/tertua/tupay/platform/gateway"
)

// allyGateway is a test-only second provider used to prove the method
// allowlist is per-provider, not method-global. It offers a neutral method id
// no built-in provider offers ("paylater"), so registering it never hijacks a
// method the other tests route. With a method-global allowlist its method
// would be vetoed (only qris is listed for the default provider); with the
// per-provider resolver it is offered.
type allyGateway struct{}

func (allyGateway) Name() string { return "ally" }

func (allyGateway) CreateTransaction(_ context.Context, _ *gateway.CreateTxRequest) (*gateway.CreateTxResponse, error) {
	return &gateway.CreateTxResponse{PaymentURL: "https://ally.test/pay", PaymentMethod: "paylater"}, nil
}

func (allyGateway) ParseAndVerify([]byte) (*gateway.NotificationResult, error) {
	return &gateway.NotificationResult{}, nil
}

func (allyGateway) Methods() []string { return []string{"paylater"} }
func (allyGateway) Configured() bool  { return true }
func (allyGateway) Sandbox() bool     { return true }

// TestPublicPaySecondProviderNotVetoedByMidtransAllowlist proves the allowlist
// is keyed by provider: the default provider's QRIS-only allowlist must not
// hide a second provider's method. The fake provider is registered inside the
// test (unique method id, so it never affects other tests regardless of order).
func TestPublicPaySecondProviderNotVetoedByMidtransAllowlist(t *testing.T) {
	gateway.Register(allyGateway{})
	t.Setenv("MIDTRANS_SERVER_KEY", "gen-server-key")
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Gen MP","email":"gen-mp@example.com","password":"secret123"}`, nil)
	if resp.StatusCode != 201 {
		t.Fatalf("register: got %d", resp.StatusCode)
	}
	cookies := resp.Cookies()
	resp.Body.Close()

	resp = doRequest(t, app, "PATCH", "/api/settings",
		`{"company_name":"Gen MP","currency":"IDR","tax_rate":0,"invoice_prefix":"INV-","usd_to_idr":"18000","provider_methods":"qris"}`, cookies)
	if resp.StatusCode != 200 {
		t.Fatalf("settings: got %d", resp.StatusCode)
	}
	resp.Body.Close()

	spec := newInvoice()
	spec.ClientID = createClient(t, app, cookies, "Gen Payer")
	spec.Items = []invoiceLine{{Description: "Service", Quantity: 1, Rate: "20000"}}
	invoiceID := createInvoiceID(t, app, cookies, spec)

	resp = doRequest(t, app, "POST", "/api/payments/online", `{"invoiceId":"`+invoiceID+`"}`, cookies)
	if resp.StatusCode != 200 {
		t.Fatalf("online payment: got %d", resp.StatusCode)
	}
	token := decodeBody(t, resp)["token"].(string)
	resp.Body.Close()

	resp = doRequest(t, app, "GET", "/api/public/pay/"+token, "", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("public pay: got %d", resp.StatusCode)
	}
	ids := map[string]bool{}
	for _, m := range decodeBody(t, resp)["methods"].([]interface{}) {
		ids[m.(map[string]interface{})["id"].(string)] = true
	}
	resp.Body.Close()
	if !ids["paylater"] {
		t.Errorf("second provider's method hidden by the default provider allowlist: %v", ids)
	}
	if !ids["qris"] {
		t.Errorf("default provider still must offer qris: %v", ids)
	}
}

// TestSettingsMethodsFlow pins the settings method-picker route and shape:
// GET /settings/methods still returns the default provider's declared methods
// as [{id,name}] and an unknown ?gateway= yields an empty list, never an error.
func TestSettingsMethodsFlow(t *testing.T) {
	t.Setenv("MIDTRANS_SERVER_KEY", "sm-server-key")
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"SM User","email":"sm-user@example.com","password":"secret123"}`, nil)
	if resp.StatusCode != 201 {
		t.Fatalf("register: got %d", resp.StatusCode)
	}
	cookies := resp.Cookies()
	resp.Body.Close()

	resp = doRequest(t, app, "GET", "/api/settings/methods", "", cookies)
	if resp.StatusCode != 200 {
		t.Fatalf("settings/methods: got %d", resp.StatusCode)
	}
	methods := decodeBody(t, resp)["methods"].([]interface{})
	if len(methods) == 0 {
		t.Fatal("settings/methods returned no methods")
	}
	byID := map[string]string{}
	for _, m := range methods {
		entry := m.(map[string]interface{})
		byID[entry["id"].(string)] = entry["name"].(string)
	}
	resp.Body.Close()
	if byID["qris"] != "QRIS" {
		t.Errorf("qris label = %q, want QRIS", byID["qris"])
	}

	resp = doRequest(t, app, "GET", "/api/settings/methods?gateway=nope", "", cookies)
	if resp.StatusCode != 200 {
		t.Fatalf("unknown gateway: got %d", resp.StatusCode)
	}
	if empty := decodeBody(t, resp)["methods"].([]interface{}); len(empty) != 0 {
		t.Errorf("unknown gateway methods = %v, want empty", empty)
	}
	resp.Body.Close()
}
