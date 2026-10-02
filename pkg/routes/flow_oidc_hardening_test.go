package routes

import (
	"net/http"
	"sync"
	"testing"

	"github.com/google/uuid"
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

// TestOIDCProvisionCompleteness proves a new SSO account lands fully wired:
// user, identity, personal org membership AND settings row all exist after one
// successful callback (F1 transactional provision, end-to-end).
func TestOIDCProvisionCompleteness(t *testing.T) {
	idp := newFakeIdP(t)
	seedOIDCEnv(t, idp)
	app := newTestApp()

	email := uniqueEmail("oidc-complete")
	_, q := oidcStart(t, app)
	idp.nextClaims = idp.claimSet("sub-"+email, email, true, q.Get("nonce"))

	resp := doRequest(t, app, "GET", oidcCallback(q.Get("state"), "test-code"), "", nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusFound, resp.StatusCode)
	require.Equal(t, "/dashboard", fetchLocation(resp))

	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	user, err := db.GetUserByEmail(email)
	require.NoError(t, err)
	assert.Empty(t, user.PasswordHash, "SSO accounts are password-less")

	identity, err := db.GetByIdentity(models.IdentityProviderOIDC, "sub-"+email)
	require.NoError(t, err)
	assert.Equal(t, user.ID, identity.UserID)

	orgID, err := db.ResolveActiveOrgID(user.ID, uuid.Nil)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, orgID)

	settings, err := db.GetSettings(orgID)
	require.NoError(t, err)
	assert.Equal(t, orgID, settings.OrgID)
}

// TestOIDCConcurrentProvision proves a concurrent create race never surfaces
// busy once an account exists: two callbacks for the same (verified) subject
// both reach a session (the loser re-fetches the winner), and exactly one user
// row is created. Tolerant of serialization.
func TestOIDCConcurrentProvision(t *testing.T) {
	idp := newFakeIdP(t)
	seedOIDCEnv(t, idp)
	app := newTestApp()

	email := uniqueEmail("oidc-race")
	sub := "sub-" + email
	states := make([]string, 2)
	for i := range states {
		_, q := oidcStart(t, app)
		states[i] = q.Get("state")
		// Pin the claims to this exact state so concurrent exchanges never
		// clobber each other's nonce.
		idp.claimsByState.Store(states[i], idp.claimSet(sub, email, true, q.Get("nonce")))
	}

	var wg sync.WaitGroup
	locations := make([]string, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp := doRequest(t, app, "GET", oidcCallback(states[i], states[i]), "", nil)
			defer resp.Body.Close()
			locations[i] = fetchLocation(resp)
		}(i)
	}
	wg.Wait()

	// Both must succeed (login or the initial register), never busy from the race.
	for _, loc := range locations {
		assert.Equal(t, "/dashboard", loc, "concurrent provision must not surface busy: %s", loc)
	}

	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	user, err := db.GetUserByEmail(email)
	require.NoError(t, err)
	var count int64
	require.NoError(t, db.UserQueries.Model(&models.User{}).Where("email = ?", email).Count(&count).Error)
	assert.EqualValues(t, 1, count, "exactly one account for the raced email")
	identity, err := db.GetByIdentity(models.IdentityProviderOIDC, sub)
	require.NoError(t, err)
	assert.Equal(t, user.ID, identity.UserID)
}
