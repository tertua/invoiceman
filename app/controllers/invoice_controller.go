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
)

// dateLayout is the date format exchanged with the frontend.
const dateLayout = "2006-01-02"

// parseDate parses an optional YYYY-MM-DD date string.
func parseDate(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(dateLayout, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

// formatDate formats an optional date for the frontend.
func formatDate(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(dateLayout)
}

// buildInvoice computes invoice and item rows from user input.
func buildInvoice(userID uuid.UUID, input *models.InvoiceInput) (*models.Invoice, []models.InvoiceItem, error) {
	issueDate, err := parseDate(input.IssueDate)
	if err != nil {
		return nil, nil, errors.New("invalid issue_date, expected YYYY-MM-DD")
	}
	dueDate, err := parseDate(input.DueDate)
	if err != nil {
		return nil, nil, errors.New("invalid due_date, expected YYYY-MM-DD")
	}

	var clientID *uuid.UUID
	if input.ClientID != nil && *input.ClientID != "" {
		parsed, err := uuid.Parse(*input.ClientID)
		if err != nil {
			return nil, nil, errors.New("invalid client_id")
		}
		clientID = &parsed
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
		amount := entry.Quantity * entry.Rate
		invoice.Subtotal += amount
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

	taxable := invoice.Subtotal - invoice.Discount
	if taxable < 0 {
		taxable = 0
	}
	invoice.TaxAmount = taxable * invoice.TaxRate / 100
	invoice.Total = taxable + invoice.TaxAmount

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
	var paid float64
	for _, payment := range payments {
		paid += payment.Amount
		paymentMaps = append(paymentMaps, fiber.Map{
			"id":      payment.ID,
			"amount":  payment.Amount,
			"paid_on": formatDate(payment.PaidOn),
			"method":  payment.Method,
			"txn_id":  payment.TxnID,
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

	return fiber.Map{
		"id":               invoice.ID,
		"invoice_number":   invoice.InvoiceNumber,
		"status":           invoice.Status,
		"effective_status": models.ResolveEffectiveStatus(invoice.Status, invoice.DueDate, invoice.Total, paid),
		"client_id":        invoice.ClientID,
		"client_name":      clientName,
		"client_company":   clientCompany,
		"client_email":     clientEmail,
		"issue_date":       formatDate(invoice.IssueDate),
		"due_date":         formatDate(invoice.DueDate),
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
		"balance":          invoice.Total - paid,
	}, nil
}

// ListInvoices returns invoices of the current user.
// @Description Get invoices of current user.
// @Summary get invoices of current user
// @Tags Invoices
// @Accept json
// @Produce json
// @Param status query string false "Filter by status"
// @Param search query string false "Search by number or client"
// @Param sort query string false "Sort column"
// @Param order query string false "Sort order"
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
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

	rows, err := db.ListInvoices(
		userID,
		c.Query("status"),
		c.Query("search"),
		c.Query("sort"),
		c.Query("order"),
	)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoices", nil)
	}

	invoices := make([]fiber.Map, 0, len(rows))
	for _, row := range rows {
		invoices = append(invoices, fiber.Map{
			"id":               row.ID,
			"invoice_number":   row.InvoiceNumber,
			"client_name":      row.ClientName,
			"client_company":   row.ClientCompany,
			"issue_date":       formatDate(row.IssueDate),
			"due_date":         formatDate(row.DueDate),
			"total":            row.Total,
			"currency":         row.Currency,
			"effective_status": row.EffectiveStatus(),
		})
	}

	return utils.OK(c, fiber.StatusOK, fiber.Map{"invoices": invoices})
}

// GetInvoice returns one invoice with items and payments.
// @Description Get invoice by ID.
// @Summary get invoice by ID
// @Tags Invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
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
// @Security ApiKeyAuth
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
		return utils.Fail(c, fiber.StatusBadRequest, err.Error(), nil)
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

	return utils.OK(c, fiber.StatusCreated, fiber.Map{"invoice": detail})
}

// UpdateInvoice replaces an invoice.
// @Description Update an invoice.
// @Summary update an invoice
// @Tags Invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Param request body models.InvoiceInput true "Update invoice payload"
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /invoices/{id} [patch]
func UpdateInvoice(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid invoice id", nil)
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

	existing, err := db.GetInvoice(userID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice", nil)
	}

	invoice, items, err := buildInvoice(userID, input)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, err.Error(), nil)
	}
	invoice.ID = existing.ID
	invoice.InvoiceNumber = existing.InvoiceNumber
	for i := range items {
		items[i].InvoiceID = existing.ID
	}

	if invoice.ClientID != nil {
		if _, err := db.GetClient(userID, *invoice.ClientID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return utils.Fail(c, fiber.StatusNotFound, "client not found", nil)
			}
			return utils.Fail(c, fiber.StatusInternalServerError, "failed to load client", nil)
		}
	}

	if err := db.UpdateInvoice(userID, invoice, items); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to update invoice", nil)
	}

	detail, err := invoiceDetail(*db, userID, existing.ID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice", nil)
	}

	return utils.OK(c, fiber.StatusOK, fiber.Map{"invoice": detail})
}

// UpdateInvoiceStatus updates only the invoice status.
// @Description Update invoice status.
// @Summary update invoice status
// @Tags Invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Param request body models.InvoiceStatusInput true "Update status payload"
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /invoices/{id}/status [patch]
func UpdateInvoiceStatus(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid invoice id", nil)
	}

	input := &models.InvoiceStatusInput{}
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

	if _, err := db.GetInvoice(userID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice", nil)
	}

	if err := db.UpdateInvoiceStatus(userID, id, input.Status); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to update invoice status", nil)
	}

	detail, err := invoiceDetail(*db, userID, id)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice", nil)
	}

	return utils.OK(c, fiber.StatusOK, fiber.Map{"invoice": detail})
}

// DeleteInvoice deletes an invoice.
// @Description Delete an invoice.
// @Summary delete an invoice
// @Tags Invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Success 204 {string} status "ok"
// @Security ApiKeyAuth
// @Router /invoices/{id} [delete]
func DeleteInvoice(c fiber.Ctx) error {
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

	if _, err := db.GetInvoice(userID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice", nil)
	}

	if err := db.DeleteInvoice(userID, id); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to delete invoice", nil)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
