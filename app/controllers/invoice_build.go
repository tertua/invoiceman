package controllers

import (
	"errors"
	"time"

	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// buildInvoice computes invoice and item rows.
func buildInvoice(userID uuid.UUID, input *models.InvoiceInput) (*models.Invoice, []models.InvoiceItem, error) {
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

// invoiceDetail loads the full invoice response for the frontend.
func invoiceDetail(db database.Queries, userID, id uuid.UUID) (fiber.Map, error) {
	invoice, err := db.GetInvoice(userID, id)
	if err != nil {
		return nil, err
	}

	items, err := db.GetInvoiceItems(id)
	if err != nil {
		return nil, err
	}
	itemMaps := make([]fiber.Map, 0, len(items))
	for _, item := range items {
		itemMaps = append(itemMaps, fiber.Map{
			"description": item.Description,
			"quantity":    item.Quantity,
			"rate":        item.Rate,
			"amount":      item.Amount,
		})
	}

	payments, err := db.GetInvoicePayments(id)
	if err != nil {
		return nil, err
	}
	paymentMaps := make([]fiber.Map, 0, len(payments))
	var paid decimal.Decimal
	for _, payment := range payments {
		paid = paid.Add(payment.Amount)
		paymentMaps = append(paymentMaps, fiber.Map{
			"id":       payment.ID,
			"amount":   payment.Amount,
			"paid_on":  utils.FormatDate(payment.PaidOn),
			"method":   payment.Method,
			"txn_id":   payment.TxnID,
			"can_void": payment.CanVoid(),
		})
	}

	var clientName, clientCompany, clientEmail string
	if invoice.ClientID != nil {
		if client, err := db.GetClient(userID, *invoice.ClientID); err == nil {
			clientName = client.Name
			clientCompany = client.Company
			clientEmail = client.Email
		}
	}

	// Expose the existing public payment link (if any) so the MPA can
	// render it persistently instead of keeping it in transient state.
	var paymentLink fiber.Map
	if link, err := db.GetPaymentLinkForInvoice(id, userID); err == nil {
		paymentLink = fiber.Map{"token": link.Token, "url": "/pay/" + link.Token}
	}

	return fiber.Map{
		"id":               invoice.ID,
		"invoice_number":   invoice.InvoiceNumber,
		"status":           invoice.Status,
		"effective_status": models.ResolveEffectiveStatus(invoice.Status, invoice.DueDate, invoice.Total, paid, db.PendingInvoiceIDs(userID)[id]),
		"payment_link":     paymentLink,
		"client_id":        invoice.ClientID,
		"client_name":      clientName,
		"client_company":   clientCompany,
		"client_email":     clientEmail,
		"issue_date":       utils.FormatDate(invoice.IssueDate),
		"due_date":         utils.FormatDate(invoice.DueDate),
		"currency":         invoice.Currency,
		"subtotal":         invoice.Subtotal,
		"discount":         invoice.Discount,
		"tax_rate":         invoice.TaxRate,
		"tax_amount":       invoice.TaxAmount,
		"total":            invoice.Total,
		"notes":            invoice.Notes,
		"terms":            invoice.Terms,
		"payment_method":   invoice.PaymentMethod,
		"items":            itemMaps,
		"payments":         paymentMaps,
		"paid_amount":      paid,
		"balance":          invoice.Total.Sub(paid),
	}, nil
}

// isPaidLocked reports whether an invoice is effectively paid (stored paid
// or payments covering the total) and must be treated as immutable.
func isPaidLocked(status string, dueDate *time.Time, total, paid decimal.Decimal) bool {
	return models.ResolveEffectiveStatus(status, dueDate, total, paid, false) == models.InvoiceStatusPaid
}
