package routes

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/tupay/pkg/middleware"
)

// TestMetricsEndpoint verifies the scrape endpoint renders counters
// for recorded routes but never records itself.
func TestMetricsEndpoint(t *testing.T) {
	app := fiber.New()
	middleware.FiberMiddleware(app)
	HealthRoutes(app)
	MetricsRoutes(app)
	PublicRoutes(app)

	// Generate traffic on a known route template.
	req := httptest.NewRequest("GET", "/api/config", http.NoBody)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)

	// Scrape twice: the second scrape must not count the first.
	for i := 0; i < 2; i++ {
		req = httptest.NewRequest("GET", "/metrics", http.NoBody)
		resp, err = app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")
		out := string(body)
		assert.Contains(t, out, "tupay_app_info")
		assert.Contains(t, out, `route="/api/config"`)
		assert.NotContains(t, out, `route="/metrics"`, "scrape must not record itself")
	}
}

// TestMetricsDisabled ensures the endpoint is gone with METRICS_ENABLED=false.
func TestMetricsDisabled(t *testing.T) {
	t.Setenv("METRICS_ENABLED", "false")

	app := fiber.New()
	MetricsRoutes(app)

	req := httptest.NewRequest("GET", "/metrics", http.NoBody)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, 404, resp.StatusCode)
}
