package routes

import (
	"context"
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
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/platform/database"
	"github.com/tertua/invoiceman/platform/outbox"
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
		// Mirror the SPA: echo the CSRF cookie as its header.
		if cookie.Name == "csrf_token" && cookie.Value != "" {
			req.Header.Set("X-CSRF-Token", cookie.Value)
		}
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

	adminCookies := adminSession(t, app, "Relay Admin", "relay-admin@example.com")

	// Register downstream project.
	resp := doRequest(t, app, "POST", "/api/admin/gateway/projects",
		`{"slug":"one-api","name":"One API","webhook_url":"`+downstream.URL+`"}`, adminCookies)
	require.Equal(t, 201, resp.StatusCode)
	created := decodeBody(t, resp)
	project := created["project"].(map[string]interface{})
	apiKey := project["api_key"].(string)
	webhookSecret := project["webhook_secret"].(string)
	require.NotEmpty(t, apiKey)
	require.NotEmpty(t, webhookSecret)

	headers := map[string]string{"X-Api-Key": apiKey, "Idempotency-Key": "relay-flow-1"}
	resp = doGatewayRequest(t, app, "POST", "/api/gateway/intents",
		`{"external_order_id":"topup_abc123","amount_idr":32000}`, headers, nil)
	require.Equal(t, 201, resp.StatusCode)
	intent := decodeBody(t, resp)
	orderID := intent["order_id"].(string)
	require.True(t, strings.HasPrefix(orderID, "EXT-one-api-topup_abc123-"), orderID)
	assert.Equal(t, "snap-tok-1", intent["snap_token"])
	assert.Equal(t, "midtrans", intent["gateway"])
	resp.Body.Close()
	// Service can create a customer and sent invoice atomically.
	runGatewayInvoiceFlow(t, app, apiKey)

	resp = doGatewayRequest(t, app, "POST", "/api/webhooks/bogus", `{}`, nil, nil)
	assert.Equal(t, 404, resp.StatusCode)
	resp.Body.Close()

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
	assert.Equal(t, true, webhookBody["queued"])

	// The background worker forwards asynchronously.
	outbox.New().ProcessOnce(context.Background())

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

func TestGatewayStatusFlow(t *testing.T) {
	t.Setenv("MIDTRANS_SERVER_KEY", "test-status-key")
	t.Setenv("NOWPAYMENTS_API_KEY", "")
	app := newTestApp()

	// NOTE: newTestApp only registers the legacy prefix; v1 coverage lives
	// in versioning_test.go via RegisterAPI.
	resp := doGatewayRequest(t, app, "GET", "/api/public/gateway/status", "", nil, nil)
	require.Equal(t, 200, resp.StatusCode)
	body := decodeBody(t, resp)
	raw, _ := json.Marshal(body)
	// No secret material may leak through the status endpoint.
	assert.NotContains(t, string(raw), "test-status-key")
	gateways, ok := body["gateways"].([]interface{})
	require.True(t, ok, "expected gateways array, got %v", body)
	byName := map[string]map[string]interface{}{}
	for _, g := range gateways {
		m := g.(map[string]interface{})
		byName[m["name"].(string)] = m
	}
	require.Contains(t, byName, "midtrans")
	require.Contains(t, byName, "nowpayments")
	assert.Equal(t, true, byName["midtrans"]["configured"])
	assert.Equal(t, false, byName["nowpayments"]["configured"])
	resp.Body.Close()

	// With a sandbox key, NOWPayments reports configured + sandbox.
	t.Setenv("NOWPAYMENTS_API_KEY", "test-status-np-key")
	t.Setenv("NOWPAYMENTS_SANDBOX", "true")
	resp = doGatewayRequest(t, app, "GET", "/api/public/gateway/status", "", nil, nil)
	require.Equal(t, 200, resp.StatusCode)
	body = decodeBody(t, resp)
	raw, _ = json.Marshal(body)
	assert.NotContains(t, string(raw), "test-status-np-key")
	for _, g := range body["gateways"].([]interface{}) {
		m := g.(map[string]interface{})
		if m["name"] == "nowpayments" {
			assert.Equal(t, true, m["configured"])
			assert.Equal(t, true, m["sandbox"])
		}
	}
	resp.Body.Close()
}

func TestNowpaymentsIntentValidation(t *testing.T) {
	// Mock NOWPayments invoice endpoint.
	npServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"inv_np_1","invoice_url":"https://nowpayments.io/payment/?iid=np1"}`))
	}))
	defer npServer.Close()

	t.Setenv("NOWPAYMENTS_API_KEY", "test-intent-key")
	t.Setenv("NOWPAYMENTS_BASE_URL", npServer.URL+"/v1")
	t.Setenv("NOWPAYMENTS_SANDBOX", "true")

	app := newTestApp()

	adminCookies := adminSession(t, app, "NP Admin", "np-admin@example.com")

	resp := doRequest(t, app, "POST", "/api/admin/gateway/projects",
		`{"slug":"np-shop","name":"NP Shop","webhook_url":"https://np-shop.example/hook","default_gateway":"nowpayments"}`, adminCookies)
	require.Equal(t, 201, resp.StatusCode)
	apiKey := decodeBody(t, resp)["project"].(map[string]interface{})["api_key"].(string)
	require.NotEmpty(t, apiKey)

	// Minor-only amounts are rejected for NOWPayments (minor/major ambiguity).
	resp = doGatewayRequest(t, app, "POST", "/api/gateway/intents",
		`{"external_order_id":"np_minor_only","gateway":"nowpayments","amount_idr":50000}`, map[string]string{"X-Api-Key": apiKey, "Idempotency-Key": "np-minor-1"}, nil)
	assert.Equal(t, 400, resp.StatusCode)
	resp.Body.Close()

	// Decimal-priced intents create a hosted invoice.
	resp = doGatewayRequest(t, app, "POST", "/api/gateway/intents",
		`{"external_order_id":"np_usd_1","gateway":"nowpayments","amount_decimal":"25.50","currency":"USD"}`, map[string]string{"X-Api-Key": apiKey, "Idempotency-Key": "np-usd-1"}, nil)
	require.Equal(t, 201, resp.StatusCode)
	intent := decodeBody(t, resp)
	assert.Equal(t, "nowpayments", intent["gateway"])
	assert.Equal(t, "https://nowpayments.io/payment/?iid=np1", intent["payment_url"])
	resp.Body.Close()
}

func TestLocalInvoiceWebhookSettlementIsAtomicAndIdempotent(t *testing.T) {
	t.Setenv("MIDTRANS_SERVER_KEY", "local-settlement-server-key")
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Settlement User","email":"settlement@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	resp.Body.Close()
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"settlement@example.com","password":"secret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	login := decodeBody(t, resp)
	userID := uuid.MustParse(login["user"].(map[string]interface{})["id"].(string))
	cookies := resp.Cookies()
	spec := newInvoice()
	spec.ClientID = createClient(t, app, cookies, "Settlement Payer")
	spec.Items = []invoiceLine{{Description: "Settlement service", Quantity: 1, Rate: "100000"}}
	invoiceID := createInvoiceID(t, app, cookies, spec)

	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	invoiceUUID := uuid.MustParse(invoiceID)
	orderID := "INV-test-settlement-001"
	require.NoError(t, db.CreateTransaction(&models.GatewayTransaction{
		OrderID: orderID, ProjectSlug: "local", Gateway: "midtrans",
		InvoiceID: &invoiceUUID, UserID: &userID, AmountIDR: 100000,
		Currency: "IDR", Status: models.GatewayStatusPending,
	}))
	trigger := `CREATE TRIGGER fail_gateway_payment BEFORE INSERT ON payments
		WHEN NEW.gateway_order_id IS NOT NULL BEGIN SELECT RAISE(ABORT, 'injected settlement failure'); END;`
	require.NoError(t, db.InvoiceQueries.Exec(trigger).Error)
	defer db.InvoiceQueries.Exec("DROP TRIGGER IF EXISTS fail_gateway_payment")

	gross := "100000.00"
	sum := sha512.Sum512([]byte(orderID + "200" + gross + "local-settlement-server-key"))
	notif, err := json.Marshal(map[string]string{
		"transaction_id":     "local-mid-txn",
		"order_id":           orderID,
		"transaction_status": "settlement",
		"payment_type":       "qris",
		"status_code":        "200",
		"gross_amount":       gross,
		"signature_key":      hex.EncodeToString(sum[:]),
	})
	require.NoError(t, err)

	// The payment insert failure must roll back the success status, so the
	// provider's retry can safely apply both records together.
	resp = doGatewayRequest(t, app, "POST", "/api/webhooks/midtrans", string(notif), nil, nil)
	assert.Equal(t, 500, resp.StatusCode)
	resp.Body.Close()
	txn, err := db.GetTransaction(orderID)
	require.NoError(t, err)
	assert.Equal(t, models.GatewayStatusPending, txn.Status)
	paid, err := db.PaidAmount(uuid.MustParse(invoiceID))
	require.NoError(t, err)
	assert.True(t, paid.IsZero())

	require.NoError(t, db.InvoiceQueries.Exec("DROP TRIGGER fail_gateway_payment").Error)
	resp = doGatewayRequest(t, app, "POST", "/api/webhooks/midtrans", string(notif), nil, nil)
	assert.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()

	// Replaying a successful notification must not insert another payment.
	resp = doGatewayRequest(t, app, "POST", "/api/webhooks/midtrans", string(notif), nil, nil)
	assert.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()

	txn, err = db.GetTransaction(orderID)
	require.NoError(t, err)
	assert.Equal(t, models.GatewayStatusSuccess, txn.Status)
	paid, err = db.PaidAmount(uuid.MustParse(invoiceID))
	require.NoError(t, err)
	assert.True(t, paid.Equal(decimal.NewFromInt(100000)))
	var paymentCount int64
	require.NoError(t, db.InvoiceQueries.Model(&models.Payment{}).
		Where("gateway_order_id = ?", orderID).Count(&paymentCount).Error)
	assert.EqualValues(t, 1, paymentCount)

	// Gateway-settled payments are not voidable from the UI, and the API
	// rejects the attempt even if a client sends it directly.
	resp = doRequest(t, app, "GET", "/api/payments", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	listed := decodeBody(t, resp)["payments"].([]interface{})
	require.Len(t, listed, 1)
	assert.Equal(t, false, listed[0].(map[string]interface{})["can_void"])
	resp.Body.Close()

	gatewayPaymentID := listed[0].(map[string]interface{})["id"].(string)
	resp = doRequest(t, app, "DELETE", "/api/payments/"+gatewayPaymentID+"?reason=wrong", "", cookies)
	assert.Equal(t, 422, resp.StatusCode)
	resp.Body.Close()
}
