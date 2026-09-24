package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The public pay page lists the single configured Midtrans method (gopay by
// default) and only opens the gateway intent after it is chosen. A USD
// invoice is shown converted (IDR via the manual rate) for the IDR-only
// provider, and unconverted for the crypto provider.
func TestPublicPayMethodSelection(t *testing.T) {
	snapServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"snap-pub","redirect_url":"https://snap.test/pub"}`))
	}))
	defer snapServer.Close()
	npServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"inv_pub","invoice_url":"https://nowpayments.io/payment/?iid=pub"}`))
	}))
	defer npServer.Close()

	t.Setenv("MIDTRANS_SERVER_KEY", "pub-server-key")
	t.Setenv("MIDTRANS_SNAP_BASE_URL", snapServer.URL)
	t.Setenv("NOWPAYMENTS_API_KEY", "pub-np-key")
	t.Setenv("NOWPAYMENTS_BASE_URL", npServer.URL+"/v1")
	t.Setenv("NOWPAYMENTS_SANDBOX", "true")

	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Public MP","email":"public-mp@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()

	// Manual rate: $1 = Rp18000.
	resp = doRequest(t, app, "PATCH", "/api/settings",
		`{"company_name":"Public MP","currency":"USD","tax_rate":0,"invoice_prefix":"INV-","usd_to_idr":"18000"}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()

	spec := newInvoice()
	spec.ClientID, spec.Currency = createClient(t, app, cookies, "Public Payer USD"), "USD"
	spec.Items = []invoiceLine{{Description: "Service", Quantity: 1, Rate: "100"}}
	invoiceID := createInvoiceID(t, app, cookies, spec)

	resp = doRequest(t, app, "POST", "/api/payments/online", `{"invoiceId":"`+invoiceID+`"}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	token := decodeBody(t, resp)["token"].(string)

	resp = doRequest(t, app, "GET", "/api/public/pay/"+token, "", nil)
	require.Equal(t, 200, resp.StatusCode)
	publicData := decodeBody(t, resp)
	methods := publicData["methods"].([]interface{})
	require.NotEmpty(t, methods)
	byID := map[string]map[string]interface{}{}
	for _, m := range methods {
		entry := m.(map[string]interface{})
		byID[entry["id"].(string)] = entry
	}
	require.Contains(t, byID, "gopay")
	assert.Equal(t, "IDR", byID["gopay"]["currency"])
	assert.Equal(t, "1800000", byID["gopay"]["amount"]) // $100 × 18000
	require.Contains(t, byID, "crypto")
	assert.Equal(t, "USD", byID["crypto"]["currency"])
	assert.Equal(t, "100", byID["crypto"]["amount"])
	resp.Body.Close()

	// Choosing gopay routes to the configured Midtrans and returns a Snap token.
	resp = doRequest(t, app, "POST", "/api/public/pay/"+token+"/transaction", `{"payment_method":"gopay"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	qrisIntent := decodeBody(t, resp)
	assert.Equal(t, "snap-pub", qrisIntent["snap_token"])
	resp.Body.Close()

	// Choosing crypto routes to NOWPayments and returns a hosted payment URL.
	resp = doRequest(t, app, "POST", "/api/public/pay/"+token+"/transaction", `{"payment_method":"crypto"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	cryptoIntent := decodeBody(t, resp)
	assert.Equal(t, "https://nowpayments.io/payment/?iid=pub", cryptoIntent["payment_url"])
	assert.Empty(t, cryptoIntent["snap_token"])
	resp.Body.Close()
}

// An owner can restrict Midtrans to a subset of methods: the public page then
// hides the rest, a hidden method is rejected, and a non-Midtrans method
// (crypto) is unaffected by the Midtrans allowlist.
func TestPublicPayMidtransMethodAllowlist(t *testing.T) {
	snapServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"snap-allow","redirect_url":"https://snap.test/allow"}`))
	}))
	defer snapServer.Close()
	npServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"inv_allow","invoice_url":"https://nowpayments.io/payment/?iid=allow"}`))
	}))
	defer npServer.Close()

	t.Setenv("MIDTRANS_SERVER_KEY", "allow-server-key")
	t.Setenv("MIDTRANS_SNAP_BASE_URL", snapServer.URL)
	t.Setenv("NOWPAYMENTS_API_KEY", "allow-np-key")
	t.Setenv("NOWPAYMENTS_BASE_URL", npServer.URL+"/v1")
	t.Setenv("NOWPAYMENTS_SANDBOX", "true")

	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Allow MP","email":"allow-mp@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	resp.Body.Close()
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"allow-mp@example.com","password":"secret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	cookies := resp.Cookies()
	resp.Body.Close()

	// Allow only QRIS on Midtrans (plus the manual rate for the USD invoice).
	resp = doRequest(t, app, "PATCH", "/api/settings",
		`{"company_name":"Allow MP","currency":"USD","tax_rate":0,"invoice_prefix":"INV-","usd_to_idr":"18000","midtrans_methods":"qris,nonsense"}`,
		cookies)
	require.Equal(t, 200, resp.StatusCode)
	settings := decodeBody(t, resp)["settings"].(map[string]interface{})
	assert.Equal(t, "qris", settings["midtrans_methods"]) // unknown id dropped
	resp.Body.Close()

	spec := newInvoice()
	spec.ClientID, spec.Currency = createClient(t, app, cookies, "Allow Payer"), "USD"
	spec.Items = []invoiceLine{{Description: "Service", Quantity: 1, Rate: "100"}}
	invoiceID := createInvoiceID(t, app, cookies, spec)

	resp = doRequest(t, app, "POST", "/api/payments/online", `{"invoiceId":"`+invoiceID+`"}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	token := decodeBody(t, resp)["token"].(string)

	resp = doRequest(t, app, "GET", "/api/public/pay/"+token, "", nil)
	require.Equal(t, 200, resp.StatusCode)
	methods := decodeBody(t, resp)["methods"].([]interface{})
	ids := map[string]bool{}
	for _, m := range methods {
		ids[m.(map[string]interface{})["id"].(string)] = true
	}
	assert.True(t, ids["qris"])
	assert.False(t, ids["gopay"], "gopay should be hidden by the allowlist")
	assert.True(t, ids["crypto"], "the Midtrans allowlist must not hide crypto")
	resp.Body.Close()

	// A method hidden by the allowlist is rejected, not silently rerouted.
	resp = doRequest(t, app, "POST", "/api/public/pay/"+token+"/transaction", `{"payment_method":"gopay"}`, nil)
	assert.Equal(t, 400, resp.StatusCode)
	resp.Body.Close()

	// The allowed method still works.
	resp = doRequest(t, app, "POST", "/api/public/pay/"+token+"/transaction", `{"payment_method":"qris"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "snap-allow", decodeBody(t, resp)["snap_token"])
	resp.Body.Close()
}
