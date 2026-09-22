package queries

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tertua/invoiceman/app/models"
	"gorm.io/gorm"
)

// InvoiceQueries struct for queries from Invoice model.
type InvoiceQueries struct {
	*gorm.DB
}

// invoiceSortColumns whitelists sortable columns for invoice listing.
var invoiceSortColumns = map[string]string{
	"issue_date": "invoices.issue_date",
	"due_date":   "invoices.due_date",
	"total":      "invoices.total",
}

// ListInvoices returns one page of invoices of a user with client names
// and paid amounts.
func (q *InvoiceQueries) ListInvoices(userID uuid.UUID, status, search, sort, order string, limit, offset int) ([]models.InvoiceListRow, error) {
	invoices := []models.InvoiceListRow{}

	tx := q.filteredInvoices(userID, status, search)

	sortColumn := "invoices.created_at"
	if column, ok := invoiceSortColumns[strings.ToLower(sort)]; ok {
		sortColumn = column
	}
	sortOrder := "DESC"
	if strings.ToLower(order) == "asc" {
		sortOrder = "ASC"
	}
	tx = tx.Order(sortColumn + " " + sortOrder).Order("invoices.created_at DESC")

	if err := tx.Limit(limit).Offset(offset).Scan(&invoices).Error; err != nil {
		return invoices, err
	}

	return invoices, nil
}

// CountInvoices returns the total invoices matching the list filters.
// It wraps the shared filter chain so the count can never diverge from
// the listing.
func (q *InvoiceQueries) CountInvoices(userID uuid.UUID, status, search string) (int64, error) {
	var total int64
	sub := q.filteredInvoices(userID, status, search).Select("invoices.id")
	if err := q.Table("(?) AS invoice_ids", sub).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// filteredInvoices builds the shared filter chain for invoice listing and
// counting so both stay in sync.
func (q *InvoiceQueries) filteredInvoices(userID uuid.UUID, status, search string) *gorm.DB {
	paidSubquery := q.Model(&models.Payment{}).
		Select("invoice_id, SUM(amount) AS paid").
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
		Where("invoices.user_id = ?", userID)

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

// ClientInvoiceRow struct to describe an invoice row for client detail.
type ClientInvoiceRow struct {
	ID            uuid.UUID  `db:"id"`
	InvoiceNumber string     `db:"invoice_number"`
	IssueDate     *time.Time `db:"issue_date"`
	DueDate       *time.Time `db:"due_date"`
	Total         float64    `db:"total"`
	Currency      string     `db:"currency"`
	Status        string     `db:"status"`
}

// ClientInvoices returns invoices of a client owned by a user.
func (q *InvoiceQueries) ClientInvoices(userID, clientID uuid.UUID) ([]ClientInvoiceRow, error) {
	rows := []ClientInvoiceRow{}

	if err := q.Model(&models.Invoice{}).
		Select("id, invoice_number, issue_date, due_date, total, currency, status").
		Where("user_id = ? AND client_id = ?", userID, clientID).
		Order("created_at DESC").
		Scan(&rows).Error; err != nil {
		return rows, err
	}

	return rows, nil
}

// GetInvoice returns one invoice of a user by ID.
func (q *InvoiceQueries) GetInvoice(userID, id uuid.UUID) (models.Invoice, error) {
	invoice := models.Invoice{}

	if err := q.Where("id = ? AND user_id = ?", id, userID).First(&invoice).Error; err != nil {
		return invoice, notFound(err)
	}

	return invoice, nil
}

// GetInvoiceItems returns lines of an invoice ordered by position.
func (q *InvoiceQueries) GetInvoiceItems(invoiceID uuid.UUID) ([]models.InvoiceItem, error) {
	items := []models.InvoiceItem{}

	if err := q.Where("invoice_id = ?", invoiceID).Order("position ASC").Find(&items).Error; err != nil {
		return items, err
	}

	return items, nil
}

// GetInvoicePayments returns payments recorded against an invoice.
func (q *InvoiceQueries) GetInvoicePayments(invoiceID uuid.UUID) ([]models.Payment, error) {
	payments := []models.Payment{}

	if err := q.Where("invoice_id = ?", invoiceID).Order("created_at ASC").Find(&payments).Error; err != nil {
		return payments, err
	}

	return payments, nil
}

// PaidAmount returns the total paid amount for an invoice.
func (q *InvoiceQueries) PaidAmount(invoiceID uuid.UUID) (float64, error) {
	var paid float64

	if err := q.Model(&models.Payment{}).
		Where("invoice_id = ?", invoiceID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&paid).Error; err != nil {
		return 0, err
	}

	return paid, nil
}

// CreateInvoice creates an invoice with items, reserving the invoice number.
// The sequence increment is atomic, so concurrent calls never produce duplicates.
func (q *InvoiceQueries) CreateInvoice(userID uuid.UUID, invoice *models.Invoice, items []models.InvoiceItem) error {
	return q.Transaction(func(tx *gorm.DB) error {
		settings := models.Settings{UserID: userID}
		if err := tx.Where("user_id = ?", userID).FirstOrCreate(
			&settings, models.Settings{UserID: userID},
		).Error; err != nil {
			return err
		}
		// Fill defaults on first creation.
		if settings.InvoicePrefix == "" {
			defaults := models.DefaultSettings(userID)
			settings.Currency = defaults.Currency
			settings.TaxRate = defaults.TaxRate
			settings.InvoicePrefix = defaults.InvoicePrefix
		}

		if err := tx.Model(&models.Settings{}).Where("user_id = ?", userID).
			UpdateColumn("invoice_seq", gorm.Expr("invoice_seq + 1")).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).First(&settings).Error; err != nil {
			return err
		}
		invoice.InvoiceNumber = fmt.Sprintf("%s%06d", settings.InvoicePrefix, settings.InvoiceSeq)

		if err := tx.Create(invoice).Error; err != nil {
			return err
		}
		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// UpdateInvoice replaces an invoice and its items.
func (q *InvoiceQueries) UpdateInvoice(userID uuid.UUID, invoice *models.Invoice, items []models.InvoiceItem) error {
	return q.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Invoice{}).Where("id = ? AND user_id = ?", invoice.ID, userID).
			Updates(map[string]interface{}{
				"updated_at": time.Now(),
				"client_id":  invoice.ClientID,
				"status":     invoice.Status,
				"issue_date": invoice.IssueDate,
				"due_date":   invoice.DueDate,
				"currency":   invoice.Currency,
				"tax_rate":   invoice.TaxRate,
				"discount":   invoice.Discount,
				"notes":      invoice.Notes,
				"terms":      invoice.Terms,
				"subtotal":   invoice.Subtotal,
				"tax_amount": invoice.TaxAmount,
				"total":      invoice.Total,
			}).Error; err != nil {
			return err
		}

		if err := tx.Where("invoice_id = ?", invoice.ID).Delete(&models.InvoiceItem{}).Error; err != nil {
			return err
		}
		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// UpdateInvoiceStatus updates only the status of an invoice.
func (q *InvoiceQueries) UpdateInvoiceStatus(userID, id uuid.UUID, status string) error {
	if err := q.Model(&models.Invoice{}).Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{
			"updated_at": time.Now(),
			"status":     status,
		}).Error; err != nil {
		return err
	}

	return nil
}

// DeleteInvoice deletes an invoice of a user.
func (q *InvoiceQueries) DeleteInvoice(userID, id uuid.UUID) error {
	if err := q.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Invoice{}).Error; err != nil {
		return err
	}

	return nil
}
