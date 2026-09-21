package models

import (
	"time"

	"github.com/google/uuid"
)

// Item is a reusable catalog item owned by a user.
type Item struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" db:"id" json:"id" validate:"required,uuid"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   *time.Time `db:"updated_at" json:"updated_at"`
	UserID      uuid.UUID  `gorm:"type:uuid" db:"user_id" json:"user_id" validate:"required,uuid"`
	Name        string     `db:"name" json:"name" validate:"required,lte=255"`
	Description string     `db:"description" json:"description" validate:"lte=1000"`
	Rate        float64    `db:"rate" json:"rate" validate:"gte=0"`
	Unit        string     `db:"unit" json:"unit" validate:"lte=50"`
}

// ItemInput describes create and update item payloads.
type ItemInput struct {
	Name        string  `json:"name" validate:"required,lte=255"`
	Description string  `json:"description" validate:"lte=1000"`
	Rate        float64 `json:"rate" validate:"gte=0"`
	Unit        string  `json:"unit" validate:"lte=50"`
}
