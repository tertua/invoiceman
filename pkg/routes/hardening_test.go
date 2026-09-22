package routes

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/invoiceman/pkg/configs"
	"github.com/tertua/invoiceman/pkg/middleware"
)

// newHardenedApp builds the app the same way main.go does:
// global middleware first, then health probes, then API routes.
func newHardenedApp() *fiber.App {
	app := fiber.New()
	middleware.FiberMiddleware(app)
	HealthRoutes(app)
	PublicRoutes(app)
	GatewayRoutes(app)
	PrivateRoutes(app)
	return app
}

// TestHealthProbes covers liveness and readiness without auth.
func TestHealthProbes(t *testing.T) {
	app := fiber.New()
	HealthRoutes(app)

	resp := doRequest(t, app, "GET", "/healthz", "", nil)
	assert.Equal(t, 200, resp.StatusCode)
	body := decodeBody(t, resp)
	assert.Equal(t, "ok", body["status"])
	assert.NotEmpty(t, body["version"])

	resp = doRequest(t, app, "GET", "/readyz", "", nil)
	assert.Equal(t, 200, resp.StatusCode)
	readyBody := decodeBody(t, resp)
	assert.Equal(t, "ready", readyBody["status"])
}

// TestSecurityHeadersAndRequestID verifies P0 hardening headers.
func TestSecurityHeadersAndRequestID(t *testing.T) {
	app := newHardenedApp()

	req := httptest.NewRequest("GET", "/healthz", nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"), "request id must be set")
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "SAMEORIGIN", resp.Header.Get("X-Frame-Options"))
	assert.Empty(t, resp.Header.Get("Content-Security-Policy"), "CSP stays off for the JSON API")

	// A client-provided request id is echoed back.
	req = httptest.NewRequest("GET", "/healthz", nil)
	req.Header.Set("X-Request-ID", "test-123")
	resp2, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	require.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, "test-123", resp2.Header.Get("X-Request-ID"))
}

// TestRecoverMiddleware ensures a panicking handler becomes a JSON 500.
func TestRecoverMiddleware(t *testing.T) {
	app := fiber.New()
	middleware.FiberMiddleware(app)
	app.Get("/panic", func(_ fiber.Ctx) error { panic("boom") })

	req := httptest.NewRequest("GET", "/panic", nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 500, resp.StatusCode)
	body := decodeBody(t, resp)
	errObj, ok := body["error"].(map[string]interface{})
	require.True(t, ok, "recover must keep the shared error envelope")
	assert.Equal(t, "internal server error", errObj["message"])
}

// TestAuthRateLimitExceeded verifies the strict auth limiter trips with
// the shared error envelope (429).
func TestAuthRateLimitExceeded(t *testing.T) {
	t.Setenv("RATE_LIMIT_AUTH", "2")

	app := fiber.New()
	PublicRoutes(app)

	for i := 0; i < 2; i++ {
		resp := doRequest(t, app, "POST", "/api/auth/login",
			`{"email":"nobody@example.com","password":"wrongpassword123"}`, nil)
		resp.Body.Close()
		require.NotEqual(t, http.StatusTooManyRequests, resp.StatusCode)
	}

	resp := doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"nobody@example.com","password":"wrongpassword123"}`, nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
}

// TestTrustedProxyIP verifies proxy trust wiring end to end: without
// TRUSTED_PROXIES a spoofed X-Forwarded-For is ignored (c.IP falls back to
// the connection IP), and with the client IP trusted the header is honored.
func TestTrustedProxyIP(t *testing.T) {
	echoIPApp := func() *fiber.App {
		app := fiber.New(configs.FiberConfig())
		app.Get("/echo-ip", func(c fiber.Ctx) error {
			return c.SendString(c.IP())
		})
		return app
	}
	getIP := func(app *fiber.App, forwardedFor string) string {
		req := httptest.NewRequest("GET", "/echo-ip", nil)
		if forwardedFor != "" {
			req.Header.Set("X-Forwarded-For", forwardedFor)
		}
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		return string(raw)
	}

	// Untrusted (default): the spoofed header is ignored.
	t.Setenv("TRUSTED_PROXIES", "")
	got := getIP(echoIPApp(), "203.0.113.9")
	assert.NotEqual(t, "203.0.113.9", got)

	// Trusted: fiber's test transport presents the remote as 0.0.0.0, so
	// trusting it makes c.IP honor the header.
	t.Setenv("TRUSTED_PROXIES", "0.0.0.0/32")
	got = getIP(echoIPApp(), "203.0.113.9")
	assert.Equal(t, "203.0.113.9", got)

	// Trusting an unrelated IP keeps the header ignored.
	t.Setenv("TRUSTED_PROXIES", "198.51.100.7")
	got = getIP(echoIPApp(), "203.0.113.9")
	assert.NotEqual(t, "203.0.113.9", got)
}
