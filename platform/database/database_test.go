package database

import (
	"strings"
	"testing"
)

// chooseDB must fail fast on DSNs it does not support (e.g. MySQL)
// instead of silently falling back to SQLite with the wrong data.
func TestChooseDBRejectsUnsupportedDSN(t *testing.T) {
	t.Setenv("TEST_SQL_DSN", "mysql://user:password@localhost:3306/invoiceman")

	db, err := chooseDB("TEST_SQL_DSN")
	if err == nil {
		t.Fatal("expected an error for a MySQL DSN, got nil")
	}
	if db != nil {
		t.Fatal("expected a nil handle for a MySQL DSN")
	}
	if !strings.Contains(err.Error(), "unsupported SQL_DSN") {
		t.Fatalf("expected an unsupported DSN error, got: %v", err)
	}
}

// An empty DSN keeps selecting the zero-config SQLite backend.
func TestChooseDBEmptySelectsSQLite(t *testing.T) {
	t.Setenv("TEST_SQL_DSN", "")
	t.Setenv("SQLITE_PATH", "file::memory:?cache=shared")

	db, err := chooseDB("TEST_SQL_DSN")
	if err != nil {
		t.Fatalf("expected SQLite for an empty DSN, got: %v", err)
	}
	if db == nil {
		t.Fatal("expected a database handle for an empty DSN, got nil")
	}
	if !UsingSQLite {
		t.Fatal("expected UsingSQLite to be true for an empty DSN")
	}
}
