package controllers

import (
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

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
	orgID, userID, err := currentUserOrg(c)
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
	if err := guardInvoiceCreate(c, input.Status); err != nil {
		return failInvoiceRule(c, err)
	}

	db, ok := openDB(c)
	if !ok {
		return nil
	}

	invoice, items, err := buildInvoice(orgID, userID, input)
	if err != nil {
		return failInvoiceRule(c, err)
	}

	if status, message, ok := orgClientStatus(*db, orgID, invoice.ClientID); ok {
		return utils.Fail(c, status, message, nil)
	}

	if err := db.CreateInvoice(orgID, invoice, items); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to create invoice", nil)
	}

	// Online invoices mint their payment link in the same request, so the redirect target already shows it (see autoOnlinePaymentLink).
	autoOnlinePaymentLink(*db, orgID, *invoice)

	detail, err := invoiceDetail(*db, orgID, invoice.ID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice", nil)
	}

	invalidateAggregates(c, orgID)
	enqueueOrgNotification(db, orgID, models.NotifEventInvoiceCreated, "",
		invoiceNotifData(db, orgID, invoice.ID))
	return utils.OK(c, fiber.StatusCreated, fiber.Map{"invoice": detail})
}
