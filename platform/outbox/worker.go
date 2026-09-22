package outbox

// Package outbox runs the in-process background worker (stdlib only).
//
// Controllers enqueue durable jobs and return fast; the worker delivers
// them with retries:
//   - mail_outbox rows -> SMTP via platform/mail (SendMail hook, testable)
//   - webhook_deliveries rows -> downstream forward via platform/relay
//   - expired idempotency_keys rows -> purged
//
// Claiming is atomic per row (single UPDATE guarded by status), so the
// worker is safe with several replicas polling the same database. Jobs are
// idempotent by design (resend = same content), making at-least-once
// delivery harmless.

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/configs"
	"github.com/tertua/invoiceman/pkg/logger"
	"github.com/tertua/invoiceman/platform/database"
	"github.com/tertua/invoiceman/platform/mail"
	"github.com/tertua/invoiceman/platform/relay"
)

// Retry policy: 1m, 2m, 4m, ... capped at 2h, dead after maxAttempts.
const (
	maxAttempts  = 10
	maxBackoff   = 2 * time.Hour
	drainTimeout = 5 * time.Second
	responseCap  = 4000
)

// Backoff returns the delay before attempt n (1-based).
func Backoff(n int) time.Duration {
	d := time.Minute << (n - 1)
	if d <= 0 || d > maxBackoff {
		return maxBackoff
	}
	return d
}

// nextRetryAt returns nil when attempts are exhausted (row goes dead).
func nextRetryAt(attempt int, now time.Time) *time.Time {
	if attempt >= maxAttempts {
		return nil
	}
	at := now.Add(Backoff(attempt))
	return &at
}

// SendMail delivers one email. It is a variable (not a direct mail call)
// so tests can capture sends without an SMTP server.
var SendMail = func(to, subject, body string) error {
	mailer, err := mail.NewFromEnv()
	if err != nil {
		return err
	}
	return mailer.Send(to, subject, body)
}

// Worker polls due jobs until its context is cancelled.
type Worker struct {
	poll  time.Duration
	batch int

	mu     sync.Mutex
	wg     sync.WaitGroup
	cancel context.CancelFunc
	done   chan struct{}
}

// New builds a worker from the central config.
func New() *Worker {
	cfg := configs.Get().Outbox
	return &Worker{poll: time.Duration(cfg.PollSeconds) * time.Second, batch: cfg.Batch}
}

// Start begins polling in the background. Stop cancels and drains.
func (w *Worker) Start(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	w.mu.Lock()
	w.cancel = cancel
	w.done = make(chan struct{})
	w.mu.Unlock()
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		defer close(w.done)
		ticker := time.NewTicker(w.poll)
		defer ticker.Stop()
		w.ProcessOnce(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.ProcessOnce(ctx)
			}
		}
	}()
}

// Stop cancels polling and waits for in-flight jobs up to drainTimeout.
func (w *Worker) Stop() {
	w.mu.Lock()
	cancel := w.cancel
	done := w.done
	w.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	select {
	case <-done:
	case <-time.After(drainTimeout):
	}
}

// ProcessOnce runs one tick: mail, deliveries, then idempotency purge.
// Exported for tests and admin-triggered drains.
func (w *Worker) ProcessOnce(ctx context.Context) {
	w.processMail(ctx)
	w.processDeliveries(ctx)
	w.purgeIdempotency()
}

func (w *Worker) processMail(ctx context.Context) {
	db, err := database.OpenDBConnection()
	if err != nil {
		logger.L().Warn("outbox mail tick skipped, database unavailable", "err", err)
		return
	}
	now := time.Now()
	rows, err := db.DueMail(now, w.batch)
	if err != nil {
		logger.L().Warn("outbox mail tick failed", "err", err)
		return
	}
	for _, m := range rows {
		select {
		case <-ctx.Done():
			return
		default:
		}
		w.sendOneMail(db, m, now)
	}
}

