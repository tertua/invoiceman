package routes

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLogoutWithoutSessionClearsCookies pins the 401 cookie cleanup: a logout
// without a live session still clears the stale access/refresh/CSRF cookies
// instead of leaving the browser holding tokens the server rejects.
func TestLogoutWithoutSessionClearsCookies(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/logout", "", nil)
	require.Equal(t, 401, resp.StatusCode)
	cleared := strings.Join(resp.Header.Values("Set-Cookie"), ";")
	assert.Contains(t, cleared, "access_token=;")
	assert.Contains(t, cleared, "refresh_token=;")
	assert.Contains(t, cleared, "csrf_token=;")
	resp.Body.Close()
}

// TestMeWithoutSessionClearsCookies pins the same cleanup on the me probe.
func TestMeWithoutSessionClearsCookies(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "GET", "/api/auth/me", "", nil)
	require.Equal(t, 401, resp.StatusCode)
	cleared := strings.Join(resp.Header.Values("Set-Cookie"), ";")
	assert.Contains(t, cleared, "access_token=;")
	assert.Contains(t, cleared, "refresh_token=;")
	assert.Contains(t, cleared, "csrf_token=;")
	resp.Body.Close()
}
