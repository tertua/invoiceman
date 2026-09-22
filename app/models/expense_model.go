package models

import (
	"time"

	"github.com/google/uuid"
)

// Expense is a business expense owned by a user.
type Expense struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" db:"id" json:"id" validate:"required,uuid"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   *time.Time `db:"updated_at" json:"updated_at"`
	UserID      uuid.UUID  `gorm:"type:uuid" db:"user_id" json:"user_id" validate:"required,uuid"`
	Vendor      string     `db:"vendor" json:"vendor" validate:"lte=255"`
	Category    string     `db:"category" json:"category" validate:"required,lte=100"`
	ExpenseDate time.Time  `db:"expense_date" json:"expense_date"`
	Amount      float64    `db:"amount" json:"amount" validate:"gte=0"`
	Currency    string     `db:"currency" json:"currency" validate:"required,lte=3"`
	Notes       string     `db:"notes" json:"notes"`
	// ReceiptURL is the storage key of the uploaded receipt (private,
	// served via the receipt proxy endpoint). Empty = no attachment.
	ReceiptURL string `db:"receipt_url" json:"receipt_url"`
}

// ExpenseInput describes create and update expense payloads.
type ExpenseInput struct {
	Vendor      string  `json:"vendor" validate:"lte=255"`
	Category    string  `json:"category" validate:"required,lte=100"`
	ExpenseDate string  `json:"expense_date" validate:"required"`
	Amount      float64 `json:"amount" validate:"gte=0"`
	Currency    string  `json:"currency" validate:"omitempty,lte=3"`
	Notes       string  `json:"notes"`
}
