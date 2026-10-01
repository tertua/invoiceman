package database

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"gorm.io/gorm"
)

// probeDB opens a private SQLite file (same isolation as TestPostgresBackend) so rollback probes never touch the shared handle.
func probeDB(t *testing.T) *gorm.DB {
	t.Helper()
	t.Setenv("SQL_DSN", "")
	t.Setenv("SQLITE_PATH", filepath.Join(t.TempDir(), "probe.db"))
	db, err := chooseDB("")
	if err != nil {
		t.Fatalf("expected probe database, got: %v", err)
	}
	return db
}

// probeMigrate creates the startup schema (all models plus the org tables) on the probe handle.
func probeMigrate(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(
		&models.User{},
		&models.Client{},
		&models.Invoice{},
		&models.InvoiceItem{},
		&models.Item{},
		&models.Expense{},
		&models.Payment{},
		&models.PaymentLink{},
		&models.GatewayProject{},
		&models.GatewayTransaction{},
		&models.GatewayEvent{},
		&models.WebhookDelivery{},
		&models.Settings{},
		&models.PasswordReset{},
		&models.EmailVerification{},
		&models.IdempotencyKey{},
		&models.MailOutbox{},
		&models.NotificationEndpoint{},
		&models.NotificationDelivery{},
		&models.SchemaMigration{},
		&models.AuditLog{},
		&models.Organization{},
		&models.Membership{},
		&models.OrgInvite{},
	); err != nil {
		t.Fatalf("expected probe migrate to succeed, got: %v", err)
	}
}

// v16Down runs the registered v16 rollback step against the probe handle.
func v16Down(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, m := range migrations {
		if m.Version != 16 {
			continue
		}
		if err := m.Down(db); err != nil {
			t.Fatalf("expected v16 rollback to succeed, got: %v", err)
		}
		return
	}
	t.Fatal("v16 rollback step missing from the migration registry")
}

// settingsPK reports settings primary-key columns as a comma-joined list.
func settingsPK(t *testing.T, db *gorm.DB) string {
	t.Helper()
	cols, err := db.Migrator().ColumnTypes("settings")
	if err != nil {
		t.Fatalf("expected settings column types, got: %v", err)
	}
	var pk []string
	for _, col := range cols {
		if isPK, ok := col.PrimaryKey(); ok && isPK {
			pk = append(pk, col.Name())
		}
	}
	return strings.Join(pk, ",")
}

// assertV16RolledBack checks every v16 org_id column and org table is gone and no scratch table is left behind.
func assertV16RolledBack(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, target := range []struct {
		model any
		col   string
	}{
		{&models.Invoice{}, "org_id"},
		{&models.Client{}, "org_id"},
		{&models.Item{}, "org_id"},
		{&models.Expense{}, "org_id"},
		{&models.Payment{}, "org_id"},
		{&models.Settings{}, "org_id"},
		{&models.GatewayProject{}, "org_id"},
		{&models.AuditLog{}, "org_id"},
	} {
		if db.Migrator().HasColumn(target.model, target.col) {
			t.Errorf("expected column %s to be dropped from the v16 target table", target.col)
		}
	}
	for _, model := range []any{&models.OrgInvite{}, &models.Membership{}, &models.Organization{}} {
		if db.Migrator().HasTable(model) {
			t.Errorf("expected table of %T to be dropped", model)
		}
	}
	if db.Migrator().HasTable(&settingsDown{}) {
		t.Error("expected the settings scratch table to be renamed into place")
	}
}

// probeSettingsRow returns the single settings row of a rolled-back probe database.
func probeSettingsRow(t *testing.T, db *gorm.DB) settingsDown {
	t.Helper()
	var rows []settingsDown
	if err := db.Table("settings").Find(&rows).Error; err != nil {
		t.Fatalf("expected settings rows after rollback, got: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 settings row after rollback, got %d", len(rows))
	}
	return rows[0]
}

