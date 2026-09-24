package controllers

import (
	"database/sql"
	"errors"

	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// UpdateInvoiceStatus updates only the invoice status.
// Flipping sent/draft while a gateway transaction is live is rejected:
// it cannot cancel the Snap intent at the provider (we never call their
// Cancel API), so the customer could still pay and the webhook would
// settle against a status the user thought was dead.
// @Description Update invoice status.
// @Summary update invoice status
// @Tags Invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Param request body models.InvoiceStatusInput true "Update status payload"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
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

	existing, err := db.GetInvoice(userID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice", nil)
	}

	if err := guardInvoiceStatus(*db, userID, id, input.Status, existing.ClientID); err != nil {
		return failInvoiceRule(c, err)
	}

	// Reopening a money-paid invoice must go through voiding payments so
	// balances stay consistent. A manually-marked paid invoice with no
	// money attached may still be reopened to sent/draft.
	if paid, err := db.PaidAmount(id); err == nil {
		if isPaidLocked(existing.Status, existing.DueDate, existing.Total, paid) &&
			input.Status != models.InvoiceStatusPaid &&
			existing.Total.GreaterThan(decimal.Zero) && paid.GreaterThanOrEqual(existing.Total) {
			return utils.Fail(c, fiber.StatusUnprocessableEntity, "invoice is already paid", nil)
		}
	} else {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice payments", nil)
	}

	if err := db.UpdateInvoiceStatus(userID, id, input.Status); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to update invoice status", nil)
	}

	detail, err := invoiceDetail(*db, userID, id)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice", nil)
	}

	invalidateAggregates(c, userID)
	enqueueNotification(db, userID, models.NotifEventInvoiceStatusUpdated, "",
		invoiceNotifData(db, userID, id))
	return utils.OK(c, fiber.StatusOK, fiber.Map{"invoice": detail})
}
