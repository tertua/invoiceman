package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/pkg/utils"
)

// InvoiceStatusCounts returns the number of invoices per status for the
// current org, powering the count badges on the invoice status tabs.
// @Description Get invoice counts per status.
// @Summary get invoice counts per status
// @Tags Invoices
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /invoices/status-counts [get]
func InvoiceStatusCounts(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	db, ok := openDB(c)
	if !ok {
		return nil
	}
	counts, err := db.InvoiceStatusCounts(orgID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice counts", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"counts": counts})
}