func (w *Worker) sendOneMail(db *database.Queries, m models.MailOutbox, now time.Time) {
	claimed, err := db.ClaimMail(m.ID, now)
	if err != nil || !claimed {
		return
	}
	if err := SendMail(m.To, m.Subject, m.Body); err != nil {
		if errors.Is(err, mail.ErrNotConfigured) {
			// No SMTP configured (dev): leave pending without burning
			// attempts; reset the claim so the next tick retries.
			_ = db.MarkMailFailed(m.ID, m.Attempt, &[]time.Time{now.Add(w.poll)}[0], now)
			logger.L().Debug("outbox mail skipped, provider not configured", "to", m.To)
			return
		}
		retry := nextRetryAt(m.Attempt+1, now)
		_ = db.MarkMailFailed(m.ID, m.Attempt+1, retry, now)
		logger.L().Warn("outbox mail failed", "to", m.To, "attempt", m.Attempt+1, "err", err)
		return
	}
	_ = db.MarkMailSent(m.ID, now)
	logger.L().Info("outbox mail sent", "to", m.To)
}

func (w *Worker) processDeliveries(ctx context.Context) {
	db, err := database.OpenDBConnection()
	if err != nil {
		logger.L().Warn("outbox delivery tick skipped, database unavailable", "err", err)
		return
	}
	now := time.Now()
	rows, err := db.PendingDeliveries(now, w.batch)
	if err != nil {
		logger.L().Warn("outbox delivery tick failed", "err", err)
		return
	}
	for _, d := range rows {
		select {
		case <-ctx.Done():
			return
		default:
		}
		w.forwardOne(db, d, now)
	}
}

func (w *Worker) forwardOne(db *database.Queries, d models.WebhookDelivery, now time.Time) {
	claimed, err := db.ClaimDelivery(d.ID, now)
	if err != nil || !claimed {
		return
	}
	project, err := db.GetProjectBySlug(d.ProjectSlug)
	if err != nil {
		retry := nextRetryAt(d.Attempt+1, now)
		failDelivery(db, d.ID, d.Attempt+1, retry, "project not found", now)
		return
	}
	payload := []byte(d.Payload)
	d.Signature = relay.SignPayload(payload, project.WebhookSecret)
	d.Attempt++
	d.UpdatedAt = now
	result, ferr := relay.Forward(context.Background(), d.TargetURL, project.Slug, "evt_retry_"+d.ID.String(), payload, project.WebhookSecret)
	if ferr != nil {
		retry := nextRetryAt(d.Attempt, now)
		failDelivery(db, d.ID, d.Attempt, retry, truncateErr(ferr.Error()), now)
		logger.L().Warn("outbox delivery failed", "order_id", d.OrderID, "attempt", d.Attempt, "err", ferr)
		return
	}
	d.RespCode = result.StatusCode
	d.RespBody = truncateBody(result.Body)
	if result.StatusCode >= 200 && result.StatusCode < 300 {
		d.Status = "delivered"
		d.NextRetryAt = nil
	} else {
		d.Status = "failed"
		d.NextRetryAt = nextRetryAt(d.Attempt, now)
		if d.NextRetryAt == nil {
			d.Status = "dead"
		}
	}
	_ = db.SaveDelivery(&d)
	if d.Status == "delivered" {
		logger.L().Info("outbox delivery sent", "order_id", d.OrderID, "attempt", d.Attempt)
	}
}

func failDelivery(db *database.Queries, id uuid.UUID, attempt int, retry *time.Time, msg string, now time.Time) {
	status := "failed"
	if retry == nil {
		status = "dead"
	}
	if err := db.FailDelivery(id, status, attempt, retry, msg, now); err != nil {
		logger.L().Warn("outbox failed to record delivery failure", "err", err)
	}
}

func (w *Worker) purgeIdempotency() {
	db, err := database.OpenDBConnection()
	if err != nil {
		return
	}
	if n, err := db.DeleteExpiredIdempotencyKeys(time.Now()); err == nil && n > 0 {
		logger.L().Info("outbox purged expired idempotency keys", "count", n)
	}
}

func truncateErr(s string) string {
	if len(s) > 1000 {
		return s[:1000]
	}
	return s
}

func truncateBody(s string) string {
	if len(s) > responseCap {
		return s[:responseCap]
	}
	return s
}
