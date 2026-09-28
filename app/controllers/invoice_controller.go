package controllers

import (
	"database/sql"
	"errors"

	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

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

	// Online invoices mint their payment link in the same request, so the
	// redirect target already shows it (see autoOnlinePaymentLink).
	autoOnlinePaymentLink(*db, userID, *invoice)

	detail, err := invoiceDetail(*db, userID, invoice.ID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice", nil)
	}

	invalidateAggregates(c, userID)
	enqueueNotification(db, userID, models.NotifEventInvoiceCreated, "",
		invoiceNotifData(db, userID, invoice.ID))
	return utils.OK(c, fiber.StatusCreated, fiber.Map{"invoice": detail})
}
