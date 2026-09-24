package models

import "github.com/google/uuid"

type GatewayOwner struct {
	OwnerUserID *uuid.UUID `gorm:"type:uuid;index" db:"owner_user_id" json:"-"`
}
