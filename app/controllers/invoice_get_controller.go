package controllers

import (
	"github.com/tertua/tupay/pkg/utils"

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
	orgID, err := utils.CurrentOrgID(c)
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

	detail, err := invoiceDetail(*db, orgID, id)
	if err != nil {
		return utils.NotFoundOrFailed(c, err, "invoice")
	}

	return utils.OK(c, fiber.StatusOK, fiber.Map{"invoice": detail})
}
