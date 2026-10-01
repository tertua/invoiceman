package models

import (
	"time"

	"github.com/google/uuid"
)

// User struct to describe User object.
type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" db:"id" json:"id" validate:"required,uuid"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
	Name         string    `db:"name" json:"name" validate:"required,lte=255"`
	Email        string    `gorm:"size:255" db:"email" json:"email" validate:"required,email,lte=255"`
	PasswordHash string    `db:"password_hash" json:"password_hash,omitempty" validate:"required,lte=255"`
	UserStatus   int       `db:"user_status" json:"user_status" validate:"oneof=0 1 2"`
	UserRole     string    `db:"user_role" json:"user_role" validate:"required,lte=25"`
}
