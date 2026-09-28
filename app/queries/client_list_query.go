package queries

import (
	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
)

// ListClients returns one page of clients of a user with billing aggregates.
func (q *ClientQueries) ListClients(userID uuid.UUID, limit, offset int) ([]models.ClientListRow, error) {
	clients := []models.ClientListRow{}

	// Billed invoices are sent + paid; drafts are not billed yet. A draft
	// with money in flight (pending) still owes, so it is billed too,
	// matching dashboard/reports. Paid must stay in total_billed so
	// auto-marking paid never shrinks client totals.
	pending := pendingInvoiceIDs(q.DB, userID)

	billedSubquery := q.Model(&models.Invoice{}).
		Select("client_id, SUM(total) AS total_billed").
		Where("user_id = ? AND status IN ?", userID, []string{models.InvoiceStatusSent, models.InvoiceStatusPaid})
	paidSubquery := q.Model(&models.Payment{}).
		Select("invoices.client_id AS client_id, SUM(payments.amount) AS paid").
		Joins("JOIN invoices ON invoices.id = payments.invoice_id").
		Where("payments.voided_at IS NULL AND invoices.user_id = ? AND invoices.status IN ?", userID, []string{models.InvoiceStatusSent, models.InvoiceStatusPaid})
	if len(pending) > 0 {
		billedSubquery = billedSubquery.Or("user_id = ? AND id IN ?", userID, pending)
		paidSubquery = paidSubquery.Or("invoices.user_id = ? AND invoices.id IN ?", userID, pending)
	}
	billedSubquery = billedSubquery.Group("client_id")
	paidSubquery = paidSubquery.Group("invoices.client_id")

	if err := q.Table("clients AS c").
		Select(`c.*, COALESCE(inv.total_billed, 0) AS total_billed,
			COALESCE(inv.total_billed, 0) - COALESCE(pay.paid, 0) AS outstanding`).
		Joins("LEFT JOIN (?) AS inv ON inv.client_id = c.id", billedSubquery).
		Joins("LEFT JOIN (?) AS pay ON pay.client_id = c.id", paidSubquery).
		Where("c.user_id = ?", userID).
		Order("c.created_at DESC").
		Limit(limit).Offset(offset).
		Scan(&clients).Error; err != nil {
		return clients, err
	}

	return clients, nil
}
