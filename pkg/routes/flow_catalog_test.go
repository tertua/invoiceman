package routes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestItemFlow(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register", `{"name":"Item User","email":"item@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()

	resp = doRequest(t, app, "POST", "/api/items", `{"name":"Consulting","description":"Hourly consulting","rate":125,"unit":"hour"}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	item := decodeBody(t, resp)["item"].(map[string]interface{})
	itemID := item["id"].(string)
	assert.Equal(t, "Consulting", item["name"])

	resp = doRequest(t, app, "GET", "/api/items", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	items := decodeBody(t, resp)["items"].([]interface{})
	require.Len(t, items, 1)

	resp = doRequest(t, app, "PATCH", "/api/items/"+itemID, `{"name":"Strategy","description":"Strategy session","rate":200,"unit":"session"}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	updated := decodeBody(t, resp)["item"].(map[string]interface{})
	assert.Equal(t, "Strategy", updated["name"])
	assert.Equal(t, float64(200), updated["rate"])

	resp = doRequest(t, app, "DELETE", "/api/items/"+itemID, "", cookies)
	assert.Equal(t, 204, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, app, "GET", "/api/items", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	assert.Empty(t, decodeBody(t, resp)["items"])
}

// TestSettingsFlow covers settings defaults and updates.
func TestSettingsFlow(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register", `{"name":"Settings User","email":"settings@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	cookies := resp.Cookies()

	resp = doRequest(t, app, "GET", "/api/settings", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	settings := decodeBody(t, resp)["settings"].(map[string]interface{})
	assert.Equal(t, "IDR", settings["currency"])
	assert.Equal(t, "INV-", settings["invoice_prefix"])
	assert.Empty(t, settings["language"])

	resp = doRequest(t, app, "PATCH", "/api/settings", `{
		"company_name":"Acme Studio",
		"currency":"IDR",
		"tax_rate":11,
		"invoice_prefix":"ACME-",
		"language":"id"
	}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	updated := decodeBody(t, resp)["settings"].(map[string]interface{})
	assert.Equal(t, "Acme Studio", updated["company_name"])
	assert.Equal(t, "IDR", updated["currency"])
	assert.Equal(t, float64(11), updated["tax_rate"])
	assert.Equal(t, "ACME-", updated["invoice_prefix"])
	assert.Equal(t, "id", updated["language"])
	resp = doRequest(t, app, "PATCH", "/api/settings", `{"currency":"IDR","invoice_prefix":"ACME-","language":""}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "id", decodeBody(t, resp)["settings"].(map[string]interface{})["language"])

	resp = doRequest(t, app, "PATCH", "/api/settings", `{"currency":"INVALID"}`, cookies)
	assert.Equal(t, 400, resp.StatusCode)
	resp.Body.Close()
}

// TestExpenseFlow covers expense CRUD, filtering, and totals.