// TestV16DownFreshDatabase rolls v16 back on a fresh schema: data and the user_id key survive the settings rebuild.
func TestV16DownFreshDatabase(t *testing.T) {
	db := probeDB(t)
	probeMigrate(t, db)

	user := models.User{ID: uuid.New(), Name: "Probe Owner", Email: "probe@example.com", PasswordHash: "hash", UserStatus: 1, UserRole: "user"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("expected probe user, got: %v", err)
	}
	row := models.Settings{OrgID: user.ID, UserID: user.ID, CompanyName: "Probe Co", Currency: "IDR", TaxRate: 11, InvoicePrefix: "PRB-", InvoiceSeq: 7}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("expected probe settings, got: %v", err)
	}
	if err := backfillOrganizations(db); err != nil {
		t.Fatalf("expected backfill to succeed, got: %v", err)
	}
	if got := settingsPK(t, db); got != "org_id" {
		t.Fatalf("expected fresh settings primary key org_id, got %q", got)
	}

	v16Down(t, db)
	assertV16RolledBack(t, db)

	if got := settingsPK(t, db); got != "user_id" {
		t.Errorf("expected settings primary key user_id after rollback, got %q", got)
	}
	got := probeSettingsRow(t, db)
	if got.UserID != user.ID || got.CompanyName != "Probe Co" || got.Currency != "IDR" || got.InvoicePrefix != "PRB-" || got.InvoiceSeq != 7 {
		t.Errorf("expected settings row preserved, got %+v", got)
	}
}

// TestV16DownUpgradedSettings rolls v16 back on a v15→v16 upgrade where settings kept the user_id primary key.
func TestV16DownUpgradedSettings(t *testing.T) {
	db := probeDB(t)
	// The v15 table is built first so AutoMigrate can only add org_id as a plain column (GORM cannot add a primary key).
	if err := db.AutoMigrate(&settingsDown{}); err != nil {
		t.Fatalf("expected v15 settings table, got: %v", err)
	}
	if err := db.Migrator().RenameTable(&settingsDown{}, "settings"); err != nil {
		t.Fatalf("expected settings rename, got: %v", err)
	}
	probeMigrate(t, db)

	user := models.User{ID: uuid.New(), Name: "Legacy Owner", Email: "legacy@example.com", PasswordHash: "hash", UserStatus: 1, UserRole: "user"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("expected probe user, got: %v", err)
	}
	row := models.Settings{OrgID: user.ID, UserID: user.ID, CompanyName: "Legacy Co", Currency: "USD", TaxRate: 10, InvoicePrefix: "OLD-", InvoiceSeq: 3}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("expected probe settings, got: %v", err)
	}
	if got := settingsPK(t, db); got != "user_id" {
		t.Fatalf("expected upgraded settings primary key user_id, got %q", got)
	}

	v16Down(t, db)
	assertV16RolledBack(t, db)

	if got := settingsPK(t, db); got != "user_id" {
		t.Errorf("expected settings primary key user_id after rollback, got %q", got)
	}
	got := probeSettingsRow(t, db)
	if got.UserID != user.ID || got.CompanyName != "Legacy Co" || got.Currency != "USD" || got.InvoicePrefix != "OLD-" || got.InvoiceSeq != 3 {
		t.Errorf("expected settings row preserved, got %+v", got)
	}
}

