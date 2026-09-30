package routes

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInvoiceStatusCountsFlow covers GET /api/invoices/status-counts: the
// badges must reflect the same effective-status rules as the listing, and
// must stay scoped to the caller's org.
func TestInvoiceStatusCountsFlow(t *testing.T) {
	app := newTestApp()

	cookies := registerUser(t, app, "status-counts@example.com", "secret123")

	// Empty org starts at zero across every bucket, including "all".
	resp := doRequest(t, app, "GET", "/api/invoices/status-counts", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	empty := decodeBody(t, resp)["counts"].(map[string]interface{})
	assert.Equal(t, float64(0), empty["all"])
	assert.Equal(t, float64(0), empty["draft"])
	assert.Equal(t, float64(0), empty["sent"])
	assert.Equal(t, float64(0), empty["paid"])
	assert.Equal(t, float64(0), empty["overdue"])

	clientID := createClient(t, app, cookies, "Counts Client")

	// One draft.
	draft := newInvoice()
	draft.ClientID, draft.Status, draft.Currency = clientID, "draft", "IDR"
	createInvoice(t, app, cookies, draft)

	// One sent invoice still within its due date.
	sent := newInvoice()
	sent.ClientID, sent.Currency = clientID, "IDR"
	createInvoice(t, app, cookies, sent)

	// One sent invoice whose due date already passed -> effective overdue.
	overdue := newInvoice()
	overdue.ClientID, overdue.Currency = clientID, "IDR"
	overdue.Issue = time.Now().AddDate(0, 0, -40).Format("2006-01-02")
	overdue.Due = time.Now().AddDate(0, 0, -10).Format("2006-01-02")
	overdueID := createInvoiceID(t, app, cookies, overdue)

	// One invoice paid in full: recording a payment covering the total makes
	// it effectively paid even though the stored status is still "sent".
	paid := newInvoice()
	paid.ClientID, paid.Currency = clientID, "IDR"
	paid.Items = []invoiceLine{{Description: "Paid work", Quantity: 1, Rate: "100"}}
	paidID := createInvoiceID(t, app, cookies, paid)
	resp = doRequest(t, app, "POST", "/api/payments",
		`{"invoiceId":"`+paidID+`","amount":100,"method":"Cash","paid_on":"`+time.Now().Format("2006-01-02")+`"}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)

	resp = doRequest(t, app, "GET", "/api/invoices/status-counts", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	counts := decodeBody(t, resp)["counts"].(map[string]interface{})
	assert.Equal(t, float64(4), counts["all"])
	assert.Equal(t, float64(1), counts["draft"])
	assert.Equal(t, float64(1), counts["paid"])
	assert.Equal(t, float64(1), counts["overdue"])
	// The remaining sent invoice (still before its due date) is the only "sent".
	assert.Equal(t, float64(1), counts["sent"])

	// The badge counts agree with what the listing filter returns.
	resp = doRequest(t, app, "GET", "/api/invoices?status=overdue", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	overdueList := decodeBody(t, resp)["invoices"].([]interface{})
	require.Len(t, overdueList, 1)
	assert.Equal(t, overdueID, overdueList[0].(map[string]interface{})["id"])

	// Counts are org-wide: a search filter on the listing must not shrink them.
	resp = doRequest(t, app, "GET", "/api/invoices/status-counts?search=nomatch", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	searched := decodeBody(t, resp)["counts"].(map[string]interface{})
	assert.Equal(t, float64(4), searched["all"])

	// Org isolation: a second user never sees the first org's counts.
	otherCookies := registerUser(t, app, "status-counts-other@example.com", "secret123")
	resp = doRequest(t, app, "GET", "/api/invoices/status-counts", "", otherCookies)
	require.Equal(t, 200, resp.StatusCode)
	otherCounts := decodeBody(t, resp)["counts"].(map[string]interface{})
	assert.Equal(t, float64(0), otherCounts["all"])
}
