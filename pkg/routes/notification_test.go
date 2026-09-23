package routes

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/platform/database"
	"github.com/tertua/invoiceman/platform/outbox"
	"github.com/tertua/invoiceman/platform/relay"
)

// TestNotificationFlow covers endpoint CRUD, event fan-out on invoice
// create/status, delivery listing, retry, and worker forwarding.
func TestNotificationFlow(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Notif User","email":"notif@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()

	// Invalid URL is rejected.
	resp = doRequest(t, app, "POST", "/api/notifications/endpoints",
		`{"target_url":"ftp://n8n.example/hook"}`, cookies)
	assert.Equal(t, 400, resp.StatusCode)
	decodeBody(t, resp)

	// Create an endpoint; secret is shown once.
	resp = doRequest(t, app, "POST", "/api/notifications/endpoints",
		`{"target_url":"https://n8n.example/webhook/invoiceman"}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	endpoint := decodeBody(t, resp)["endpoint"].(map[string]interface{})
	endpointID := endpoint["id"].(string)
	require.NotEmpty(t, endpointID)
	require.NotEmpty(t, endpoint["secret"])

	// Secret is hidden on list.
	resp = doRequest(t, app, "GET", "/api/notifications/endpoints", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	listed := decodeBody(t, resp)["endpoints"].([]interface{})
	require.Len(t, listed, 1)
	assert.NotContains(t, listed[0], "secret")

	// Other users cannot see it.
	resp = doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Other User","email":"notif-other@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	otherCookies := resp.Cookies()
	resp = doRequest(t, app, "GET", "/api/notifications/endpoints", "", otherCookies)
	require.Equal(t, 200, resp.StatusCode)
	assert.Empty(t, decodeBody(t, resp)["endpoints"])

	// Create an invoice: fans out invoice.created to the endpoint.
	resp = doRequest(t, app, "POST", "/api/invoices", `{
		"status": "draft",
		"issue_date": "2026-09-01",
		"due_date": "2026-09-30",
		"currency": "IDR",
		"tax_rate": 0,
		"discount": 0,
		"items": [{"description": "Design", "quantity": 1, "rate": 160000}]
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	invoiceID := decodeBody(t, resp)["invoice"].(map[string]interface{})["id"].(string)

	// Status change fans out invoice.status_updated.
	resp = doRequest(t, app, "PATCH", "/api/invoices/"+invoiceID+"/status",
		`{"status":"sent"}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	decodeBody(t, resp)

	// Two deliveries queued (created + status_updated).
	resp = doRequest(t, app, "GET", "/api/notifications/deliveries", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	deliveries := decodeBody(t, resp)["deliveries"].([]interface{})
	require.Len(t, deliveries, 2)
	types := map[string]bool{}
	for _, d := range deliveries {
		types[d.(map[string]interface{})["event_type"].(string)] = true
	}
	assert.True(t, types["invoice.created"])
	assert.True(t, types["invoice.status_updated"])

	// Event filter works.
	resp = doRequest(t, app, "GET", "/api/notifications/deliveries?event_type=invoice.created", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	assert.Len(t, decodeBody(t, resp)["deliveries"], 1)

	// Stub the forwarder: first send succeeds, worker marks delivered.
	calls := 0
	targets := map[string]bool{}
	oldForward := outbox.ForwardNotification
	outbox.ForwardNotification = func(ctx context.Context, targetURL, eventID string, payload []byte, secret string) (*relay.ForwardResult, error) {
		calls++
		targets[targetURL] = true
		require.NotEmpty(t, secret)
		return &relay.ForwardResult{StatusCode: 200, Body: "ok"}, nil
	}
	defer func() { outbox.ForwardNotification = oldForward }()

	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	w := outbox.New()
	w.ProcessOnce(context.Background())
	assert.Equal(t, 2, calls)

	resp = doRequest(t, app, "GET", "/api/notifications/deliveries?status=delivered", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	assert.Len(t, decodeBody(t, resp)["deliveries"], 2)

	// A filtered-out endpoint receives nothing: subscribe to payment only,
	// then create another invoice.
	secondID := ""
	resp = doRequest(t, app, "POST", "/api/notifications/endpoints",
		`{"target_url":"https://n8n.example/webhook/payments-only","events":"payment.created"}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	secondID = decodeBody(t, resp)["endpoint"].(map[string]interface{})["id"].(string)
	calls = 0
	targets = map[string]bool{}
	resp = doRequest(t, app, "POST", "/api/invoices", `{
		"status": "draft",
		"issue_date": "2026-09-01",
		"due_date": "2026-09-30",
		"currency": "IDR",
		"tax_rate": 0,
		"discount": 0,
		"items": [{"description": "Extra", "quantity": 1, "rate": 50000}]
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	w.ProcessOnce(context.Background())
	assert.Equal(t, 1, calls, "only the all-events endpoint should receive invoice.created")
	assert.True(t, targets["https://n8n.example/webhook/invoiceman"])
	assert.False(t, targets["https://n8n.example/webhook/payments-only"])

	// payment.created fans out to both endpoints.
	paymentKey := uuid.NewString()
	resp = doRequestWithHeaders(t, app, "POST", "/api/payments", `{
		"invoiceId": "`+invoiceID+`",
		"amount": 100000,
		"method": "transfer",
		"paid_on": "2026-09-10"
	}`, cookies, map[string]string{"Idempotency-Key": paymentKey})
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	calls = 0
	targets = map[string]bool{}
	w.ProcessOnce(context.Background())
	assert.Equal(t, 2, calls, "both endpoints should receive payment.created")
	assert.True(t, targets["https://n8n.example/webhook/invoiceman"])
	assert.True(t, targets["https://n8n.example/webhook/payments-only"])

	// Test fires only at the selected endpoint, even when it is not
	// subscribed to the test event and other endpoints exist.
	calls = 0
	targets = map[string]bool{}
	resp = doRequest(t, app, "POST", "/api/notifications/endpoints/"+secondID+"/test", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	decodeBody(t, resp)
	w.ProcessOnce(context.Background())
	assert.Equal(t, 1, calls, "test must target only the selected endpoint")
	assert.True(t, targets["https://n8n.example/webhook/payments-only"])
	assert.False(t, targets["https://n8n.example/webhook/invoiceman"])

	// Rotate secret returns a new one-time secret.
	resp = doRequest(t, app, "POST", "/api/notifications/endpoints/"+endpointID+"/rotate-secret", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	rotated := decodeBody(t, resp)["endpoint"].(map[string]interface{})
	require.NotEmpty(t, rotated["secret"])
	assert.NotEqual(t, endpoint["secret"], rotated["secret"])

	// Retry requeues the row to pending; the worker owns the HTTP attempt.
	resp = doRequest(t, app, "GET", "/api/notifications/deliveries?event_type=invoice.created&status=delivered", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	createdDeliveries := decodeBody(t, resp)["deliveries"].([]interface{})
	require.NotEmpty(t, createdDeliveries)
	firstID := createdDeliveries[0].(map[string]interface{})["id"].(string)
	resp = doRequest(t, app, "POST", "/api/notifications/deliveries/"+firstID+"/retry", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	retried := decodeBody(t, resp)["delivery"].(map[string]interface{})
	assert.Equal(t, "pending", retried["status"])
	calls = 0
	w.ProcessOnce(context.Background())
	assert.Equal(t, 1, calls, "requeued delivery must be picked up by the worker")

	// Delete second endpoint; its future events stop.
	resp = doRequest(t, app, "DELETE", "/api/notifications/endpoints/"+secondID, "", cookies)
	assert.Equal(t, 204, resp.StatusCode)
	resp.Body.Close()

	// Unknown delivery id 404s.
	resp = doRequest(t, app, "POST", "/api/notifications/deliveries/"+uuid.NewString()+"/retry", "", cookies)
	assert.Equal(t, 404, resp.StatusCode)
	decodeBody(t, resp)

	// Worker dead-letters after exhausted attempts: mark directly and
	// verify the row never re-enters the due queue.
	meResp := doRequest(t, app, "GET", "/api/auth/me", "", cookies)
	meID := uuid.MustParse(decodeBody(t, meResp)["user"].(map[string]interface{})["id"].(string))
	deadRow := &models.NotificationDelivery{
		UserID:     meID,
		EndpointID: uuid.MustParse(endpointID),
		EventID:    "evt_dead_" + uuid.NewString()[:8],
		EventType:  models.NotifEventTest,
		TargetURL:  "https://n8n.example/webhook/invoiceman",
		Payload:    `{"type":"notification.test"}`,
	}
	require.NoError(t, db.EnqueueDelivery(deadRow))
	require.NoError(t, db.MarkNotificationFailed(deadRow.ID, 10, nil, 0, "down", time.Now()))
	dead, err := db.CountDeliveriesByUser(meID, "", "dead")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, dead, int64(1))
	due, err := db.DueNotifications(time.Now().Add(48*time.Hour), 100)
	require.NoError(t, err)
	for _, r := range due {
		assert.NotEqual(t, deadRow.ID, r.ID, "dead rows must never re-enter the queue")
	}
}
