package models

import (
	"time"

	"github.com/google/uuid"
)

// AdminClaim is the singleton first-admin bootstrap row (ID is always 1).
// The first register to insert it wins the admin role; the primary key
// makes the claim atomic across concurrent registers on both SQLite and
// PostgreSQL, closing the CountUsers()+CreateUser race.
type AdminClaim struct {
	ID        int       `gorm:"primaryKey" db:"id" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid" db:"user_id" json:"user_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// TableName keeps the plural convention used by other models.
func (AdminClaim) TableName() string { return "admin_claims" }

// MigrateDownInput describes an admin schema-rollback request. Rollbacks
// destroy schema (never data rows, but dropped columns lose their values),
// so the caller must opt in twice: an in-range target plus confirm=true.
type MigrateDownInput struct {
	TargetVersion int  `json:"target_version" validate:"required,min=1"`
	Confirm       bool `json:"confirm" validate:"eq=true"`
}
