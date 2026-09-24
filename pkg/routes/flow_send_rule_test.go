package routes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSendRequiresClient covers the cross-field invariant: an invoice may be
// saved as a draft without a client, but it cannot be sent until one is set.
// This is the behavior contract behind `validateInvoice` (app/controllers).
func TestSendRequiresClient(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Send Rule User","email":"sendrule@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()

	// Creating directly as sent without a client is rejected.
	sent := newInvoice()
	sent.ClientID = ""
	resp = doRequest(t, app, "POST", "/api/invoices", sent.body(t), cookies)
	require.Equal(t, 422, resp.StatusCode)
	assert.Equal(t, "client is required to send an invoice",
		decodeBody(t, resp)["error"].(map[string]interface{})["message"])

	// The same invoice as a draft is fine.
	draft := newInvoice()
	draft.Status, draft.ClientID = "draft", ""
	draftID := createInvoiceID(t, app, cookies, draft)

	// Flipping that draft to sent is rejected...
	resp = doRequest(t, app, "PATCH", "/api/invoices/"+draftID+"/status", `{"status":"sent"}`, cookies)
	require.Equal(t, 422, resp.StatusCode)
	assert.Equal(t, "client is required to send an invoice",
		decodeBody(t, resp)["error"].(map[string]interface{})["message"])

	// ...and allowed once the invoice names a client.
	withClient := newInvoice()
	withClient.ClientID = createClient(t, app, cookies, "Send Rule Client")
	resp = doRequest(t, app, "PATCH", "/api/invoices/"+draftID, withClient.body(t), cookies)
	require.Equal(t, 200, resp.StatusCode)
	resp = doRequest(t, app, "PATCH", "/api/invoices/"+draftID+"/status", `{"status":"sent"}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "sent", decodeBody(t, resp)["invoice"].(map[string]interface{})["effective_status"])
}
