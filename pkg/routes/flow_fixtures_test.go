package routes

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
	"github.com/tertua/tupay/app/models"
)

// invoiceLine is one line item in an invoice spec.
type invoiceLine struct {
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	Rate        string  `json:"rate"`
}

// invoiceSpec describes the fields a flow test cares about; everything else
// falls back to newInvoice()'s defaults. Add fields here as tests need them
// rather than hand-writing JSON at each call site.
//
// ClientID is deliberately explicit: a test that bills someone calls
// createClient() first (see the helper below) so it also holds the id for
// later assertions. There is no implicit client creation — that ambiguity is
// how fixtures rot.
type invoiceSpec struct {
	Status        string        `json:"status"`
	Currency      string        `json:"currency"`
	ClientID      string        `json:"client_id,omitempty"`
	Issue         string        `json:"issue_date"`
	Due           string        `json:"due_date"`
	TaxRate       float64       `json:"tax_rate,omitempty"`
	Discount      string        `json:"discount,omitempty"`
	PaymentMethod string        `json:"payment_method,omitempty"`
	Items         []invoiceLine `json:"items"`
}

// newInvoice returns a sent IDR invoice with one line and stable dates.
// Tests override only the fields they exercise.
func newInvoice() invoiceSpec {
	return invoiceSpec{
		Status:   models.InvoiceStatusSent,
		Currency: "IDR",
		Issue:    "2026-09-01",
		Due:      "2026-09-30",
		Items:    []invoiceLine{{Description: "Service", Quantity: 1, Rate: "100000"}},
	}
}

// body renders the spec as the JSON request body.
func (s invoiceSpec) body(t *testing.T) string {
	t.Helper()
	raw, err := json.Marshal(s)
	require.NoError(t, err)
	return string(raw)
}

// createInvoice posts the spec and returns the decoded "invoice" object.
// It fails the test on any non-201 response, so callers read as setup.
func createInvoice(t *testing.T, app *fiber.App, cookies []*http.Cookie, spec invoiceSpec) map[string]interface{} {
	t.Helper()
	resp := doRequest(t, app, "POST", "/api/invoices", spec.body(t), cookies)
	require.Equal(t, 201, resp.StatusCode, "createInvoice: %s", spec.body(t))
	return decodeBody(t, resp)["invoice"].(map[string]interface{})
}

// createInvoiceID is createInvoice for the common case where only the id matters.
func createInvoiceID(t *testing.T, app *fiber.App, cookies []*http.Cookie, spec invoiceSpec) string {
	t.Helper()
	return createInvoice(t, app, cookies, spec)["id"].(string)
}

// createClient makes a client and returns its id. Sent invoices must name a
// client, so flow tests that bill someone need this fixture.
func createClient(t *testing.T, app *fiber.App, cookies []*http.Cookie, name string) string {
	t.Helper()
	resp := doRequest(t, app, "POST", "/api/clients", `{"name":"`+name+`"}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	return decodeBody(t, resp)["client"].(map[string]interface{})["id"].(string)
}
