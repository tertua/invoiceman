package database

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/app/queries"
)

// TestPostgresBackend validates the PostgreSQL path on a real server. It
// runs only when INVOICEMAN_TEST_PG_DSN points at a SCRATCH database (tables
// are created and a DDL round trip is executed); otherwise it skips, so
// resource-limited builds never need a PostgreSQL service. It uses a fresh
// handle via chooseDB and never touches the shared test handle.
func TestPostgresBackend(t *testing.T) {
	dsn := os.Getenv("INVOICEMAN_TEST_PG_DSN")
	if dsn == "" {
		t.Skip("set INVOICEMAN_TEST_PG_DSN to a scratch PostgreSQL database to run")
	}

	db, err := chooseDB(dsn)
	if err != nil {
		t.Fatalf("expected PostgreSQL handle, got: %v", err)
	}
	if !UsingPostgreSQL {
		t.Fatal("expected UsingPostgreSQL to be true for a postgres DSN")
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("expected sql handle, got: %v", err)
	}
	defer sqlDB.Close()

	if err := db.AutoMigrate(&models.User{}, &models.Expense{}, &models.MailOutbox{}); err != nil {
		t.Fatalf("expected automigrate on PostgreSQL, got: %v", err)
	}

	// CRUD smoke through the shared query layer (backend-agnostic code).
	uq := &queries.UserQueries{DB: db}
	email := "pg-smoke-" + uuid.NewString() + "@example.com"
	u := &models.User{
		ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(),
		Name: "PG Smoke", Email: email,
		PasswordHash: "x", UserStatus: 1, UserRole: "user",
	}
	if err := uq.CreateUser(u); err != nil {
		t.Fatalf("expected user create on PostgreSQL, got: %v", err)
	}
	got, err := uq.GetUserByEmail(email)
	if err != nil || got.ID != u.ID {
		t.Fatalf("expected user read back on PostgreSQL, got %+v (%v)", got, err)
	}

	// DDL round trip: the same DropColumn the v2 rollback uses must work
	// on PostgreSQL, then AutoMigrate restores it.
	if err := db.Migrator().DropColumn(&models.Expense{}, "receipt_url"); err != nil {
		t.Fatalf("expected DROP COLUMN on PostgreSQL, got: %v", err)
	}
	if db.Migrator().HasColumn(&models.Expense{}, "receipt_url") {
		t.Fatal("expected receipt_url gone after DROP COLUMN on PostgreSQL")
	}
	if err := db.AutoMigrate(&models.Expense{}); err != nil {
		t.Fatalf("expected re-migrate on PostgreSQL, got: %v", err)
	}
	if !db.Migrator().HasColumn(&models.Expense{}, "receipt_url") {
		t.Fatal("expected receipt_url restored on PostgreSQL")
	}

	if err := db.Delete(&models.User{}, "id = ?", u.ID).Error; err != nil {
		t.Fatalf("expected cleanup delete, got: %v", err)
	}
}
