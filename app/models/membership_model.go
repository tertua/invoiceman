package models

import (
	"time"

	"github.com/google/uuid"
)

// Membership roles within an organization.
const (
	RoleOwner = "owner"
	RoleStaff = "staff"
)

// Membership links a user to an organization with a role.
type Membership struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" db:"id" json:"id"`
	OrgID     uuid.UUID `gorm:"type:uuid;index;uniqueIndex:idx_membership_org_user" db:"org_id" json:"org_id"`
	UserID    uuid.UUID `gorm:"type:uuid;index;uniqueIndex:idx_membership_org_user" db:"user_id" json:"user_id"`
	Role      string    `gorm:"size:16" db:"role" json:"role"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// TableName keeps the plural convention used by other models.
func (Membership) TableName() string { return "memberships" }
