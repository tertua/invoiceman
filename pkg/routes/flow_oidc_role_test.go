package routes

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/tupay/platform/database"
)

// imposeRole sets a user's platform role directly (test-only, mirrors the admin
// console path without needing an admin session).
func imposeRole(t *testing.T, email, role string) {
	t.Helper()
	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	user, err := db.GetUserByEmail(email)
	require.NoError(t, err)
	require.NoError(t, db.UpdateUserRole(user.ID, role))
}

// roleLogin drives one full SSO login for sub/email and returns the session
// cookies plus the final callback redirect.
func roleLogin(t *testing.T, app *fiber.App, idp *fakeIdP, sub, email string) []*http.Cookie {
	t.Helper()
	_, q := oidcStart(t, app)
	idp.nextClaims = idp.claimSet(sub, email, true, q.Get("nonce"))
	resp := doRequest(t, app, "GET", oidcCallback(q.Get("state"), "test-code"), "", nil)
	require.Equal(t, http.StatusFound, resp.StatusCode)
	require.Equal(t, "/dashboard", fetchLocation(resp))
	return resp.Cookies()
}

// meRole reads the caller's platform role from /api/auth/me.
func meRole(t *testing.T, app *fiber.App, cookies []*http.Cookie) string {
	t.Helper()
	me := doRequest(t, app, "GET", "/api/auth/me", "", cookies)
	require.Equal(t, http.StatusOK, me.StatusCode)
	body := decodeBody(t, me)
	return body["user"].(map[string]interface{})["role"].(string)
}

// TestOIDCFlowRoleSync covers mapping the configured claim to the platform role
// on every successful SSO login.
func TestOIDCFlowRoleSync(t *testing.T) {
	idp := newFakeIdP(t)
	seedOIDCEnv(t, idp)
	app := newTestApp()
	t.Setenv("OIDC_ROLE_CLAIM", "realm_access.roles")

	t.Run("admin claim grants admin", func(t *testing.T) {
		idp.tokenOverride = mutatingTokenOverride(func(c map[string]any) {
			c["realm_access"] = map[string]any{"roles": []any{"admin"}}
		})
		t.Cleanup(func() { idp.tokenOverride = nil })

		cookies := roleLogin(t, app, idp, "sub-grant", "grant@example.com")
		assert.Equal(t, "admin", meRole(t, app, cookies))
	})

	t.Run("case-insensitive admin match", func(t *testing.T) {
		idp.tokenOverride = mutatingTokenOverride(func(c map[string]any) {
			c["realm_access"] = map[string]any{"roles": []any{"ADMIN"}}
		})
		t.Cleanup(func() { idp.tokenOverride = nil })

		cookies := roleLogin(t, app, idp, "sub-case", "case@example.com")
		assert.Equal(t, "admin", meRole(t, app, cookies))
	})

	t.Run("non-admin value stays user", func(t *testing.T) {
		idp.tokenOverride = mutatingTokenOverride(func(c map[string]any) {
			c["realm_access"] = map[string]any{"roles": []any{"viewer"}}
		})
		t.Cleanup(func() { idp.tokenOverride = nil })

		cookies := roleLogin(t, app, idp, "sub-viewer", "viewer@example.com")
		assert.Equal(t, "user", meRole(t, app, cookies))
	})

	t.Run("admin grant then revoke demotes on next login", func(t *testing.T) {
		idp.tokenOverride = mutatingTokenOverride(func(c map[string]any) {
			c["realm_access"] = map[string]any{"roles": []any{"admin"}}
		})
		cookies := roleLogin(t, app, idp, "sub-revoke", "revoke@example.com")
		assert.Equal(t, "admin", meRole(t, app, cookies))

		idp.tokenOverride = mutatingTokenOverride(func(c map[string]any) {
			c["realm_access"] = map[string]any{"roles": []any{"user"}}
		})
		t.Cleanup(func() { idp.tokenOverride = nil })
		cookies = roleLogin(t, app, idp, "sub-revoke", "revoke@example.com")
		assert.Equal(t, "user", meRole(t, app, cookies))
	})

	t.Run("absent claim demotes fail-closed", func(t *testing.T) {
		idp.tokenOverride = mutatingTokenOverride(func(c map[string]any) {
			c["realm_access"] = map[string]any{"roles": []any{"admin"}}
		})
		cookies := roleLogin(t, app, idp, "sub-absent", "absent@example.com")
		assert.Equal(t, "admin", meRole(t, app, cookies))

		// Second login: no realm_access at all -> fail-closed demote, still 302.
		idp.tokenOverride = mutatingTokenOverride(func(c map[string]any) { delete(c, "realm_access") })
		t.Cleanup(func() { idp.tokenOverride = nil })
		cookies = roleLogin(t, app, idp, "sub-absent", "absent@example.com")
		assert.Equal(t, "user", meRole(t, app, cookies))
	})

	t.Run("existing email account is synced on link login", func(t *testing.T) {
		// Local password account, then SSO with the same verified email links it.
		registerAndLogin(t, app, "linked@example.com", "secret123")

		idp.tokenOverride = mutatingTokenOverride(func(c map[string]any) {
			c["realm_access"] = map[string]any{"roles": []any{"admin"}}
		})
		t.Cleanup(func() { idp.tokenOverride = nil })

		cookies := roleLogin(t, app, idp, "sub-linked", "linked@example.com")
		assert.Equal(t, "admin", meRole(t, app, cookies))
	})
}

// TestOIDCFlowRoleSyncDisabled verifies an empty claim path is a no-op: a
// console-promoted admin is not demoted by an SSO login (plan D7).
func TestOIDCFlowRoleSyncDisabled(t *testing.T) {
	idp := newFakeIdP(t)
	seedOIDCEnv(t, idp)
	app := newTestApp()

	// No OIDC_ROLE_CLAIM: feature off. Promote the account via the local path.
	registerAndLogin(t, app, "keepadmin@example.com", "secret123")
	imposeRole(t, "keepadmin@example.com", "admin")

	// SSO login for the same email with no role claim must keep admin.
	idp.tokenOverride = mutatingTokenOverride(func(c map[string]any) { delete(c, "realm_access") })
	t.Cleanup(func() { idp.tokenOverride = nil })

	_, q := oidcStart(t, app)
	idp.nextClaims = idp.claimSet("sub-keep", "keepadmin@example.com", true, q.Get("nonce"))
	resp := doRequest(t, app, "GET", oidcCallback(q.Get("state"), "test-code"), "", nil)
	require.Equal(t, http.StatusFound, resp.StatusCode)
	require.Equal(t, "/dashboard", fetchLocation(resp))

	assert.Equal(t, "admin", meRole(t, app, resp.Cookies()))
}
