package routes

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/platform/database"
)

// adminSession registers (or reuses) an account, promotes it to admin, and
// returns a fresh session. Promotion is direct so the test is independent
// of execution order (only the first-ever account is admin by default).
func adminSession(t *testing.T, app *fiber.App, name, email string) []*http.Cookie {
	t.Helper()
	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"`+name+`","email":"`+email+`","password":"secret123"}`, nil)
	require.True(t, resp.StatusCode == 201 || resp.StatusCode == 409)
	resp.Body.Close()
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"`+email+`","password":"secret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	adminID := decodeBody(t, resp)["user"].(map[string]interface{})["id"].(string)
	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	require.NoError(t, db.UpdateUserRole(uuid.MustParse(adminID), "admin"))
	// Refresh session so the role change takes effect.
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"`+email+`","password":"secret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	cookies := resp.Cookies()
	resp.Body.Close()
	return cookies
}

// TestMigrateDownFlow covers the admin schema-rollback endpoint: validation,
// refusal, roles, and the round trip. The shared test database is restored
// to the current version at the end so later flow tests keep their columns.
func TestMigrateDownFlow(t *testing.T) {
	defer func() {
		require.NoError(t, database.Migrate())
	}()

	app := newTestApp()
	adminCookies := adminSession(t, app, "Mig Admin", "migadmin@example.com")

	// Confirmation is mandatory.
	resp := doRequest(t, app, "POST", "/api/admin/migrate/down",
		`{"target_version":1,"confirm":false}`, adminCookies)
	assert.Equal(t, 400, resp.StatusCode)
	decodeBody(t, resp)

	// Non-admins are rejected by roles, not version logic — before rollback, since v1 lacks new columns.
	resp = doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Mig User","email":"miguser@example.com","password":"secret123"}`, nil)
	require.True(t, resp.StatusCode == 201 || resp.StatusCode == 409)
	resp.Body.Close()
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"miguser@example.com","password":"secret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	role := decodeBody(t, resp)["user"].(map[string]interface{})["role"].(string)
	require.NotEqual(t, "admin", role)
	userCookies := resp.Cookies()
	resp.Body.Close()
	resp = doRequest(t, app, "POST", "/api/admin/migrate/down",
		`{"target_version":1,"confirm":true}`, userCookies)
	assert.Equal(t, 403, resp.StatusCode)
	decodeBody(t, resp)

	// Roll back to v1.
	resp = doRequest(t, app, "POST", "/api/admin/migrate/down",
		`{"target_version":1,"confirm":true}`, adminCookies)
	assert.Equal(t, 200, resp.StatusCode)
	body := decodeBody(t, resp)
	assert.Equal(t, float64(1), body["version"])
	stamp, err := database.CurrentSchemaVersion()
	require.NoError(t, err)
	assert.Equal(t, 1, stamp)

	// Same target again is refused.
	resp = doRequest(t, app, "POST", "/api/admin/migrate/down",
		`{"target_version":1,"confirm":true}`, adminCookies)
	assert.Equal(t, 400, resp.StatusCode)
	decodeBody(t, resp)

	// The suite shares one database: remove our accounts so user counts
	// in other flow tests stay exact.
	q, err := database.OpenDBConnection()
	require.NoError(t, err)
	h := q.InvoiceQueries.DB
	require.NoError(t, h.Where("email IN ?", []string{"migadmin@example.com", "miguser@example.com"}).Delete(&models.User{}).Error)
}
