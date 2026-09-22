package models

import (
	"time"

	"github.com/google/uuid"
)

// IdempotencyKey stores one executed mutating request so client retries
// (double-click, timeout) replay the stored response instead of executing
// twice. Rows expire after IdempotencyTTL and are purged by the worker.
type IdempotencyKey struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	KeyHash      string    `gorm:"size:64;uniqueIndex:idx_idem_scope_key;not null" json:"-"`
	Scope        string    `gorm:"size:128;uniqueIndex:idx_idem_scope_key;not null" json:"-"`
	Method       string    `gorm:"size:16" json:"-"`
	Path         string    `gorm:"size:512" json:"-"`
	RequestHash  string    `gorm:"size:64" json:"-"`
	StatusCode   int       `json:"-"`
	ResponseBody string    `gorm:"type:text" json:"-"`
	ExpiresAt    time.Time `gorm:"index" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// TableName keeps the plural convention used by other models.
func (IdempotencyKey) TableName() string { return "idempotency_keys" }
