package routes

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientInvoiceFlow(t *testing.T) {
	app := newTestApp()

	// Register and keep the session cookies.
	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Invoice User","email":"invoice@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()

	// Create a client.
	resp = doRequest(t, app, "POST", "/api/clients",
		`{"name":"Acme","email":"billing@acme.test","company":"Acme Inc"}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	clientID := decodeBody(t, resp)["client"].(map[string]interface{})["id"].(string)
	require.NotEmpty(t, clientID)

	// Client detail starts empty.
	resp = doRequest(t, app, "GET", "/api/clients/"+clientID, "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	detail := decodeBody(t, resp)
	assert.Equal(t, float64(0), detail["stats"].(map[string]interface{})["count"])

	// Create an invoice: subtotal 250, discount 10, tax 10% -> total 264.
	resp = doRequest(t, app, "POST", "/api/invoices", `{
		"client_id": "`+clientID+`",
		"status": "draft",
		"issue_date": "2026-09-01",
		"due_date": "2026-09-30",
		"currency": "USD",
		"tax_rate": 10,
		"discount": 10,
		"items": [
			{"description": "Design", "quantity": 2, "rate": 100},
			{"description": "Hosting", "quantity": 1, "rate": 50}
		]
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	invoice := decodeBody(t, resp)["invoice"].(map[string]interface{})
	assert.Equal(t, float64(264), invoice["total"])
	assert.Equal(t, "draft", invoice["effective_status"])
	assert.True(t, strings.HasPrefix(invoice["invoice_number"].(string), "INV-"))
	invoiceID := invoice["id"].(string)

	// List shows the invoice.
	resp = doRequest(t, app, "GET", "/api/invoices", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	listed := decodeBody(t, resp)["invoices"].([]interface{})
	require.Len(t, listed, 1)

	// Drafts are not receivables: dashboard, client list and client detail
	// must all report zero outstanding while the invoice is still a draft.
	resp = doRequest(t, app, "GET", "/api/dashboard", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, float64(0), decodeBody(t, resp)["stats"].(map[string]interface{})["outstanding"])

	resp = doRequest(t, app, "GET", "/api/clients", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	draftClients := decodeBody(t, resp)["clients"].([]interface{})
	require.Len(t, draftClients, 1)
	assert.Equal(t, float64(0), draftClients[0].(map[string]interface{})["outstanding"])

	resp = doRequest(t, app, "GET", "/api/clients/"+clientID, "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, float64(0), decodeBody(t, resp)["stats"].(map[string]interface{})["outstanding"])

	// Mark as sent.
	resp = doRequest(t, app, "PATCH", "/api/invoices/"+invoiceID+`/status`,
		`{"status":"sent"}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	updated := decodeBody(t, resp)["invoice"].(map[string]interface{})
	assert.Equal(t, "sent", updated["status"])

	// Dashboard reflects the new data.
	resp = doRequest(t, app, "GET", "/api/dashboard", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	dashboard := decodeBody(t, resp)
	stats := dashboard["stats"].(map[string]interface{})
	assert.Equal(t, float64(1), stats["invoiceCount"])
	assert.Equal(t, float64(1), stats["clientCount"])
	assert.Equal(t, float64(264), stats["outstanding"])

	// Sent invoices count as receivables in the client list and detail.
	resp = doRequest(t, app, "GET", "/api/clients", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	sentClients := decodeBody(t, resp)["clients"].([]interface{})
	require.Len(t, sentClients, 1)
	assert.Equal(t, float64(264), sentClients[0].(map[string]interface{})["outstanding"])

	resp = doRequest(t, app, "GET", "/api/clients/"+clientID, "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	sentDetail := decodeBody(t, resp)["stats"].(map[string]interface{})
	assert.Equal(t, float64(264), sentDetail["outstanding"])
	assert.Equal(t, float64(264), sentDetail["totalBilled"])
	assert.Len(t, dashboard["revenueSeries"].([]interface{}), 6)
	// Revenue points expose a stable YYYY-MM key for frontend localization.
	for _, item := range dashboard["revenueSeries"].([]interface{}) {
		point := item.(map[string]interface{})
		assert.Regexp(t, `^\d{4}-\d{2}$`, point["key"].(string))
		assert.NotEmpty(t, point["label"])
	}
	assert.Len(t, dashboard["recentInvoices"].([]interface{}), 1)

	// Currency filtering keeps dashboard totals from mixing currencies.
	resp = doRequest(t, app, "GET", "/api/dashboard?currency=IDR", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	idrDashboard := decodeBody(t, resp)["stats"].(map[string]interface{})
	assert.Equal(t, float64(0), idrDashboard["invoiceCount"])

	resp = doRequest(t, app, "GET", "/api/dashboard?currency=USD", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	usdDashboard := decodeBody(t, resp)["stats"].(map[string]interface{})
	assert.Equal(t, float64(1), usdDashboard["invoiceCount"])

	// Cleanup.
	resp = doRequest(t, app, "DELETE", "/api/invoices/"+invoiceID, "", cookies)
	assert.Equal(t, 204, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, app, "DELETE", "/api/clients/"+clientID, "", cookies)
	assert.Equal(t, 204, resp.StatusCode)
	resp.Body.Close()
}

// TestItemFlow covers catalog item CRUD and per-user ownership.
