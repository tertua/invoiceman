package controllers

import (
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"

	"github.com/gofiber/fiber/v3"
)

// GetDashboard returns dashboard aggregates for the current user.
// @Description Get dashboard aggregates.
// @Summary get dashboard aggregates
// @Tags Dashboard
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /dashboard [get]
func GetDashboard(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}

	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}

	stats, err := db.GetStats(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load dashboard stats", nil)
	}

	series, err := db.GetRevenueSeries(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load revenue series", nil)
	}

	recent, err := db.GetRecentInvoices(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load recent invoices", nil)
	}

	return utils.OK(c, fiber.StatusOK, fiber.Map{
		"stats":          stats,
		"revenueSeries":  series,
		"recentInvoices": recent,
	})
}
