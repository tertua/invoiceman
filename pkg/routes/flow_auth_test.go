package routes

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/tupay/platform/database"
	"github.com/tertua/tupay/platform/relay"
)

func TestAuthFlow(t *testing.T) {
	// The flow performs ~11 auth calls (registers, logins, resets); raise
	// the per-test budget so the strict default limiter stays out of the way
	// (dedicated coverage lives in TestAuthRateLimitExceeded).
	t.Setenv("RATE_LIMIT_AUTH", "100")
	app := newTestApp()

	// Register a new user.
	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Flow User","email":"flow@example.com","password":"secret123"}`, nil)
	assert.Equal(t, 201, resp.StatusCode)
	body := decodeBody(t, resp)
	assert.Equal(t, "flow@example.com", body["user"].(map[string]interface{})["email"])
	// Order-proof: only the first-ever account is admin by default, so
	// promote explicitly when an earlier test already claimed it.
	if body["user"].(map[string]interface{})["role"] != "admin" {
		flowID := body["user"].(map[string]interface{})["id"].(string)
		db, err := database.OpenDBConnection()
		require.NoError(t, err)
		require.NoError(t, db.UpdateUserRole(uuid.MustParse(flowID), "admin"))
		resp = doRequest(t, app, "POST", "/api/auth/login",
			`{"email":"flow@example.com","password":"secret123"}`, nil)
		require.Equal(t, 200, resp.StatusCode)
		body = decodeBody(t, resp)
	}
	assert.Equal(t, "admin", body["user"].(map[string]interface{})["role"])
	cookies := resp.Cookies()
	require.NotEmpty(t, cookies)

	// Duplicate email is rejected.
	resp = doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Flow User","email":"flow@example.com","password":"secret123"}`, nil)
	assert.Equal(t, 409, resp.StatusCode)
	resp.Body.Close()

	// Session user.
	resp = doRequest(t, app, "GET", "/api/auth/me", "", cookies)
	assert.Equal(t, 200, resp.StatusCode)
	body = decodeBody(t, resp)
	assert.Equal(t, "Flow User", body["user"].(map[string]interface{})["name"])

	// Update profile.
	resp = doRequest(t, app, "PATCH", "/api/auth/profile", `{"name":"Renamed"}`, cookies)
	assert.Equal(t, 200, resp.StatusCode)
	body = decodeBody(t, resp)
	assert.Equal(t, "Renamed", body["user"].(map[string]interface{})["name"])

	// Wrong password is rejected.
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"flow@example.com","password":"wrongpass"}`, nil)
	assert.Equal(t, 401, resp.StatusCode)
	resp.Body.Close()

	// Login works.
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"flow@example.com","password":"secret123"}`, nil)
	assert.Equal(t, 200, resp.StatusCode)
	loginBody := decodeBody(t, resp)
	userID := loginBody["user"].(map[string]interface{})["id"].(string)
	cookies = resp.Cookies()

	// Expired access tokens are refreshed transparently via the refresh cookie.
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  userID,
		"exp": time.Now().Add(-time.Hour).Unix(),
	})
	expiredString, err := expiredToken.SignedString([]byte(os.Getenv("JWT_SECRET_KEY")))
	require.NoError(t, err)
	var refreshCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "refresh_token" {
			refreshCookie = cookie
		}
	}
	require.NotNil(t, refreshCookie)
	resp = doRequest(t, app, "GET", "/api/auth/me", "", []*http.Cookie{
		{Name: "access_token", Value: expiredString},
		refreshCookie,
	})
	assert.Equal(t, 200, resp.StatusCode)
	refreshedBody := decodeBody(t, resp)
	assert.Equal(t, "flow@example.com", refreshedBody["user"].(map[string]interface{})["email"])
	renewed := false
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "access_token" && cookie.Value != "" && cookie.Value != expiredString {
			renewed = true
		}
	}
	assert.True(t, renewed, "expected a fresh access_token cookie")
	cookies = mergeCookies(cookies, resp.Cookies())

	// The renewed access cookie outlives the JWT inside it: its Expires
	// must reach the refresh horizon, otherwise browsers drop the expired
	// hint after 15 minutes of idle and transparent refresh can never fire.
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "access_token" && cookie.Value != "" {
			assert.WithinDuration(t, time.Now().Add(720*time.Hour), cookie.Expires, 30*time.Minute)
		}
	}

	// Change password.
	resp = doRequest(t, app, "PATCH", "/api/auth/password",
		`{"currentPassword":"secret123","newPassword":"newsecret123"}`, cookies)
	assert.Equal(t, 200, resp.StatusCode)
	decodeBody(t, resp)

	// Login with the new password.
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"flow@example.com","password":"newsecret123"}`, nil)
	assert.Equal(t, 200, resp.StatusCode)
	decodeBody(t, resp)
	cookies = resp.Cookies()
	refreshCookie = nil
	for _, cookie := range cookies {
		if cookie.Name == "refresh_token" {
			refreshCookie = cookie
		}
	}
	require.NotNil(t, refreshCookie)

	// Forgot password always succeeds generically.
	resp = doRequest(t, app, "POST", "/api/auth/forgot-password",
		`{"email":"flow@example.com"}`, nil)
	assert.Equal(t, 200, resp.StatusCode)
	decodeBody(t, resp)

	// Reset with an unknown token fails.
	resp = doRequest(t, app, "POST", "/api/auth/reset-password",
		`{"token":"nope","new_password":"anothersecret123"}`, nil)
	assert.Equal(t, 400, resp.StatusCode)
	decodeBody(t, resp)

	// Reset tokens are stored hashed: the DB row holds relay.HashKey(raw),
	// never the raw link, and the raw token still resets the password.
	rawToken := strings.Repeat("a", 64)
	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	require.NoError(t, db.DeletePasswordResetsByUser(uuid.MustParse(userID)))
	require.NoError(t, db.CreatePasswordReset(uuid.MustParse(userID), relay.HashKey(rawToken), time.Now().Add(time.Hour)))
	stored, err := db.GetPasswordReset(relay.HashKey(rawToken))
	require.NoError(t, err)
	assert.Equal(t, relay.HashKey(rawToken), stored.Token)
	assert.NotEqual(t, rawToken, stored.Token)
	resp = doRequest(t, app, "POST", "/api/auth/reset-password",
		`{"token":"`+rawToken+`","new_password":"hashedtoken123"}`, nil)
	assert.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()

	// A password reset kills the live session: the old cookies are dead.
	resp = doRequest(t, app, "GET", "/api/auth/me", "", cookies)
	assert.Equal(t, 401, resp.StatusCode)
	resp.Body.Close()

	// Login with the reset password.
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"flow@example.com","password":"hashedtoken123"}`, nil)
	assert.Equal(t, 200, resp.StatusCode)
	decodeBody(t, resp)
	cookies = resp.Cookies()
	refreshCookie = nil
	for _, cookie := range cookies {
		if cookie.Name == "refresh_token" {
			refreshCookie = cookie
		}
	}
	require.NotNil(t, refreshCookie)

	// Logout ends the session: cookies are cleared and renewal is revoked.
	resp = doRequest(t, app, "POST", "/api/auth/logout", "", cookies)
	assert.Equal(t, 204, resp.StatusCode)
	cleared := strings.Join(resp.Header.Values("Set-Cookie"), ";")
	assert.Contains(t, cleared, "access_token=;")
	assert.Contains(t, cleared, "refresh_token=;")
	resp.Body.Close()

	// The not-yet-expired access token is still cryptographically valid
	// (stateless JWT), but it can no longer be renewed after logout.
	resp = doRequest(t, app, "GET", "/api/auth/me", "", []*http.Cookie{
		{Name: "access_token", Value: expiredString},
		refreshCookie,
	})
	assert.Equal(t, 401, resp.StatusCode)
	resp.Body.Close()
}

