package routes

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/tupay/platform/outbox"
	"github.com/tertua/tupay/platform/relay"
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

// TestUpdateToSentFansOutStatusUpdated covers the editor "save & send" path:
// flipping draft to sent through PATCH /invoices/{id} must enqueue
// invoice.status_updated like the status toggle does, while an update that
// keeps the status must not enqueue anything.
func TestUpdateToSentFansOutStatusUpdated(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Status Fanout User","email":"statusfanout@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()

	resp = doRequest(t, app, "POST", "/api/notifications/endpoints",
		`{"target_url":"https://n8n.example/webhook/tupay"}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)

	clientID := createClient(t, app, cookies, "Status Fanout Client")
	draft := newInvoice()
	draft.Status, draft.ClientID = "draft", clientID
	draftID := createInvoiceID(t, app, cookies, draft)

	statusUpdated := func() []interface{} {
		resp := doRequest(t, app, "GET", "/api/notifications/deliveries?event_type=invoice.status_updated", "", cookies)
		require.Equal(t, 200, resp.StatusCode)
		return decodeBody(t, resp)["deliveries"].([]interface{})
	}
	require.Empty(t, statusUpdated())

	// Same-status update enqueues nothing.
	same := newInvoice()
	same.Status, same.ClientID = "draft", clientID
	resp = doRequest(t, app, "PATCH", "/api/invoices/"+draftID, same.body(t), cookies)
	require.Equal(t, 200, resp.StatusCode)
	decodeBody(t, resp)
	assert.Empty(t, statusUpdated())

	// Draft -> sent through the update endpoint fans out status_updated.
	send := newInvoice()
	send.Status, send.ClientID = "sent", clientID
	resp = doRequest(t, app, "PATCH", "/api/invoices/"+draftID, send.body(t), cookies)
	require.Equal(t, 200, resp.StatusCode)
	decodeBody(t, resp)
	require.Len(t, statusUpdated(), 1)

	// Drain the queued rows: route tests share one in-memory database, so
	// pending deliveries would leak into tests that count worker forwards.
	oldForward := outbox.ForwardNotification
	outbox.ForwardNotification = func(ctx context.Context, targetURL, eventID string, payload []byte, secret string) (*relay.ForwardResult, error) {
		return &relay.ForwardResult{StatusCode: 200, Body: "ok"}, nil
	}
	outbox.New().ProcessOnce(context.Background())
	outbox.ForwardNotification = oldForward
}
