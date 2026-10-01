package routes

import (
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/app/queries"
	"github.com/tertua/tupay/platform/database"
)

// meOrg returns the /auth/me "org" block: the resolved org plus every membership of the caller.
func meOrg(t *testing.T, app *fiber.App, cookies []*http.Cookie) map[string]interface{} {
	t.Helper()
	resp := doRequest(t, app, "GET", "/api/auth/me", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	return decodeBody(t, resp)["org"].(map[string]interface{})
}

// TestOrgInviteFlow covers minting a token, redeeming it at register or accept, the active-org switch, revocation and the stale-token answers.
func TestOrgInviteFlow(t *testing.T) {
	t.Setenv("RATE_LIMIT_AUTH", "100")
	app := newTestApp()

	owner := registerUser(t, app, "invite-owner@example.com", "secret123")
	ownerOrg := myOrgID(t, app, owner)

	// Create invite → token.
	joinToken := inviteToken(t, app, owner)

	// Registering with the token lands the new account in the inviting org.
	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Joiner","email":"invite-joiner@example.com","password":"secret123","invite_token":"`+joinToken+`"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	resp.Body.Close()
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"invite-joiner@example.com","password":"secret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	joiner := resp.Cookies()

	// No personal tenant was provisioned: one membership, and it is the inviting org.
	joined := meOrg(t, app, joiner)
	assert.Equal(t, ownerOrg, joined["id"])
	assert.Equal(t, "staff", joined["role"])
	require.Len(t, joined["memberships"].([]interface{}), 1)

	// Redeeming the token again fails and leaves that single membership alone.
	resp = doRequest(t, app, "POST", "/api/orgs/invites/accept", `{"token":"`+joinToken+`"}`, joiner)
	require.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "invite already accepted", envelopeMessage(t, resp))
	require.Len(t, meOrg(t, app, joiner)["memberships"].([]interface{}), 1)

	// An existing user redeems a second invite: memberships grow to two.
	existing := registerUser(t, app, "invite-existing@example.com", "secret123")
	personalOrg := myOrgID(t, app, existing)
	resp = doRequest(t, app, "POST", "/api/orgs/invites/accept",
		`{"token":"`+inviteToken(t, app, owner)+`"}`, existing)
	require.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()
	assert.GreaterOrEqual(t, len(meOrg(t, app, existing)["memberships"].([]interface{})), 2)
	// The accept switched the session's active org: /auth/me must report the joined org's staff role, not the oldest membership's owner role.
	assert.Equal(t, ownerOrg, meOrg(t, app, existing)["id"])
	assert.Equal(t, "staff", meOrg(t, app, existing)["role"])

	// Activate moves the session between the two memberships and back.
	resp = doRequest(t, app, "POST", "/api/orgs/"+personalOrg+"/activate", "", existing)
	require.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()
	assert.Equal(t, personalOrg, myOrgID(t, app, existing))
	assert.Equal(t, "owner", meOrg(t, app, existing)["role"])
	resp = doRequest(t, app, "POST", "/api/orgs/"+ownerOrg+"/activate", "", existing)
	require.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()
	assert.Equal(t, ownerOrg, myOrgID(t, app, existing))

	// A revoked invite is refused with the stable message.
	resp = doRequest(t, app, "POST", "/api/orgs/invites", `{}`, owner)
	require.Equal(t, 201, resp.StatusCode)
	invite := decodeBody(t, resp)["invite"].(map[string]interface{})
	resp = doRequest(t, app, "DELETE", "/api/orgs/invites/"+invite["id"].(string), "", owner)
	require.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()
	resp = doRequest(t, app, "POST", "/api/orgs/invites/accept",
		`{"token":"`+invite["token"].(string)+`"}`,
		registerUser(t, app, "invite-revoked@example.com", "secret123"))
	require.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "invite revoked", envelopeMessage(t, resp))

	// An unknown invite id cannot be revoked.
	resp = doRequest(t, app, "DELETE", "/api/orgs/invites/"+uuid.NewString(), "", owner)
	require.Equal(t, 404, resp.StatusCode)
	assert.Equal(t, "invite not found", envelopeMessage(t, resp))

	// An expired invite is refused with the stable "expired" message.
	expiredToken := inviteToken(t, app, owner)
	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	require.NoError(t, db.OrgInviteQueries.Model(&models.OrgInvite{}).
		Where("token = ?", expiredToken).Update("expires_at", time.Now().Add(-time.Hour)).Error)
	resp = doRequest(t, app, "POST", "/api/orgs/invites/accept",
		`{"token":"`+expiredToken+`"}`,
		registerUser(t, app, "invite-expired@example.com", "secret123"))
	require.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "invite expired", envelopeMessage(t, resp))
}

