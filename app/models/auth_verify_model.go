package models

import (
	"time"

	"github.com/google/uuid"
)

// Account status values. A pending account may not log in until the email
// verification link is followed, which flips it to active.
const (
	UserStatusBlocked = 0
	UserStatusActive  = 1
	UserStatusPending = 2
)

// VerifyEmail struct to describe the email verification payload.
type VerifyEmail struct {
	Token string `json:"token" validate:"required,lte=255"`
}

// ResendVerification struct to describe the resend verification payload.
type ResendVerification struct {
	Email string `json:"email" validate:"required,email,lte=255"`
}

// EmailVerification struct to describe a stored email verification token.
type EmailVerification struct {
	Token     string    `gorm:"primaryKey" db:"token" json:"token"`
	UserID    uuid.UUID `gorm:"type:uuid" db:"user_id" json:"user_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
}
