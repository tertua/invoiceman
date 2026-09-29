package models

import (
	"time"

	"github.com/google/uuid"
)

// Organization describes a tenant organization that owns users, projects, and billing.
type Organization struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey" db:"id" json:"id" validate:"required,uuid"`
	Name      string     `gorm:"size:255" db:"name" json:"name" validate:"required,lte=255"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt *time.Time `db:"updated_at" json:"updated_at"`
}

// TableName keeps the plural convention used by other models.
func (Organization) TableName() string { return "organizations" }
