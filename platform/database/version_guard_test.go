package database

import (
	"testing"

	"github.com/tertua/invoiceman/app/models"
)

// TestSchemaVersionGuard stamps fresh databases and refuses newer ones.
func TestSchemaVersionGuard(t *testing.T) {
	if err := Migrate(); err != nil {
		t.Fatalf("expected migrate to succeed, got: %v", err)
	}

	db, err := openShared()
	if err != nil {
		t.Fatalf("expected shared handle, got: %v", err)
	}
	row := models.SchemaMigration{}
	if err := db.Where("id = ?", 1).First(&row).Error; err != nil {
		t.Fatalf("expected version row, got: %v", err)
	}
	if row.Version != SchemaVersion {
		t.Fatalf("expected stamped version %d, got %d", SchemaVersion, row.Version)
	}

	// Simulate a newer database (e.g. binary rolled back): startup fails.
	row.Version = SchemaVersion + 1
	if err := db.Save(&row).Error; err != nil {
		t.Fatalf("failed to bump version row: %v", err)
	}
	if err := Migrate(); err == nil {
		t.Fatal("expected migrate to refuse a newer database, got nil")
	}

	// Restore so the shared handle stays usable.
	row.Version = SchemaVersion
	if err := db.Save(&row).Error; err != nil {
		t.Fatalf("failed to restore version row: %v", err)
	}
}
