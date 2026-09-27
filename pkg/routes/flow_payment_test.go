package routes

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentFlow(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Payment User","email":"payment@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()
	spec := newInvoice()
	spec.ClientID, spec.Currency = createClient(t, app, cookies, "Payer"), "USD"
	spec.Items = []invoiceLine{{Description: "Service", Quantity: 1, Rate: "100"}}
	invoiceID := createInvoiceID(t, app, cookies, spec)

	resp = doRequest(t, app, "POST", "/api/payments", `{
		"invoiceId":"`+invoiceID+`",
		"amount":40,
		"method":"Bank transfer",
		"paid_on":"2026-09-21",
		"notes":"Deposit"
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	payment := decodeBody(t, resp)["payment"].(map[string]interface{})
	paymentID := payment["id"].(string)

	resp = doRequest(t, app, "GET", "/api/payments", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	body := decodeBody(t, resp)
	assert.Len(t, body["payments"].([]interface{}), 1)
	assert.Equal(t, "40", body["totals"].(map[string]interface{})["total"])

	resp = doRequest(t, app, "POST", "/api/payments", `{
		"invoiceId":"`+invoiceID+`",
		"amount":61,
		"method":"Cash",
		"paid_on":"2026-09-21"
	}`, cookies)
	assert.Equal(t, 400, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, app, "GET", "/api/invoices/"+invoiceID, "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	detail := decodeBody(t, resp)["invoice"].(map[string]interface{})
	assert.Equal(t, "40", detail["paid_amount"])
	assert.Equal(t, "sent", detail["effective_status"])

	// Void requires a reason.
	resp = doRequest(t, app, "DELETE", "/api/payments/"+paymentID, "", cookies)
	assert.Equal(t, 400, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, app, "DELETE", "/api/payments/"+paymentID+"?reason=duplicate", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	voided := decodeBody(t, resp)["payment"].(map[string]interface{})
	assert.Equal(t, true, voided["voided"])
	resp.Body.Close()

	// A voided payment disappears from list, totals and invoice balances.
	resp = doRequest(t, app, "GET", "/api/payments", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	body = decodeBody(t, resp)
	assert.Empty(t, body["payments"])
	assert.Equal(t, "0", body["totals"].(map[string]interface{})["total"])

	resp = doRequest(t, app, "GET", "/api/invoices/"+invoiceID, "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	detail = decodeBody(t, resp)["invoice"].(map[string]interface{})
	assert.Equal(t, "0", detail["paid_amount"])
	assert.Empty(t, detail["payments"])
	assert.Equal(t, "sent", detail["effective_status"])

	// Double void is rejected.
	resp = doRequest(t, app, "DELETE", "/api/payments/"+paymentID+"?reason=duplicate", "", cookies)
	assert.Equal(t, 409, resp.StatusCode)
	resp.Body.Close()
}

// TestReportsFlow covers report totals and breakdowns across features.
func TestPublicPaymentFlow(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Public User","email":"public@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()
	spec := newInvoice()
	spec.ClientID = createClient(t, app, cookies, "Public Payer")
	spec.Items = []invoiceLine{{Description: "Public service", Quantity: 1, Rate: "100000"}}
	invoiceID := createInvoiceID(t, app, cookies, spec)

	resp = doRequest(t, app, "POST", "/api/payments/online", `{"invoiceId":"`+invoiceID+`"}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	link := decodeBody(t, resp)
	token := link["token"].(string)
	assert.Contains(t, link["url"], "/pay/")

	resp = doRequest(t, app, "GET", "/api/public/pay/"+token, "", nil)
	require.Equal(t, 200, resp.StatusCode)
	publicData := decodeBody(t, resp)
	assert.True(t, publicData["can_pay"].(bool))
	pubInvoice := publicData["invoice"].(map[string]interface{})
	assert.Equal(t, invoiceID, pubInvoice["id"].(string))
	assert.Nil(t, publicData["gateway"].(map[string]interface{})["name"])
	assert.NotContains(t, pubInvoice, "client_id")
	assert.NotContains(t, pubInvoice, "client_email")
	assert.NotContains(t, pubInvoice, "payment_link")

	resp = doRequest(t, app, "POST", "/api/public/pay/"+token+"/transaction", "", nil)
	require.Equal(t, 501, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, app, "GET", "/api/public/pay/"+token+"/status", "", nil)
	require.Equal(t, 200, resp.StatusCode)
	status := decodeBody(t, resp)
	assert.NotEqual(t, "paid", status["status"])
	assert.NotEqual(t, "0", status["balance"])

	resp = doRequest(t, app, "POST", "/api/public/pay/"+token+"/transaction", "", nil)
	assert.Equal(t, 501, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, app, "POST", "/api/payments/online/send", `{"invoiceId":"`+invoiceID+`","email":"client@example.com"}`, cookies)
	assert.Equal(t, 501, resp.StatusCode)
	resp.Body.Close()
}

// TestPaidInvoiceLocked covers the paid immutability contract: content
// edits, status downgrades and deletes are rejected once payments cover
// the total, while the public payment link stays in the DB.
func TestPaidInvoiceLocked(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Paid Lock User","email":"paidlock@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()
	today := time.Now().Format("2006-01-02")
	clientID := createClient(t, app, cookies, "Locked Payer")
	spec := newInvoice()
	spec.ClientID, spec.Issue, spec.Due = clientID, today, today
	spec.Items = []invoiceLine{{Description: "Locked service", Quantity: 1, Rate: "50000"}}
	invoiceID := createInvoiceID(t, app, cookies, spec)

	resp = doRequest(t, app, "POST", "/api/payments/online", `{"invoiceId":"`+invoiceID+`"}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	decodeBody(t, resp)

	resp = doRequest(t, app, "POST", "/api/payments", `{
		"invoiceId":"`+invoiceID+`","amount":50000,"method":"Cash","paid_on":"`+today+`"
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)

	// Detail reports paid and still exposes the persisted link.
	resp = doRequest(t, app, "GET", "/api/invoices/"+invoiceID, "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	detail := decodeBody(t, resp)["invoice"].(map[string]interface{})
	assert.Equal(t, "paid", detail["effective_status"])
	assert.NotNil(t, detail["payment_link"])

	// Content edit is rejected.
	edit := newInvoice()
	edit.ClientID, edit.Issue, edit.Due = clientID, today, today
	edit.Items = []invoiceLine{{Description: "Changed", Quantity: 1, Rate: "1"}}
	resp = doRequest(t, app, "PATCH", "/api/invoices/"+invoiceID, edit.body(t), cookies)
	assert.Equal(t, 422, resp.StatusCode)
	resp.Body.Close()

	// Status downgrade without voiding is rejected.
	resp = doRequest(t, app, "PATCH", "/api/invoices/"+invoiceID+"/status", `{"status":"sent"}`, cookies)
	assert.Equal(t, 422, resp.StatusCode)
	resp.Body.Close()

	// Paid invoices are never hard-deleted.
	resp = doRequest(t, app, "DELETE", "/api/invoices/"+invoiceID, "", cookies)
	assert.Equal(t, 422, resp.StatusCode)
	resp.Body.Close()
}

// TestAIContractFlow covers auth, validation, and unconfigured-provider behavior.
func TestDraftOnlinePaymentBlocked(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Draft Guard User","email":"draftguard@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()
	clientID := createClient(t, app, cookies, "Guarded Payer")

	makeInvoice := func(status string) string {
		spec := newInvoice()
		spec.ClientID, spec.Status = clientID, status
		spec.Items = []invoiceLine{{Description: "Guarded service", Quantity: 1, Rate: "50000"}}
		return createInvoiceID(t, app, cookies, spec)
	}
	onlineErr := func(method, path, body string, cookies []*http.Cookie) (int, string) {
		resp := doRequest(t, app, method, path, body, cookies)
		defer resp.Body.Close()
		if resp.StatusCode < 400 {
			return resp.StatusCode, ""
		}
		return resp.StatusCode, decodeBody(t, resp)["error"].(map[string]interface{})["message"].(string)
	}

	// Draft invoices cannot get a public link at all.
	draftID := makeInvoice("draft")
	code, msg := onlineErr("POST", "/api/payments/online", `{"invoiceId":"`+draftID+`"}`, cookies)
	assert.Equal(t, 422, code)
	assert.Equal(t, "invoice is still a draft", msg)

	code, _ = onlineErr("POST", "/api/payments/online/send", `{"invoiceId":"`+draftID+`","email":"client@example.com"}`, cookies)
	assert.Equal(t, 422, code)

	// A link created while sent stops working once the invoice goes back to draft.
	sentID := makeInvoice("sent")
	resp = doRequest(t, app, "POST", "/api/payments/online", `{"invoiceId":"`+sentID+`"}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	token := decodeBody(t, resp)["token"].(string)

	resp = doRequest(t, app, "PATCH", "/api/invoices/"+sentID+"/status", `{"status":"draft"}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()

	code, _ = onlineErr("GET", "/api/public/pay/"+token, "", nil)
	assert.Equal(t, 422, code)
	code, _ = onlineErr("POST", "/api/public/pay/"+token+"/transaction", "", nil)
	assert.Equal(t, 422, code)

	// Paid invoices cannot get new links...
	paidID := makeInvoice("sent")
	resp = doRequest(t, app, "POST", "/api/payments", `{
		"invoiceId":"`+paidID+`","amount":50000,"method":"Cash","paid_on":"2026-09-21"
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	resp.Body.Close()

	code, msg = onlineErr("POST", "/api/payments/online", `{"invoiceId":"`+paidID+`"}`, cookies)
	assert.Equal(t, 400, code)
	assert.Equal(t, "invoice is already paid", msg)

	// ...but an existing link stays open as a receipt.
	receiptID := makeInvoice("sent")
	resp = doRequest(t, app, "POST", "/api/payments/online", `{"invoiceId":"`+receiptID+`"}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	receiptToken := decodeBody(t, resp)["token"].(string)

	resp = doRequest(t, app, "POST", "/api/payments", `{
		"invoiceId":"`+receiptID+`","amount":50000,"method":"Cash","paid_on":"2026-09-21"
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, app, "POST", "/api/payments/online", `{"invoiceId":"`+receiptID+`"}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, receiptToken, decodeBody(t, resp)["token"])

	resp = doRequest(t, app, "GET", "/api/public/pay/"+receiptToken, "", nil)
	require.Equal(t, 200, resp.StatusCode)
	assert.False(t, decodeBody(t, resp)["can_pay"].(bool))
}
