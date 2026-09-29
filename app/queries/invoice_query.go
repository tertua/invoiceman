package queries

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tertua/tupay/app/models"
	"gorm.io/gorm"
)

// InvoiceQueries provides invoice queries.
type InvoiceQueries struct {
	*gorm.DB
}

// ClientInvoiceRow struct to describe an invoice row for client detail.
type ClientInvoiceRow struct {
	ID            uuid.UUID       `db:"id"`
	InvoiceNumber string          `db:"invoice_number"`
	IssueDate     *time.Time      `db:"issue_date"`
	DueDate       *time.Time      `db:"due_date"`
	Total         decimal.Decimal `db:"total"`
	Currency      string          `db:"currency"`
	Status        string          `db:"status"`
	PaidAmount    decimal.Decimal `db:"paid_amount"`
}

// EffectiveStatus resolves the display status for a client invoice row,
// matching the /invoices listing (paid via payments counts as paid).
func (r ClientInvoiceRow) EffectiveStatus(pending bool) string {
	return models.ResolveEffectiveStatus(r.Status, r.DueDate, r.Total, r.PaidAmount, pending)
}

// GetInvoice returns one invoice of an org by ID.
func (q *InvoiceQueries) GetInvoice(orgID, id uuid.UUID) (models.Invoice, error) {
	invoice := models.Invoice{}

	if err := q.Where("id = ? AND org_id = ?", id, orgID).First(&invoice).Error; err != nil {
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

// GetInvoicePayments returns non-voided payments recorded against an
// invoice. Voided rows stay in the table for audit but are invisible here.
func (q *InvoiceQueries) GetInvoicePayments(invoiceID uuid.UUID) ([]models.Payment, error) {
	payments := []models.Payment{}

	if err := q.Where("invoice_id = ? AND voided_at IS NULL", invoiceID).Order("created_at ASC").Find(&payments).Error; err != nil {
		return payments, err
	}

	return payments, nil
}

// PaidAmount returns the total non-voided paid amount for an invoice.
func (q *InvoiceQueries) PaidAmount(invoiceID uuid.UUID) (decimal.Decimal, error) {
	var paid decimal.Decimal

	if err := q.Model(&models.Payment{}).
		Where("invoice_id = ? AND voided_at IS NULL", invoiceID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&paid).Error; err != nil {
		return decimal.Zero, err
	}

	return paid, nil
}

// CreateInvoice creates an invoice with items, reserving the invoice number.
// The sequence increment is atomic, so concurrent calls never produce duplicates.
func (q *InvoiceQueries) CreateInvoice(orgID uuid.UUID, invoice *models.Invoice, items []models.InvoiceItem) error {
	return DoRetry(func() error {
		return q.Transaction(func(tx *gorm.DB) error {
			number, err := q.ReserveInvoiceNumber(tx, orgID)
			if err != nil {
				return err
			}
			invoice.InvoiceNumber = number

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
	})
}

// UpdateInvoice replaces an invoice and its items.
func (q *InvoiceQueries) UpdateInvoice(orgID uuid.UUID, invoice *models.Invoice, items []models.InvoiceItem) error {
	return DoRetry(func() error {
		return q.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&models.Invoice{}).Where("id = ? AND org_id = ?", invoice.ID, orgID).
				Updates(map[string]any{
					"updated_at":     time.Now(),
					"client_id":      invoice.ClientID,
					"status":         invoice.Status,
					"issue_date":     invoice.IssueDate,
					"due_date":       invoice.DueDate,
					"currency":       invoice.Currency,
					"tax_rate":       invoice.TaxRate,
					"discount":       invoice.Discount,
					"notes":          invoice.Notes,
					"terms":          invoice.Terms,
					"payment_method": invoice.PaymentMethod,
					"subtotal":       invoice.Subtotal,
					"tax_amount":     invoice.TaxAmount,
					"total":          invoice.Total,
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
	})
}

// UpdateInvoiceStatus updates only the status of an invoice.
func (q *InvoiceQueries) UpdateInvoiceStatus(orgID, id uuid.UUID, status string) error {
	if err := q.Model(&models.Invoice{}).Where("id = ? AND org_id = ?", id, orgID).
		Updates(map[string]any{
			"updated_at": time.Now(),
			"status":     status,
		}).Error; err != nil {
		return err
	}

	return nil
}

// DeleteInvoice deletes an invoice of an org.
func (q *InvoiceQueries) DeleteInvoice(orgID, id uuid.UUID) error {
	return DoRetry(func() error {
		return q.Where("id = ? AND org_id = ?", id, orgID).Delete(&models.Invoice{}).Error
	})
}