// TestStrictSingleSession covers kick-on-relogin: a second login mints a
// new sid, and the previous access + refresh tokens die immediately.
func TestStrictSingleSession(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Single User","email":"single@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	userID := decodeBody(t, resp)["user"].(map[string]interface{})["id"].(string)

	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"single@example.com","password":"secret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	first := resp.Cookies()
	resp.Body.Close()

	resp = doRequest(t, app, "GET", "/api/auth/me", "", first)
	assert.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()

	// Second login kicks the first session.
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"single@example.com","password":"secret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	second := resp.Cookies()
	resp.Body.Close()

	resp = doRequest(t, app, "GET", "/api/auth/me", "", second)
	assert.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()

	// Old access token is dead even though it is not expired yet.
	resp = doRequest(t, app, "GET", "/api/auth/me", "", first)
	assert.Equal(t, 401, resp.StatusCode)
	resp.Body.Close()

	// Old refresh token cannot renew either: expired access + old refresh.
	var firstAccessValue, firstRefreshValue string
	for _, c := range first {
		switch c.Name {
		case "access_token":
			firstAccessValue = c.Value
		case "refresh_token":
			firstRefreshValue = c.Value
		}
	}
	require.NotEmpty(t, firstAccessValue)
	require.NotEmpty(t, firstRefreshValue)
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  userID,
		"exp": time.Now().Add(-time.Hour).Unix(),
	})
	expiredString, err := expiredToken.SignedString([]byte(os.Getenv("JWT_SECRET_KEY")))
	require.NoError(t, err)
	resp = doRequest(t, app, "GET", "/api/auth/me", "", []*http.Cookie{
		{Name: "access_token", Value: expiredString},
		{Name: "refresh_token", Value: firstRefreshValue},
	})
	assert.Equal(t, 401, resp.StatusCode)
	resp.Body.Close()
}

// TestClientInvoiceFlow covers clients, invoices and dashboard.
func TestRegistrationToggle(t *testing.T) {
	t.Setenv("RATE_LIMIT_AUTH", "100")
	t.Setenv("ALLOW_REGISTRATION", "false")
	app := newTestApp()

	resp := doRequest(t, app, "GET", "/api/config", "", nil)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, false, decodeBody(t, resp)["allowRegistration"])

	resp = doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Closed User","email":"closed-reg@example.com","password":"secret123"}`, nil)
	assert.Equal(t, 403, resp.StatusCode)
	assert.Equal(t, "registration is disabled", decodeBody(t, resp)["error"].(map[string]interface{})["message"])
}

// TestDraftOnlinePaymentBlocked covers the draft/paid guards on public links:
// drafts can never get a link or be opened/paid publicly, and paid invoices
// cannot get new links (existing links stay open as receipts).
