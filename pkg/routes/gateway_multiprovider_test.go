package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/invoiceman/platform/database"
)

// multiProviderAdmin boots an app with a configured Midtrans and returns the
// admin session plus a helper to register projects.
func multiProviderAdmin(t *testing.T, app *fiber.App) (cookies []*http.Cookie, create func(body string) map[string]interface{}) {
	t.Helper()
	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"MP Admin","email":"mp-admin@example.com","password":"secret123"}`, nil)
	require.True(t, resp.StatusCode == 201 || resp.StatusCode == 409)
	resp.Body.Close()
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"mp-admin@example.com","password":"secret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	adminID := uuid.MustParse(decodeBody(t, resp)["user"].(map[string]interface{})["id"].(string))
	resp.Body.Close()
	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	require.NoError(t, db.UpdateUserRole(adminID, "admin"))
	// Refresh session so the new role is active.
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"mp-admin@example.com","password":"secret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	cookies = resp.Cookies()
	resp.Body.Close()

	create = func(body string) map[string]interface{} {
		resp := doRequest(t, app, "POST", "/api/admin/gateway/projects", body, cookies)
		require.Equal(t, 201, resp.StatusCode)
		return decodeBody(t, resp)["project"].(map[string]interface{})
	}
	return cookies, create
}

// A method wins over the project default: a NOWPayments-default project that
// asks for QRIS must be routed to the configured Midtrans.
func TestIntentMethodRoutesAcrossProviders(t *testing.T) {
	snapServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"snap-mp","redirect_url":"https://snap.test/mp"}`))
	}))
	defer snapServer.Close()
	t.Setenv("MIDTRANS_SERVER_KEY", "mp-server-key")
	t.Setenv("MIDTRANS_SNAP_BASE_URL", snapServer.URL)

	app := newTestApp()
	_, create := multiProviderAdmin(t, app)
	project := create(`{"slug":"mp-shop","name":"MP Shop","webhook_url":"https://mp.example/hook","default_gateway":"nowpayments"}`)
	apiKey := project["api_key"].(string)

	resp := doGatewayRequest(t, app, "POST", "/api/gateway/intents",
		`{"external_order_id":"mp-qris-1","payment_method":"qris","amount_idr":32000}`,
		map[string]string{"X-Api-Key": apiKey}, nil)
	require.Equal(t, 201, resp.StatusCode)
	intent := decodeBody(t, resp)
	assert.Equal(t, "midtrans", intent["gateway"])
	assert.Equal(t, "qris", intent["payment_method"])
	resp.Body.Close()
}

// A USD intent charged through the IDR-only provider uses the manual rate.
func TestIntentConvertsUSDWithManualRate(t *testing.T) {
	snapServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"snap-fx","redirect_url":"https://snap.test/fx"}`))
	}))
	defer snapServer.Close()
	t.Setenv("MIDTRANS_SERVER_KEY", "fx-server-key")
	t.Setenv("MIDTRANS_SNAP_BASE_URL", snapServer.URL)

	app := newTestApp()
	cookies, create := multiProviderAdmin(t, app)

	// Manual rate: $1 = Rp18000 (money crosses as a decimal string).
	resp := doRequest(t, app, "PATCH", "/api/settings",
		`{"company_name":"MP","currency":"IDR","tax_rate":0,"invoice_prefix":"INV-","usd_to_idr":"18000"}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()

	project := create(`{"slug":"fx-shop","name":"FX Shop","webhook_url":"https://fx.example/hook","default_gateway":"midtrans"}`)
	apiKey := project["api_key"].(string)

	resp = doGatewayRequest(t, app, "POST", "/api/gateway/intents",
		`{"external_order_id":"fx-usd-1","payment_method":"qris","amount_decimal":"25.50","currency":"USD"}`,
		map[string]string{"X-Api-Key": apiKey}, nil)
	require.Equal(t, 201, resp.StatusCode)
	intent := decodeBody(t, resp)
	assert.Equal(t, "midtrans", intent["gateway"])
	assert.EqualValues(t, 459000, intent["amount_idr"])
	assert.Equal(t, "IDR", intent["currency"])
	assert.Equal(t, "USD", intent["invoice_currency"])
	resp.Body.Close()
}
