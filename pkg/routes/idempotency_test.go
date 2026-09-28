package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// doRequestWithHeaders is doRequest plus extra headers (e.g. Idempotency-Key).
func doRequestWithHeaders(t *testing.T, app *fiber.App, method, route, body string, cookies []*http.Cookie, headers map[string]string) *http.Response {
	t.Helper()

	req := httptest.NewRequest(method, route, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
		// Mirror the MPA: echo the CSRF cookie as its header.
		if cookie.Name == "csrf_token" && cookie.Value != "" {
			req.Header.Set("X-CSRF-Token", cookie.Value)
		}
	}

	resp, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	require.NoError(t, err)
	return resp
}

func readBody(t *testing.T, resp *http.Response) (code int, body map[string]interface{}, header http.Header) {
	t.Helper()
	defer resp.Body.Close()
	body = map[string]interface{}{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	return resp.StatusCode, body, resp.Header
}

// TestPaymentIdempotency covers replay, mismatch rejection and opt-in.
func TestPaymentIdempotency(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Idem User","email":"idem@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	resp.Body.Close()
	cookies := resp.Cookies()

	clientID := createClient(t, app, cookies, "Idem Co")

	spec := newInvoice()
	spec.ClientID, spec.Currency = clientID, "USD"
	spec.Items = []invoiceLine{{Description: "Work", Quantity: 1, Rate: "100"}}
	invoiceID := createInvoiceID(t, app, cookies, spec)

	payload := fmt.Sprintf(`{"invoiceId":%q,"amount":40,"method":"cash","paid_on":"2026-09-10"}`, invoiceID)
	key := "test-key-idempotency-1"

	// First execution creates the payment.
	resp = doRequestWithHeaders(t, app, "POST", "/api/payments", payload, cookies,
		map[string]string{"Idempotency-Key": key})
	status, first, headers := readBody(t, resp)
	require.Equal(t, 201, status)
	assert.Empty(t, headers.Get("Idempotent-Replayed"))
	firstID := first["payment"].(map[string]interface{})["id"].(string)
	require.NotEmpty(t, firstID)

	// Retry with the same key replays the stored response, no second row.
	resp = doRequestWithHeaders(t, app, "POST", "/api/payments", payload, cookies,
		map[string]string{"Idempotency-Key": key})
	status, second, headers := readBody(t, resp)
	require.Equal(t, 201, status)
	assert.Equal(t, "true", headers.Get("Idempotent-Replayed"))
	assert.Equal(t, firstID, second["payment"].(map[string]interface{})["id"])

	resp = doRequest(t, app, "GET", "/api/payments", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	list := decodeBody(t, resp)
	assert.Len(t, list["payments"], 1)

	// Same key with a different payload is rejected.
	other := fmt.Sprintf(`{"invoiceId":%q,"amount":50,"method":"cash","paid_on":"2026-09-10"}`, invoiceID)
	resp = doRequestWithHeaders(t, app, "POST", "/api/payments", other, cookies,
		map[string]string{"Idempotency-Key": key})
	status, mismatched, _ := readBody(t, resp)
	require.Equal(t, 422, status)
	assert.Contains(t, mismatched["error"].(map[string]interface{})["message"], "idempotency key")

	// Without a key the endpoint behaves as before (second payment).
	resp = doRequest(t, app, "POST", "/api/payments", payload, cookies)
	require.Equal(t, 201, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, app, "GET", "/api/payments", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	assert.Len(t, decodeBody(t, resp)["payments"], 2)
}
