package queries

import (
	"time"

	"github.com/google/uuid"
	"github.com/tertua/invoiceman/app/models"
	"gorm.io/gorm"
)

// MailOutboxQueries persists outgoing emails for the background worker.
type MailOutboxQueries struct {
	*gorm.DB
}

// EnqueueMail stores one email for async delivery.
func (q *MailOutboxQueries) EnqueueMail(m *models.MailOutbox) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	now := time.Now()
	m.Status = models.MailStatusPending
	m.CreatedAt = now
	m.UpdatedAt = now
	return q.Create(m).Error
}

// DueMail returns pending/failed emails whose retry is due, oldest first.
func (q *MailOutboxQueries) DueMail(now time.Time, limit int) ([]models.MailOutbox, error) {
	out := []models.MailOutbox{}
	if err := q.Where("status IN ? AND (next_retry_at IS NULL OR next_retry_at <= ?)",
		[]string{models.MailStatusPending, models.MailStatusFailed}, now).
		Order("created_at ASC").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// ClaimMail atomically marks one due email as processing. Only one worker
// wins the claim; losers get claimed=false and skip the row.
func (q *MailOutboxQueries) ClaimMail(id uuid.UUID, now time.Time) (bool, error) {
	return DoRetryValue(func() (bool, error) {
		res := q.Model(&models.MailOutbox{}).
			Where("id = ? AND status IN ? AND (next_retry_at IS NULL OR next_retry_at <= ?)",
				id, []string{models.MailStatusPending, models.MailStatusFailed}, now).
			Updates(map[string]interface{}{"status": models.MailStatusProcessing, "updated_at": now})
		return res.RowsAffected > 0, res.Error
	})
}

// MarkMailSent records a successful delivery.
func (q *MailOutboxQueries) MarkMailSent(id uuid.UUID, now time.Time) error {
	return DoRetry(func() error {
		return q.Model(&models.MailOutbox{}).Where("id = ?", id).
			Updates(map[string]interface{}{
				"status": models.MailStatusSent, "updated_at": now, "next_retry_at": nil,
			}).Error
	})
}

// MarkMailFailed records a failed attempt with the next retry time,
// or parks the row as dead when attempts are exhausted.
func (q *MailOutboxQueries) MarkMailFailed(id uuid.UUID, attempt int, retryAt *time.Time, now time.Time) error {
	status := models.MailStatusFailed
	if retryAt == nil {
		status = models.MailStatusDead
	}
	return DoRetry(func() error {
		return q.Model(&models.MailOutbox{}).Where("id = ?", id).
			Updates(map[string]interface{}{
				"status": status, "attempt": attempt, "next_retry_at": retryAt, "updated_at": now,
			}).Error
	})
}
