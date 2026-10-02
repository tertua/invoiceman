package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/pkg/utils"
)

// GetSettlement returns gateway transaction counts and summed IDR amounts grouped by status.
// @Description Settlement summary grouped by transaction status.
// @Summary settlement summary
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /admin/gateway/settlement [get]
func GetSettlement(c fiber.Ctx) error {
	db, ok := openDB(c)
	if !ok {
		return nil
	}
	rows, err := db.SettlementSummary()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load settlement summary", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"summary": rows})
}
