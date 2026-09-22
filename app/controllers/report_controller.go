package controllers

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"
)

// GetReports returns financial reports for the current user.
// @Description Get financial reports.
// @Summary get financial reports
// @Tags Reports
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /reports [get]
func GetReports(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	report, err := db.GetReports(userID, strings.TrimSpace(c.Query("currency")))
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load reports", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{
		"totals":          report.Totals,
		"monthly":         report.Monthly,
		"aging":           report.Aging,
		"topClients":      report.TopClients,
		"statusBreakdown": report.StatusBreakdown,
	})
}
