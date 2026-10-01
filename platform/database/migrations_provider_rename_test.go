package database

import (
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"gorm.io/gorm"
)

// v16GatewayTransaction mirrors models.GatewayTransaction as of v16: the old
// provider-named columns and no provider-neutral ones. TableName pins the
// plural table so the v16 shape lands on the same table the real model uses.
type v16GatewayTransaction struct {
	OrderID       string `gorm:"primaryKey;size:128" db:"order_id"`
	ProjectSlug   string `db:"project_slug"`
	Gateway       string `gorm:"size:32" db:"gateway"`
	SnapToken     string `gorm:"size:255" db:"snap_token"`
	MidtransTxnID string `gorm:"size:128" db:"midtrans_txn_id"`
}

func (v16GatewayTransaction) TableName() string { return "gateway_transactions" }

// v16Settings mirrors models.Settings as of v16 with only the columns this test
// needs; it carries the old midtrans_methods column and the org_id key.
type v16Settings struct {
	OrgID           uuid.UUID `gorm:"type:uuid;primaryKey" db:"org_id"`
	UserID          uuid.UUID `gorm:"type:uuid;not null" db:"user_id"`
	CompanyName     string    `db:"company_name"`
	Currency        string    `db:"currency"`
	MidtransMethods string    `gorm:"size:255" db:"midtrans_methods"`
}

func (v16Settings) TableName() string { return "settings" }

// seedV16Schema builds the old-shape tables and inserts one row each with known
// values in the provider-named columns, then runs the real startup schema on top
// (AutoMigrate adds the neutral columns empty) and the v17 copy step.
func seedV16Schema(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(&v16GatewayTransaction{}, &v16Settings{}); err != nil {
		t.Fatalf("expected v16-shape tables, got: %v", err)
	}
	txn := v16GatewayTransaction{
		OrderID: "PAY-V17-1", ProjectSlug: "local", Gateway: "midtrans",
		SnapToken: "snap-old-token", MidtransTxnID: "mtx-old-id",
	}
	if err := db.Create(&txn).Error; err != nil {
		t.Fatalf("expected v16 transaction row, got: %v", err)
	}
	orgID := uuid.New()
	row := v16Settings{
		OrgID: orgID, UserID: uuid.New(), CompanyName: "Legacy Co",
		Currency: "IDR", MidtransMethods: "qris",
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("expected v16 settings row, got: %v", err)
	}
	// probeMigrate adds every current model (new columns included, old columns
	// preserved as unmanaged extras on the same tables).
	probeMigrate(t, db)
}

// TestProviderRenameUpCopiesAndDrops is the forward half: both columns present →
// data copied into the neutral column, old column dropped, copy idempotent.
func TestProviderRenameUpCopiesAndDrops(t *testing.T) {
	db := probeDB(t)
	seedV16Schema(t, db)

	for _, r := range providerColumnRenames {
		if !db.Migrator().HasColumn(r.model, r.oldCol) {
			t.Fatalf("expected pre-copy column %s.%s", r.table, r.oldCol)
		}
		if !db.Migrator().HasColumn(r.model, r.newCol) {
			t.Fatalf("expected AutoMigrate to add %s.%s", r.table, r.newCol)
		}
	}

	if err := migrateProviderColumnsUp(db); err != nil {
		t.Fatalf("expected v17 copy to succeed, got: %v", err)
	}

	var txn models.GatewayTransaction
	if err := db.First(&txn, "order_id = ?", "PAY-V17-1").Error; err != nil {
		t.Fatalf("expected copied transaction, got: %v", err)
	}
	if txn.ProviderToken != "snap-old-token" {
		t.Errorf("expected provider_token copied from snap_token, got %q", txn.ProviderToken)
	}
	if txn.ProviderTxnID != "mtx-old-id" {
		t.Errorf("expected provider_txn_id copied from midtrans_txn_id, got %q", txn.ProviderTxnID)
	}

	var settings models.Settings
	if err := db.First(&settings).Error; err != nil {
		t.Fatalf("expected copied settings, got: %v", err)
	}
	if settings.ProviderMethods != "qris" {
		t.Errorf("expected provider_methods copied from midtrans_methods, got %q", settings.ProviderMethods)
	}

	for _, r := range providerColumnRenames {
		if db.Migrator().HasColumn(r.model, r.oldCol) {
			t.Errorf("expected old column %s.%s dropped after copy", r.table, r.oldCol)
		}
	}

	// Idempotent: a second run on the migrated shape is a no-op, not an error.
	if err := migrateProviderColumnsUp(db); err != nil {
		t.Fatalf("expected repeat v17 copy to be a no-op, got: %v", err)
	}
}

// TestProviderRenameFreshAndMigratedAreNoops pins the skip paths: a fresh schema
// (only the new column) and a fully-migrated schema both leave the copy step a
// silent no-op.
func TestProviderRenameFreshAndMigratedAreNoops(t *testing.T) {
	db := probeDB(t)
	probeMigrate(t, db) // fresh: AutoMigrate creates only the neutral columns
	for _, r := range providerColumnRenames {
		if db.Migrator().HasColumn(r.model, r.oldCol) {
			t.Fatalf("fresh schema must not carry the old column %s.%s", r.table, r.oldCol)
		}
	}
	if err := migrateProviderColumnsUp(db); err != nil {
		t.Fatalf("expected fresh-schema copy to be a no-op, got: %v", err)
	}
}

