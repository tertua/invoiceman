package controllers

import (
	"context"
	"strings"

	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/cache"

	"github.com/gofiber/fiber/v3"
)

// GetDashboard returns dashboard aggregates for the current organization.
// @Description Get dashboard aggregates.
// @Summary get dashboard aggregates
// @Tags Dashboard
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /dashboard [get]
func GetDashboard(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}

	db, ok := openDB(c)
	if !ok {
		return nil
	}

	currency := strings.TrimSpace(c.Query("currency"))
	ctx := c.Context()
	stats, err := cache.FetchJSON(ctx, cache.AggKey(orgID.String(), "dashboard:stats", currency),
		func(ctx context.Context) (models.DashboardStats, error) {
			return db.GetStats(orgID, currency)
		})
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load dashboard stats", nil)
	}

	series, err := cache.FetchJSON(ctx, cache.AggKey(orgID.String(), "dashboard:series", currency),
		func(ctx context.Context) ([]models.RevenuePoint, error) {
			return db.GetRevenueSeries(orgID, currency)
		})
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load revenue series", nil)
	}

	recent, err := cache.FetchJSON(ctx, cache.AggKey(orgID.String(), "dashboard:recent", currency),
		func(ctx context.Context) ([]models.RecentInvoice, error) {
			return db.GetRecentInvoices(orgID, currency)
		})
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load recent invoices", nil)
	}

	return utils.OK(c, fiber.StatusOK, dashboardResponse{
		Stats:          stats,
		RevenueSeries:  series,
		RecentInvoices: recent,
	})
}
