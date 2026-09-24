package models

import "github.com/google/uuid"

// GatewayProjectOwner is embedded on GatewayProject to track which user
// owns the project and receives invoices created through it.
type GatewayProjectOwner struct {
	OwnerUserID *uuid.UUID `gorm:"type:uuid;index" db:"owner_user_id" json:"-"`
}
