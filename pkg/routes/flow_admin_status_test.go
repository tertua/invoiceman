package routes

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/tupay/platform/database"
)

// TestAdminStatusFlow walks PATCH /admin/users/:id/status: an admin blocks a
// plain user, which also revokes their live session and refuses their next
// login, then unblocks them so login works again. Self-change, bad payloads,
// malformed ids, non-admin callers and unknown targets all answer with the
// stable codes, and each successful change leaves a user.status.update audit
// entry attributed to the admin.
func TestAdminStatusFlow(t *testing.T) {
	t.Setenv("RATE_LIMIT_AUTH", "100")
	app := newTestApp()

	admin := adminSession(t, app, "Status Admin", "status-admin@example.com")
	adminID := userID(t, app, admin)
	target := registerUser(t, app, "status-target@example.com", "secret123")
	targetID := userID(t, app, target)
	db, err := database.OpenDBConnection()
	require.NoError(t, err)

	// Block the target: status flips to 0 and the response mirrors it.
	resp := doRequest(t, app, "PATCH", "/api/admin/users/"+targetID+"/status", `{"status":0}`, admin)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, float64(0), decodeBody(t, resp)["user"].(map[string]interface{})["status"])
	stored, err := db.GetUserByID(uuid.MustParse(targetID))
	require.NoError(t, err)
	assert.Equal(t, 0, stored.UserStatus)

	// The block revokes the target's live session: their old cookies no longer pass.
	resp = doRequest(t, app, "GET", "/api/auth/me", "", target)
	require.Equal(t, 401, resp.StatusCode)
	resp.Body.Close()

	// A blocked account cannot log back in.
	resp = doRequest(t, app, "POST", "/api/auth/login", `{"email":"status-target@example.com","password":"secret123"}`, nil)
	require.Equal(t, 403, resp.StatusCode)
	assert.Equal(t, "account is blocked", envelopeMessage(t, resp))

	// The change is on the audit trail, attributed to the admin.
	logs, err := db.ListAuditLogs(50, 0)
	require.NoError(t, err)
	found := false
	for _, entry := range logs {
		if entry.Action == "user.status.update" && entry.Entity == "user" && entry.EntityID == targetID {
			assert.Equal(t, uuid.MustParse(adminID), entry.UserID)
			found = true
			break
		}
	}
	assert.True(t, found, "user.status.update audit entry not found")

	// Unblock the target: status flips back to 1 and login works again.
	resp = doRequest(t, app, "PATCH", "/api/admin/users/"+targetID+"/status", `{"status":1}`, admin)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, float64(1), decodeBody(t, resp)["user"].(map[string]interface{})["status"])
	stored, err = db.GetUserByID(uuid.MustParse(targetID))
	require.NoError(t, err)
	assert.Equal(t, 1, stored.UserStatus)
	resp = doRequest(t, app, "POST", "/api/auth/login", `{"email":"status-target@example.com","password":"secret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	target = resp.Cookies()
	resp.Body.Close()

	// An admin may not change their own status.
	resp = doRequest(t, app, "PATCH", "/api/admin/users/"+adminID+"/status", `{"status":0}`, admin)
	require.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "you cannot change your own status", envelopeMessage(t, resp))

	// Only the terminal statuses are accepted; pending (2), an out-of-range
	// value and a missing field are all 400s, and the account keeps its status.
	resp = doRequest(t, app, "PATCH", "/api/admin/users/"+targetID+"/status", `{"status":2}`, admin)
	require.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "validation failed", envelopeMessage(t, resp))
	resp = doRequest(t, app, "PATCH", "/api/admin/users/"+targetID+"/status", `{"status":5}`, admin)
	require.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "validation failed", envelopeMessage(t, resp))
	resp = doRequest(t, app, "PATCH", "/api/admin/users/"+targetID+"/status", `{}`, admin)
	require.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "validation failed", envelopeMessage(t, resp))
	stored, err = db.GetUserByID(uuid.MustParse(targetID))
	require.NoError(t, err)
	assert.Equal(t, 1, stored.UserStatus)

	// A malformed id is refused before any lookup.
	resp = doRequest(t, app, "PATCH", "/api/admin/users/not-a-uuid/status", `{"status":0}`, admin)
	require.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "invalid user id", envelopeMessage(t, resp))

	// A non-admin caller is refused by RequireRoles.
	resp = doRequest(t, app, "PATCH", "/api/admin/users/"+targetID+"/status", `{"status":0}`, target)
	require.Equal(t, 403, resp.StatusCode)
	resp.Body.Close()

	// An unknown target is a 404.
	resp = doRequest(t, app, "PATCH", "/api/admin/users/"+uuid.NewString()+"/status", `{"status":0}`, admin)
	require.Equal(t, 404, resp.StatusCode)
	assert.Equal(t, "user not found", envelopeMessage(t, resp))
}
