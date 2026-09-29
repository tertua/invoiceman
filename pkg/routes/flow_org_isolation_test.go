package routes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOrgIsolationFlow proves the tenant boundary: two personal orgs never see each other's invoices, clients, dashboard counts, invoice numbers or settings.
func TestOrgIsolationFlow(t *testing.T) {
	t.Setenv("RATE_LIMIT_AUTH", "100")
	app := newTestApp()

	alpha := registerUser(t, app, "iso-alpha@example.com", "secret123")
	beta := registerUser(t, app, "iso-beta@example.com", "secret123")
	orgAlpha, orgBeta := myOrgID(t, app, alpha), myOrgID(t, app, beta)
	require.NotEqual(t, orgAlpha, orgBeta)

	spec := newInvoice()
	spec.ClientID = createClient(t, app, alpha, "Alpha Client")
	alphaInvoice := createInvoice(t, app, alpha, spec)
	alphaInvoiceID := alphaInvoice["id"].(string)
	assert.Equal(t, "INV-000001", alphaInvoice["invoice_number"])

	// Each org lists only its own rows: org B starts empty.
	resp := doRequest(t, app, "GET", "/api/invoices", "", alpha)
	require.Equal(t, 200, resp.StatusCode)
	assert.Len(t, decodeBody(t, resp)["invoices"], 1)
	resp = doRequest(t, app, "GET", "/api/invoices", "", beta)
	require.Equal(t, 200, resp.StatusCode)
	assert.Empty(t, decodeBody(t, resp)["invoices"])

	// Org A's invoice is invisible from org B on every verb, as if it never existed.
	resp = doRequest(t, app, "GET", "/api/invoices/"+alphaInvoiceID, "", beta)
	require.Equal(t, 404, resp.StatusCode)
	resp.Body.Close()
	foreign := newInvoice()
	foreign.ClientID = spec.ClientID
	resp = doRequest(t, app, "PATCH", "/api/invoices/"+alphaInvoiceID, foreign.body(t), beta)
	require.Equal(t, 404, resp.StatusCode)
	resp.Body.Close()
	resp = doRequest(t, app, "DELETE", "/api/invoices/"+alphaInvoiceID, "", beta)
	require.Equal(t, 404, resp.StatusCode)
	resp.Body.Close()
	resp = doRequest(t, app, "GET", "/api/clients/"+spec.ClientID, "", beta)
	require.Equal(t, 404, resp.StatusCode)
	resp.Body.Close()
	resp = doRequest(t, app, "PATCH", "/api/clients/"+spec.ClientID, `{"name":"Beta Client"}`, beta)
	require.Equal(t, 404, resp.StatusCode)
	resp.Body.Close()
	resp = doRequest(t, app, "DELETE", "/api/clients/"+spec.ClientID, "", beta)
	require.Equal(t, 404, resp.StatusCode)
	resp.Body.Close()

	// The client list is scoped the same way, so org B sees nobody.
	resp = doRequest(t, app, "GET", "/api/clients", "", alpha)
	require.Equal(t, 200, resp.StatusCode)
	assert.Len(t, decodeBody(t, resp)["clients"], 1)
	resp = doRequest(t, app, "GET", "/api/clients", "", beta)
	require.Equal(t, 200, resp.StatusCode)
	assert.Empty(t, decodeBody(t, resp)["clients"])

	// The refused writes left org A's client editable.
	resp = doRequest(t, app, "GET", "/api/clients/"+spec.ClientID, "", alpha)
	require.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()
	resp = doRequest(t, app, "PATCH", "/api/clients/"+spec.ClientID, `{"name":"Beta Renamed"}`, beta)
	require.Equal(t, 404, resp.StatusCode)
	assert.Equal(t, "client not found", envelopeMessage(t, resp))
	resp = doRequest(t, app, "DELETE", "/api/clients/"+spec.ClientID, "", beta)
	require.Equal(t, 404, resp.StatusCode)
	assert.Equal(t, "client not found", envelopeMessage(t, resp))

	// The client list of org B holds only its own rows.
	resp = doRequest(t, app, "GET", "/api/clients", "", beta)
	require.Equal(t, 200, resp.StatusCode)
	assert.Empty(t, decodeBody(t, resp)["clients"])

	// The refused reads and deletes left org A's invoice untouched.
	resp = doRequest(t, app, "GET", "/api/invoices/"+alphaInvoiceID, "", alpha)
	require.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()

	// Dashboard counts are scoped per org: alpha owns one invoice and one client, beta none.
	resp = doRequest(t, app, "GET", "/api/dashboard?currency=IDR", "", alpha)
	require.Equal(t, 200, resp.StatusCode)
	alphaStats := decodeBody(t, resp)["stats"].(map[string]interface{})
	assert.Equal(t, float64(1), alphaStats["invoiceCount"])
	assert.Equal(t, float64(1), alphaStats["clientCount"])
	resp = doRequest(t, app, "GET", "/api/dashboard?currency=IDR", "", beta)
	require.Equal(t, 200, resp.StatusCode)
	betaStats := decodeBody(t, resp)["stats"].(map[string]interface{})
	assert.Equal(t, float64(0), betaStats["invoiceCount"])
	assert.Equal(t, float64(0), betaStats["clientCount"])

	// The sequence lives in each org's settings row, so org B's first invoice restarts at the same number.
	betaSpec := newInvoice()
	betaSpec.Status = "draft"
	betaInvoice := createInvoice(t, app, beta, betaSpec)
	assert.Equal(t, "INV-000001", betaInvoice["invoice_number"])

	// Settings stay per org: beta renames itself without touching alpha.
	resp = doRequest(t, app, "PATCH", "/api/settings",
		`{"company_name":"Alpha Co","currency":"IDR","tax_rate":11,"invoice_prefix":"INV-","usd_to_idr":"18000"}`, alpha)
	require.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()
	resp = doRequest(t, app, "GET", "/api/settings", "", beta)
	require.Equal(t, 200, resp.StatusCode)
	assert.NotEqual(t, "Alpha Co", decodeBody(t, resp)["settings"].(map[string]interface{})["company_name"])
	resp = doRequest(t, app, "PATCH", "/api/settings",
		`{"company_name":"Beta Co","currency":"IDR","tax_rate":11,"invoice_prefix":"INV-","usd_to_idr":"18000"}`, beta)
	require.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()
	resp = doRequest(t, app, "GET", "/api/settings", "", alpha)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "Alpha Co", decodeBody(t, resp)["settings"].(map[string]interface{})["company_name"])
}
