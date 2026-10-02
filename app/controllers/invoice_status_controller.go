package controllers

import (
	"errors"

	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
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
	orgID, err := utils.CurrentOrgID(c)
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

	db, ok := openDB(c)
	if !ok {
		return nil
	}

	existing, err := db.GetInvoice(orgID, id)
	if err != nil {
		return utils.NotFoundOrFailed(c, err, "invoice")
	}

	if err := guardInvoiceStatus(c, *db, orgID, id, existing, input.Status); err != nil {
		return failInvoiceRule(c, err)
	}

	if err := guardPaidReopen(*db, id, existing, input.Status); err != nil {
		if errors.Is(err, ErrInvoicePaid) {
			return failInvoiceRule(c, err)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice payments", nil)
	}

	if err := db.UpdateInvoiceStatus(orgID, id, input.Status); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to update invoice status", nil)
	}

	detail, err := invoiceDetail(*db, orgID, id)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice", nil)
	}

	invalidateAggregates(c, orgID)
	enqueueOrgNotification(db, orgID, models.NotifEventInvoiceStatusUpdated, "",
		invoiceNotifData(db, orgID, id))
	return utils.OK(c, fiber.StatusOK, fiber.Map{"invoice": detail})
}
