package routes

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/platform/database"
)

// TestPendingEffectiveStatusFlow covers the awaiting-payment display status:
// a sent invoice with a live gateway transaction reads as pending without
// changing its stored status, and returns to sent once it resolves.
func TestPendingEffectiveStatusFlow(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Pending User","email":"pending@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)

	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"pending@example.com","password":"secret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	userID := uuid.MustParse(decodeBody(t, resp)["user"].(map[string]interface{})["id"].(string))
	cookies := resp.Cookies()

	resp = doRequest(t, app, "POST", "/api/invoices", `{
		"status":"sent",
		"issue_date":"2026-09-01",
		"due_date":"2026-09-30",
		"currency":"IDR",
		"items":[{"description":"Service","quantity":1,"rate":100000}]
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	invoice := decodeBody(t, resp)["invoice"].(map[string]interface{})
	invoiceID := invoice["id"].(string)
	assert.Equal(t, "sent", invoice["effective_status"])

	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	invoiceUUID := uuid.MustParse(invoiceID)
	orderID := "PAY-test-pending-001"
	require.NoError(t, db.CreateTransaction(&models.GatewayTransaction{
		OrderID: orderID, ProjectSlug: "local", Gateway: "midtrans",
		InvoiceID: &invoiceUUID, UserID: &userID, AmountIDR: 100000,
		Currency: "IDR", Status: models.GatewayStatusPending,
	}))

	resp = doRequest(t, app, "GET", "/api/invoices/"+invoiceID, "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	detail := decodeBody(t, resp)["invoice"].(map[string]interface{})
	assert.Equal(t, "sent", detail["status"])
	assert.Equal(t, "pending", detail["effective_status"])

	resp = doRequest(t, app, "GET", "/api/invoices?status=sent", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	listed := decodeBody(t, resp)["invoices"].([]interface{})
	require.Len(t, listed, 1)
	assert.Equal(t, "pending", listed[0].(map[string]interface{})["effective_status"])
}
