package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/invoiceman/platform/database"
)

// sessionCookies returns cookies without the CSRF token (attacker's view:
// stolen session cookies cannot be paired with the header).
func sessionCookies(cookies []*http.Cookie) []*http.Cookie {
	out := []*http.Cookie{}
	for _, c := range cookies {
		if c.Name == "csrf_token" {
			continue
		}
		out = append(out, c)
	}
	return out
}

// TestCSRFProtection rejects cookie-session mutations without the header.
func TestCSRFProtection(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"CSRF User","email":"csrf@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	cookies := resp.Cookies()
	resp.Body.Close()

	// Mutation with only session cookies (no CSRF header) is rejected.
	req := httptest.NewRequest("POST", "/api/clients", strings.NewReader(`{"name":"Nope"}`))
	req.Header.Set("Content-Type", "application/json")
	for _, c := range sessionCookies(cookies) {
		req.AddCookie(c)
	}
	forged, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	require.NoError(t, err)
	forged.Body.Close()
	assert.Equal(t, 403, forged.StatusCode)

	// Same request with the header (mirrored by doRequest) succeeds.
	resp = doRequest(t, app, "POST", "/api/clients", `{"name":"Yep"}`, cookies)
	assert.Equal(t, 201, resp.StatusCode)
	resp.Body.Close()

	// Safe methods never need the header.
	resp = doRequest(t, app, "GET", "/api/clients", "", sessionCookies(cookies))
	assert.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()
}

// TestCaptchaEnforcement fails closed when configured without a token.
func TestCaptchaEnforcement(t *testing.T) {
	t.Setenv("TURNSTILE_SECRET", "dummy-secret")
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Bot","email":"bot@example.com","password":"secret123"}`, nil)
	require.Equal(t, 403, resp.StatusCode)
	body := decodeBody(t, resp)
	assert.Equal(t, "captcha verification failed", body["error"].(map[string]interface{})["message"])

	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"bot@example.com","password":"secret123"}`, nil)
	require.Equal(t, 403, resp.StatusCode)
	resp.Body.Close()
}
func TestAuditTrail(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Audit Admin","email":"audit-admin@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	adminID := decodeBody(t, resp)["user"].(map[string]interface{})["id"].(string)
	adminCookies := resp.Cookies()
	resp.Body.Close()

	// Promote explicitly: other tests may have registered first users.
	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	require.NoError(t, db.UpdateUserRole(uuid.MustParse(adminID), "admin"))

	resp = doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Audit User","email":"audit-user@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	targetID := decodeBody(t, resp)["user"].(map[string]interface{})["id"].(string)

	// Promote the user (admin-only action).
	resp = doRequest(t, app, "PATCH", "/api/admin/users/"+targetID+`/role`,
		`{"role":"moderator"}`, adminCookies)
	require.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()

	// The trail contains the change with actor context.
	resp = doRequest(t, app, "GET", "/api/admin/audit-logs", "", adminCookies)
	require.Equal(t, 200, resp.StatusCode)
	body := decodeBody(t, resp)
	logs := body["audit_logs"].([]interface{})
	require.NotEmpty(t, logs)
	found := false
	for _, l := range logs {
		entry := l.(map[string]interface{})
		if entry["action"] == "user.role.update" && entry["entity_id"] == targetID {
			found = true
			assert.NotEmpty(t, entry["user_id"])
		}
	}
	assert.True(t, found, "expected user.role.update in audit trail")
	assert.Contains(t, body, "meta")
}
