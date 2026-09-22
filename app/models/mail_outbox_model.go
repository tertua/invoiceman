package models

import (
	"time"

	"github.com/google/uuid"
)

// Mail outbox statuses. Rows are written by controllers and consumed by the
// background worker; "processing" is a short-lived claim state.
const (
	MailStatusPending    = "pending"
	MailStatusProcessing = "processing"
	MailStatusSent       = "sent"
	MailStatusFailed     = "failed"
	MailStatusDead       = "dead"
)

// MailOutbox is one email waiting for (re)delivery by the worker.
// Controllers enqueue and return fast; SMTP latency and retries never
// block API responses.
type MailOutbox struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" db:"id" json:"id"`
	To          string     `gorm:"size:255;index" db:"to" json:"-"`
	Subject     string     `gorm:"size:255" db:"subject" json:"-"`
	Body        string     `gorm:"type:text" db:"body" json:"-"`
	Status      string     `gorm:"size:16;index;default:pending" db:"status" json:"-"`
	Attempt     int        `db:"attempt" json:"-"`
	NextRetryAt *time.Time `gorm:"index" db:"next_retry_at" json:"-"`
	CreatedAt   time.Time  `db:"created_at" json:"-"`
	UpdatedAt   time.Time  `db:"updated_at" json:"-"`
}

// TableName keeps the plural convention used by other models.
func (MailOutbox) TableName() string { return "mail_outbox" }
