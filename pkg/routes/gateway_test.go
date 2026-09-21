package routes

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/invoiceman/platform/database"
	"github.com/tertua/invoiceman/platform/relay"
)

type receivedRelay struct {
	mu     sync.Mutex
	count  int
	bodies [][]byte
	sigs   []string
}

func doGatewayRequest(t *testing.T, app *fiber.App, method, route, body string, headers map[string]string, cookies []*http.Cookie) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, route, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	resp, err := app.Test(req)
	require.NoError(t, err)
	return resp
}

func TestGatewayRelayFlow(t *testing.T) {
	// Mock Snap endpoint.
	snapServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"snap-tok-1","redirect_url":"https://snap.test/1"}`))
	}))
	defer snapServer.Close()

	// Mock downstream project receiver.
	receiver := &receivedRelay{}
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		_ = r.Body.Close()
		receiver.mu.Lock()
		receiver.count++
		receiver.bodies = append(receiver.bodies, body)
		receiver.sigs = append(receiver.sigs, r.Header.Get("X-Relay-Signature"))
		receiver.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer downstream.Close()

	os.Setenv("MIDTRANS_SERVER_KEY", "test-server-key")
	os.Setenv("MIDTRANS_SNAP_BASE_URL", snapServer.URL)
	defer os.Unsetenv("MIDTRANS_SERVER_KEY")
	defer os.Unsetenv("MIDTRANS_SNAP_BASE_URL")

	app := newTestApp()

	// Admin session (promote directly so the test is independent of execution order).
	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Relay Admin","email":"relay-admin@example.com","password":"secret123"}`, nil)
	require.True(t, resp.StatusCode == 201 || resp.StatusCode == 409)
	resp.Body.Close()
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"relay-admin@example.com","password":"secret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	adminID := decodeBody(t, resp)["user"].(map[string]interface{})["id"].(string)
	adminCookies := resp.Cookies()
	resp.Body.Close()
	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	require.NoError(t, db.UpdateUserRole(uuid.MustParse(adminID), "admin"))
	// Refresh session so the role change takes effect on subsequent requests.
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"relay-admin@example.com","password":"secret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	adminCookies = resp.Cookies()
	resp.Body.Close()

	// Register downstream project.
	resp = doRequest(t, app, "POST", "/api/admin/gateway/projects",
		`{"slug":"one-api","name":"One API","webhook_url":"`+downstream.URL+`"}`, adminCookies)
	require.Equal(t, 201, resp.StatusCode)
	created := decodeBody(t, resp)
	project := created["project"].(map[string]interface{})
	apiKey := project["api_key"].(string)
	webhookSecret := project["webhook_secret"].(string)
	require.NotEmpty(t, apiKey)
	require.NotEmpty(t, webhookSecret)

	// Service creates an intent (identity from header, no slug in body).
	headers := map[string]string{"X-Api-Key": apiKey}
	resp = doGatewayRequest(t, app, "POST", "/api/gateway/intents",
		`{"external_order_id":"topup_abc123","amount_idr":32000}`, headers, nil)
	require.Equal(t, 201, resp.StatusCode)
	intent := decodeBody(t, resp)
	orderID := intent["order_id"].(string)
	require.True(t, strings.HasPrefix(orderID, "EXT-one-api-topup_abc123-"), orderID)
	assert.Equal(t, "snap-tok-1", intent["snap_token"])
	assert.Equal(t, "midtrans", intent["gateway"])
	resp.Body.Close()

	// Unknown gateway webhooks are rejected.
	resp = doGatewayRequest(t, app, "POST", "/api/webhooks/bogus", `{}`, nil, nil)
	assert.Equal(t, 404, resp.StatusCode)
	resp.Body.Close()

	// Missing key is rejected.
	resp = doGatewayRequest(t, app, "POST", "/api/gateway/intents",
		`{"external_order_id":"topup_nope","amount_idr":1000}`, nil, nil)
	assert.Equal(t, 401, resp.StatusCode)
	resp.Body.Close()

	// Status polling works for the owner.
	resp = doGatewayRequest(t, app, "GET", "/api/gateway/intents/"+orderID, "", headers, nil)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "pending", decodeBody(t, resp)["status"])
	resp.Body.Close()

	// Midtrans notification with a valid signature.
	gross := "32000.00"
	sum := sha512.Sum512([]byte(orderID + "200" + gross + "test-server-key"))
	notif, _ := json.Marshal(map[string]string{
		"transaction_id":     "mid-txn-1",
		"order_id":           orderID,
		"transaction_status": "settlement",
		"payment_type":       "qris",
		"status_code":        "200",
		"gross_amount":       gross,
		"signature_key":      hex.EncodeToString(sum[:]),
	})
	resp = doGatewayRequest(t, app, "POST", "/api/webhooks/midtrans", string(notif), nil, nil)
	require.Equal(t, 200, resp.StatusCode)
	webhookBody := decodeBody(t, resp)
	assert.Equal(t, true, webhookBody["success"])
	assert.Equal(t, true, webhookBody["relayed"])

	// Downstream received exactly one signed forward.
	receiver.mu.Lock()
	require.Equal(t, 1, receiver.count)
	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(receiver.bodies[0], &payload))
	sig := receiver.sigs[0]
	receiver.mu.Unlock()
	assert.Equal(t, "topup_abc123", payload["external_order_id"])
	assert.Equal(t, "success", payload["status"])
	assert.Equal(t, "midtrans", payload["gateway"])
	assert.True(t, relay.VerifySignature(receiver.bodies[0], webhookSecret, sig))

	// Replay is idempotent: no second forward.
	resp = doGatewayRequest(t, app, "POST", "/api/webhooks/midtrans", string(notif), nil, nil)
	assert.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()
	receiver.mu.Lock()
	assert.Equal(t, 1, receiver.count)
	receiver.mu.Unlock()

	// Status is now success.
	resp = doGatewayRequest(t, app, "GET", "/api/gateway/intents/"+orderID, "", headers, nil)
	assert.Equal(t, "success", decodeBody(t, resp)["status"])
	resp.Body.Close()

	// Tampered signature is rejected.
	bad, _ := json.Marshal(map[string]string{
		"transaction_id": "mid-txn-1", "order_id": orderID,
		"transaction_status": "settlement", "payment_type": "qris",
		"status_code": "200", "gross_amount": gross, "signature_key": "bad",
	})
	resp = doGatewayRequest(t, app, "POST", "/api/webhooks/midtrans", string(bad), nil, nil)
	assert.Equal(t, 401, resp.StatusCode)
	resp.Body.Close()
}
