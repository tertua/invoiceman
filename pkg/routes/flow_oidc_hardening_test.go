package routes

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/platform/database"
	"github.com/tertua/tupay/platform/relay"
)

// TestOIDCAuditReasonPending pins the shared status gate: an SSO login against a
// pending account is redirected to the stable denied code AND the audit row
// records reason "pending" (not the old blanket "blocked").
func TestOIDCAuditReasonPending(t *testing.T) {
	idp := newFakeIdP(t)
	seedOIDCEnv(t, idp)
	app := newTestApp()

	// A pending account: verification is required and the install already has
	// users (count > 0), so the registration lands pending instead of active.
	// verifyFlagOn seeds a bootstrap user so the count==0 first-install
	// exemption never turns this account active when the test runs alone.
	verifyFlagOn(t, app)
	email := uniqueEmail("oidc-pending")
	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"OIDC Pending","email":"`+email+`","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	assert.Equal(t, "verification_required", decodeBody(t, resp)["status"])
	// New registrations go back to active for the rest of the flow; the account
	// under test is already stamped pending.
	t.Setenv("REQUIRE_EMAIL_VERIFICATION", "false")

	// SSO callback with the same (verified) email links the identity, then the
	// shared gate rejects the pending account.
	_, q := oidcStart(t, app)
	idp.nextClaims = idp.claimSet("sub-pending", email, true, q.Get("nonce"))
	resp = doRequest(t, app, "GET", oidcCallback(q.Get("state"), "test-code"), "", nil)
	require.Equal(t, http.StatusFound, resp.StatusCode)
	assert.Equal(t, "/login?oidc_error=denied", fetchLocation(resp))
	resp.Body.Close()

	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	var rows []models.AuditLog
	require.NoError(t, db.AuditQueries.Model(&models.AuditLog{}).
		Where("action = ? AND entity_id = ?", "auth.login.failed", relay.HashKey(email)).
		Order("created_at DESC").Limit(1).Find(&rows).Error)
	require.Len(t, rows, 1, "expected an auth.login.failed audit row for the pending OIDC account")
	assert.Contains(t, rows[0].Meta, `"reason":"pending"`)
	assert.NotContains(t, rows[0].Meta, `"reason":"blocked"`)
}
