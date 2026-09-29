package database

import (
	"testing"

	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"gorm.io/gorm"
)

// orgUpModels lists every model that gains an org_id column with the v16 migration.
var orgUpModels = []any{
	&models.Invoice{}, &models.Client{}, &models.Item{}, &models.Expense{},
	&models.Payment{}, &models.Settings{}, &models.GatewayProject{}, &models.AuditLog{},
}

// orgUpTables pairs each org model with the literal table name its TableName method must declare.
var orgUpTables = []struct {
	model interface{ TableName() string }
	name  string
}{
	{models.Organization{}, "organizations"},
	{models.Membership{}, "memberships"},
	{models.OrgInvite{}, "org_invites"},
}

// assertOrgUp checks every v16 org_id column and org table exists, pinning each table to its model TableName literal.
func assertOrgUp(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, model := range orgUpModels {
		if !db.Migrator().HasColumn(model, "org_id") {
			t.Errorf("expected column org_id on %T after the v16 up", model)
		}
	}
	for _, entry := range orgUpTables {
		if got := entry.model.TableName(); got != entry.name {
			t.Errorf("expected %T TableName %q, got %q", entry.model, entry.name, got)
		}
		if !db.Migrator().HasTable(entry.model) {
			t.Errorf("expected table %q after the v16 up", entry.name)
		}
	}
}

// seedOrgRoundTrip inserts the core rows (one user plus one invoice) that must survive the whole round trip.
func seedOrgRoundTrip(t *testing.T, db *gorm.DB) (models.User, models.Invoice) {
	t.Helper()
	user := models.User{ID: uuid.New(), Name: "Roundtrip Owner", Email: "roundtrip@example.com", PasswordHash: "hash", UserStatus: 1, UserRole: "user"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("expected roundtrip user, got: %v", err)
	}
	invoice := models.Invoice{
		ID: uuid.New(), UserID: user.ID, InvoiceNumber: "RT-000001",
		Status: models.InvoiceStatusDraft, Currency: "IDR", TaxRate: 11,
		Subtotal: models.MoneyFromMinor(150000), TaxAmount: models.ZeroMoney,
		Discount: models.ZeroMoney, Total: models.MoneyFromMinor(150000),
	}
	if err := db.Create(&invoice).Error; err != nil {
		t.Fatalf("expected roundtrip invoice, got: %v", err)
	}
	return user, invoice
}

// assertOrgCoreRows probes the seeded rows by text keys so it also works while the org_id column is dropped.
func assertOrgCoreRows(t *testing.T, db *gorm.DB, user models.User, invoice models.Invoice) {
	t.Helper()
	var users, invoices int64
	if err := db.Table("users").Where("email = ?", user.Email).Count(&users).Error; err != nil {
		t.Fatalf("expected user probe, got: %v", err)
	}
	if users != 1 {
		t.Errorf("expected the seeded user to survive, got %d rows", users)
	}
	if err := db.Table("invoices").Where("invoice_number = ?", invoice.InvoiceNumber).Count(&invoices).Error; err != nil {
		t.Fatalf("expected invoice probe, got: %v", err)
	}
	if invoices != 1 {
		t.Errorf("expected the seeded invoice to survive, got %d rows", invoices)
	}
}

// TestOrgMigrationRoundTrip drives Migrate to 16, MigrateDownTo(15) and Migrate again on one isolated in-memory database.
func TestOrgMigrationRoundTrip(t *testing.T) {
	t.Setenv("SQL_DSN", "")
	t.Setenv("SQLITE_PATH", "file::memory:?cache=shared")

	if err := Migrate(); err != nil {
		t.Fatalf("expected migrate to v16, got: %v", err)
	}
	db, err := openShared()
	if err != nil {
		t.Fatalf("expected shared handle, got: %v", err)
	}
	if stamp, err := CurrentSchemaVersion(); err != nil || stamp != SchemaVersion {
		t.Fatalf("expected stamp %d after migrate, got %d (%v)", SchemaVersion, stamp, err)
	}
	assertOrgUp(t, db)
	if got := settingsPK(t, db); got != "org_id" {
		t.Fatalf("expected settings primary key org_id after migrate, got %q", got)
	}
	user, invoice := seedOrgRoundTrip(t, db)

	ver, err := MigrateDownTo(15)
	if err != nil {
		t.Fatalf("expected rollback to 15, got: %v", err)
	}
	if ver != 15 {
		t.Fatalf("expected returned version 15, got %d", ver)
	}
	assertV16RolledBack(t, db)
	if got := settingsPK(t, db); got != "user_id" {
		t.Errorf("expected settings primary key user_id after rollback, got %q", got)
	}
	assertOrgCoreRows(t, db, user, invoice)
	if stamp, err := CurrentSchemaVersion(); err != nil || stamp != 15 {
		t.Fatalf("expected stamp 15 after rollback, got %d (%v)", stamp, err)
	}

	if err := Migrate(); err != nil {
		t.Fatalf("expected re-migrate to v16, got: %v", err)
	}
	assertOrgUp(t, db)
	if got := settingsPK(t, db); got != "org_id" {
		t.Errorf("expected settings primary key org_id after re-migrate, got %q", got)
	}
	assertOrgCoreRows(t, db, user, invoice)
	if stamp, err := CurrentSchemaVersion(); err != nil || stamp != SchemaVersion {
		t.Fatalf("expected stamp %d after re-migrate, got %d (%v)", SchemaVersion, stamp, err)
	}
	restored := models.Invoice{}
	if err := db.Where("invoice_number = ?", invoice.InvoiceNumber).First(&restored).Error; err != nil {
		t.Fatalf("expected seeded invoice after re-migrate, got: %v", err)
	}
	if restored.OrgID == uuid.Nil {
		t.Error("expected the restored invoice to be stamped with an org_id by the backfill")
	}
}
