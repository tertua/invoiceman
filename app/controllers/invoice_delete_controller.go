package controllers

import (
	"github.com/tertua/tupay/pkg/utils"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// DeleteInvoice deletes an invoice; paid or in-flight ones stay locked (hard-deleting money in flight orphans the payment) and a non-draft delete needs the owner role.
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
	orgID, userID, err := currentUserOrg(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid invoice id", nil)
	}

	db, ok := openDB(c)
	if !ok {
		return nil
	}

	existing, err := db.GetInvoice(orgID, id)
	if err != nil {
		return utils.NotFoundOrFailed(c, err, "invoice")
	}

	if err := guardInvoiceDelete(c, existing.Status); err != nil {
		return failInvoiceRule(c, err)
	}
	if paid, err := db.PaidAmount(id); err == nil {
		pending := db.PendingInvoiceIDs(orgID)[id]
		if pending {
			return utils.Fail(c, fiber.StatusUnprocessableEntity, "invoice has a pending payment", nil)
		}
		if isPaidLocked(existing.Status, existing.DueDate, existing.Total, paid) {
			return utils.Fail(c, fiber.StatusUnprocessableEntity, "invoice is already paid", nil)
		}
	} else {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice payments", nil)
	}

	if err := db.DeleteInvoice(orgID, id); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to delete invoice", nil)
	}
	recordAudit(c, db, userID, "invoice.delete", "invoice", id.String(), "")
	invalidateAggregates(c, orgID)

	return c.SendStatus(fiber.StatusNoContent)
}
