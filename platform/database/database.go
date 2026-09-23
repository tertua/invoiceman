package database

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	sqliteDriver "github.com/glebarez/sqlite"
	"github.com/tertua/invoiceman/pkg/configs"
	"github.com/tertua/invoiceman/pkg/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Database backend flags, following the one-api pattern.
var UsingSQLite = false
var UsingPostgreSQL = false

// chooseDB opens a database handle from the central config:
//   - empty DSN -> SQLite file (dev, auto-created; path from SQLITE_PATH)
//   - "postgres://..." or "postgresql://..." prefix -> PostgreSQL (production)
//   - anything else -> error (fail fast instead of silently using SQLite,
//     e.g. a MySQL DSN which this app does not support)
func chooseDB(dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return openSQLite()
	}
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		return openPostgreSQL(dsn)
	}
	return nil, fmt.Errorf(
		"unsupported SQL_DSN %q: leave it empty for SQLite (set SQLITE_PATH for the file) or use a postgres://... DSN for PostgreSQL",
		dsn,
	)
}

// openPostgreSQL opens a PostgreSQL connection with pool settings from the
// central config.
func openPostgreSQL(dsn string) (*gorm.DB, error) {
	logger.L().Info("database: using PostgreSQL")
	UsingPostgreSQL = true

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt: true,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	pool := configs.Get().DB
	sqlDB.SetMaxOpenConns(pool.MaxConn)
	sqlDB.SetMaxIdleConns(pool.MaxIdle)
	sqlDB.SetConnMaxLifetime(time.Duration(pool.MaxLifetimeSec) * time.Second)

	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// openSQLite opens a SQLite database file, creating it when missing.
// A single connection is used because SQLite serializes writers;
// this avoids "database is locked" errors entirely.
func openSQLite() (*gorm.DB, error) {
	path := configs.Get().DB.SQLitePath
	logger.L().Info("database: using SQLite", "path", path)
	UsingSQLite = true

	if !strings.HasPrefix(path, "file:") {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			return nil, err
		}
	}

	db, err := gorm.Open(sqliteDriver.Open(path), &gorm.Config{
		PrepareStmt: true,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)

	// WAL mode is file-only; in-memory databases ignore it.
	if !strings.HasPrefix(path, "file::memory:") {
		if err := db.Exec(`PRAGMA journal_mode=WAL`).Error; err != nil {
			return nil, err
		}
	}
	if err := db.Exec(`PRAGMA busy_timeout=5000`).Error; err != nil {
		return nil, err
	}
	// Foreign keys are off by default in SQLite; the schema relies on
	// ON DELETE CASCADE, so they must be enabled per connection.
	if err := db.Exec(`PRAGMA foreign_keys=ON`).Error; err != nil {
		return nil, err
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
