package database

import (
	"path/filepath"
	"testing"

	"github.com/tertua/tupay/app/models"
	"gorm.io/gorm"
)

// v20Down runs the registered v20 rollback step against the probe handle.
func v20Down(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, m := range migrations {
		if m.Version != 20 {
			continue
		}
		if err := m.Down(db); err != nil {
			t.Fatalf("expected v20 rollback to succeed, got: %v", err)
		}
		return
	}
	t.Fatal("v20 rollback step missing from the migration registry")
}

// TestOIDCMigrationUpDownRoundTrip pins the v20 schema step: the
// user_identities table appears on up, disappears on rollback, and comes
// back when the migration is re-applied (AutoMigrate).
func TestOIDCMigrationUpDownRoundTrip(t *testing.T) {
	t.Setenv("SQL_DSN", "")
	t.Setenv("SQLITE_PATH", filepath.Join(t.TempDir(), "oidc-probe.db"))
	db, err := chooseDB("")
	if err != nil {
		t.Fatalf("expected probe database, got: %v", err)
	}

	// Up: the model table exists with its columns.
	if err := db.AutoMigrate(&models.UserIdentity{}); err != nil {
		t.Fatalf("expected v20 up to create user_identities, got: %v", err)
	}
	if got := (models.UserIdentity{}).TableName(); got != "user_identities" {
		t.Fatalf("expected TableName user_identities, got %q", got)
	}
	if !db.Migrator().HasTable(&models.UserIdentity{}) {
		t.Fatal("expected user_identities table after the v20 up")
	}
	if !db.Migrator().HasColumn(&models.UserIdentity{}, "sub") {
		t.Fatal("expected sub column on user_identities")
	}

	// Down: the rollback drops the table.
	v20Down(t, db)
	if db.Migrator().HasTable(&models.UserIdentity{}) {
		t.Fatal("expected user_identities table gone after the v20 down")
	}

	// Re-apply: the round trip restores the table.
	if err := db.AutoMigrate(&models.UserIdentity{}); err != nil {
		t.Fatalf("expected v20 re-apply to restore user_identities, got: %v", err)
	}
	if !db.Migrator().HasTable(&models.UserIdentity{}) {
		t.Fatal("expected user_identities table after re-applying the v20 up")
	}
}
