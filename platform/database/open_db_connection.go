package database

import (
	"sync"

	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/app/queries"
	"gorm.io/gorm"
)

// Queries struct for collect all app queries.
type Queries struct {
	*queries.UserQueries      // load queries from User model
	*queries.ClientQueries    // load queries from Client model
	*queries.InvoiceQueries   // load queries from Invoice model
	*queries.SettingsQueries  // load queries from Settings model
	*queries.DashboardQueries // load queries for Dashboard aggregates
}

var (
	sharedDB  *gorm.DB
	sharedErr error
	dbOnce    sync.Once
)

// openShared opens the database handle once per process.
func openShared() (*gorm.DB, error) {
	dbOnce.Do(func() {
		sharedDB, sharedErr = chooseDB("SQL_DSN")
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
		UserQueries:      &queries.UserQueries{DB: db},      // from User model
		ClientQueries:    &queries.ClientQueries{DB: db},    // from Client model
		InvoiceQueries:   &queries.InvoiceQueries{DB: db},   // from Invoice model
		SettingsQueries:  &queries.SettingsQueries{DB: db},  // from Settings model
		DashboardQueries: &queries.DashboardQueries{DB: db}, // for Dashboard aggregates
	}, nil
}

// Migrate creates or updates tables from models.
// Add Fase 2 models (items, expenses, payment links) to this list when built.
func Migrate() error {
	db, err := openShared()
	if err != nil {
		return err
	}

	return db.AutoMigrate(
		&models.User{},
		&models.Client{},
		&models.Invoice{},
		&models.InvoiceItem{},
		&models.Payment{},
		&models.Settings{},
		&models.PasswordReset{},
	)
}
