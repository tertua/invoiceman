package routes

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/tupay/platform/gateway"
)

// TestGatewayConfigDefaultProvider verifies GET /gateway/config returns the
// browser payer config for the default provider with the same JSON keys the
// pay page has always read (client_key / is_production / configured).
func TestGatewayConfigDefaultProvider(t *testing.T) {
	t.Setenv("MIDTRANS_CLIENT_KEY", "ck-public-1")
	t.Setenv("MIDTRANS_SERVER_KEY", "sk-secret-1")
	t.Setenv("MIDTRANS_IS_PROD", "true")
	app := newTestApp()

	resp := doGatewayRequest(t, app, "GET", "/api/public/gateway/config", "", nil, nil)
	require.Equal(t, 200, resp.StatusCode)
	body := decodeBody(t, resp)
	assert.Equal(t, "midtrans", body["gateway"])
	assert.Equal(t, "ck-public-1", body["client_key"])
	assert.Equal(t, true, body["is_production"])
	assert.Equal(t, true, body["configured"])

	// Server credentials must never leak through the public config endpoint.
	raw, _ := json.Marshal(body)
	assert.NotContains(t, string(raw), "sk-secret-1")
	assert.NotContains(t, string(raw), "server_key")
	resp.Body.Close()
}

// TestGatewayConfigExplicitProvider verifies ?gateway= selects a provider and
// an unknown name fails with 400 (stable error, no panic).
func TestGatewayConfigExplicitProvider(t *testing.T) {
	t.Setenv("NOWPAYMENTS_API_KEY", "np-public-1")
	app := newTestApp()

	resp := doGatewayRequest(t, app, "GET", "/api/public/gateway/config?gateway=nowpayments", "", nil, nil)
	require.Equal(t, 200, resp.StatusCode)
	body := decodeBody(t, resp)
	assert.Equal(t, "nowpayments", body["gateway"])
	assert.Equal(t, true, body["configured"])
	// NOWPayments declares no payer config: no client_key is added.
	assert.NotContains(t, body, "client_key")
	resp.Body.Close()

	resp = doGatewayRequest(t, app, "GET", "/api/public/gateway/config?gateway=nope", "", nil, nil)
	require.Equal(t, 400, resp.StatusCode)
	body = decodeBody(t, resp)
	require.NotNil(t, body["error"])
	assert.Equal(t, "unknown payment gateway", body["error"].(map[string]interface{})["message"])
	resp.Body.Close()
}

// TestGatewayConfigCaseInsensitive verifies the provider name is normalized
// (trim + lowercase) before registry lookup.
func TestGatewayConfigCaseInsensitive(t *testing.T) {
	t.Setenv("MIDTRANS_CLIENT_KEY", "ck-case")
	app := newTestApp()
	resp := doGatewayRequest(t, app, "GET", "/api/public/gateway/config?gateway=%20MIDTRANS%20", "", nil, nil)
	require.Equal(t, 200, resp.StatusCode)
	body := decodeBody(t, resp)
	assert.Equal(t, "midtrans", body["gateway"])
	assert.Equal(t, "ck-case", body["client_key"])
	resp.Body.Close()
}

// TestPayerConfigHidesServerKey asserts on the capability map directly: a
// provider's PayerConfig must never contain server-side keys.
func TestPayerConfigHidesServerKey(t *testing.T) {
	t.Setenv("MIDTRANS_SERVER_KEY", "sk-hidden")
	t.Setenv("MIDTRANS_CLIENT_KEY", "ck-visible")

	gw, err := gateway.Get("midtrans")
	require.NoError(t, err)
	provider, ok := gw.(gateway.PayerConfigProvider)
	require.True(t, ok, "midtrans must implement PayerConfigProvider")

	conf := provider.PayerConfig()
	assert.Equal(t, "ck-visible", conf["client_key"])
	assert.Contains(t, conf, "is_production")
	for key := range conf {
		assert.NotContains(t, key, "server")
	}
}

// TestGatewayStatusViaProviderMap verifies GET /public/gateway/status still
// reports the provider as configured once its env credential is set through
// the provider-keyed config map (regression guard for the FromEnv rewiring).
func TestGatewayStatusViaProviderMap(t *testing.T) {
	t.Setenv("MIDTRANS_SERVER_KEY", "sk-status-1")
	app := newTestApp()

	resp := doGatewayRequest(t, app, "GET", "/api/public/gateway/status", "", nil, nil)
	require.Equal(t, 200, resp.StatusCode)
	body := decodeBody(t, resp)
	raw, _ := json.Marshal(body)
	assert.NotContains(t, string(raw), "sk-status-1")
	for _, g := range body["gateways"].([]interface{}) {
		m := g.(map[string]interface{})
		if m["name"] == "midtrans" {
			assert.Equal(t, true, m["configured"])
		}
	}
	resp.Body.Close()
}

// TestPublicPayPayloadGatewayBlock verifies the public pay payload still
// exposes gateway.client_key / gateway.is_production for the default provider
// (response-shape compatibility for the frontend).
func TestPublicPayPayloadGatewayBlock(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Config Block","email":"pay-block@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()
	spec := newInvoice()
	spec.ClientID = createClient(t, app, cookies, "Config Payer")
	spec.Items = []invoiceLine{{Description: "Config service", Quantity: 1, Rate: "100000"}}
	invoiceID := createInvoiceID(t, app, cookies, spec)

	resp = doRequest(t, app, "POST", "/api/payments/online", `{"invoiceId":"`+invoiceID+`"}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	token := decodeBody(t, resp)["token"].(string)

	t.Setenv("MIDTRANS_CLIENT_KEY", "ck-pay")
	t.Setenv("MIDTRANS_IS_PROD", "false")
	resp = doRequest(t, app, "GET", "/api/public/pay/"+token, "", nil)
	require.Equal(t, 200, resp.StatusCode)
	body := decodeBody(t, resp)
	gwBlock, ok := body["gateway"].(map[string]interface{})
	require.True(t, ok, "expected gateway block, got %v", body["gateway"])
	assert.Equal(t, "ck-pay", gwBlock["client_key"])
	assert.Equal(t, false, gwBlock["is_production"])
	resp.Body.Close()
}
