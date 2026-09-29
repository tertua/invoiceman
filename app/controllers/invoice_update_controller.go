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

// UpdateInvoice replaces an invoice.
// Paid invoices are immutable. Invoices with a live gateway transaction
// (effective status pending) are also locked: the Snap intent snapshots
// amount/total at creation, so editing items/totals mid-flight would
// settle the wrong amount against the provider's payment.
// @Description Update an invoice.
// @Summary update an invoice
// @Tags Invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Param request body models.InvoiceInput true "Update invoice payload"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /invoices/{id} [patch]
func UpdateInvoice(c fiber.Ctx) error {
	orgID, userID, err := currentUserOrg(c)
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

	existing, err := db.GetInvoice(orgID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice", nil)
	}

	if err := rejectLockedInvoice(*db, orgID, id, existing); err != nil {
		if errors.Is(err, ErrPendingPayment) || errors.Is(err, ErrInvoicePaid) {
			return failInvoiceRule(c, err)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice payments", nil)
	}

	invoice, items, err := buildInvoice(orgID, userID, input)
	if err != nil {
		return failInvoiceRule(c, err)
	}
	invoice.ID = existing.ID
	invoice.InvoiceNumber = existing.InvoiceNumber
	for i := range items {
		items[i].InvoiceID = existing.ID
	}

	if invoice.ClientID != nil {
		if _, err := db.GetClient(orgID, *invoice.ClientID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return utils.Fail(c, fiber.StatusNotFound, "client not found", nil)
			}
			return utils.Fail(c, fiber.StatusInternalServerError, "failed to load client", nil)
		}
	}

	if err := db.UpdateInvoice(orgID, invoice, items); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to update invoice", nil)
	}

	detail, err := invoiceDetail(*db, orgID, existing.ID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice", nil)
	}

	invalidateAggregates(c, orgID)
	if existing.Status != invoice.Status {
		enqueueOrgNotification(db, orgID, models.NotifEventInvoiceStatusUpdated, "",
			invoiceNotifData(db, orgID, existing.ID))
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"invoice": detail})
}
