package routes

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newVersionedApp mirrors main.go registration (v1 before legacy).
func newVersionedApp() *fiber.App {
	app := fiber.New()
	HealthRoutes(app)
	app.Use(DeprecationHeaders())
	RegisterAPI(app, APIV1Prefix)
	RegisterAPI(app, APILegacyPrefix)
	return app
}

// TestVersionParity verifies /api/v1 mirrors /api and only legacy
// responses carry deprecation headers.
func TestVersionParity(t *testing.T) {
	app := newVersionedApp()

	legacyReq := httptest.NewRequest("GET", "/api/config", nil)
	legacyResp, err := app.Test(legacyReq, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	require.NoError(t, err)
	legacyBody := decodeBody(t, legacyResp)
	require.Equal(t, 200, legacyResp.StatusCode)
	assert.Equal(t, "true", legacyResp.Header.Get("Deprecation"))
	assert.NotEmpty(t, legacyResp.Header.Get("Sunset"))

	v1Req := httptest.NewRequest("GET", "/api/v1/config", nil)
	v1Resp, err := app.Test(v1Req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	require.NoError(t, err)
	v1Body := decodeBody(t, v1Resp)
	require.Equal(t, 200, v1Resp.StatusCode)
	assert.Empty(t, v1Resp.Header.Get("Deprecation"))
	assert.Empty(t, v1Resp.Header.Get("Sunset"))

	assert.Equal(t, legacyBody, v1Body)
}

// TestVersionedAuthFlow proves session auth works under the v1 prefix.
func TestVersionedAuthFlow(t *testing.T) {
	app := newVersionedApp()

	resp := doRequest(t, app, "POST", "/api/v1/auth/register",
		`{"name":"V1 User","email":"v1@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	cookies := resp.Cookies()
	resp.Body.Close()

	resp = doRequest(t, app, "GET", "/api/v1/auth/me", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	body := decodeBody(t, resp)
	assert.Equal(t, "V1 User", body["user"].(map[string]interface{})["name"])
}
