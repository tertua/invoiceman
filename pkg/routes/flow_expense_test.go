package routes

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpenseFlow(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Expense User","email":"expense@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()
	today := time.Now().Format("2006-01-02")

	resp = doRequest(t, app, "POST", "/api/expenses", `{
		"vendor":"Cloud Provider",
		"category":"Software",
		"expense_date":"`+today+`",
		"amount":100,
		"notes":"Monthly hosting"
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	expense := decodeBody(t, resp)["expense"].(map[string]interface{})
	expenseID := expense["id"].(string)
	assert.Equal(t, "IDR", expense["currency"])

	resp = doRequest(t, app, "POST", "/api/expenses", `{
		"vendor":"Office Supply",
		"category":"Office",
		"expense_date":"`+today+`",
		"amount":50,
		"currency":"IDR"
	}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)

	resp = doRequest(t, app, "GET", "/api/expenses?category=Software", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	body := decodeBody(t, resp)
	assert.Len(t, body["expenses"].([]interface{}), 1)
	assert.Equal(t, "100", body["totals"].(map[string]interface{})["total"])
	assert.Equal(t, "100", body["totals"].(map[string]interface{})["thisMonth"])

	resp = doRequest(t, app, "PATCH", "/api/expenses/"+expenseID, `{
		"vendor":"Cloud Provider Updated",
		"category":"Hosting",
		"expense_date":"`+today+`",
		"amount":125,
		"notes":"Updated"
	}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	updated := decodeBody(t, resp)["expense"].(map[string]interface{})
	assert.Equal(t, "Hosting", updated["category"])
	assert.Equal(t, "125", updated["amount"])

	resp = doRequest(t, app, "DELETE", "/api/expenses/"+expenseID, "", cookies)
	assert.Equal(t, 204, resp.StatusCode)
	resp.Body.Close()
}

// TestPaymentFlow covers recording, listing, balance validation, and voiding payments.
