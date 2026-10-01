package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	resp = doRequest(t, app, "POST", "/api/auth/reset-password",
		`{"token":"nope","new_password":"anothersecret123"}`, nil)
	require.Equal(t, 403, resp.StatusCode)
	resp.Body.Close()
}

// TestCSRFBindingEnforced cross-checks the token against the session
// store: the cookie/header pair alone is not enough.
func TestCSRFBindingEnforced(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Bound User","email":"bound@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	cookies := resp.Cookies()
	resp.Body.Close()

	// A self-consistent forged pair (cookie + header agree) from another
	// session is rejected: it is not bound to this session.
	req := httptest.NewRequest("POST", "/api/clients", strings.NewReader(`{"name":"Forged"}`))
	req.Header.Set("Content-Type", "application/json")
	for _, c := range sessionCookies(cookies) {
		req.AddCookie(c)
	}
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: strings.Repeat("f", 64)})
	req.Header.Set("X-CSRF-Token", strings.Repeat("f", 64))
	forged, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	require.NoError(t, err)
	forged.Body.Close()
	assert.Equal(t, 403, forged.StatusCode)
}

// TestCSRFRotationOnPasswordChange proves the old token dies server-side
// after a password change (privilege moment), while the fresh cookie works.
func TestCSRFRotationOnPasswordChange(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Rotate User","email":"rotate@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	cookies := resp.Cookies()
	resp.Body.Close()

	var oldCSRF string
	for _, c := range cookies {
		if c.Name == "csrf_token" {
			oldCSRF = c.Value
		}
	}
	require.NotEmpty(t, oldCSRF)

	resp = doRequest(t, app, "PATCH", "/api/auth/password",
		`{"currentPassword":"secret123","newPassword":"rotatedsecret123"}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	fresh := mergeCookies(cookies, resp.Cookies())
	resp.Body.Close()

	// Stale pair replays are rejected even though cookie and header agree.
	req := httptest.NewRequest("POST", "/api/clients", strings.NewReader(`{"name":"Stale"}`))
	req.Header.Set("Content-Type", "application/json")
	for _, c := range sessionCookies(fresh) {
		req.AddCookie(c)
	}
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: oldCSRF})
	req.Header.Set("X-CSRF-Token", oldCSRF)
	stale, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	require.NoError(t, err)
	stale.Body.Close()
	assert.Equal(t, 403, stale.StatusCode)

	// The fresh cookie/header pair works.
	resp = doRequest(t, app, "POST", "/api/clients", `{"name":"Fresh"}`, fresh)
	assert.Equal(t, 201, resp.StatusCode)
	resp.Body.Close()
}

// TestAuthAuditTrail covers login/register/logout entries: successes carry
// the user id, failures carry a hashed email (never plaintext) + reason.
func TestAuthAuditTrail(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Trail User","email":"trail@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	trailID := decodeBody(t, resp)["user"].(map[string]interface{})["id"].(string)
	trailCookies := resp.Cookies()
	resp.Body.Close()

	// Failed login (wrong password) + successful login + logout.
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"trail@example.com","password":"wrongpass"}`, nil)
	require.Equal(t, 401, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"trail@example.com","password":"secret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	loginCookies := resp.Cookies()
	resp.Body.Close()

	resp = doRequest(t, app, "POST", "/api/auth/logout", "", loginCookies)
	require.Equal(t, 204, resp.StatusCode)
	resp.Body.Close()

	// Promote to read the trail (this register is not necessarily the first account).
	adminCookies := adminSession(t, app, "Trail User", "trail@example.com")

	resp = doRequest(t, app, "GET", "/api/admin/audit-logs?per_page=100", "", adminCookies)
	require.Equal(t, 200, resp.StatusCode)
	body := decodeBody(t, resp)
	logs := body["audit_logs"].([]interface{})

	seen := map[string]bool{}
	var failedEntity string
	for _, l := range logs {
		entry := l.(map[string]interface{})
		action, _ := entry["action"].(string)
		entity, _ := entry["entity_id"].(string)
		switch action {
		case "auth.register", "auth.login.success", "auth.logout":
			if entity == trailID {
				seen[action] = true
			}
		case "auth.login.failed":
			failedEntity = entity
			seen[action] = true
		}
	}
	assert.True(t, seen["auth.register"], "expected auth.register in trail")
	assert.True(t, seen["auth.login.success"], "expected auth.login.success in trail")
	assert.True(t, seen["auth.login.failed"], "expected auth.login.failed in trail")
	assert.True(t, seen["auth.logout"], "expected auth.logout in trail")
	assert.NotEmpty(t, failedEntity, "failed login must carry an entity id")
	assert.NotContains(t, failedEntity, "trail@example.com", "email must be hashed, never plaintext")

	_ = trailCookies
}

func TestAuditTrail(t *testing.T) {
	app := newTestApp()

	adminCookies := adminSession(t, app, "Audit Admin", "audit-admin@example.com")

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Audit User","email":"audit-user@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	targetID := decodeBody(t, resp)["user"].(map[string]interface{})["id"].(string)

	// The trail contains the change with actor context.
	resp = doRequest(t, app, "GET", "/api/admin/audit-logs", "", adminCookies)
	require.Equal(t, 200, resp.StatusCode)
	body := decodeBody(t, resp)
	logs := body["audit_logs"].([]interface{})
	require.NotEmpty(t, logs)
	found := false
	for _, l := range logs {
		entry := l.(map[string]interface{})
		if entry["action"] == "auth.register" && entry["entity_id"] == targetID {
			found = true
			assert.NotEmpty(t, entry["user_id"])
		}
	}
	assert.True(t, found, "expected auth.register in audit trail")
	assert.Contains(t, body, "meta")
}
