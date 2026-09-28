package outbox

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/platform/database"
)

// TestMain runs the worker tests against in-memory SQLite.
func TestMain(m *testing.M) {
	os.Setenv("SQL_DSN", "")
	os.Setenv("SQLITE_PATH", "file::memory:?cache=shared")
	os.Setenv("REDIS_HOST", "")
	if err := database.Migrate(); err != nil {
		panic("test migrate failed: " + err.Error())
	}
	os.Exit(m.Run())
}

func testDB(t *testing.T) *database.Queries {
	t.Helper()
	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	return db
}

// TestBackoff covers the retry schedule and exhaustion.
func TestBackoff(t *testing.T) {
	assert.Equal(t, time.Minute, Backoff(1))
	assert.Equal(t, 2*time.Minute, Backoff(2))
	assert.Equal(t, 2*time.Hour, Backoff(20))
	assert.Nil(t, nextRetryAt(10, time.Now()))
	require.NotNil(t, nextRetryAt(9, time.Now()))
}

// TestMailSent verifies a queued email is delivered and marked sent.
func TestMailSent(t *testing.T) {
	var mu sync.Mutex
	var gotTo, gotSubject string
	old := SendMail
	SendMail = func(to, subject, textBody, htmlBody string) error {
		mu.Lock()
		gotTo, gotSubject = to, subject
		mu.Unlock()
		assert.NotEmpty(t, htmlBody, "queued mail carries the HTML part")
		return nil
	}
	defer func() { SendMail = old }()

	db := testDB(t)
	require.NoError(t, db.EnqueueMail(&models.MailOutbox{
		To: "user@example.com", Subject: "hi", Body: "hello", HtmlBody: "<p>hello</p>",
	}))

	New().ProcessOnce(context.Background())

	mu.Lock()
	assert.Equal(t, "user@example.com", gotTo)
	assert.Equal(t, "hi", gotSubject)
	mu.Unlock()

	rows, err := db.DueMail(time.Now(), 10)
	require.NoError(t, err)
	assert.Empty(t, rows, "sent mail must leave the due queue")
}

// TestMailFailureRetries verifies failures reschedule with backoff and
// eventually park as dead.
func TestMailFailureRetries(t *testing.T) {
	old := SendMail
	SendMail = func(to, subject, textBody, htmlBody string) error { return errors.New("smtp down") }
	defer func() { SendMail = old }()

	db := testDB(t)
	m := &models.MailOutbox{To: "fail@example.com", Subject: "x", Body: "y"}
	require.NoError(t, db.EnqueueMail(m))

	New().ProcessOnce(context.Background())

	due, err := db.DueMail(time.Now().Add(time.Hour), 10)
	require.NoError(t, err)
	require.Len(t, due, 1)
	assert.Equal(t, models.MailStatusFailed, due[0].Status)
	assert.Equal(t, 1, due[0].Attempt)
	require.NotNil(t, due[0].NextRetryAt)

	// Exhausted attempts go dead instead of retrying forever.
	require.NoError(t, db.MarkMailFailed(due[0].ID, 10, nil, time.Now()))
	rows, err := db.DueMail(time.Now().Add(48*time.Hour), 50)
	require.NoError(t, err)
	for _, r := range rows {
		assert.NotEqual(t, due[0].ID, r.ID, "dead mail must never re-enter the queue")
	}
}

