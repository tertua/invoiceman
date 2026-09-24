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

	spec := newInvoice()
	spec.ClientID = createClient(t, app, cookies, "Acme")
	spec.Items = []invoiceLine{{Description: "Service", Quantity: 1, Rate: "100000"}}
	invoice := createInvoice(t, app, cookies, spec)
	invoiceID := invoice["id"].(string)
	clientID := spec.ClientID
	assert.Equal(t, "sent", invoice["effective_status"])

	// The public link is minted before any money moves; it must stay usable
	// while the payment is in flight.
	resp = doRequest(t, app, "POST", "/api/payments/online", `{"invoiceId":"`+invoiceID+`"}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	token := decodeBody(t, resp)["token"].(string)

	// A partial manual payment lands before the Snap intent opens.
	resp = doRequest(t, app, "POST", "/api/payments", `{
		"invoiceId":"`+invoiceID+`",
		"amount":10000,
		"method":"Cash",
		"paid_on":"2026-09-05"
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	paymentID := decodeBody(t, resp)["payment"].(map[string]interface{})["id"].(string)

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

	// Pending still owes: it counts toward outstanding on both dashboard
	// and reports, and shows its own donut slice.
	resp = doRequest(t, app, "GET", "/api/dashboard?currency=IDR", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "90000", decodeBody(t, resp)["stats"].(map[string]interface{})["outstanding"])
	resp = doRequest(t, app, "GET", "/api/reports?currency=IDR", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	body := decodeBody(t, resp)
	assert.Equal(t, "90000", body["totals"].(map[string]interface{})["outstanding"])
	slices := map[string]string{}
	for _, s := range body["statusBreakdown"].([]interface{}) {
		m := s.(map[string]interface{})
		slices[m["key"].(string)] = m["value"].(string)
	}
	// The donut breaks down invoiced totals per status (a paid slice would
	// read zero otherwise), so pending shows the full 100000 here.
	assert.Equal(t, "100000", slices["pending"])

	// Client billing agrees too: a pending overlay is a receivable on both
	// the client list and the client detail.
	resp = doRequest(t, app, "GET", "/api/clients", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	clients := decodeBody(t, resp)["clients"].([]interface{})
	require.Len(t, clients, 1)
	assert.Equal(t, "100000", clients[0].(map[string]interface{})["total_billed"])
	assert.Equal(t, "90000", clients[0].(map[string]interface{})["outstanding"])
	resp = doRequest(t, app, "GET", "/api/clients/"+clientID, "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	clientStats := decodeBody(t, resp)["stats"].(map[string]interface{})
	assert.Equal(t, "100000", clientStats["totalBilled"])
	assert.Equal(t, "90000", clientStats["outstanding"])

	// Legacy flipped-back draft with money in flight: both outstanding
	// cards still count it (the Snap intent stays live at the provider).
	require.NoError(t, db.UpdateInvoiceStatus(userID, invoiceUUID, "draft"))
	resp = doRequest(t, app, "GET", "/api/dashboard?currency=IDR", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "90000", decodeBody(t, resp)["stats"].(map[string]interface{})["outstanding"])
	resp = doRequest(t, app, "GET", "/api/reports?currency=IDR", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "90000", decodeBody(t, resp)["totals"].(map[string]interface{})["outstanding"])
	// Client billing tracks the same overlay, not the flipped-back stored status.
	resp = doRequest(t, app, "GET", "/api/clients", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	draftClients := decodeBody(t, resp)["clients"].([]interface{})
	require.Len(t, draftClients, 1)
	assert.Equal(t, "90000", draftClients[0].(map[string]interface{})["outstanding"])
	resp = doRequest(t, app, "GET", "/api/clients/"+clientID, "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "90000", decodeBody(t, resp)["stats"].(map[string]interface{})["outstanding"])
	// The public pay page tracks the overlay too: a stored draft with money
	// in flight is payable, not "still a draft".
	resp = doRequest(t, app, "GET", "/api/public/pay/"+token, "", nil)
	require.Equal(t, 200, resp.StatusCode)
	pub := decodeBody(t, resp)["invoice"].(map[string]interface{})
	assert.Equal(t, "pending", pub["effective_status"])
	resp = doRequest(t, app, "POST", "/api/public/pay/"+token+"/transaction", "", nil)
	assert.Equal(t, 501, resp.StatusCode) // gateway unconfigured, not draft-blocked
	resp.Body.Close()
	require.NoError(t, db.UpdateInvoiceStatus(userID, invoiceUUID, "sent"))

	// While money is in flight the invoice is locked: editing totals or
	// hard-deleting would orphan the provider payment, so both are 422.
	edit := newInvoice()
	edit.ClientID = clientID
	edit.Items = []invoiceLine{{Description: "Service", Quantity: 1, Rate: "100000"}}
	editBody := edit.body(t)
	resp = doRequest(t, app, "PATCH", "/api/invoices/"+invoiceID, editBody, cookies)
	require.Equal(t, 422, resp.StatusCode)
	assert.Equal(t, "invoice has a pending payment",
		decodeBody(t, resp)["error"].(map[string]interface{})["message"])
	resp = doRequest(t, app, "DELETE", "/api/invoices/"+invoiceID, "", cookies)
	require.Equal(t, 422, resp.StatusCode)
	assert.Equal(t, "invoice has a pending payment",
		decodeBody(t, resp)["error"].(map[string]interface{})["message"])

	// Flipping sent/draft mid-flight is locked too: it cannot cancel the
	// Snap intent at the provider, so the payment could still settle.
	resp = doRequest(t, app, "PATCH", "/api/invoices/"+invoiceID+"/status", `{"status":"draft"}`, cookies)
	require.Equal(t, 422, resp.StatusCode)
	assert.Equal(t, "invoice has a pending payment",
		decodeBody(t, resp)["error"].(map[string]interface{})["message"])

	// Money movement is locked too: a manual record, a void, or a second
	// link would settle against a different balance than the live intent.
	// (The existing public link stays usable — no new link is made here.)
	resp = doRequest(t, app, "POST", "/api/payments", `{
		"invoiceId":"`+invoiceID+`",
		"amount":10000,
		"method":"Cash",
		"paid_on":"2026-09-06"
	}`, cookies)
	require.Equal(t, 422, resp.StatusCode)
	assert.Equal(t, "invoice has a pending payment",
		decodeBody(t, resp)["error"].(map[string]interface{})["message"])
	resp = doRequest(t, app, "DELETE", "/api/payments/"+paymentID+"?reason=duplicate", "", cookies)
	require.Equal(t, 422, resp.StatusCode)
	assert.Equal(t, "invoice has a pending payment",
		decodeBody(t, resp)["error"].(map[string]interface{})["message"])
	resp = doRequest(t, app, "POST", "/api/payments/online", `{"invoiceId":"`+invoiceID+`"}`, cookies)
	require.Equal(t, 422, resp.StatusCode)
	assert.Equal(t, "invoice has a pending payment",
		decodeBody(t, resp)["error"].(map[string]interface{})["message"])

	// Once the transaction resolves the lock lifts and the invoice is
	// editable/deletable again.
	txn, err := db.GetTransaction(orderID)
	require.NoError(t, err)
	txn.Status = models.GatewayStatusExpired
	require.NoError(t, db.SaveTransaction(&txn))
	resp = doRequest(t, app, "PATCH", "/api/invoices/"+invoiceID, editBody, cookies)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "sent", decodeBody(t, resp)["invoice"].(map[string]interface{})["effective_status"])
	resp = doRequest(t, app, "PATCH", "/api/invoices/"+invoiceID+"/status", `{"status":"draft"}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	resp = doRequest(t, app, "DELETE", "/api/invoices/"+invoiceID, "", cookies)
	require.Equal(t, 204, resp.StatusCode)
}
