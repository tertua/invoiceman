package database

import (
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/app/queries"
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
	if !db.Migrator().HasTable(&models.AdminClaim{}) {
		t.Fatal("expected admin_claims table after migrate")
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
	if db.Migrator().HasTable(&models.AdminClaim{}) {
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
	if !db.Migrator().HasTable(&models.AdminClaim{}) {
		t.Error("expected admin_claims table restored after re-migrate")
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

// TestConcurrentAdminClaim proves exactly one concurrent claimant wins the
// singleton first-admin row (the CountUsers+CreateUser race is closed by
// the primary key, not by timing).
func TestConcurrentAdminClaim(t *testing.T) {
	t.Setenv("SQL_DSN", "")
	t.Setenv("SQLITE_PATH", filepath.Join(t.TempDir(), "claim.db"))

	if err := Migrate(); err != nil {
		t.Fatalf("expected migrate to succeed, got: %v", err)
	}
	db, err := openShared()
	if err != nil {
		t.Fatalf("expected shared handle, got: %v", err)
	}
	q := &queries.UserQueries{DB: db}

	const racers = 8
	var wins atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			won, err := q.ClaimFirstAdmin(uuid.New())
			if err != nil {
				t.Errorf("expected claim attempt to resolve, got: %v", err)
				return
			}
			if won {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("expected exactly 1 winner, got %d", wins.Load())
	}
	var count int64
	if err := db.Model(&models.AdminClaim{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("expected 1 claim row, got %d (%v)", count, err)
	}

	// A late claim after the winner loses cleanly (no error, no new row).
	if won, err := q.ClaimFirstAdmin(uuid.New()); err != nil || won {
		t.Fatalf("expected late claim to lose, got won=%v err=%v", won, err)
	}
}
