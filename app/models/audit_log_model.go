package models

import (
	"time"

	"github.com/google/uuid"
)

// AuditLog records one sensitive user action for accountability.
// Written explicitly by controllers (role changes, deletes, payments,
// settings, secret rotations, password changes); listed by admins.
type AuditLog struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" db:"id" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;index" db:"user_id" json:"user_id"`
	Action    string    `gorm:"size:64;index" db:"action" json:"action"`
	Entity    string    `gorm:"size:64;index" db:"entity" json:"entity"`
	EntityID  string    `gorm:"size:128;index" db:"entity_id" json:"entity_id"`
	Meta      string    `gorm:"type:text" db:"meta" json:"-"`
	IP        string    `gorm:"size:64" db:"ip" json:"ip"`
	CreatedAt time.Time `gorm:"index" db:"created_at" json:"created_at"`
}

// TableName keeps the plural convention used by other models.
func (AuditLog) TableName() string { return "audit_logs" }
