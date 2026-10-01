package database

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"gorm.io/gorm"
)

// newEmailProbe opens an isolated SQLite file with the users table so the email
// normalization + index steps run against a known shape, independent of the
// shared handle other tests stamp.
func newEmailProbe(t *testing.T) *gorm.DB {
	t.Helper()
	t.Setenv("SQL_DSN", "")
	t.Setenv("SQLITE_PATH", filepath.Join(t.TempDir(), "email-probe.db"))
	db, err := chooseDB("")
	if err != nil {
		t.Fatalf("expected probe database, got: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("expected users table, got: %v", err)
	}
	return db
}

// seedUser inserts one user with the given raw email (case/space preserved).
func seedUser(t *testing.T, db *gorm.DB, email string) {
	t.Helper()
	user := models.User{
		ID: uuid.New(), Name: "Email Probe", Email: email,
		PasswordHash: "hash", UserStatus: 1, UserRole: "user",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("expected to seed %q, got: %v", email, err)
	}
}

// emailsInTable returns every stored email in insertion-independent order.
func emailsInTable(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	var emails []string
	if err := db.Table("users").Order("created_at ASC").Pluck("email", &emails).Error; err != nil {
		t.Fatalf("expected to read emails, got: %v", err)
	}
	return emails
}

// TestNormalizeUserEmailsRoundTrip pins the v21 data step: mixed-case padded
// emails collapse to lowercase/trimmed values, and the unique index builds on
// the cleaned column.
func TestNormalizeUserEmailsRoundTrip(t *testing.T) {
	db := newEmailProbe(t)
	seedUser(t, db, "A@x.com")
	seedUser(t, db, " b@x.com ")

	if err := normalizeUserEmails(db); err != nil {
		t.Fatalf("expected normalization to succeed, got: %v", err)
	}
	got := emailsInTable(t, db)
	want := []string{"a@x.com", "b@x.com"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("expected normalized emails %v, got %v", want, got)
	}

	// Once normalized (and unique), the index builds cleanly.
	if err := ensureUserEmailIndex(db); err != nil {
		t.Fatalf("expected the unique index build to succeed, got: %v", err)
	}
	if !db.Migrator().HasIndex(&models.User{}, userEmailIndexName) {
		t.Fatal("expected the unique users.email index after the step")
	}
}

// TestNormalizeUserEmailsIdempotent proves a clean table is untouched by a
// second run (no spurious writes, no error).
func TestNormalizeUserEmailsIdempotent(t *testing.T) {
	db := newEmailProbe(t)
	seedUser(t, db, "stable@x.com")

	for i := 0; i < 2; i++ {
		if err := normalizeUserEmails(db); err != nil {
			t.Fatalf("expected run %d to be a no-op, got: %v", i+1, err)
		}
	}
	got := emailsInTable(t, db)
	if len(got) != 1 || got[0] != "stable@x.com" {
		t.Fatalf("expected the email unchanged, got %v", got)
	}
}

// TestNormalizeUserEmailsRejectsDuplicates pins the fail-fast path: a
// case-insensitive collision after normalization aborts with a message naming
// the offending address rather than deleting anything.
func TestNormalizeUserEmailsRejectsDuplicates(t *testing.T) {
	db := newEmailProbe(t)
	seedUser(t, db, "dup@x.com")
	seedUser(t, db, "DUP@X.com")

	err := normalizeUserEmails(db)
	if err == nil {
		t.Fatal("expected normalization to reject case-insensitive duplicates")
	}
	if !strings.Contains(err.Error(), "dup@x.com") {
		t.Fatalf("expected the error to name the colliding address, got: %v", err)
	}
	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		t.Fatalf("expected count, got: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected both rows preserved (no data loss), got %d", count)
	}
}

// TestUserEmailIndexRejectsDuplicate proves the created index is enforced by the
// database: a second row with the same (already normalized) email is rejected.
func TestUserEmailIndexRejectsDuplicate(t *testing.T) {
	db := newEmailProbe(t)
	seedUser(t, db, "solo@x.com")

	if err := normalizeUserEmails(db); err != nil {
		t.Fatalf("expected normalization, got: %v", err)
	}
	if err := ensureUserEmailIndex(db); err != nil {
		t.Fatalf("expected the unique index, got: %v", err)
	}

	err := db.Create(&models.User{
		ID: uuid.New(), Name: "Clash", Email: "solo@x.com",
		PasswordHash: "hash", UserStatus: 1, UserRole: "user",
	}).Error
	if err == nil {
		t.Fatal("expected the unique index to reject the duplicate email")
	}
}

// TestUserEmailIndexDown pins the registered v21 rollback: it drops the index
// (and is a no-op when the index is already absent).
func TestUserEmailIndexDown(t *testing.T) {
	db := newEmailProbe(t)
	seedUser(t, db, "rollback@x.com")
	if err := ensureUserEmailIndex(db); err != nil {
		t.Fatalf("expected the unique index, got: %v", err)
	}

	var down func(*gorm.DB) error
	for _, m := range migrations {
		if m.Version == 21 {
			down = m.Down
		}
	}
	if down == nil {
		t.Fatal("v21 rollback step missing from the migration registry")
	}
	if err := down(db); err != nil {
		t.Fatalf("expected the v21 rollback to succeed, got: %v", err)
	}
	if db.Migrator().HasIndex(&models.User{}, userEmailIndexName) {
		t.Fatal("expected the unique index gone after the v21 rollback")
	}
	// Idempotent: running it again on an index-less schema is a no-op.
	if err := down(db); err != nil {
		t.Fatalf("expected the v21 rollback to be idempotent, got: %v", err)
	}
}
