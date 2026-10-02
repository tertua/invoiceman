package controllers

import (
	"errors"
	"time"

	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// buildInvoice computes invoice and item rows.
func buildInvoice(orgID, userID uuid.UUID, input *models.InvoiceInput) (*models.Invoice, []models.InvoiceItem, error) {
	issueDate, err := utils.ParseDate(input.IssueDate)
	if err != nil {
		return nil, nil, errors.New("invalid issue_date, expected YYYY-MM-DD")
	}
	dueDate, err := utils.ParseDate(input.DueDate)
	if err != nil {
		return nil, nil, errors.New("invalid due_date, expected YYYY-MM-DD")
	}

	clientID, err := resolveClient(input)
	if err != nil {
		return nil, nil, err
	}

	now := time.Now()
	invoice := &models.Invoice{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: &now,
		OrgID:     orgID,
		UserID:    userID,
		ClientID:  clientID,
		Status:    input.Status,
		IssueDate: issueDate,
		DueDate:   dueDate,
		Currency:  input.Currency,
		TaxRate:   input.TaxRate,
		Discount:  input.Discount,
		Notes:     input.Notes,
		Terms:     input.Terms,
		// Informational "how to pay" choice, not a gateway routing key.
		PaymentMethod: input.PaymentMethod,
	}

	items, err := addItemRows(invoice, input.Items)
	if err != nil {
		return nil, nil, err
	}
	applyInvoiceTotals(invoice)

	return invoice, items, nil
}

// addItemRows builds the item rows and accumulates the invoice subtotal, rejecting negative money values; both create paths run it.
func addItemRows(invoice *models.Invoice, entries []models.InvoiceItemInput) ([]models.InvoiceItem, error) {
	items := make([]models.InvoiceItem, 0, len(entries))
	for position, entry := range entries {
		if entry.Rate.IsNegative() || invoice.Discount.IsNegative() {
			return nil, errors.New("money values cannot be negative")
		}
		amount := models.DecimalFromFloat(entry.Quantity).Mul(entry.Rate)
		invoice.Subtotal = invoice.Subtotal.Add(amount)
		items = append(items, models.InvoiceItem{
			ID:          uuid.New(),
			InvoiceID:   invoice.ID,
			Description: entry.Description,
			Quantity:    entry.Quantity,
			Rate:        entry.Rate,
			Amount:      amount,
			Position:    position,
		})
	}
	return items, nil
}

// applyInvoiceTotals derives taxable, tax and total from the subtotal, the discount and the tax rate.
func applyInvoiceTotals(invoice *models.Invoice) {
	taxable := invoice.Subtotal.Sub(invoice.Discount)
	if taxable.IsNegative() {
		taxable = decimal.Zero
	}
	invoice.TaxAmount = taxable.Mul(models.DecimalFromFloat(invoice.TaxRate)).Div(decimal.NewFromInt(100))
	invoice.Total = taxable.Add(invoice.TaxAmount)
}

// isPaidLocked reports whether an invoice is effectively paid (stored paid or payments covering the total) and must be treated as immutable.
func isPaidLocked(status string, dueDate *time.Time, total, paid decimal.Decimal) bool {
	return models.ResolveEffectiveStatus(status, dueDate, total, paid, false) == models.InvoiceStatusPaid
}
