package controllers

import (
	"database/sql"
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
	}

	items := make([]models.InvoiceItem, 0, len(input.Items))
	for position, entry := range input.Items {
		if entry.Rate.IsNegative() || input.Discount.IsNegative() {
			return nil, nil, errors.New("money values cannot be negative")
		}
		amount := models.MoneyFromFloat(entry.Quantity).Mul(entry.Rate)
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

	taxable := invoice.Subtotal.Sub(invoice.Discount)
	if taxable.IsNegative() {
		taxable = decimal.Zero
	}
	invoice.TaxAmount = taxable.Mul(models.MoneyFromFloat(invoice.TaxRate)).Div(decimal.NewFromInt(100))
	invoice.Total = taxable.Add(invoice.TaxAmount)

	return invoice, items, nil
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

	// Expose the existing public payment link (if any) so the SPA can
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

// ListInvoices returns one page of invoices of the current user.
// @Description Get invoices of current user.
// @Summary get invoices of current user
// @Tags Invoices
// @Accept json
// @Produce json
// @Param status query string false "Filter by status"
// @Param search query string false "Search by number or client"
// @Param sort query string false "Sort column"
// @Param order query string false "Sort order"
// @Param page query int false "Page number (default 1)"
// @Param per_page query int false "Items per page (default 20, max 100)"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /invoices [get]
func ListInvoices(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}

	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}

	status, search := c.Query("status"), c.Query("search")
	paging := utils.ParsePagination(c)
	rows, err := db.ListInvoices(
		userID,
		status,
		search,
		c.Query("sort"),
		c.Query("order"),
		paging.Limit(),
		paging.Offset(),
	)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoices", nil)
	}
	total, err := db.CountInvoices(userID, status, search)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to count invoices", nil)
	}

	invoices := make([]fiber.Map, 0, len(rows))
	pending := db.PendingInvoiceIDs(userID)
	for _, row := range rows {
		invoices = append(invoices, fiber.Map{
			"id":               row.ID,
			"invoice_number":   row.InvoiceNumber,
			"client_name":      row.ClientName,
			"client_company":   row.ClientCompany,
			"issue_date":       utils.FormatDate(row.IssueDate),
			"due_date":         utils.FormatDate(row.DueDate),
			"total":            row.Total,
			"currency":         row.Currency,
			"effective_status": row.EffectiveStatus(pending[row.ID]),
		})
	}

	return utils.OK(c, fiber.StatusOK, fiber.Map{"invoices": invoices, "meta": paging.Meta(total)})
}

// GetInvoice returns one invoice with items and payments.
// @Description Get invoice by ID.
// @Summary get invoice by ID
// @Tags Invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /invoices/{id} [get]
func GetInvoice(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid invoice id", nil)
	}

	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}

	detail, err := invoiceDetail(*db, userID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice", nil)
	}

	return utils.OK(c, fiber.StatusOK, fiber.Map{"invoice": detail})
}

// CreateInvoice creates a new invoice.
// @Description Create a new invoice.
// @Summary create a new invoice
// @Tags Invoices
// @Accept json
// @Produce json
// @Param request body models.InvoiceInput true "Create invoice payload"
// @Success 201 {object} map[string]interface{}
// @Security SessionCookie
// @Router /invoices [post]
func CreateInvoice(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}

	input := &models.InvoiceInput{}
	if err := c.Bind().Body(input); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
	}
	if err := utils.NewValidator().Struct(input); err != nil {
		return utils.ValidationFailed(c, err)
	}

	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}

	invoice, items, err := buildInvoice(userID, input)
	if err != nil {
		return failInvoiceRule(c, err)
	}

	if invoice.ClientID != nil {
		if _, err := db.GetClient(userID, *invoice.ClientID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return utils.Fail(c, fiber.StatusNotFound, "client not found", nil)
			}
			return utils.Fail(c, fiber.StatusInternalServerError, "failed to load client", nil)
		}
	}

	if err := db.CreateInvoice(userID, invoice, items); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to create invoice", nil)
	}

	detail, err := invoiceDetail(*db, userID, invoice.ID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice", nil)
	}

	invalidateAggregates(c, userID)
	enqueueNotification(db, userID, models.NotifEventInvoiceCreated, "",
		invoiceNotifData(db, userID, invoice.ID))
	return utils.OK(c, fiber.StatusCreated, fiber.Map{"invoice": detail})
}
