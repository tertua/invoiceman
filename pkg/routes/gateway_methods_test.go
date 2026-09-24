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
}
