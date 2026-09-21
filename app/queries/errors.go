package queries

import (
	"database/sql"
	"errors"

	"gorm.io/gorm"
)

// notFound translates GORM not-found errors to sql.ErrNoRows,
// keeping controller error handling unchanged across drivers.
func notFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return sql.ErrNoRows
	}
	return err
}
