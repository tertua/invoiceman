package routes

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runGatewayMethodsFlow(t *testing.T, app *fiber.App, apiKey string) {
	t.Helper()
	resp := doGatewayRequest(t, app, "GET", "/api/gateway/methods", "", map[string]string{"X-Api-Key": apiKey}, nil)
	require.Equal(t, 200, resp.StatusCode)
	methods := decodeBody(t, resp)["methods"].([]interface{})
	assert.NotEmpty(t, methods)
	resp.Body.Close()

	// JSON compatibility (Phase 4): the provider-neutral alias keys carry the
	// same values as the legacy keys. Both must be present so relay clients
	// can migrate without a flag day. A fresh intent is read back from its
	// status route, which uses intentResponse too.
	headers := map[string]string{"X-Api-Key": apiKey, "Idempotency-Key": "alias-keys-1"}
	resp = doGatewayRequest(t, app, "POST", "/api/gateway/intents",
		`{"external_order_id":"alias_keys_order","amount_idr":15000}`, headers, nil)
	require.Equal(t, 201, resp.StatusCode)
	created := decodeBody(t, resp)
	orderID, _ := created["order_id"].(string)
	require.NotEmpty(t, orderID)
	require.NotEmpty(t, created["snap_token"], "midtrans intent must carry a widget token")
	assert.Equal(t, created["snap_token"], created["provider_token"], "provider_token aliases snap_token")
	assert.Equal(t, created["midtrans_txn_id"], created["provider_txn_id"], "provider_txn_id aliases midtrans_txn_id")
	resp.Body.Close()

	resp = doGatewayRequest(t, app, "GET", "/api/gateway/intents/"+orderID, "", map[string]string{"X-Api-Key": apiKey}, nil)
	require.Equal(t, 200, resp.StatusCode)
	fetched := decodeBody(t, resp)
	assert.Equal(t, fetched["snap_token"], fetched["provider_token"])
	resp.Body.Close()
}
