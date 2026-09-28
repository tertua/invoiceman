package queries

import (
	"time"

	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"gorm.io/gorm"
)

// StalePendingTransactions returns pending gateway transactions whose state
// has not been refreshed since before, oldest first. The reconciler polls
// these against the provider instead of trusting the stored status.
func (q *GatewayQueries) StalePendingTransactions(before time.Time, limit int) ([]models.GatewayTransaction, error) {
	out := []models.GatewayTransaction{}
	if err := q.Where("status = ? AND updated_at < ?", models.GatewayStatusPending, before).
		Order("updated_at ASC").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// PendingInvoiceIDs returns invoice IDs with a live gateway transaction.
// Callers overlay this onto the computed effective status so an open Snap
// payment reads as awaiting payment instead of merely sent. Errors fail open
// to an empty set: this only decorates display status.
func (q *GatewayQueries) PendingInvoiceIDs(userID uuid.UUID) map[uuid.UUID]bool {
	return pendingInvoiceSet(q.DB, userID)
}

// pendingInvoiceSet collects invoice IDs with a pending gateway transaction
// for any query struct sharing the same database handle.
func pendingInvoiceSet(db *gorm.DB, userID uuid.UUID) map[uuid.UUID]bool {
	out := map[uuid.UUID]bool{}
	for _, id := range pendingInvoiceIDs(db, userID) {
		out[id] = true
	}
	return out
}

// pendingInvoiceIDs lists invoice IDs with a pending gateway transaction.
// Invoice IDs go through string parsing so SQLite and PostgreSQL UUID forms
// both decode.
func pendingInvoiceIDs(db *gorm.DB, userID uuid.UUID) []uuid.UUID {
	out := []uuid.UUID{}
	var found []string
	if err := db.Model(&models.GatewayTransaction{}).
		Where("user_id = ? AND invoice_id IS NOT NULL AND status = ?", userID, models.GatewayStatusPending).
		Distinct().Pluck("invoice_id", &found).Error; err != nil {
		return out
	}
	for _, raw := range found {
		if id, err := uuid.Parse(raw); err == nil {
			out = append(out, id)
		}
	}
	return out
}
