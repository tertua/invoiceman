package routes

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCurrentMonthTotalsUseUTCMonthBounds pins the "this month" filters to
// UTC-midnight bounds. Date-only columns (expense_date, paid_on) hold UTC
// midnights, but SQLite compares the query bound rendered in the server's
// local zone as text, so a local-zone lower bound ("...+07:00") against
// stored "...+00:00" drops the start edge of the month whenever the host
// zone is east of UTC.
func TestCurrentMonthTotalsUseUTCMonthBounds(t *testing.T) {
	prevLocal := time.Local
	time.Local = time.FixedZone("UTC+07", 7*60*60)
	t.Cleanup(func() { time.Local = prevLocal })

	// First day of the server's current month (the month MonthBounds uses),
	// stored as UTC midnight: the exact day a local-zone lower bound loses
	// under SQLite's text comparison.
	now := time.Now()
	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")

	app := newTestApp()
	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Bounds User","email":"bounds@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()

	resp = doRequest(t, app, "POST", "/api/expenses", `{
		"vendor":"First Day","category":"Software",
		"expense_date":"`+firstOfMonth+`","amount":100
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)

	resp = doRequest(t, app, "GET", "/api/expenses", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	totals := decodeBody(t, resp)["totals"].(map[string]interface{})
	assert.Equal(t, "100", totals["thisMonth"],
		"expense dated %s must count in the current month", firstOfMonth)

	// The same edge for payments.paid_on.
	spec := newInvoice()
	spec.ClientID, spec.Currency = createClient(t, app, cookies, "Bounds Payer"), "USD"
	spec.Items = []invoiceLine{{Description: "Service", Quantity: 1, Rate: "100"}}
	invoiceID := createInvoiceID(t, app, cookies, spec)

	resp = doRequest(t, app, "POST", "/api/payments", `{
		"invoiceId":"`+invoiceID+`","amount":40,"method":"Cash",
		"paid_on":"`+firstOfMonth+`"
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)

	resp = doRequest(t, app, "GET", "/api/payments", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	pTotals := decodeBody(t, resp)["totals"].(map[string]interface{})
	assert.Equal(t, "40", pTotals["thisMonth"],
		"payment paid on %s must count in the current month", firstOfMonth)
}
