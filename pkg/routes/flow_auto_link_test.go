package routes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOnlineInvoiceAutoPaymentLink covers the Save & send shortcut: an
// invoice created with payment_method "Online" and status sent gets its
// public payment link in the create response itself, so the detail page
// opens with the link already visible — no manual share click. Drafts and
// other methods stay link-free.
func TestOnlineInvoiceAutoPaymentLink(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Auto Link User","email":"autolink@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()
	clientID := createClient(t, app, cookies, "Auto Payer")

	spec := newInvoice()
	spec.ClientID, spec.PaymentMethod = clientID, "Online"
	invoice := createInvoice(t, app, cookies, spec)
	id := invoice["id"].(string)
	createdLink, ok := invoice["payment_link"].(map[string]interface{})
	require.True(t, ok, "Online invoice create response should carry payment_link")
	token, ok := createdLink["token"].(string)
	require.True(t, ok)
	assert.Contains(t, createdLink["url"], "/pay/")

	// Detail echoes the same link instead of minting a second one.
	resp = doRequest(t, app, "GET", "/api/invoices/"+id, "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	detail := decodeBody(t, resp)["invoice"].(map[string]interface{})
	detailLink, ok := detail["payment_link"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, token, detailLink["token"])

	// The manual endpoint stays get-or-create on top of the auto link.
	resp = doRequest(t, app, "POST", "/api/payments/online", `{"invoiceId":"`+id+`"}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, token, decodeBody(t, resp)["token"])

	// The link is publicly reachable like any manually shared one.
	resp = doRequest(t, app, "GET", "/api/public/pay/"+token, "", nil)
	require.Equal(t, 200, resp.StatusCode)
	assert.True(t, decodeBody(t, resp)["can_pay"].(bool))

	// Other methods do not get a link on create.
	spec = newInvoice()
	spec.ClientID, spec.PaymentMethod = clientID, "Cash"
	cashInvoice := createInvoice(t, app, cookies, spec)
	assert.Nil(t, cashInvoice["payment_link"])

	// Drafts never get a link, even with method Online.
	spec = newInvoice()
	spec.ClientID, spec.PaymentMethod, spec.Status = clientID, "Online", "draft"
	draftInvoice := createInvoice(t, app, cookies, spec)
	assert.Nil(t, draftInvoice["payment_link"])
}
