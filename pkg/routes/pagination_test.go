package routes

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClientPagination verifies page/per_page slicing and meta totals.
func TestClientPagination(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Page User","email":"page@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	cookies := resp.Cookies()
	resp.Body.Close()

	for i := 0; i < 5; i++ {
		resp = doRequest(t, app, "POST", "/api/clients",
			fmt.Sprintf(`{"name":"Client %d"}`, i), cookies)
		require.Equal(t, 201, resp.StatusCode)
		resp.Body.Close()
	}

	// Page 2 of 5 with 2 per page.
	resp = doRequest(t, app, "GET", "/api/clients?per_page=2&page=2", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	body := decodeBody(t, resp)
	assert.Len(t, body["clients"], 2)
	meta := body["meta"].(map[string]interface{})
	assert.Equal(t, float64(2), meta["page"])
	assert.Equal(t, float64(2), meta["per_page"])
	assert.Equal(t, float64(5), meta["total"])
	assert.Equal(t, float64(3), meta["total_pages"])

	// Page 3 holds the remainder.
	resp = doRequest(t, app, "GET", "/api/clients?per_page=2&page=3", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	body = decodeBody(t, resp)
	assert.Len(t, body["clients"], 1)

	// Invalid values fall back to defaults; oversized clamps to 100.
	resp = doRequest(t, app, "GET", "/api/clients?page=abc&per_page=9999", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	body = decodeBody(t, resp)
	meta = body["meta"].(map[string]interface{})
	assert.Equal(t, float64(1), meta["page"])
	assert.Equal(t, float64(100), meta["per_page"])
	assert.Equal(t, float64(5), meta["total"])

	// No params keeps the legacy shape plus meta defaults.
	resp = doRequest(t, app, "GET", "/api/clients", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	body = decodeBody(t, resp)
	assert.Len(t, body["clients"], 5)
	assert.Contains(t, body, "meta")
}
