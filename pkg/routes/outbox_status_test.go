package routes

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/platform/database"
)

// TestOutboxStatusEndpoint covers the admin outbox status endpoint: guards,
// queue shape, and the pending-age signal after an enqueue.
func TestOutboxStatusEndpoint(t *testing.T) {
	app := newTestApp()
	adminCookies := adminSession(t, app, "Outbox Admin", "outboxadmin@example.com")

	// Enqueue one mail so the pending bucket and age signal are non-empty.
	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	require.NoError(t, db.EnqueueMail(&models.MailOutbox{
		To: "ops@example.com", Subject: "status probe", Body: "probe",
	}))

	// No session -> 401.
	resp := doRequest(t, app, "GET", "/api/admin/outbox/status", "", nil)
	assert.Equal(t, 401, resp.StatusCode)
	resp.Body.Close()

	// Non-admin session -> 403 from the role guard.
	resp = doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Outbox User","email":"outboxuser@example.com","password":"secret123"}`, nil)
	require.True(t, resp.StatusCode == 201 || resp.StatusCode == 409)
	resp.Body.Close()
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"outboxuser@example.com","password":"secret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	userCookies := resp.Cookies()
	resp.Body.Close()
	resp = doRequest(t, app, "GET", "/api/admin/outbox/status", "", userCookies)
	assert.Equal(t, 403, resp.StatusCode)
	resp.Body.Close()

	// Admin sees all four queues with zero-filled statuses and age signals.
	resp = doRequest(t, app, "GET", "/api/admin/outbox/status", "", adminCookies)
	require.Equal(t, 200, resp.StatusCode)
	body := decodeBody(t, resp)
	resp.Body.Close()

	queues, ok := body["queues"].([]any)
	require.True(t, ok, "queues must be an array")
	require.Len(t, queues, 4)
	names := make([]string, 0, 4)
	for _, q := range queues {
		qm := q.(map[string]any)
		names = append(names, qm["queue"].(string))
	}
	assert.Equal(t, []string{"mail", "notification", "webhook", "gateway_pending"}, names)

	mail := queues[0].(map[string]any)
	statuses := mail["statuses"].(map[string]any)
	assert.GreaterOrEqual(t, statuses["pending"].(float64), float64(1))
	require.NotNil(t, mail["oldest_pending_seconds"], "pending age signal missing")
	assert.GreaterOrEqual(t, mail["oldest_pending_seconds"].(float64), float64(0))
	assert.Nil(t, mail["oldest_processing_seconds"])

	assert.Greater(t, body["poll_seconds"].(float64), float64(0))
	assert.Greater(t, body["batch_size"].(float64), float64(0))
}

// TestOutboxMetricsExposed verifies the scrape endpoint exposes outbox gauges.
func TestOutboxMetricsExposed(t *testing.T) {
	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	require.NoError(t, db.EnqueueMail(&models.MailOutbox{
		To: "scrape@example.com", Subject: "metrics probe", Body: "probe",
	}))

	app := fiber.New()
	MetricsRoutes(app)

	req := httptest.NewRequest("GET", "/metrics", http.NoBody)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	out := string(body)
	assert.Contains(t, out, `tupay_outbox_queue_depth{queue="mail"`)
	assert.Contains(t, out, `tupay_outbox_oldest_seconds{queue="mail"`)
}
