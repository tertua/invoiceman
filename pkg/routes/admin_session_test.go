package routes

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/tertua/tupay/platform/database"
)

// adminSession registers (or reuses) an account, promotes it to admin and returns a fresh session, so a test is independent of execution order (only the first-ever account is admin by default).
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
	resp.Body.Close()
	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	require.NoError(t, db.UpdateUserRole(uuid.MustParse(adminID), "admin"))
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"`+email+`","password":"secret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	cookies := resp.Cookies()
	resp.Body.Close()
	return cookies
}
