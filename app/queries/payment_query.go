package queries

import (
	"github.com/google/uuid"
	"github.com/tertua/invoiceman/app/models"
	"gorm.io/gorm"
)

// PaymentQueries provides payment persistence operations.
type PaymentQueries struct {
	*gorm.DB
}

// ListPayments returns payments belonging to invoices owned by a user.
func (q *PaymentQueries) ListPayments(userID uuid.UUID) ([]models.PaymentListRow, error) {
	payments := []models.PaymentListRow{}
	err := q.Table("payments").
		Select(`payments.id AS payment_id, payments.invoice_id, invoices.invoice_number,
			COALESCE(clients.name, '') AS client_name, invoices.currency AS invoice_currency,
			payments.amount, payments.method, payments.paid_on, payments.txn_id, payments.notes`).
		Joins("JOIN invoices ON invoices.id = payments.invoice_id").
		Joins("LEFT JOIN clients ON clients.id = invoices.client_id").
		Where("payments.user_id = ? AND invoices.user_id = ?", userID, userID).
		Order("payments.paid_on DESC").Order("payments.created_at DESC").
		Scan(&payments).Error
	return payments, err
}

// GetPayment returns a payment owned by a user.
func (q *PaymentQueries) GetPayment(userID, id uuid.UUID) (models.Payment, error) {
	payment := models.Payment{}
	if err := q.Where("id = ? AND user_id = ?", id, userID).First(&payment).Error; err != nil {
		return payment, notFound(err)
	}
	return payment, nil
}

// CreatePayment persists a payment.
func (q *PaymentQueries) CreatePayment(payment *models.Payment) error {
	return q.Create(payment).Error
}

// DeletePayment deletes a payment owned by a user.
func (q *PaymentQueries) DeletePayment(userID, id uuid.UUID) error {
	return q.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Payment{}).Error
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

// GetPaymentLinkForInvoice returns the existing public link for an invoice.
func (q *PaymentQueries) GetPaymentLinkForInvoice(invoiceID, userID uuid.UUID) (models.PaymentLink, error) {
	link := models.PaymentLink{}
	if err := q.Where("invoice_id = ? AND user_id = ?", invoiceID, userID).First(&link).Error; err != nil {
		return link, notFound(err)
	}
	return link, nil
}
