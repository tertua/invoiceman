package routes

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/platform/database"
)

// TestDashboardCacheInvalidation proves cached aggregates stay fresh:
// a dashboard read, then a write, then a dashboard read must reflect the
// write immediately (no TTL-stale window), on either cache backend.
func TestDashboardCacheInvalidation(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Cache User","email":"cache@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	userID := decodeBody(t, resp)["user"].(map[string]interface{})["id"].(string)
	cookies := resp.Cookies()

	dashboardStats := func() map[string]interface{} {
		resp := doRequest(t, app, "GET", "/api/dashboard?currency=USD", "", cookies)
		require.Equal(t, 200, resp.StatusCode)
		return decodeBody(t, resp)["stats"].(map[string]interface{})
	}

	before := dashboardStats()
	countBefore := before["invoiceCount"]

	// Create a client + invoice (both invalidate aggregates).
	resp = doRequest(t, app, "POST", "/api/clients",
		`{"name":"Cache Co","email":"c@cache.test"}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	clientID := decodeBody(t, resp)["client"].(map[string]interface{})["id"].(string)

	resp = doRequest(t, app, "POST", "/api/invoices", `{
		"client_id": "`+clientID+`",
		"status": "draft",
		"issue_date": "2026-09-01",
		"due_date": "2026-09-30",
		"currency": "USD",
		"tax_rate": 0,
		"discount": 0,
		"items": [{"description": "Work", "quantity": 1, "rate": 100}]
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	invoiceID := decodeBody(t, resp)["invoice"].(map[string]interface{})["id"].(string)

	after := dashboardStats()
	assert.Equal(t, countBefore.(float64)+1, after["invoiceCount"])

	// The suite shares one database: remove our rows so user/list counts
	// in other flow tests stay exact (FK order: items, invoice, client, user).
	q, err := database.OpenDBConnection()
	require.NoError(t, err)
	h := q.InvoiceQueries.DB
	require.NoError(t, h.Where("invoice_id = ?", invoiceID).Delete(&models.InvoiceItem{}).Error)
	require.NoError(t, h.Where("id = ?", invoiceID).Delete(&models.Invoice{}).Error)
	require.NoError(t, h.Where("id = ?", uuid.MustParse(clientID)).Delete(&models.Client{}).Error)
	require.NoError(t, h.Where("id = ?", uuid.MustParse(userID)).Delete(&models.User{}).Error)
}
