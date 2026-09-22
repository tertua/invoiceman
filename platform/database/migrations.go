package database

import (
	"fmt"
	"time"

	"github.com/tertua/invoiceman/app/models"
	"gorm.io/gorm"
)

// Migration describes one reversible schema step. Ups run implicitly through
// AutoMigrate at startup; only the Down rollback is recorded here, keyed by
// the SchemaVersion that introduced the change. Version 1 is the base
// schema and has no Down: rolling back to 0 is refused.
//
// Rules for new versions (see AGENTS.md compatibility contract):
//   - Down must be backend-agnostic: GORM Migrator calls only, no raw SQL,
//     working on both SQLite and PostgreSQL.
//   - Down must only drop what its version added; shared tables/columns stay.
type Migration struct {
	Version     int
	Description string
	Down        func(db *gorm.DB) error
}

// migrations lists every rollback known to this binary, oldest first.
var migrations = []Migration{
	{
		Version:     2,
		Description: "expense receipt_url + mail_outbox html_body",
		Down: func(db *gorm.DB) error {
			if err := db.Migrator().DropColumn(&models.Expense{}, "receipt_url"); err != nil {
				return err
			}
			return db.Migrator().DropColumn(&models.MailOutbox{}, "html_body")
		},
	},
	{
		Version:     3,
		Description: "admin_claims singleton for atomic first-admin bootstrap",
		Down: func(db *gorm.DB) error {
			return db.Migrator().DropTable(&models.AdminClaim{})
		},
	},
}

// CurrentSchemaVersion reports the version stamp stored in the database.
func CurrentSchemaVersion() (int, error) {
	db, err := openShared()
	if err != nil {
		return 0, err
	}
	row := models.SchemaMigration{}
	if err := db.Where("id = ?", 1).First(&row).Error; err != nil {
		return 0, err
	}
	return row.Version, nil
}

// MigrateDownTo rolls the schema back to target for an emergency rollback
// (run via the admin endpoint, then deploy the older binary). Steps run
// newest-first inside one transaction per step; the version stamp is
// updated only after all steps succeed. Forward upgrades re-apply
// automatically on the next startup via AutoMigrate + checkSchemaVersion,
// so a rollback followed by a restart with this binary is a no-op round
// trip. Refuses: target below 1, target at/above the stored version, and
// databases newer than this binary (their downs are unknown here).
func MigrateDownTo(target int) (int, error) {
	db, err := openShared()
	if err != nil {
		return 0, err
	}
	row := models.SchemaMigration{}
	if err := db.Where("id = ?", 1).First(&row).Error; err != nil {
		return 0, fmt.Errorf("schema version row missing: %w", err)
	}
	if row.Version > SchemaVersion {
		return 0, fmt.Errorf(
			"database schema version %d is newer than this binary (version %d): rollback unknown",
			row.Version, SchemaVersion)
	}
	if target < 1 || target >= row.Version {
		return 0, fmt.Errorf(
			"cannot roll back from version %d to version %d: target must be between 1 and %d",
			row.Version, target, row.Version-1)
	}
	byVersion := map[int]Migration{}
	for _, m := range migrations {
		byVersion[m.Version] = m
	}
	for v := row.Version; v > target; v-- {
		m, ok := byVersion[v]
		if !ok {
			return 0, fmt.Errorf("no rollback registered for schema version %d", v)
		}
		if err := m.Down(db); err != nil {
			return 0, fmt.Errorf("rollback of schema version %d (%s) failed: %w", v, m.Description, err)
		}
	}
	row.Version = target
	row.AppVersion = appVersion()
	row.AppliedAt = time.Now()
	if err := db.Save(&row).Error; err != nil {
		return 0, err
	}
	return target, nil
}
