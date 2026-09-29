package routes

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/platform/database"
	"github.com/tertua/tupay/platform/events"
)

// nextStatusEvent waits on the shared broker for the owner's invoice.status_updated frame, skipping unrelated frames such as invoice.created.
func nextStatusEvent(t *testing.T, ch <-chan events.Event) events.Event {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case ev := <-ch:
			if ev.Type == models.NotifEventInvoiceStatusUpdated {
				return ev
			}
		case <-deadline:
			t.Fatal("expected an invoice.status_updated event")
			return events.Event{}
		}
	}
}

// auditOrgOf loads the newest audit row for action/entity through the GORM builder (no raw SQL) and returns the org stamped on it.
func auditOrgOf(t *testing.T, action, entityID string) string {
	t.Helper()
	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	var rows []models.AuditLog
	require.NoError(t, db.AuditQueries.Model(&models.AuditLog{}).
		Where("action = ? AND entity_id = ?", action, entityID).
		Order("created_at DESC").Limit(1).Find(&rows).Error)
	require.Len(t, rows, 1, "audit row for "+action)
	return rows[0].OrgID.String()
}

// TestInvoiceApprovalFlow covers draft → pending → sent, the reject path back to draft, the invalid transitions, the org-scoped audit trail and the owner's status event.
func TestInvoiceApprovalFlow(t *testing.T) {
	t.Setenv("RATE_LIMIT_AUTH", "100")
	app := newTestApp()

	cookies := registerUser(t, app, "approval-owner@example.com", "secret123")
	resp := doRequest(t, app, "GET", "/api/auth/me", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	ownerUserID := decodeBody(t, resp)["user"].(map[string]interface{})["id"].(string)
	orgID := myOrgID(t, app, cookies)

	clientID := createClient(t, app, cookies, "Approval Client")
	draft := newInvoice()
	draft.Status, draft.ClientID = "draft", clientID
	invoiceID := createInvoiceID(t, app, cookies, draft)

	ch, unsubscribe := events.Default.Subscribe(ownerUserID)
	defer unsubscribe()

	// draft → submit → pending, audited and announced to the owner.
	resp = doRequest(t, app, "POST", "/api/invoices/"+invoiceID+"/submit", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "pending", decodeBody(t, resp)["invoice"].(map[string]interface{})["status"])
	require.Equal(t, orgID, auditOrgOf(t, "invoice.submitted", invoiceID))
	assert.Equal(t, models.NotifEventInvoiceStatusUpdated, nextStatusEvent(t, ch).Type)

	// pending → approve → sent, again with its own audit row and event.
	resp = doRequest(t, app, "POST", "/api/invoices/"+invoiceID+"/approve", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "sent", decodeBody(t, resp)["invoice"].(map[string]interface{})["status"])
	require.Equal(t, orgID, auditOrgOf(t, "invoice.approved", invoiceID))
	nextStatusEvent(t, ch)

	// Approving an invoice that is no longer pending is an invalid transition.
	resp = doRequest(t, app, "POST", "/api/invoices/"+invoiceID+"/approve", "", cookies)
	require.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "invalid status transition", envelopeMessage(t, resp))

	// pending → reject sends the invoice back to draft.
	second := newInvoice()
	second.Status, second.ClientID = "draft", clientID
	secondID := createInvoiceID(t, app, cookies, second)
	resp = doRequest(t, app, "POST", "/api/invoices/"+secondID+"/submit", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()
	resp = doRequest(t, app, "POST", "/api/invoices/"+secondID+"/reject", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "draft", decodeBody(t, resp)["invoice"].(map[string]interface{})["status"])
	require.Equal(t, orgID, auditOrgOf(t, "invoice.rejected", secondID))
	nextStatusEvent(t, ch)

	// A draft cannot be approved either.
	resp = doRequest(t, app, "POST", "/api/invoices/"+secondID+"/approve", "", cookies)
	require.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "invalid status transition", envelopeMessage(t, resp))
}
