package middleware

import (
	"github.com/google/uuid"
	"github.com/tertua/invoiceman/pkg/logger"
	"github.com/tertua/invoiceman/platform/database"
)

// deleteIdempotencyKey removes a key best-effort. Row expiry bounds the
// staleness of a failed delete, so it is debug-logged instead of failing
// the request.
func deleteIdempotencyKey(db *database.Queries, id uuid.UUID) {
	if err := db.DeleteIdempotencyKey(id); err != nil {
		logger.L().Debug("idempotency cleanup failed", "err", err)
	}
}
