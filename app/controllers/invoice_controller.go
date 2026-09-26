package controllers

import (
	"database/sql"
	"errors"

	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

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
