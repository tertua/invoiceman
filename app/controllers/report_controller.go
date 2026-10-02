package controllers

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/cache"
)

// GetReports returns financial reports for the current organization.
// @Description Get financial reports.
// @Summary get financial reports
// @Tags Reports
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /reports [get]
func GetReports(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	db, ok := openDB(c)
	if !ok {
		return nil
	}
	report, err := cache.FetchJSON(c.Context(), cache.AggKey(orgID.String(), "reports", strings.TrimSpace(c.Query("currency"))),
		func(ctx context.Context) (models.Reports, error) {
			return db.GetReports(orgID, strings.TrimSpace(c.Query("currency")))
		})
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load reports", nil)
	}
	return utils.OK(c, fiber.StatusOK, newReportResponse(report))
}
