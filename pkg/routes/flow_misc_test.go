package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/invoiceman/pkg/middleware"
)

func TestAIContractFlow(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/ai/business-summary", "", nil)
	assert.Equal(t, 401, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"AI User","email":"ai@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()

	// Valid requests return a clear configuration response without contacting Gemini.
	resp = doRequest(t, app, "POST", "/api/ai/business-summary", "", cookies)
	assert.Equal(t, 501, resp.StatusCode)
	assert.Contains(t, decodeBody(t, resp)["error"].(map[string]interface{})["message"], "not configured")

	resp = doRequest(t, app, "POST", "/api/ai/write-note", `{"kind":"invalid"}`, cookies)
	assert.Equal(t, 400, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, app, "POST", "/api/ai/write-note", `{"kind":"terms"}`, cookies)
	assert.Equal(t, 501, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, app, "POST", "/api/ai/payment-reminder", `{"invoiceId":"not-a-uuid","tone":"friendly"}`, cookies)
	assert.Equal(t, 400, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, app, "POST", "/api/ai/receipt-parse", "", cookies)
	assert.Equal(t, 400, resp.StatusCode)
	resp.Body.Close()
}

// TestCORSOrigins covers credentialed CORS for configured frontend origins.
func TestCORSOrigins(t *testing.T) {
	t.Setenv("CORS_ORIGINS", "https://app.example.com, https://admin.example.com")
	// Mirror main.go wiring: middleware first so CORS headers apply to routes.
	app := fiber.New()
	middleware.FiberMiddleware(app)
	PublicRoutes(app)
	GatewayRoutes(app)
	PrivateRoutes(app)

	preflight := func(origin string) *http.Response {
		req := httptest.NewRequest("OPTIONS", "/api/config", http.NoBody)
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", "GET")
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
		require.NoError(t, err)
		return resp
	}

	resp := preflight("https://app.example.com")
	assert.Equal(t, "https://app.example.com", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", resp.Header.Get("Access-Control-Allow-Credentials"))
	resp.Body.Close()

	resp = preflight("https://evil.example.com")
	assert.NotEqual(t, "https://evil.example.com", resp.Header.Get("Access-Control-Allow-Origin"))
	resp.Body.Close()
}

// TestAppConfig covers the public branding endpoint and APP_NAME override.
func TestAppConfig(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "GET", "/api/config", "", nil)
	require.Equal(t, 200, resp.StatusCode)
	body := decodeBody(t, resp)
	assert.Equal(t, "Invoiceman", body["appName"])
	assert.Equal(t, true, body["allowRegistration"])

	t.Setenv("APP_NAME", "  Acme Billing  ")
	resp = doRequest(t, app, "GET", "/api/config", "", nil)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "Acme Billing", decodeBody(t, resp)["appName"])
	resp.Body.Close()
}

// TestRegistrationToggle covers the ALLOW_REGISTRATION kill switch:
// closed mode rejects new signups with 403 while login keeps working.
