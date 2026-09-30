package queries

import (
	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
)

// InvoiceStatusCounts returns per-status invoice counts for an org using the
// same effective-status rules as the listing, so status-tab badges never
// diverge from the rows beneath them. Counts are always org-wide: search and
// currency are deliberately not applied, because a badge that shrinks while
// the payer types reads as a bug, not a filter.
//
// The effective status (paid via full payment, overdue via due date, pending
// via a live gateway transaction) is computed in Go — never a raw SQL
// GROUP BY — so "paid" and "overdue" match models.ResolveEffectiveStatus.
func (q *InvoiceQueries) InvoiceStatusCounts(orgID uuid.UUID) (map[string]int, error) {
	rows := []models.InvoiceListRow{}
	if err := q.filteredInvoices(orgID, "", "").Scan(&rows).Error; err != nil {
		return nil, err
	}

	pending := pendingInvoiceSet(q.DB, orgID)

	counts := map[string]int{"all": 0, "draft": 0, "sent": 0, "paid": 0, "overdue": 0}
	for _, row := range rows {
		counts["all"]++
		if _, ok := counts[row.EffectiveStatus(pending[row.ID])]; ok {
			counts[row.EffectiveStatus(pending[row.ID])]++
		}
	}

	return counts, nil
}
