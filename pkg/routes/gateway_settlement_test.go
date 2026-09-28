package routes

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/platform/database"
)

type settlementRow struct{ count, amount float64 }

func fetchSettlement(t *testing.T, app *fiber.App, cookies []*http.Cookie) map[string]settlementRow {
	t.Helper()
	resp := doRequest(t, app, "GET", "/api/admin/gateway/settlement", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	summary, ok := decodeBody(t, resp)["summary"].([]interface{})
	require.True(t, ok, "response must carry a summary array")
	out := map[string]settlementRow{}
	for _, row := range summary {
		r := row.(map[string]interface{})
		out[r["status"].(string)] = settlementRow{
			count:  r["count"].(float64),
			amount: r["amount_idr"].(float64),
		}
	}
	return out
}

// TestGatewaySettlementSummary asserts deltas because the suite shares one in-memory database with the other flow tests.
func TestGatewaySettlementSummary(t *testing.T) {
	app := newTestApp()
	adminCookies := adminSession(t, app, "Settle Admin", "settle-admin@example.com")

	before := fetchSettlement(t, app, adminCookies)

	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	require.NoError(t, db.CreateTransaction(&models.GatewayTransaction{
		OrderID: "SETTLE-sum-001", ProjectSlug: "local", Gateway: "midtrans",
		AmountIDR: 150000, Currency: "IDR", Status: models.GatewayStatusSuccess,
	}))
	require.NoError(t, db.CreateTransaction(&models.GatewayTransaction{
		OrderID: "SETTLE-sum-002", ProjectSlug: "local", Gateway: "midtrans",
		AmountIDR: 50000, Currency: "IDR", Status: models.GatewayStatusPending,
	}))

	after := fetchSettlement(t, app, adminCookies)
	assert.Equal(t, before["success"].count+1, after["success"].count)
	assert.Equal(t, before["success"].amount+150000, after["success"].amount)
	assert.Equal(t, before["pending"].count+1, after["pending"].count)
	assert.Equal(t, before["pending"].amount+50000, after["pending"].amount)

	resp := doRequest(t, app, "GET", "/api/admin/gateway/settlement", "", nil)
	require.Equal(t, 401, resp.StatusCode)

	// A signed-in non-admin is rejected by the role guard, not the data layer.
	resp = doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Settle User","email":"settle-user@example.com","password":"secret123"}`, nil)
	require.True(t, resp.StatusCode == 201 || resp.StatusCode == 409)
	resp.Body.Close()
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"settle-user@example.com","password":"secret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	role := decodeBody(t, resp)["user"].(map[string]interface{})["role"].(string)
	require.NotEqual(t, "admin", role)
	userCookies := resp.Cookies()
	resp.Body.Close()
	resp = doRequest(t, app, "GET", "/api/admin/gateway/settlement", "", userCookies)
	require.Equal(t, 403, resp.StatusCode)
}
