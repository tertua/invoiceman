package controllers

import (
	"database/sql"
	"errors"

	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// DeleteInvoice deletes an invoice.
// Paid invoices are voided/reopened, never hard-deleted. Invoices with a
// live gateway transaction (effective status pending) are also locked:
// deleting while money is in flight would orphan the payment.
// @Description Delete an invoice.
// @Summary delete an invoice
// @Tags Invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Success 204 {string} status "ok"
// @Security SessionCookie
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

	existing, err := db.GetInvoice(userID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice", nil)
	}

	if paid, err := db.PaidAmount(id); err == nil {
		pending := db.PendingInvoiceIDs(userID)[id]
		if pending {
			return utils.Fail(c, fiber.StatusUnprocessableEntity, "invoice has a pending payment", nil)
		}
		if isPaidLocked(existing.Status, existing.DueDate, existing.Total, paid) {
			return utils.Fail(c, fiber.StatusUnprocessableEntity, "invoice is already paid", nil)
		}
	} else {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice payments", nil)
	}

	if err := db.DeleteInvoice(userID, id); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to delete invoice", nil)
	}
	recordAudit(c, db, userID, "invoice.delete", "invoice", id.String(), "")
	invalidateAggregates(c, userID)

	return c.SendStatus(fiber.StatusNoContent)
}
