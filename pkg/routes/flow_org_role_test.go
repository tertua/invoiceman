package routes

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// envelopeMessage reads the stable English message out of a utils.Fail response.
func envelopeMessage(t *testing.T, resp *http.Response) string {
	t.Helper()
	return decodeBody(t, resp)["error"].(map[string]interface{})["message"].(string)
}

// TestOrgRoleFlow walks the owner/staff matrix on the owner's real organization: staff may draft and submit, only the owner may approve, delete a non-draft or change settings.
func TestOrgRoleFlow(t *testing.T) {
	t.Setenv("RATE_LIMIT_AUTH", "100")
	app := newTestApp()

	owner := registerUser(t, app, "role-owner@example.com", "secret123")
	staff := registerUser(t, app, "role-staff@example.com", "secret123")
	inviteAndAccept(t, app, owner, staff)

	// Staff drafts an invoice that still needs the owner to send it.
	spec := newInvoice()
	spec.Status = "draft"
	spec.ClientID = createClient(t, app, owner, "Role Client")
	invoiceID := createInvoiceID(t, app, staff, spec)

	resp := doRequest(t, app, "POST", "/api/invoices/"+invoiceID+"/submit", "", staff)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "pending", decodeBody(t, resp)["invoice"].(map[string]interface{})["status"])

	// Staff may not approve, send a pending invoice, delete a non-draft, or edit the org's settings.
	resp = doRequest(t, app, "POST", "/api/invoices/"+invoiceID+"/approve", "", staff)
	require.Equal(t, 403, resp.StatusCode)
	assert.Equal(t, "org.ownerRequired", envelopeMessage(t, resp))
	resp = doRequest(t, app, "PATCH", "/api/invoices/"+invoiceID+"/status", `{"status":"sent"}`, staff)
	require.Equal(t, 403, resp.StatusCode)
	assert.Equal(t, "org.ownerRequired", envelopeMessage(t, resp))
	resp = doRequest(t, app, "DELETE", "/api/invoices/"+invoiceID, "", staff)
	require.Equal(t, 403, resp.StatusCode)
	assert.Equal(t, "org.ownerRequired", envelopeMessage(t, resp))
	resp = doRequest(t, app, "PATCH", "/api/settings",
		`{"company_name":"Staff Co","currency":"IDR","tax_rate":11,"invoice_prefix":"INV-"}`, staff)
	require.Equal(t, 403, resp.StatusCode)
	assert.Equal(t, "org.ownerRequired", envelopeMessage(t, resp))

	// Listing members needs no owner role: the route carries no RequireOrgRole.
	resp = doRequest(t, app, "GET", "/api/orgs/members", "", staff)
	require.Equal(t, 200, resp.StatusCode)
	assert.Len(t, decodeBody(t, resp)["members"], 2)

	// The owner holds exactly what staff lacks.
	resp = doRequest(t, app, "POST", "/api/invoices/"+invoiceID+"/approve", "", owner)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "sent", decodeBody(t, resp)["invoice"].(map[string]interface{})["status"])
	resp = doRequest(t, app, "PATCH", "/api/settings",
		`{"company_name":"Owner Co","currency":"IDR","tax_rate":11,"invoice_prefix":"INV-"}`, owner)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "Owner Co", decodeBody(t, resp)["settings"].(map[string]interface{})["company_name"])
}

// TestOrgRoleInvoiceWriteGuards closes the audit gaps the route-level guards cannot see: staff may not mint a non-draft, rewrite a non-draft, promote a draft through the full update, or reach the owner-only logo/online-send routes.
func TestOrgRoleInvoiceWriteGuards(t *testing.T) {
	t.Setenv("RATE_LIMIT_AUTH", "100")
	app := newTestApp()

	owner := registerUser(t, app, "guard-owner@example.com", "secret123")
	staff := registerUser(t, app, "guard-staff@example.com", "secret123")
	inviteAndAccept(t, app, owner, staff)
	clientID := createClient(t, app, owner, "Guard Client")

	sent := newInvoice()
	sent.ClientID = clientID

	// Creating straight into sent is owner-only (permission matrix: staff start at draft).
	resp := doRequest(t, app, "POST", "/api/invoices", sent.body(t), staff)
	require.Equal(t, 403, resp.StatusCode)
	assert.Equal(t, "org.ownerRequired", envelopeMessage(t, resp))

	// Staff may still draft with a client attached.
	draft := newInvoice()
	draft.Status = "draft"
	draft.ClientID = clientID
	staffDraft := createInvoiceID(t, app, staff, draft)

	// The full update cannot promote that draft to sent either.
	resp = doRequest(t, app, "PATCH", "/api/invoices/"+staffDraft, sent.body(t), staff)
	require.Equal(t, 403, resp.StatusCode)
	assert.Equal(t, "org.ownerRequired", envelopeMessage(t, resp))

	// Submit moves it to pending; from there any rewrite by staff is refused, even with the status left alone.
	resp = doRequest(t, app, "POST", "/api/invoices/"+staffDraft+"/submit", "", staff)
	require.Equal(t, 200, resp.StatusCode)
	resp = doRequest(t, app, "PATCH", "/api/invoices/"+staffDraft, draft.body(t), staff)
	require.Equal(t, 403, resp.StatusCode)
	assert.Equal(t, "org.ownerRequired", envelopeMessage(t, resp))

	// The owner keeps the same move: draft through the full update lands as sent.
	resp = doRequest(t, app, "PATCH", "/api/invoices/"+createInvoiceID(t, app, owner, draft), sent.body(t), owner)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "sent", decodeBody(t, resp)["invoice"].(map[string]interface{})["status"])

	// Owner-only routes stay owner-only: logo upload and the online pay-link email.
	resp = multipartFile(t, app, "POST", "/api/settings/logo", "logo", "logo.png", "image/png", tinyPNG, staff)
	require.Equal(t, 403, resp.StatusCode)
	assert.Equal(t, "org.ownerRequired", envelopeMessage(t, resp))
	resp = doRequest(t, app, "POST", "/api/payments/online/send",
		`{"invoiceId":"`+staffDraft+`","email":"client@example.com"}`, staff)
	require.Equal(t, 403, resp.StatusCode)
	assert.Equal(t, "org.ownerRequired", envelopeMessage(t, resp))
}
