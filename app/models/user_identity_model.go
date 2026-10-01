package models

import (
	"time"

	"github.com/google/uuid"
)

// IdentityProviderOIDC is the provider name stored for OIDC SSO identities.
const IdentityProviderOIDC = "oidc"

// UserIdentity links one external identity (provider + subject) to a local
// user. A user may have several identities (one row per provider), and the
// (provider, sub) pair is unique so a subject maps to exactly one account.
type UserIdentity struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" db:"id" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;index" db:"user_id" json:"user_id"`
	Provider  string    `gorm:"type:varchar(32);uniqueIndex:idx_identity_provider_sub" db:"provider" json:"provider"`
	Sub       string    `gorm:"type:varchar(255);uniqueIndex:idx_identity_provider_sub" db:"sub" json:"sub"`
	Email     string    `gorm:"type:varchar(255)" db:"email" json:"email"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// TableName keeps the plural table convention used by the other models.
func (UserIdentity) TableName() string { return "user_identities" }