// TestInviteOnlyRegistrationFlow proves that while ALLOW_REGISTRATION is disabled a valid org invite is the only way a new account may join, and that each bad-token shape reports its own stable message.
func TestInviteOnlyRegistrationFlow(t *testing.T) {
	t.Setenv("RATE_LIMIT_AUTH", "100")
	app := newTestApp()

	// The owner is provisioned while registration is still open; the gate is applied for the invite-only assertions below.
	owner := registerUser(t, app, "inviteonly-owner@example.com", "secret123")
	ownerOrg := myOrgID(t, app, owner)
	t.Setenv("ALLOW_REGISTRATION", "false")

	// A token-less register is now refused (the install has users).
	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Nope","email":"inviteonly-nope@example.com","password":"secret123"}`, nil)
	require.Equal(t, 403, resp.StatusCode)
	assert.Equal(t, "registration is disabled", envelopeMessage(t, resp))

	// 2. A fresh valid invite registers the newcomer straight into the inviting org: one membership, staff role, no personal tenant.
	validToken := inviteToken(t, app, owner)
	resp = doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Invited","email":"inviteonly-joiner@example.com","password":"secret123","invite_token":"`+validToken+`"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	resp.Body.Close()
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"inviteonly-joiner@example.com","password":"secret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	joiner := resp.Cookies()
	resp.Body.Close()
	joined := meOrg(t, app, joiner)
	assert.Equal(t, ownerOrg, joined["id"])
	assert.Equal(t, "staff", joined["role"])
	require.Len(t, joined["memberships"].([]interface{}), 1)

	// 3. A garbage token passes the coarse gate and fails in resolution with the invite message.
	resp = doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Bad","email":"inviteonly-bad@example.com","password":"secret123","invite_token":"not-a-real-token"}`, nil)
	require.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "invite expired", envelopeMessage(t, resp))

	// 4. An expired invite is refused with the stable "expired" message.
	expiredToken := inviteToken(t, app, owner)
	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	require.NoError(t, db.OrgInviteQueries.Model(&models.OrgInvite{}).
		Where("token = ?", expiredToken).Update("expires_at", time.Now().Add(-time.Hour)).Error)
	resp = doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Expired","email":"inviteonly-expired@example.com","password":"secret123","invite_token":"`+expiredToken+`"}`, nil)
	require.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "invite expired", envelopeMessage(t, resp))

	// 5. A revoked invite is refused with the stable "revoked" message.
	resp = doRequest(t, app, "POST", "/api/orgs/invites", `{}`, owner)
	require.Equal(t, 201, resp.StatusCode)
	revoked := decodeBody(t, resp)["invite"].(map[string]interface{})
	resp = doRequest(t, app, "DELETE", "/api/orgs/invites/"+revoked["id"].(string), "", owner)
	require.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()
	resp = doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Revoked","email":"inviteonly-revoked@example.com","password":"secret123","invite_token":"`+revoked["token"].(string)+`"}`, nil)
	require.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "invite revoked", envelopeMessage(t, resp))

	// 6. A token already redeemed once cannot register a second account.
	resp = doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Reuse","email":"inviteonly-reuse@example.com","password":"secret123","invite_token":"`+validToken+`"}`, nil)
	require.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "invite already accepted", envelopeMessage(t, resp))
}

// TestInviteClaimSingleUse pins the claim primitive behind the accept flow: the second claim of one token is refused at the query layer too, so the fix does not depend on the controller's pre-read check.
func TestInviteClaimSingleUse(t *testing.T) {
	t.Setenv("RATE_LIMIT_AUTH", "100")
	app := newTestApp()

	owner := registerUser(t, app, "claim-owner@example.com", "secret123")
	token := inviteToken(t, app, owner)

	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	var invite models.OrgInvite
	require.NoError(t, db.OrgInviteQueries.Where("token = ?", token).First(&invite).Error)

	require.NoError(t, db.MarkAccepted(invite.ID, uuid.New()))
	require.ErrorIs(t, db.MarkAccepted(invite.ID, uuid.New()), queries.ErrInviteAccepted)
}

// TestOrgInviteRoleFlow proves the invite honours the requested role: owner invites join as owner, a role-less invite still joins as staff, and an unknown role is refused.
func TestOrgInviteRoleFlow(t *testing.T) {
	t.Setenv("RATE_LIMIT_AUTH", "100")
	app := newTestApp()

	owner := registerUser(t, app, "inviterole-owner@example.com", "secret123")
	ownerOrg := myOrgID(t, app, owner)

	// role=owner in the payload → the accepter lands as owner of the inviting org.
	resp := doRequest(t, app, "POST", "/api/orgs/invites", `{"role":"owner"}`, owner)
	require.Equal(t, 201, resp.StatusCode)
	ownerInvite := decodeBody(t, resp)["invite"].(map[string]interface{})
	assert.Equal(t, "owner", ownerInvite["role"])
	joinOwner := registerUser(t, app, "inviterole-joiner@example.com", "secret123")
	resp = doRequest(t, app, "POST", "/api/orgs/invites/accept", `{"token":"`+ownerInvite["token"].(string)+`"}`, joinOwner)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "owner", decodeBody(t, resp)["role"])
	assert.Equal(t, ownerOrg, myOrgID(t, app, joinOwner))
	assert.Equal(t, "owner", meOrg(t, app, joinOwner)["role"])

	// No role in the payload → staff, and the minted invite records staff.
	resp = doRequest(t, app, "POST", "/api/orgs/invites", `{}`, owner)
	require.Equal(t, 201, resp.StatusCode)
	staffInvite := decodeBody(t, resp)["invite"].(map[string]interface{})
	assert.Equal(t, "staff", staffInvite["role"])
	joinStaff := registerUser(t, app, "inviterole-staff@example.com", "secret123")
	resp = doRequest(t, app, "POST", "/api/orgs/invites/accept", `{"token":"`+staffInvite["token"].(string)+`"}`, joinStaff)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "staff", decodeBody(t, resp)["role"])

	// An unknown role is rejected before a token is minted.
	resp = doRequest(t, app, "POST", "/api/orgs/invites", `{"role":"moderator"}`, owner)
	require.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "invalid invite role", envelopeMessage(t, resp))
}
