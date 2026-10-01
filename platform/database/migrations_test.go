package database

import (
	"path/filepath"
	"testing"

	"github.com/tertua/tupay/app/models"
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
	if db.Migrator().HasTable(&legacyAdminClaim{}) {
		t.Fatal("expected no admin_claims table after migrate")
	}
	if !db.Migrator().HasColumn(&models.Payment{}, "gateway_order_id") {
		t.Fatal("expected gateway_order_id column after migrate")
	}
	if !db.Migrator().HasTable(&models.NotificationEndpoint{}) {
		t.Fatal("expected notification_endpoints table after migrate")
	}
	if !db.Migrator().HasTable(&models.NotificationDelivery{}) {
		t.Fatal("expected notification_deliveries table after migrate")
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
	if db.Migrator().HasTable(&legacyAdminClaim{}) {
		t.Error("expected admin_claims table dropped after rollback")
	}
	if db.Migrator().HasColumn(&models.Payment{}, "gateway_order_id") {
		t.Error("expected gateway_order_id column dropped after rollback")
	}
	if db.Migrator().HasTable(&models.NotificationEndpoint{}) {
		t.Error("expected notification_endpoints table dropped after rollback")
	}
	if db.Migrator().HasTable(&models.NotificationDelivery{}) {
		t.Error("expected notification_deliveries table dropped after rollback")
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
	if db.Migrator().HasTable(&legacyAdminClaim{}) {
		t.Error("expected no admin_claims table after re-migrate")
	}
	if !db.Migrator().HasColumn(&models.Payment{}, "gateway_order_id") {
		t.Error("expected gateway_order_id column restored after re-migrate")
	}
	if !db.Migrator().HasTable(&models.NotificationEndpoint{}) {
		t.Error("expected notification_endpoints table restored after re-migrate")
	}
	if !db.Migrator().HasTable(&models.NotificationDelivery{}) {
		t.Error("expected notification_deliveries table restored after re-migrate")
	}
	if stamp, err := CurrentSchemaVersion(); err != nil || stamp != SchemaVersion {
		t.Fatalf("expected stamp %d after re-migrate, got %d (%v)", SchemaVersion, stamp, err)
	}
}

// TestDropLegacyAdminClaims proves the v18 startup step removes the orphaned
// claim table that databases upgraded from the claim-based bootstrap still
// carry: AutoMigrate only ever adds, so startup has to drop it explicitly.
func TestDropLegacyAdminClaims(t *testing.T) {
	t.Setenv("SQL_DSN", "")
	t.Setenv("SQLITE_PATH", filepath.Join(t.TempDir(), "legacy.db"))

	if err := Migrate(); err != nil {
		t.Fatalf("expected migrate to succeed, got: %v", err)
	}
	db, err := openShared()
	if err != nil {
		t.Fatalf("expected shared handle, got: %v", err)
	}

	// Re-create the table a pre-v18 database would still carry.
	if err := db.AutoMigrate(&legacyAdminClaim{}); err != nil {
		t.Fatalf("expected legacy table to be recreated, got: %v", err)
	}
	if err := Migrate(); err != nil {
		t.Fatalf("expected re-migrate to succeed, got: %v", err)
	}
	if db.Migrator().HasTable(&legacyAdminClaim{}) {
		t.Error("expected startup to drop the legacy admin_claims table")
	}
}
