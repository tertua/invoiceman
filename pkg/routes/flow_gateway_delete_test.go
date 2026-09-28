package routes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/platform/database"
)

// TestDeleteGatewayProject covers the guarded delete: unknown slug 404s,
// a used project 409s, an unused project deletes and disappears.
func TestDeleteGatewayProject(t *testing.T) {
	app := newTestApp()
	adminCookies := adminSession(t, app, "Delete Admin", "delete-admin@example.com")

	resp := doRequest(t, app, "DELETE", "/api/admin/gateway/projects/nope", "", adminCookies)
	require.Equal(t, 404, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, app, "POST", "/api/admin/gateway/projects",
		`{"slug":"doomed","name":"Doomed","webhook_url":"https://doomed.example/hook"}`, adminCookies)
	require.Equal(t, 201, resp.StatusCode)
	resp.Body.Close()

	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	require.NoError(t, db.CreateTransaction(&models.GatewayTransaction{
		OrderID: "DOOMED-1", ProjectSlug: "doomed", Gateway: "midtrans",
		AmountIDR: 10000, Currency: "IDR", Status: models.GatewayStatusPending,
	}))

	resp = doRequest(t, app, "DELETE", "/api/admin/gateway/projects/doomed", "", adminCookies)
	require.Equal(t, 409, resp.StatusCode)
	errBody := decodeBody(t, resp)["error"].(map[string]interface{})
	assert.Contains(t, errBody["message"], "disable it instead")

	resp = doRequest(t, app, "POST", "/api/admin/gateway/projects",
		`{"slug":"typo","name":"Typo","webhook_url":"https://typo.example/hook"}`, adminCookies)
	require.Equal(t, 201, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, app, "DELETE", "/api/admin/gateway/projects/typo", "", adminCookies)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "project deleted", decodeBody(t, resp)["message"])

	resp = doRequest(t, app, "GET", "/api/admin/gateway/projects", "", adminCookies)
	require.Equal(t, 200, resp.StatusCode)
	for _, row := range decodeBody(t, resp)["projects"].([]interface{}) {
		assert.NotEqual(t, "typo", row.(map[string]interface{})["slug"])
	}

	resp = doRequest(t, app, "DELETE", "/api/admin/gateway/projects/typo", "", nil)
	require.Equal(t, 401, resp.StatusCode)
	resp.Body.Close()
}
