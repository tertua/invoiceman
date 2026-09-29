package models

import (
	"time"

	"github.com/google/uuid"
)

// OrgInviteDefaultTTL is the validity window of a freshly minted invite token.
const OrgInviteDefaultTTL = 7 * 24 * time.Hour

// OrgInvite is a single-use token granting join access to an org with Role (design §Invite flow).
type OrgInvite struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey" db:"id" json:"id"`
	OrgID      uuid.UUID  `gorm:"type:uuid;index" db:"org_id" json:"org_id"`
	Token      string     `gorm:"size:64;uniqueIndex" db:"token" json:"token"`
	Role       string     `gorm:"size:16;default:staff" db:"role" json:"role"`
	Email      string     `gorm:"size:255" db:"email" json:"email,omitempty"`
	ExpiresAt  time.Time  `gorm:"index" db:"expires_at" json:"expires_at"`
	AcceptedBy *uuid.UUID `gorm:"type:uuid" db:"accepted_by" json:"accepted_by,omitempty"`
	RevokedAt  *time.Time `gorm:"index" db:"revoked_at" json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
}

// TableName keeps the plural convention used by other models.
func (OrgInvite) TableName() string { return "org_invites" }