// TestProviderRenameDownRenamesBack is the rollback half: the neutral columns
// are renamed back to the provider-named ones with the data intact.
func TestProviderRenameDownRenamesBack(t *testing.T) {
	db := probeDB(t)
	seedV16Schema(t, db)
	if err := migrateProviderColumnsUp(db); err != nil {
		t.Fatalf("expected v17 copy to succeed, got: %v", err)
	}

	if err := migrateProviderColumnsDown(db); err != nil {
		t.Fatalf("expected v17 rollback to succeed, got: %v", err)
	}

	if db.Migrator().HasColumn(&models.GatewayTransaction{}, "provider_txn_id") {
		t.Error("expected provider_txn_id renamed back to midtrans_txn_id")
	}
	if db.Migrator().HasColumn(&models.Settings{}, "provider_methods") {
		t.Error("expected provider_methods renamed back to midtrans_methods")
	}

	var out v16GatewayTransaction
	if err := db.First(&out, "order_id = ?", "PAY-V17-1").Error; err != nil {
		t.Fatalf("expected rolled-back transaction, got: %v", err)
	}
	if out.SnapToken != "snap-old-token" || out.MidtransTxnID != "mtx-old-id" {
		t.Errorf("expected data intact after rollback, got %+v", out)
	}
	var settings v16Settings
	if err := db.First(&settings).Error; err != nil {
		t.Fatalf("expected rolled-back settings, got: %v", err)
	}
	if settings.MidtransMethods != "qris" {
		t.Errorf("expected midtrans_methods intact after rollback, got %q", settings.MidtransMethods)
	}

	// A rollback followed by a rollback is a no-op (the new columns are gone).
	if err := migrateProviderColumnsDown(db); err != nil {
		t.Fatalf("expected repeat rollback to be a no-op, got: %v", err)
	}
}

// TestProviderRenameMigrateRoundTrip drives the full public path: Migrate()
// lands on the current SchemaVersion, MigrateDownTo(16) renames the columns
// back with data intact, and a re-Migrate() returns to the current version.
// It uses the package's shared handle (established once per process) exactly
// like migrations_test.go.
func TestProviderRenameMigrateRoundTrip(t *testing.T) {
	t.Setenv("SQL_DSN", "")
	t.Setenv("SQLITE_PATH", filepath.Join(t.TempDir(), "v17.db"))

	if err := Migrate(); err != nil {
		t.Fatalf("expected migrate to succeed, got: %v", err)
	}
	db, err := openShared()
	if err != nil {
		t.Fatalf("expected shared handle, got: %v", err)
	}
	if stamp, err := CurrentSchemaVersion(); err != nil || stamp != SchemaVersion {
		t.Fatalf("expected stamp %d after migrate, got %d (%v)", SchemaVersion, stamp, err)
	}
	for _, r := range providerColumnRenames {
		if db.Migrator().HasColumn(r.model, r.oldCol) {
			t.Errorf("expected old column %s.%s gone after migrate", r.table, r.oldCol)
		}
		if !db.Migrator().HasColumn(r.model, r.newCol) {
			t.Errorf("expected new column %s.%s after migrate", r.table, r.newCol)
		}
	}

	// A row written under the neutral schema must survive the down/up cycle.
	txn := models.GatewayTransaction{
		OrderID: "PAY-RT-1", ProjectSlug: "local", Gateway: "midtrans",
		ProviderToken: "tok-rt", ProviderTxnID: "txn-id-rt",
	}
	if err := db.Create(&txn).Error; err != nil {
		t.Fatalf("expected round-trip transaction, got: %v", err)
	}

	if _, err := MigrateDownTo(16); err != nil {
		t.Fatalf("expected rollback to 16, got: %v", err)
	}
	if stamp, err := CurrentSchemaVersion(); err != nil || stamp != 16 {
		t.Fatalf("expected stamp 16 after rollback, got %d (%v)", stamp, err)
	}
	if !db.Migrator().HasColumn(&models.GatewayTransaction{}, "midtrans_txn_id") {
		t.Error("expected midtrans_txn_id restored after rollback")
	}
	if db.Migrator().HasColumn(&models.GatewayTransaction{}, "provider_txn_id") {
		t.Error("expected provider_txn_id gone after rollback")
	}

	if err := Migrate(); err != nil {
		t.Fatalf("expected re-migrate to succeed, got: %v", err)
	}
	if stamp, err := CurrentSchemaVersion(); err != nil || stamp != SchemaVersion {
		t.Fatalf("expected stamp %d after re-migrate, got %d (%v)", SchemaVersion, stamp, err)
	}
	var out models.GatewayTransaction
	if err := db.First(&out, "order_id = ?", "PAY-RT-1").Error; err != nil {
		t.Fatalf("expected round-trip row after re-migrate, got: %v", err)
	}
	if out.ProviderToken != "tok-rt" || out.ProviderTxnID != "txn-id-rt" {
		t.Errorf("expected data intact after round trip, got %+v", out)
	}
}