// TestDeliveryForwarded verifies a pending delivery reaches downstream
// and is marked delivered.
func TestDeliveryForwarded(t *testing.T) {
	var mu sync.Mutex
	count := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		mu.Lock()
		count++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	db := testDB(t)
	slug := "outbox-test-" + uuid.NewString()[:8]
	require.NoError(t, db.CreateProject(&models.GatewayProject{
		Slug: slug, Name: "Outbox Test", APIKeyHash: "hash-" + slug,
		WebhookURL:    server.URL,
		WebhookSecret: "whsec_test", IsActive: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, db.CreateDelivery(&models.WebhookDelivery{
		ID: uuid.New(), OrderID: "EXT-test-1", ProjectSlug: slug,
		Gateway: "midtrans", TargetURL: server.URL,
		Payload:   `{"event_id":"evt_1"}`,
		Status:    "pending",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	New().ProcessOnce(context.Background())

	mu.Lock()
	assert.Equal(t, 1, count)
	mu.Unlock()

	deliveries, err := db.ListDeliveriesByOrder("EXT-test-1")
	require.NoError(t, err)
	require.Len(t, deliveries, 1)
	assert.Equal(t, "delivered", deliveries[0].Status)
	assert.Equal(t, 200, deliveries[0].RespCode)
}

// TestDeliveryFailureRetries verifies an unreachable downstream reschedules.
func TestDeliveryFailureRetries(t *testing.T) {
	db := testDB(t)
	slug := "outbox-fail-" + uuid.NewString()[:8]
	require.NoError(t, db.CreateProject(&models.GatewayProject{
		Slug: slug, Name: "Outbox Fail", APIKeyHash: "hash-" + slug,
		WebhookURL:    "http://127.0.0.1:1/",
		WebhookSecret: "whsec_test", IsActive: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, db.CreateDelivery(&models.WebhookDelivery{
		ID: uuid.New(), OrderID: "EXT-fail-1", ProjectSlug: slug,
		Gateway: "midtrans", TargetURL: "http://127.0.0.1:1/",
		Payload:   `{"event_id":"evt_2"}`,
		Status:    "pending",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	New().ProcessOnce(context.Background())

	deliveries, err := db.ListDeliveriesByOrder("EXT-fail-1")
	require.NoError(t, err)
	require.Len(t, deliveries, 1)
	assert.Equal(t, "failed", deliveries[0].Status)
	assert.Equal(t, 1, deliveries[0].Attempt)
	require.NotNil(t, deliveries[0].NextRetryAt)
}

// TestNotificationForwarded verifies a pending notification reaches the
// endpoint with its secret and is marked delivered.
func TestNotificationForwarded(t *testing.T) {
	var mu sync.Mutex
	var gotSecret string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		mu.Lock()
		gotSecret = r.Header.Get("X-Relay-Signature")
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	db := testDB(t)
	uid := uuid.New()
	endpoint := &models.NotificationEndpoint{
		UserID: uid, TargetURL: server.URL,
		Secret: "whsec_notif", IsActive: true,
	}
	require.NoError(t, db.CreateEndpoint(endpoint))
	require.NoError(t, db.EnqueueDelivery(&models.NotificationDelivery{
		UserID: uid, EndpointID: endpoint.ID,
		EventID:   "evt_notif_" + uuid.NewString()[:8],
		EventType: models.NotifEventInvoiceCreated,
		TargetURL: server.URL,
		Payload:   `{"type":"invoice.created"}`,
	}))

	New().ProcessOnce(context.Background())

	mu.Lock()
	assert.NotEmpty(t, gotSecret, "forward must sign with the endpoint secret")
	mu.Unlock()

	rows, err := db.DueNotifications(time.Now(), 10)
	require.NoError(t, err)
	assert.Empty(t, rows, "delivered notifications must leave the due queue")
}

// TestNotificationFailureRetries verifies an unreachable endpoint reschedules.
func TestNotificationFailureRetries(t *testing.T) {
	db := testDB(t)
	uid := uuid.New()
	endpoint := &models.NotificationEndpoint{
		UserID: uid, TargetURL: "http://127.0.0.1:1/",
		Secret: "whsec_notif", IsActive: true,
	}
	require.NoError(t, db.CreateEndpoint(endpoint))
	d := &models.NotificationDelivery{
		UserID: uid, EndpointID: endpoint.ID,
		EventID:   "evt_notif_fail_" + uuid.NewString()[:8],
		EventType: models.NotifEventPaymentCreated,
		TargetURL: "http://127.0.0.1:1/",
		Payload:   `{"type":"payment.created"}`,
	}
	require.NoError(t, db.EnqueueDelivery(d))

	New().ProcessOnce(context.Background())

	due, err := db.DueNotifications(time.Now().Add(time.Hour), 10)
	require.NoError(t, err)
	found := false
	for _, r := range due {
		if r.ID == d.ID {
			found = true
			assert.Equal(t, models.NotifStatusFailed, r.Status)
			assert.Equal(t, 1, r.Attempt)
			require.NotNil(t, r.NextRetryAt)
		}
	}
	assert.True(t, found, "failed notification must reschedule")
}

// TestPurgeIdempotency verifies expired keys are cleaned on each tick.
func TestPurgeIdempotency(t *testing.T) {
	db := testDB(t)
	past := time.Now().Add(-time.Hour)
	require.NoError(t, db.CreateIdempotencyPlaceholder(&models.IdempotencyKey{
		KeyHash: "deadbeef", Scope: "user:test", Method: "POST",
		Path: "/api/payments", RequestHash: "abc", StatusCode: 201,
		ResponseBody: `{"ok":true}`, ExpiresAt: past, CreatedAt: past,
	}))

	New().ProcessOnce(context.Background())

	n, err := db.DeleteExpiredIdempotencyKeys(time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(0), n, "purge must have removed the expired row")
}
