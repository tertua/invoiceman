package routes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInvoiceCreateIdempotency proves a double-fired Save creates one
// invoice: same key + body replays the first row, same key + other body is
// rejected, and keyless requests behave as before.
func TestInvoiceCreateIdempotency(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Inv Idem","email":"invoice-idem@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	resp.Body.Close()
	cookies := resp.Cookies()

	spec := newInvoice()
	spec.ClientID = createClient(t, app, cookies, "Inv Idem Co")
	key := "invoice-create-idem-1"

	// First execution creates the invoice.
	resp = doRequestWithHeaders(t, app, "POST", "/api/invoices", spec.body(t), cookies,
		map[string]string{"Idempotency-Key": key})
	status, first, headers := readBody(t, resp)
	require.Equal(t, 201, status)
	assert.Empty(t, headers.Get("Idempotent-Replayed"))
	firstID := first["invoice"].(map[string]interface{})["id"].(string)
	require.NotEmpty(t, firstID)

	// Retry with the same key replays the stored response, no second row.
	resp = doRequestWithHeaders(t, app, "POST", "/api/invoices", spec.body(t), cookies,
		map[string]string{"Idempotency-Key": key})
	status, second, headers := readBody(t, resp)
	require.Equal(t, 201, status)
	assert.Equal(t, "true", headers.Get("Idempotent-Replayed"))
	assert.Equal(t, firstID, second["invoice"].(map[string]interface{})["id"])

	resp = doRequest(t, app, "GET", "/api/invoices", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	assert.Len(t, decodeBody(t, resp)["invoices"], 1)

	// Same key with a different payload is rejected.
	other := newInvoice()
	other.ClientID, other.Currency = spec.ClientID, "USD"
	resp = doRequestWithHeaders(t, app, "POST", "/api/invoices", other.body(t), cookies,
		map[string]string{"Idempotency-Key": key})
	status, mismatched, _ := readBody(t, resp)
	require.Equal(t, 422, status)
	assert.Contains(t, mismatched["error"].(map[string]interface{})["message"], "idempotency key")

	// Without a key the endpoint behaves as before (second invoice).
	resp = doRequest(t, app, "POST", "/api/invoices", spec.body(t), cookies)
	require.Equal(t, 201, resp.StatusCode)
	resp.Body.Close()
}

// TestInvoiceIdempotencyIsPerOrg pins the dedup scope to the active org: the
// same key and body in another org must execute there (not replay the first
// org's response), or a switch would silently skip the write and hand back
// the other tenant's invoice.
func TestInvoiceIdempotencyIsPerOrg(t *testing.T) {
	t.Setenv("RATE_LIMIT_AUTH", "100")
	app := newTestApp()

	owner := registerUser(t, app, "idem-org-owner@example.com", "secret123")
	joiner := registerUser(t, app, "idem-org-joiner@example.com", "secret123")
	personalOrg := myOrgID(t, app, joiner)

	// A draft needs no client, so one body is valid in both orgs.
	spec := newInvoice()
	spec.Status = "draft"
	key := "invoice-create-cross-org-1"

	resp := doRequestWithHeaders(t, app, "POST", "/api/invoices", spec.body(t), joiner,
		map[string]string{"Idempotency-Key": key})
	status, first, _ := readBody(t, resp)
	require.Equal(t, 201, status)
	firstID := first["invoice"].(map[string]interface{})["id"].(string)

	// Joining the owner's org switches the session's active org.
	inviteAndAccept(t, app, owner, joiner)
	ownerOrg := myOrgID(t, app, owner)
	require.NotEqual(t, personalOrg, ownerOrg)
	assert.Equal(t, ownerOrg, myOrgID(t, app, joiner))

	// The same key + body in the new org must create a fresh invoice there.
	resp = doRequestWithHeaders(t, app, "POST", "/api/invoices", spec.body(t), joiner,
		map[string]string{"Idempotency-Key": key})
	status, second, headers := readBody(t, resp)
	require.Equal(t, 201, status)
	assert.Empty(t, headers.Get("Idempotent-Replayed"))
	assert.NotEqual(t, firstID, second["invoice"].(map[string]interface{})["id"])

	// Each org holds exactly its own invoice.
	resp = doRequest(t, app, "GET", "/api/invoices", "", owner)
	require.Equal(t, 200, resp.StatusCode)
	assert.Len(t, decodeBody(t, resp)["invoices"], 1, "owner org")
}
