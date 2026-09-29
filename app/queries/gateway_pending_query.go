package queries

import (
	"time"

	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"gorm.io/gorm"
)

// StalePendingTransactions returns pending gateway transactions not refreshed since before, oldest first, for the reconciler poll.
func (q *GatewayQueries) StalePendingTransactions(before time.Time, limit int) ([]models.GatewayTransaction, error) {
	out := []models.GatewayTransaction{}
	if err := q.Where("status = ? AND updated_at < ?", models.GatewayStatusPending, before).
		Order("updated_at ASC").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// PendingInvoiceIDs returns org-scoped invoice IDs with a live gateway transaction; callers overlay them onto the computed effective status, and errors fail open to an empty set.
func (q *GatewayQueries) PendingInvoiceIDs(orgID uuid.UUID) map[uuid.UUID]bool {
	return pendingInvoiceSet(q.DB, orgID)
}

// pendingInvoiceSet collects org-scoped invoice IDs with a pending gateway transaction for any query struct sharing the same database handle.
func pendingInvoiceSet(db *gorm.DB, orgID uuid.UUID) map[uuid.UUID]bool {
	out := map[uuid.UUID]bool{}
	for _, id := range pendingInvoiceIDs(db, orgID) {
		out[id] = true
	}
	return out
}

// pendingInvoiceIDs lists org-scoped invoice IDs with a pending gateway transaction; string parsing keeps SQLite and PostgreSQL UUID forms decoding.
func pendingInvoiceIDs(db *gorm.DB, orgID uuid.UUID) []uuid.UUID {
	out := []uuid.UUID{}
	var found []string
	if err := db.Model(&models.GatewayTransaction{}).
		Joins("JOIN invoices ON invoices.id = gateway_transactions.invoice_id").
		Where("invoices.org_id = ? AND gateway_transactions.invoice_id IS NOT NULL AND gateway_transactions.status = ?", orgID, models.GatewayStatusPending).
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
