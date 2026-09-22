package database

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/app/queries"
	"github.com/tertua/invoiceman/pkg/configs"
	"gorm.io/gorm"
)

// Queries struct for collect all app queries.
type Queries struct {
	*queries.UserQueries        // load queries from User model
	*queries.ClientQueries      // load queries from Client model
	*queries.InvoiceQueries     // load queries from Invoice model
	*queries.ItemQueries        // load queries from Item model
	*queries.ExpenseQueries     // load queries from Expense model
	*queries.PaymentQueries     // load queries from Payment model
	*queries.GatewayQueries     // load queries for central payment relay
	*queries.ReportQueries      // load queries for Reports aggregates
	*queries.SettingsQueries    // load queries from Settings model
	*queries.DashboardQueries   // load queries for Dashboard aggregates
	*queries.IdempotencyQueries // load queries for idempotency keys
	*queries.MailOutboxQueries  // load queries for mail outbox
	*queries.AuditQueries       // load queries for audit trail
}

var (
	sharedDB  *gorm.DB
	sharedErr error
	dbOnce    sync.Once
)

// openShared opens the database handle once per process.
func openShared() (*gorm.DB, error) {
	dbOnce.Do(func() {
		sharedDB, sharedErr = chooseDB(configs.Get().DSN())
	})
	return sharedDB, sharedErr
}

// OpenDBConnection func for opening database connection.
func OpenDBConnection() (*Queries, error) {
	db, err := openShared()
	if err != nil {
		return nil, err
	}

	return &Queries{
		// Set queries from models:
		UserQueries:        &queries.UserQueries{DB: db},        // from User model
		ClientQueries:      &queries.ClientQueries{DB: db},      // from Client model
		InvoiceQueries:     &queries.InvoiceQueries{DB: db},     // from Invoice model
		ItemQueries:        &queries.ItemQueries{DB: db},        // from Item model
		ExpenseQueries:     &queries.ExpenseQueries{DB: db},     // from Expense model
		PaymentQueries:     &queries.PaymentQueries{DB: db},     // from Payment model
		GatewayQueries:     &queries.GatewayQueries{DB: db},     // for central payment relay
		ReportQueries:      &queries.ReportQueries{DB: db},      // for Reports aggregates
		SettingsQueries:    &queries.SettingsQueries{DB: db},    // from Settings model
		DashboardQueries:   &queries.DashboardQueries{DB: db},   // for Dashboard aggregates
		IdempotencyQueries: &queries.IdempotencyQueries{DB: db}, // for idempotency keys
		MailOutboxQueries:  &queries.MailOutboxQueries{DB: db},  // for mail outbox
		AuditQueries:       &queries.AuditQueries{DB: db},       // for audit trail
	}, nil
}

// SchemaVersion is the current schema revision. Bump it by 1 every time a
// model changes so the version guard below can detect newer databases.
const SchemaVersion = 2

// Migrate creates or updates tables from models, then enforces the schema
// version guard (forward-only upgrades; newer DB than binary is fatal).
func Migrate() error {
	db, err := openShared()
	if err != nil {
		return err
	}

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
		&models.IdempotencyKey{},
		&models.MailOutbox{},
		&models.SchemaMigration{},
		&models.AuditLog{},
	); err != nil {
		return err
	}

	return checkSchemaVersion(db)
}

// checkSchemaVersion implements the forward-only guard.
func checkSchemaVersion(db *gorm.DB) error {
	row := models.SchemaMigration{}
	err := db.Where("id = ?", 1).First(&row).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		// Fresh database: stamp it.
		return db.Create(&models.SchemaMigration{
			ID: 1, Version: SchemaVersion,
			AppVersion: appVersion(), AppliedAt: time.Now(),
		}).Error
	}
	if row.Version > SchemaVersion {
		return fmt.Errorf(
			"database schema version %d is newer than this binary (version %d): refusing to start",
			row.Version, SchemaVersion)
	}
	if row.Version < SchemaVersion {
		row.Version = SchemaVersion
		row.AppVersion = appVersion()
		row.AppliedAt = time.Now()
		return db.Save(&row).Error
	}
	return nil
}

// appVersion reads the single-source VERSION file, falling back to "dev".
func appVersion() string {
	raw, err := os.ReadFile("VERSION")
	if err != nil {
		return "dev"
	}
	if v := strings.TrimSpace(string(raw)); v != "" {
		return v
	}
	return "dev"
}
