package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The local Snap intent is a session route. It must not be swept up by the
// API-key gateway middleware mounted at /api/gateway: sending a valid session
// cookie has to reach the handler, and an anonymous call must fail with the
// session error, never "missing api key".
func TestInvoiceIntentUsesSessionAuth(t *testing.T) {
	app := newTestApp()

	// Anonymous: rejected by session auth, not by the API-key middleware.
	resp := doGatewayRequest(t, app, "POST", "/api/invoices/"+uuid.NewString()+"/intents", "", nil, nil)
	require.Equal(t, 401, resp.StatusCode)
	errObject, ok := decodeBody(t, resp)["error"].(map[string]interface{})
	require.True(t, ok, "expected error envelope")
	assert.NotEqual(t, "missing api key", errObject["message"])
	resp.Body.Close()
}

// End-to-end: a signed-in user can open a Snap intent for a sent invoice.
func TestInvoiceIntentSessionFlow(t *testing.T) {
	snapServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"snap-tok-local","redirect_url":"https://snap.test/local"}`))
	}))
	defer snapServer.Close()
	t.Setenv("MIDTRANS_SERVER_KEY", "test-local-intent-key")
	t.Setenv("MIDTRANS_SNAP_BASE_URL", snapServer.URL)

	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Intent User","email":"intent-user@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	resp.Body.Close()
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"intent-user@example.com","password":"secret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	cookies := resp.Cookies()
	resp.Body.Close()

	spec := newInvoice()
	spec.ClientID = createClient(t, app, cookies, "Intent Payer")
	invoiceID := createInvoiceID(t, app, cookies, spec)

	resp = doRequest(t, app, "POST", "/api/invoices/"+invoiceID+"/intents", "", cookies)
	require.Equal(t, 201, resp.StatusCode)
	intent := decodeBody(t, resp)
	assert.Equal(t, "snap-tok-local", intent["snap_token"])
	assert.Equal(t, intent["snap_token"], intent["provider_token"], "provider_token aliases snap_token")
	assert.Equal(t, "midtrans", intent["gateway"])
	resp.Body.Close()
}
