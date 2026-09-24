package routes

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runGatewayInvoiceFlow(t *testing.T, app *fiber.App, apiKey string) {
	t.Helper()
	headers := map[string]string{"X-Api-Key": apiKey, "Idempotency-Key": "invoice-create-1"}
	body := `{"external_id":"order-invoice-1","status":"sent","currency":"IDR","customer":{"external_id":"customer-1","name":"PT Invoice Test","email":"billing@example.com"},"items":[{"description":"Subscription","quantity":1,"rate":"150000"}]}`
	resp := doGatewayRequest(t, app, "POST", "/api/gateway/invoices", body, headers, nil)
	require.Equal(t, 201, resp.StatusCode)
	created := decodeBody(t, resp)["invoice"].(map[string]interface{})
	assert.Equal(t, "order-invoice-1", created["external_id"])
	assert.Equal(t, "sent", created["status"])
	resp.Body.Close()

	resp = doGatewayRequest(t, app, "POST", "/api/gateway/invoices", body, headers, nil)
	require.Equal(t, 201, resp.StatusCode)
	replayed := decodeBody(t, resp)["invoice"].(map[string]interface{})
	assert.Equal(t, created["id"], replayed["id"])
	resp.Body.Close()

	resp = doGatewayRequest(t, app, "POST", "/api/gateway/invoices", body,
		map[string]string{"X-Api-Key": apiKey, "Idempotency-Key": "invoice-create-2"}, nil)
	assert.Equal(t, 409, resp.StatusCode)
	resp.Body.Close()
}
