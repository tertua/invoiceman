package database

import (
	"path/filepath"
	"testing"

	"github.com/tertua/invoiceman/app/models"
)

// TestMigrateDownUpRoundTrip rolls v2 columns away and back on an isolated
// SQLite file, leaving the shared handle stamped at SchemaVersion.
func TestMigrateDownUpRoundTrip(t *testing.T) {
	t.Setenv("SQL_DSN", "")
	t.Setenv("SQLITE_PATH", filepath.Join(t.TempDir(), "mig.db"))

	if err := Migrate(); err != nil {
		t.Fatalf("expected migrate to succeed, got: %v", err)
	}
	db, err := openShared()
	if err != nil {
		t.Fatalf("expected shared handle, got: %v", err)
	}
	if !db.Migrator().HasColumn(&models.Expense{}, "receipt_url") {
		t.Fatal("expected receipt_url column after migrate")
	}
	if !db.Migrator().HasColumn(&models.MailOutbox{}, "html_body") {
		t.Fatal("expected html_body column after migrate")
	}

	ver, err := MigrateDownTo(1)
	if err != nil {
		t.Fatalf("expected rollback to 1, got: %v", err)
	}
	if ver != 1 {
		t.Fatalf("expected returned version 1, got %d", ver)
	}
	if db.Migrator().HasColumn(&models.Expense{}, "receipt_url") {
		t.Error("expected receipt_url column dropped after rollback")
	}
	if db.Migrator().HasColumn(&models.MailOutbox{}, "html_body") {
		t.Error("expected html_body column dropped after rollback")
	}
	if stamp, err := CurrentSchemaVersion(); err != nil || stamp != 1 {
		t.Fatalf("expected stamp 1 after rollback, got %d (%v)", stamp, err)
	}

	// Refusals: same version, below 1, above current.
	for _, target := range []int{1, 0, 5} {
		if _, err := MigrateDownTo(target); err == nil {
			t.Fatalf("expected refusal for target %d, got nil", target)
		}
	}

	// Forward re-applies automatically on next startup.
	if err := Migrate(); err != nil {
		t.Fatalf("expected re-migrate to succeed, got: %v", err)
	}
	if !db.Migrator().HasColumn(&models.Expense{}, "receipt_url") {
		t.Error("expected receipt_url column restored after re-migrate")
	}
	if stamp, err := CurrentSchemaVersion(); err != nil || stamp != SchemaVersion {
		t.Fatalf("expected stamp %d after re-migrate, got %d (%v)", SchemaVersion, stamp, err)
	}
}
