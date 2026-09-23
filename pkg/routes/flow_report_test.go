package routes

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportsFlow(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Reports User","email":"reports@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()
	today := time.Now().Format("2006-01-02")

	// Empty reports use arrays instead of null values for frontend safety.
	resp = doRequest(t, app, "GET", "/api/reports", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	emptyReport := decodeBody(t, resp)
	assert.NotNil(t, emptyReport["monthly"])
	assert.NotNil(t, emptyReport["aging"])
	assert.Empty(t, emptyReport["topClients"])
	assert.NotNil(t, emptyReport["statusBreakdown"])

	resp = doRequest(t, app, "POST", "/api/clients", `{"name":"Report Client"}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	clientID := decodeBody(t, resp)["client"].(map[string]interface{})["id"].(string)

	resp = doRequest(t, app, "POST", "/api/invoices", `{
		"client_id":"`+clientID+`",
		"status":"sent",
		"issue_date":"`+today+`",
		"due_date":"`+today+`",
		"currency":"USD",
		"items":[{"description":"Report work","quantity":1,"rate":100}]
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	invoiceID := decodeBody(t, resp)["invoice"].(map[string]interface{})["id"].(string)

	resp = doRequest(t, app, "POST", "/api/payments", `{
		"invoiceId":"`+invoiceID+`","amount":40,"method":"Cash","paid_on":"`+today+`"
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)

	resp = doRequest(t, app, "POST", "/api/expenses", `{
		"vendor":"Report Expense","category":"Software","expense_date":"`+today+`","amount":15
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)

	resp = doRequest(t, app, "GET", "/api/reports", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	report := decodeBody(t, resp)
	totals := report["totals"].(map[string]interface{})
	assert.Equal(t, "40", totals["revenue"])
	assert.Equal(t, "15", totals["expenses"])
	assert.Equal(t, "25", totals["netProfit"])
	assert.Equal(t, "60", totals["outstanding"])

	// A draft invoice must not inflate report outstanding or aging.
	resp = doRequest(t, app, "POST", "/api/invoices", `{
		"client_id":"`+clientID+`",
		"status":"draft",
		"issue_date":"`+today+`",
		"due_date":"`+today+`",
		"currency":"USD",
		"items":[{"description":"Draft work","quantity":2,"rate":100}]
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)

	resp = doRequest(t, app, "GET", "/api/reports", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	reportAfterDraft := decodeBody(t, resp)
	assert.Equal(t, "60", reportAfterDraft["totals"].(map[string]interface{})["outstanding"])
	agingSum := "0"
	for _, item := range reportAfterDraft["aging"].([]interface{}) {
		value := item.(map[string]interface{})["value"].(string)
		if value != "0" {
			agingSum = value
		}
	}
	assert.Equal(t, "60", agingSum)
	assert.Len(t, report["monthly"].([]interface{}), 6)
	for _, item := range report["monthly"].([]interface{}) {
		point := item.(map[string]interface{})
		assert.Regexp(t, `^\d{4}-\d{2}$`, point["key"].(string))
		assert.NotEmpty(t, point["label"])
	}
	assert.Len(t, report["aging"].([]interface{}), 5)
	// Aging buckets expose stable keys for frontend localization.
	agingKeys := make([]string, 0, 5)
	for _, item := range report["aging"].([]interface{}) {
		agingKeys = append(agingKeys, item.(map[string]interface{})["bucket"].(string))
	}
	assert.Equal(t, []string{"current", "d1_30", "d31_60", "d61_90", "d90_plus"}, agingKeys)
	assert.Len(t, report["topClients"].([]interface{}), 1)
	assert.Len(t, report["statusBreakdown"].([]interface{}), 4)
}

// TestPublicPaymentFlow covers public payment links and unconfigured gateway behavior.
