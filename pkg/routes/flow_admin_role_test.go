package routes

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/tupay/platform/database"
)

// TestAdminRoleFlow walks PATCH /admin/users/:id/role: an admin promotes a plain
// user, then demotes them again, while self-change, non-admin callers and an
// unknown role are all refused with the stable answers.
func TestAdminRoleFlow(t *testing.T) {
	t.Setenv("RATE_LIMIT_AUTH", "100")
	app := newTestApp()

	admin := adminSession(t, app, "Role Admin", "role-admin@example.com")
	target := registerUser(t, app, "role-target@example.com", "secret123")
	targetID := userID(t, app, target)

	// Promote user → admin. Each success rotates the caller's CSRF token, so merge it back before the next write.
	resp := doRequest(t, app, "PATCH", "/api/admin/users/"+targetID+"/role", `{"role":"admin"}`, admin)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "admin", decodeBody(t, resp)["user"].(map[string]interface{})["role"])
	assert.Equal(t, "admin", userRole(t, targetID))
	admin = mergeCookies(admin, resp.Cookies())

	// Demote admin → user (both directions must work; the target's old admin role is not protected).
	resp = doRequest(t, app, "PATCH", "/api/admin/users/"+targetID+"/role", `{"role":"user"}`, admin)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "user", userRole(t, targetID))
	admin = mergeCookies(admin, resp.Cookies())

	// An admin may not change their own role.
	selfID := userID(t, app, admin)
	resp = doRequest(t, app, "PATCH", "/api/admin/users/"+selfID+"/role", `{"role":"user"}`, admin)
	require.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "you cannot change your own role", envelopeMessage(t, resp))

	// A non-admin caller is refused.
	resp = doRequest(t, app, "PATCH", "/api/admin/users/"+targetID+"/role", `{"role":"admin"}`, target)
	require.Equal(t, 403, resp.StatusCode)

	// An unknown role fails validation; the account keeps its role.
	resp = doRequest(t, app, "PATCH", "/api/admin/users/"+targetID+"/role", `{"role":"moderator"}`, admin)
	require.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "user", userRole(t, targetID))
}

// userID reads the session user's id from /auth/me.
func userID(t *testing.T, app *fiber.App, cookies []*http.Cookie) string {
	t.Helper()
	resp := doRequest(t, app, "GET", "/api/auth/me", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	return decodeBody(t, resp)["user"].(map[string]interface{})["id"].(string)
}

// userRole reads one account's platform role straight from the database.
func userRole(t *testing.T, id string) string {
	t.Helper()
	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	user, err := db.GetUserByID(uuid.MustParse(id))
	require.NoError(t, err)
	return user.UserRole
}
