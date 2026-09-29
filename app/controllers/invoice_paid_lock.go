package controllers

import (
	"fmt"

	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/platform/database"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// guardPaidReopen blocks reopening an invoice whose recorded payments already cover its total.
func guardPaidReopen(db database.Queries, id uuid.UUID, existing models.Invoice, targetStatus string) error {
	paid, err := db.PaidAmount(id)
	if err != nil {
		return fmt.Errorf("failed to load invoice payments: %w", err)
	}
	// Reopening a money-paid invoice must void payments first; a manually-paid invoice without money may reopen.
	moneyPaid := isPaidLocked(existing.Status, existing.DueDate, existing.Total, paid) &&
		existing.Total.GreaterThan(decimal.Zero) && paid.GreaterThanOrEqual(existing.Total)
	if moneyPaid && targetStatus != models.InvoiceStatusPaid {
		return ErrInvoicePaid
	}
	return nil
}
