package routes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The invoice payment_method field is informational (how the client should
// pay), not gateway routing: it round-trips through create/detail/update and
// only accepts the shared enum from web/src/lib/paymentMethods.js.
func TestInvoicePaymentMethodFlow(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Method User","email":"method@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()

	clientID := createClient(t, app, cookies, "Acme")

	spec := newInvoice()
	spec.ClientID = clientID
	spec.Status = "draft"
	spec.PaymentMethod = "Bank transfer"
	invoice := createInvoice(t, app, cookies, spec)
	assert.Equal(t, "Bank transfer", invoice["payment_method"])
	id := invoice["id"].(string)

	// Detail echoes the stored value.
	resp = doRequest(t, app, "GET", "/api/invoices/"+id, "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	detail := decodeBody(t, resp)["invoice"].(map[string]interface{})
	assert.Equal(t, "Bank transfer", detail["payment_method"])

	// Update rewrites it.
	spec.PaymentMethod = "Cash"
	resp = doRequest(t, app, "PATCH", "/api/invoices/"+id, spec.body(t), cookies)
	require.Equal(t, 200, resp.StatusCode, "PATCH: %s", spec.body(t))
	updated := decodeBody(t, resp)["invoice"].(map[string]interface{})
	assert.Equal(t, "Cash", updated["payment_method"])

	// Values outside the enum are rejected before anything is written.
	spec.PaymentMethod = "Dogecoin"
	resp = doRequest(t, app, "PATCH", "/api/invoices/"+id, spec.body(t), cookies)
	require.Equal(t, 400, resp.StatusCode, "PATCH: %s", spec.body(t))

	// Clearing it is allowed (omitempty drops the key entirely).
	spec.PaymentMethod = ""
	resp = doRequest(t, app, "PATCH", "/api/invoices/"+id, spec.body(t), cookies)
	require.Equal(t, 200, resp.StatusCode, "PATCH: %s", spec.body(t))
	cleared := decodeBody(t, resp)["invoice"].(map[string]interface{})
	assert.Equal(t, "", cleared["payment_method"])
}