// TestBackfillRebuildsUpgradedSettingsPK runs the v16 backfill on a v15-upgraded settings table: org_id must become the primary key with both rows intact (stamped org for the owner, user-id fallback for the staff member).
func TestBackfillRebuildsUpgradedSettingsPK(t *testing.T) {
	db := probeDB(t)
	if err := db.AutoMigrate(&settingsDown{}); err != nil {
		t.Fatalf("expected v15 settings table, got: %v", err)
	}
	if err := db.Migrator().RenameTable(&settingsDown{}, "settings"); err != nil {
		t.Fatalf("expected settings rename, got: %v", err)
	}
	probeMigrate(t, db)

	owner := models.User{ID: uuid.New(), Name: "Up Owner", Email: "upowner@example.com", PasswordHash: "hash", UserStatus: 1, UserRole: "user"}
	staff := models.User{ID: uuid.New(), Name: "Up Staff", Email: "upstaff@example.com", PasswordHash: "hash", UserStatus: 1, UserRole: "user"}
	for _, u := range []models.User{owner, staff} {
		if err := db.Create(&u).Error; err != nil {
			t.Fatalf("expected probe user, got: %v", err)
		}
	}
	// The staff member already sits in a foreign org, so the backfill leaves their settings row unstamped.
	foreign := models.Organization{ID: uuid.New(), Name: "Foreign Org"}
	if err := db.Create(&foreign).Error; err != nil {
		t.Fatalf("expected foreign org, got: %v", err)
	}
	m := models.Membership{ID: uuid.New(), OrgID: foreign.ID, UserID: staff.ID, Role: models.RoleStaff}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("expected staff membership, got: %v", err)
	}

	legacy := []models.Settings{
		{OrgID: uuid.Nil, UserID: owner.ID, CompanyName: "Owner Co", Currency: "IDR", TaxRate: 11, InvoicePrefix: "OWN-", InvoiceSeq: 1},
		{OrgID: uuid.Nil, UserID: staff.ID, CompanyName: "Staff Co", Currency: "USD", TaxRate: 10, InvoicePrefix: "STF-", InvoiceSeq: 2},
	}
	if err := db.CreateInBatches(&legacy, 10).Error; err != nil {
		t.Fatalf("expected legacy settings rows, got: %v", err)
	}
	if got := settingsPK(t, db); got != "user_id" {
		t.Fatalf("expected v15 primary key user_id, got %q", got)
	}

	if err := backfillOrganizations(db); err != nil {
		t.Fatalf("expected backfill to rebuild settings, got: %v", err)
	}
	if got := settingsPK(t, db); got != "org_id" {
		t.Fatalf("expected rebuilt primary key org_id, got %q", got)
	}

	var rebuilt []models.Settings
	if err := db.Find(&rebuilt).Error; err != nil {
		t.Fatalf("expected rebuilt settings rows, got: %v", err)
	}
	if len(rebuilt) != 2 {
		t.Fatalf("expected 2 settings rows after rebuild, got %d", len(rebuilt))
	}
	byUser := map[uuid.UUID]models.Settings{}
	for _, row := range rebuilt {
		byUser[row.UserID] = row
	}
	ownerRow, staffRow := byUser[owner.ID], byUser[staff.ID]
	if ownerRow.OrgID == uuid.Nil || ownerRow.OrgID == owner.ID {
		t.Errorf("expected owner settings stamped with the personal org, got %s", ownerRow.OrgID)
	}
	if staffRow.OrgID != staff.ID {
		t.Errorf("expected staff settings to fall back to the user id, got %s", staffRow.OrgID)
	}
	if ownerRow.CompanyName != "Owner Co" || staffRow.CompanyName != "Staff Co" || ownerRow.InvoiceSeq != 1 || staffRow.InvoiceSeq != 2 {
		t.Errorf("expected settings values preserved, got %+v and %+v", ownerRow, staffRow)
	}
}

// TestV19DownDropsEmailVerifications rolls the v19 step back: the verification table is gone while the schema stays usable.
func TestV19DownDropsEmailVerifications(t *testing.T) {
	db := probeDB(t)
	probeMigrate(t, db)

	user := models.User{ID: uuid.New(), Name: "Verify Probe", Email: "verify@example.com", PasswordHash: "hash", UserStatus: models.UserStatusPending, UserRole: "user"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("expected probe user, got: %v", err)
	}
	row := models.EmailVerification{Token: "probe-token", UserID: user.ID, CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour)}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("expected probe verification row, got: %v", err)
	}

	var down bool
	for _, m := range migrations {
		if m.Version != 19 {
			continue
		}
		if err := m.Down(db); err != nil {
			t.Fatalf("expected v19 rollback to succeed, got: %v", err)
		}
		down = true
	}
	if !down {
		t.Fatal("v19 rollback step missing from the migration registry")
	}
	if db.Migrator().HasTable(&models.EmailVerification{}) {
		t.Error("expected the email_verifications table to be dropped")
	}
}
