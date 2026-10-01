package routes

import (
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/platform/database"
)

// TestAdminCreateOrgFlow walks POST /admin/orgs: an admin bootstraps a tenant for
// another user, and the target claims it by activating the new org. Validation,
// non-admin callers, a missing owner and a blocked owner all answer with the
// stable codes, and the successful create leaves an org.create audit entry.
func TestAdminCreateOrgFlow(t *testing.T) {
	t.Setenv("RATE_LIMIT_AUTH", "100")
	app := newTestApp()

	admin := adminSession(t, app, "Org Admin", "org-admin@example.com")
	adminID := userID(t, app, admin)
	target := registerUser(t, app, "org-target@example.com", "secret123")
	targetID := userID(t, app, target)
	db, err := database.OpenDBConnection()
	require.NoError(t, err)

	// Happy path: the admin names the target as owner of a fresh org.
	resp := doRequest(t, app, "POST", "/api/admin/orgs", `{"name":"Acme","owner_user_id":"`+targetID+`"}`, admin)
	require.Equal(t, 201, resp.StatusCode)
	body := decodeBody(t, resp)
	assert.Equal(t, "owner", body["role"])
	assert.Equal(t, targetID, body["owner_user_id"])
	org := body["org"].(map[string]interface{})
	assert.Equal(t, "Acme", org["name"])
	orgID := org["id"].(string)

	// Persisted: the target owns the new org and the admin is NOT a member.
	role, err := db.GetRole(uuid.MustParse(orgID), uuid.MustParse(targetID))
	require.NoError(t, err)
	assert.Equal(t, models.RoleOwner, role)
	stored, err := db.GetOrg(uuid.MustParse(orgID))
	require.NoError(t, err)
	assert.Equal(t, "Acme", stored.Name)
	_, err = db.GetRole(uuid.MustParse(orgID), uuid.MustParse(adminID))
	require.ErrorIs(t, err, sql.ErrNoRows)

	// The target activates the bootstrap org and lands in it as owner.
	resp = doRequest(t, app, "POST", "/api/orgs/"+orgID+"/activate", "", target)
	require.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()
	assert.Equal(t, orgID, myOrgID(t, app, target))
	assert.Equal(t, "owner", meOrg(t, app, target)["role"])

	// The create is on the audit trail, attributed to the admin.
	logs, err := db.ListAuditLogs(50, 0)
	require.NoError(t, err)
	found := false
	for _, entry := range logs {
		if entry.Action == "org.create" && entry.Entity == "org" && entry.EntityID == orgID {
			assert.Equal(t, uuid.MustParse(adminID), entry.UserID)
			found = true
			break
		}
	}
	assert.True(t, found, "org.create audit entry not found")

	// Validation: a blank name (after trim) and a malformed owner id are 400s.
	resp = doRequest(t, app, "POST", "/api/admin/orgs", `{"name":"   ","owner_user_id":"`+targetID+`"}`, admin)
	require.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "name is required", envelopeMessage(t, resp))
	resp = doRequest(t, app, "POST", "/api/admin/orgs", `{"name":"Beta","owner_user_id":"not-a-uuid"}`, admin)
	require.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "validation failed", envelopeMessage(t, resp))

	// A non-admin caller is refused by RequireRoles.
	resp = doRequest(t, app, "POST", "/api/admin/orgs", `{"name":"Beta","owner_user_id":"`+targetID+`"}`, target)
	require.Equal(t, 403, resp.StatusCode)

	// An unknown owner id is a 404.
	resp = doRequest(t, app, "POST", "/api/admin/orgs", `{"name":"Gamma","owner_user_id":"`+uuid.NewString()+`"}`, admin)
	require.Equal(t, 404, resp.StatusCode)
	assert.Equal(t, "user not found", envelopeMessage(t, resp))

	// A blocked owner is refused and no org is created for them.
	blocked := registerUser(t, app, "org-blocked@example.com", "secret123")
	blockedID := userID(t, app, blocked)
	require.NoError(t, db.UserQueries.Model(&models.User{}).
		Where("id = ?", uuid.MustParse(blockedID)).Update("user_status", 0).Error)
	resp = doRequest(t, app, "POST", "/api/admin/orgs", `{"name":"Delta","owner_user_id":"`+blockedID+`"}`, admin)
	require.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "owner user is blocked", envelopeMessage(t, resp))
}
