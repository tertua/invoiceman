package queries

import (
	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
)

// GetInvoiceUnscoped loads one invoice by ID alone: these callers are authorized by possession of the payment-link token / gateway project, and the returned row carries OrgID for all further scoping.
func (q *InvoiceQueries) GetInvoiceUnscoped(invoiceID uuid.UUID) (models.Invoice, error) {
	invoice := models.Invoice{}

	if err := q.First(&invoice, "id = ?", invoiceID).Error; err != nil {
		return invoice, notFound(err)
	}

	return invoice, nil
}
