package queries

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"
	"gorm.io/gorm"
)

// PaymentQueries provides payment persistence.
type PaymentQueries struct {
	*gorm.DB
}

// ListPayments returns one page of non-voided payments over the org's invoices; voided rows stay for audit but stay invisible here.
func (q *PaymentQueries) ListPayments(orgID uuid.UUID, limit, offset int) ([]models.PaymentListRow, error) {
	payments := []models.PaymentListRow{}
	err := q.Table("payments").
		Select(`payments.id AS payment_id, payments.invoice_id, invoices.invoice_number,
			COALESCE(clients.name, '') AS client_name, invoices.currency AS invoice_currency,
			payments.amount, payments.method, payments.paid_on, payments.txn_id, payments.notes,
			payments.gateway_order_id`).
		Joins("JOIN invoices ON invoices.id = payments.invoice_id").
		Joins("LEFT JOIN clients ON clients.id = invoices.client_id").
		Where("payments.org_id = ? AND invoices.org_id = ? AND payments.voided_at IS NULL", orgID, orgID).
		Order("payments.paid_on DESC").Order("payments.created_at DESC").
		Limit(limit).Offset(offset).
		Scan(&payments).Error
	return payments, err
}

// CountPayments returns the total non-voided payments over the org's invoices.
func (q *PaymentQueries) CountPayments(orgID uuid.UUID) (int64, error) {
	var total int64
	err := q.Table("payments").
		Joins("JOIN invoices ON invoices.id = payments.invoice_id").
		Where("payments.org_id = ? AND invoices.org_id = ? AND payments.voided_at IS NULL", orgID, orgID).
		Count(&total).Error
	return total, err
}

// PaymentTotals holds payment aggregates for an org.
type PaymentTotals struct {
	Total     decimal.Decimal
	ThisMonth decimal.Decimal
}

// GetPaymentTotals returns all-time and current-month sums over the org's non-voided payments.
// Bounds come from utils.MonthBounds: paid_on is date-only (UTC midnight), so a
// local-zone bound would read differently in SQLite (text) than in PostgreSQL.
func (q *PaymentQueries) GetPaymentTotals(orgID uuid.UUID) (PaymentTotals, error) {
	totals := PaymentTotals{}
	base := q.Table("payments").
		Joins("JOIN invoices ON invoices.id = payments.invoice_id").
		Where("payments.org_id = ? AND invoices.org_id = ? AND payments.voided_at IS NULL", orgID, orgID)
	if err := base.Select("COALESCE(SUM(payments.amount), 0)").Scan(&totals.Total).Error; err != nil {
		return totals, err
	}
	monthStart, nextMonth := utils.MonthBounds(time.Now())
	monthTx := q.Table("payments").
		Joins("JOIN invoices ON invoices.id = payments.invoice_id").
		Where("payments.org_id = ? AND invoices.org_id = ? AND payments.voided_at IS NULL", orgID, orgID).
		Where("payments.paid_on >= ? AND payments.paid_on < ?", monthStart, nextMonth)
	var thisMonth decimal.Decimal
	if err := monthTx.Select("COALESCE(SUM(payments.amount), 0)").Scan(&thisMonth).Error; err != nil {
		return totals, err
	}
	totals.ThisMonth = thisMonth
	return totals, nil
}

// GetPayment returns a payment of the org.
func (q *PaymentQueries) GetPayment(orgID, id uuid.UUID) (models.Payment, error) {
	payment := models.Payment{}
	if err := q.Where("id = ? AND org_id = ?", id, orgID).First(&payment).Error; err != nil {
		return payment, notFound(err)
	}
	return payment, nil
}

// CreatePayment persists a payment; an unset OrgID rides as the creator UserID (rides-as), UserID stays audit.
func (q *PaymentQueries) CreatePayment(payment *models.Payment) error {
	return DoRetry(func() error {
		return rideOrg(q.DB, payment, &payment.OrgID, payment.UserID)
	})
}

// VoidPayment marks a payment void instead of deleting it: the row stays for audit, drops out of every balance/list/aggregate, and only the first void wins.
func (q *PaymentQueries) VoidPayment(orgID, id uuid.UUID, reason string) error {
	return DoRetry(func() error {
		res := q.Model(&models.Payment{}).
			Where("id = ? AND org_id = ? AND voided_at IS NULL", id, orgID).
			Updates(map[string]any{
				"voided_at":   time.Now(),
				"void_reason": reason,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

// CreatePaymentLink stores a public link for an invoice.
func (q *PaymentQueries) CreatePaymentLink(link *models.PaymentLink) error {
	return q.Create(link).Error
}

// GetPaymentLink returns a payment link by its public token.
func (q *PaymentQueries) GetPaymentLink(token string) (models.PaymentLink, error) {
	link := models.PaymentLink{}
	if err := q.Where("token = ?", token).First(&link).Error; err != nil {
		return link, notFound(err)
	}
	return link, nil
}

// GetPaymentLinkForInvoice returns the public link for an invoice, scoped by the invoice's org because payment_links carries no org_id column.
func (q *PaymentQueries) GetPaymentLinkForInvoice(invoiceID, orgID uuid.UUID) (models.PaymentLink, error) {
	link := models.PaymentLink{}
	if err := q.Table("payment_links").Select("payment_links.*").
		Joins("JOIN invoices ON invoices.id = payment_links.invoice_id").
		Where("payment_links.invoice_id = ? AND invoices.org_id = ?", invoiceID, orgID).First(&link).Error; err != nil {
		return link, notFound(err)
	}
	return link, nil
}
