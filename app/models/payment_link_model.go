package models

import (
	"time"

	"github.com/google/uuid"
)

// PaymentLink exposes one invoice through a public, unguessable token.
type PaymentLink struct {
	Token     string    `gorm:"primaryKey" db:"token" json:"token"`
	InvoiceID uuid.UUID `gorm:"type:uuid;uniqueIndex" db:"invoice_id" json:"invoice_id"`
	UserID    uuid.UUID `gorm:"type:uuid;index" db:"user_id" json:"user_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
