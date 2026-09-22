package models

import (
	"time"
)

// SchemaMigration tracks the schema version of this database in a single
// row (id 1). The version is bumped in code (database.SchemaVersion) every
// time a model changes; startup refuses to run when the database is newer
// than the binary, so accidental rollbacks fail fast instead of corrupting
// data. Forward upgrades run through the usual AutoMigrate.
type SchemaMigration struct {
	ID         int       `gorm:"primaryKey" db:"id" json:"-"`
	Version    int       `db:"version" json:"version"`
	AppVersion string    `gorm:"size:32" db:"app_version" json:"app_version"`
	AppliedAt  time.Time `db:"applied_at" json:"applied_at"`
}

// TableName keeps the plural convention used by other models.
func (SchemaMigration) TableName() string { return "schema_migrations" }
