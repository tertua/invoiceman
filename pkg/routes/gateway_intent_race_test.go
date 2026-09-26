package routes

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/invoiceman/platform/database"
)

// relayRaceFixture boots an app with a stubbed Snap endpoint and a service
// project, returning its API key. Snap hits are counted so a test can tell
// one provider charge from two.
func relayRaceFixture(t *testing.T, app *fiber.App, slug, email string, delay time.Duration, calls *atomic.Int32) string {
	t.Helper()
	snap := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if delay > 0 {
			time.Sleep(delay)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"token":"snap-race-%d","redirect_url":"https://snap.test/race/%d"}`, n, n)
	}))
	t.Cleanup(snap.Close)
	t.Setenv("MIDTRANS_SERVER_KEY", "relay-race-key-"+slug)
	t.Setenv("MIDTRANS_SNAP_BASE_URL", snap.URL)

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Race Admin","email":"`+email+`","password":"secret123"}`, nil)
	require.True(t, resp.StatusCode == 201 || resp.StatusCode == 409)
	resp.Body.Close()
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"`+email+`","password":"secret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	adminID := uuid.MustParse(decodeBody(t, resp)["user"].(map[string]interface{})["id"].(string))
	resp.Body.Close()
	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	require.NoError(t, db.UpdateUserRole(adminID, "admin"))
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"`+email+`","password":"secret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	adminCookies := resp.Cookies()
	resp.Body.Close()

	resp = doRequest(t, app, "POST", "/api/admin/gateway/projects",
		`{"slug":"`+slug+`","name":"Race Shop","webhook_url":"https://race.example/hook"}`, adminCookies)
	require.Equal(t, 201, resp.StatusCode)
	apiKey := decodeBody(t, resp)["project"].(map[string]interface{})["api_key"].(string)
	require.NotEmpty(t, apiKey)
	resp.Body.Close()
	return apiKey
}

// Without an Idempotency-Key the relay refuses to charge: exact retries must
// be replayable, and the key is what makes two submits the same intent.
func TestRelayIntentRequiresIdempotencyKey(t *testing.T) {
	calls := &atomic.Int32{}
	app := newTestApp()
	apiKey := relayRaceFixture(t, app, "race-req", "race-req@example.com", 0, calls)

	resp := doGatewayRequest(t, app, "POST", "/api/gateway/intents",
		`{"external_order_id":"no_key","amount_idr":32000}`, map[string]string{"X-Api-Key": apiKey}, nil)
	require.Equal(t, 400, resp.StatusCode)
	body := decodeBody(t, resp)
	assert.Contains(t, body["error"].(map[string]interface{})["message"], "Idempotency-Key")
	resp.Body.Close()
	assert.EqualValues(t, 0, calls.Load(), "a keyless submit must never reach the gateway")
}

// Two sequential submits with the same key charge once; the second replays.
func TestRelayIntentSameKeyReplaysSingleCharge(t *testing.T) {
	calls := &atomic.Int32{}
	app := newTestApp()
	apiKey := relayRaceFixture(t, app, "race-replay", "race-replay@example.com", 0, calls)

	headers := map[string]string{"X-Api-Key": apiKey, "Idempotency-Key": "relay-replay-1"}
	resp := doGatewayRequest(t, app, "POST", "/api/gateway/intents",
		`{"external_order_id":"replay_1","amount_idr":32000}`, headers, nil)
	require.Equal(t, 201, resp.StatusCode)
	first := decodeBody(t, resp)
	resp.Body.Close()

	resp = doGatewayRequest(t, app, "POST", "/api/gateway/intents",
		`{"external_order_id":"replay_1","amount_idr":32000}`, headers, nil)
	status, second, replayHeaders := readBody(t, resp)
	require.Equal(t, 201, status)
	assert.Equal(t, "true", replayHeaders.Get("Idempotent-Replayed"))
	assert.Equal(t, first["order_id"], second["order_id"])
	assert.EqualValues(t, 1, calls.Load(), "replay must not recharge the gateway")
}

// Two concurrent submits with different keys for the same external id must
// still collapse onto one provider charge: the loser joins the winner's
// claim instead of opening a second payable intent.
func TestRelayIntentConcurrentCollapse(t *testing.T) {
	calls := &atomic.Int32{}
	app := newTestApp()
	apiKey := relayRaceFixture(t, app, "race-collapse", "race-collapse@example.com", 150*time.Millisecond, calls)

	type outcome struct {
		status int
		body   map[string]interface{}
	}
	results := make([]outcome, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			headers := map[string]string{"X-Api-Key": apiKey, "Idempotency-Key": fmt.Sprintf("relay-collapse-%d", i)}
			resp := doGatewayRequest(t, app, "POST", "/api/gateway/intents",
				`{"external_order_id":"collapse_1","amount_idr":32000}`, headers, nil)
			results[i] = outcome{status: resp.StatusCode, body: decodeBody(t, resp)}
		}(i)
	}
	wg.Wait()

	for _, r := range results {
		require.Contains(t, []int{200, 201}, r.status, "concurrent submit must not fail")
	}
	assert.Equal(t, results[0].body["order_id"], results[1].body["order_id"], "both submits collapse onto one intent")
	assert.EqualValues(t, 1, calls.Load(), "concurrent submits must open a single provider charge")
}
