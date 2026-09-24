package routes

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runGatewayInvoiceFlow(t *testing.T, app *fiber.App, apiKey string) {
	t.Helper()

	// Happy path: create sent invoice with customer.
	headers := map[string]string{"X-Api-Key": apiKey, "Idempotency-Key": "invoice-create-1"}
	baseBody := `{"external_id":"order-invoice-1","status":"sent","currency":"IDR","customer":{"external_id":"customer-1","name":"PT Invoice Test","email":"billing@example.com"},"items":[{"description":"Subscription","quantity":1,"rate":"150000"}]}`
	resp := doGatewayRequest(t, app, "POST", "/api/gateway/invoices", baseBody, headers, nil)
	require.Equal(t, 201, resp.StatusCode)
	created := decodeBody(t, resp)["invoice"].(map[string]interface{})
	assert.Equal(t, "order-invoice-1", created["external_id"])
	assert.Equal(t, "sent", created["status"])
	resp.Body.Close()

	// Idempotent replay: same key returns same invoice.
	resp = doGatewayRequest(t, app, "POST", "/api/gateway/invoices", baseBody, headers, nil)
	require.Equal(t, 201, resp.StatusCode)
	replayed := decodeBody(t, resp)["invoice"].(map[string]interface{})
	assert.Equal(t, created["id"], replayed["id"])
	resp.Body.Close()

	// Duplicate external_id with different key returns 409.
	resp = doGatewayRequest(t, app, "POST", "/api/gateway/invoices", baseBody,
		map[string]string{"X-Api-Key": apiKey, "Idempotency-Key": "invoice-create-2"}, nil)
	assert.Equal(t, 409, resp.StatusCode)
	resp.Body.Close()

	// Missing Idempotency-Key returns 400.
	resp = doGatewayRequest(t, app, "POST", "/api/gateway/invoices",
		`{"external_id":"no-key","status":"draft","currency":"IDR","customer":{"external_id":"c-no-key","name":"No Key"},"items":[{"description":"x","quantity":1,"rate":"1000"}]}`,
		map[string]string{"X-Api-Key": apiKey}, nil)
	assert.Equal(t, 400, resp.StatusCode)
	resp.Body.Close()

	// Invalid currency (too long) returns 400.
	resp = doGatewayRequest(t, app, "POST", "/api/gateway/invoices",
		`{"external_id":"bad-currency","status":"draft","currency":"ABCD","customer":{"external_id":"c-bad-ccy","name":"Bad Ccy"},"items":[{"description":"x","quantity":1,"rate":"1000"}]}`,
		map[string]string{"X-Api-Key": apiKey, "Idempotency-Key": "inv-bad-ccy"}, nil)
	assert.Equal(t, 400, resp.StatusCode)
	resp.Body.Close()

	// Empty items returns 400.
	resp = doGatewayRequest(t, app, "POST", "/api/gateway/invoices",
		`{"external_id":"empty-items","status":"draft","currency":"IDR","customer":{"external_id":"c-empty","name":"Empty Items"},"items":[]}`,
		map[string]string{"X-Api-Key": apiKey, "Idempotency-Key": "inv-empty"}, nil)
	assert.Equal(t, 400, resp.StatusCode)
	resp.Body.Close()

	// Invalid date returns 400.
	resp = doGatewayRequest(t, app, "POST", "/api/gateway/invoices",
		`{"external_id":"bad-date","status":"draft","currency":"IDR","issue_date":"not-a-date","customer":{"external_id":"c-bad-date","name":"Bad Date"},"items":[{"description":"x","quantity":1,"rate":"1000"}]}`,
		map[string]string{"X-Api-Key": apiKey, "Idempotency-Key": "inv-bad-date"}, nil)
	assert.Equal(t, 400, resp.StatusCode)
	resp.Body.Close()

	// Negative discount returns 400.
	resp = doGatewayRequest(t, app, "POST", "/api/gateway/invoices",
		`{"external_id":"neg-discount","status":"draft","currency":"IDR","discount":"-5000","customer":{"external_id":"c-neg-disc","name":"Neg Discount"},"items":[{"description":"x","quantity":1,"rate":"1000"}]}`,
		map[string]string{"X-Api-Key": apiKey, "Idempotency-Key": "inv-neg-disc"}, nil)
	assert.Equal(t, 400, resp.StatusCode)
	resp.Body.Close()

	// Negative item rate returns 400.
	resp = doGatewayRequest(t, app, "POST", "/api/gateway/invoices",
		`{"external_id":"neg-rate","status":"draft","currency":"IDR","customer":{"external_id":"c-neg-rate","name":"Neg Rate"},"items":[{"description":"x","quantity":1,"rate":"-1000"}]}`,
		map[string]string{"X-Api-Key": apiKey, "Idempotency-Key": "inv-neg-rate"}, nil)
	assert.Equal(t, 400, resp.StatusCode)
	resp.Body.Close()

	// Missing X-Api-Key returns 401.
	resp = doGatewayRequest(t, app, "POST", "/api/gateway/invoices", baseBody,
		map[string]string{"Idempotency-Key": "invoice-no-key"}, nil)
	assert.Equal(t, 401, resp.StatusCode)
	resp.Body.Close()
}
