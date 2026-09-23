package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Client describes a client.
type Client struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey" db:"id" json:"id" validate:"required,uuid"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt *time.Time `db:"updated_at" json:"updated_at"`
	UserID    uuid.UUID  `gorm:"type:uuid" db:"user_id" json:"user_id" validate:"required,uuid"`
	Name      string     `db:"name" json:"name" validate:"required,lte=255"`
	Email     string     `db:"email" json:"email" validate:"omitempty,email,lte=255"`
	Company   string     `db:"company" json:"company" validate:"lte=255"`
	Phone     string     `db:"phone" json:"phone" validate:"lte=100"`
	Address   string     `db:"address" json:"address"`
	Notes     string     `db:"notes" json:"notes"`
}

// ClientInput struct to describe create/update client payload.
type ClientInput struct {
	Name    string `json:"name" validate:"required,lte=255"`
	Email   string `json:"email" validate:"omitempty,email,lte=255"`
	Company string `json:"company" validate:"lte=255"`
	Phone   string `json:"phone" validate:"lte=100"`
	Address string `json:"address"`
	Notes   string `json:"notes"`
}

// ClientListRow struct to describe a client row with billing aggregates.
type ClientListRow struct {
	Client
	TotalBilled decimal.Decimal `db:"total_billed" json:"total_billed"`
	Outstanding decimal.Decimal `db:"outstanding" json:"outstanding"`
}

// ClientStats struct to describe client detail statistics.
type ClientStats struct {
	Count       int             `json:"count"`
	TotalBilled decimal.Decimal `json:"totalBilled"`
	Outstanding decimal.Decimal `json:"outstanding"`
}
