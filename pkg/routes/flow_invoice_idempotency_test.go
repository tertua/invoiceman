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
