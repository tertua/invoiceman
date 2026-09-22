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
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/platform/database"
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
	SendMail = func(to, subject, body string) error {
		mu.Lock()
		gotTo, gotSubject = to, subject
		mu.Unlock()
		return nil
	}
	defer func() { SendMail = old }()

	db := testDB(t)
	require.NoError(t, db.EnqueueMail(&models.MailOutbox{
		To: "user@example.com", Subject: "hi", Body: "hello",
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
	SendMail = func(to, subject, body string) error { return errors.New("smtp down") }
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
