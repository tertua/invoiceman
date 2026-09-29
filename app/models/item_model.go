package models

import (
	"time"

	"github.com/google/uuid"
)

// Item is a reusable catalog item.
type Item struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" db:"id" json:"id" validate:"required,uuid"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   *time.Time `db:"updated_at" json:"updated_at"`
	UserID      uuid.UUID  `gorm:"type:uuid" db:"user_id" json:"user_id" validate:"required,uuid"`
	OrgID       uuid.UUID  `gorm:"type:uuid;index" db:"org_id" json:"org_id"`
	Name        string     `db:"name" json:"name" validate:"required,lte=255"`
	Description string     `db:"description" json:"description" validate:"lte=1000"`
	Rate        Money      `gorm:"type:decimal(19,4)" db:"rate" json:"rate"`
	Unit        string     `db:"unit" json:"unit" validate:"lte=50"`
}

// ItemInput describes create and update item payloads.
type ItemInput struct {
	Name        string `json:"name" validate:"required,lte=255"`
	Description string `json:"description" validate:"lte=1000"`
	Rate        Money  `json:"rate"`
	Unit        string `json:"unit" validate:"lte=50"`
}
