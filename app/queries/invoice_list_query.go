package queries

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"gorm.io/gorm"
)

// invoiceSortColumns whitelists sortable columns for invoice listing.
var invoiceSortColumns = map[string]string{
	"issue_date": "invoices.issue_date",
	"due_date":   "invoices.due_date",
	"total":      "invoices.total",
}

// ListInvoices returns one page of invoices of an org with client names and paid amounts.
func (q *InvoiceQueries) ListInvoices(orgID uuid.UUID, status, search, sort, order string, limit, offset int) ([]models.InvoiceListRow, error) {
	invoices := []models.InvoiceListRow{}

	tx := q.filteredInvoices(orgID, status, search)

	sortColumn := "invoices.created_at"
	if column, ok := invoiceSortColumns[strings.ToLower(sort)]; ok {
		sortColumn = column
	}
	sortOrder := "DESC"
	if strings.EqualFold(order, "asc") {
		sortOrder = "ASC"
	}
	tx = tx.Order(sortColumn + " " + sortOrder).Order("invoices.created_at DESC")

	if err := tx.Limit(limit).Offset(offset).Scan(&invoices).Error; err != nil {
		return invoices, err
	}

	return invoices, nil
}

// CountInvoices returns the total invoices matching the list filters via the shared filter chain, so the count never diverges from the listing.
func (q *InvoiceQueries) CountInvoices(orgID uuid.UUID, status, search string) (int64, error) {
	var total int64
	sub := q.filteredInvoices(orgID, status, search).Select("invoices.id")
	if err := q.Table("(?) AS invoice_ids", sub).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// filteredInvoices builds the shared filter chain for invoice listing and counting so both stay in sync.
func (q *InvoiceQueries) filteredInvoices(orgID uuid.UUID, status, search string) *gorm.DB {
	paidSubquery := q.Model(&models.Payment{}).
		Select("invoice_id, SUM(amount) AS paid").
		Where("voided_at IS NULL").
		Group("invoice_id")

	tx := q.Table("invoices").
		Select(`invoices.id, invoices.invoice_number,
			COALESCE(clients.name, '') AS client_name,
			COALESCE(clients.company, '') AS client_company,
			invoices.issue_date, invoices.due_date, invoices.total,
			invoices.currency, invoices.status,
			COALESCE(pay.paid, 0) AS paid_amount`).
		Joins("LEFT JOIN clients ON clients.id = invoices.client_id").
		Joins("LEFT JOIN (?) AS pay ON pay.invoice_id = invoices.id", paidSubquery).
		Where("invoices.org_id = ?", orgID)

	switch status {
	case models.InvoiceStatusDraft, models.InvoiceStatusSent:
		tx = tx.Where("invoices.status = ?", status)
	case models.InvoiceStatusPaid:
		tx = tx.Where("invoices.status = ? OR (invoices.total > 0 AND COALESCE(pay.paid, 0) >= invoices.total)",
			models.InvoiceStatusPaid)
	case models.InvoiceEffectiveOverdue:
		tx = tx.Where("invoices.status = ? AND invoices.due_date IS NOT NULL AND invoices.due_date < ?",
			models.InvoiceStatusSent, time.Now())
	}

	if search != "" {
		like := "%" + search + "%"
		tx = tx.Where("invoices.invoice_number LIKE ? OR clients.name LIKE ?", like, like)
	}

	return tx
}

// ClientInvoices returns invoices of a client owned by an org.
func (q *InvoiceQueries) ClientInvoices(orgID, clientID uuid.UUID) ([]ClientInvoiceRow, error) {
	rows := []ClientInvoiceRow{}

	paidSubquery := q.Model(&models.Payment{}).
		Select("invoice_id, SUM(amount) AS paid").
		Where("voided_at IS NULL").
		Group("invoice_id")

	if err := q.Table("invoices").
		Select(`invoices.id, invoices.invoice_number, invoices.issue_date, invoices.due_date,
			invoices.total, invoices.currency, invoices.status,
			COALESCE(pay.paid, 0) AS paid_amount`).
		Joins("LEFT JOIN (?) AS pay ON pay.invoice_id = invoices.id", paidSubquery).
		Where("invoices.org_id = ? AND invoices.client_id = ?", orgID, clientID).
		Order("invoices.created_at DESC").
		Scan(&rows).Error; err != nil {
		return rows, err
	}

	return rows, nil
}
