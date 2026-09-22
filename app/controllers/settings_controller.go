package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"
)

// GetSettings returns settings for the current user.
// @Description Get settings of current user.
// @Summary get current user settings
// @Tags Settings
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /settings [get]
func GetSettings(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	settings, err := db.GetSettings(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load settings", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"settings": settings})
}

// UpdateSettings updates settings for the current user.
// @Description Update settings of current user.
// @Summary update current user settings
// @Tags Settings
// @Accept json
// @Produce json
// @Param request body models.SettingsInput true "Settings payload"
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /settings [patch]
func UpdateSettings(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	input := &models.SettingsInput{}
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
	settings, err := db.GetSettings(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load settings", nil)
	}
	settings.CompanyName = input.CompanyName
	settings.Email = input.Email
	settings.Phone = input.Phone
	settings.Address = input.Address
	settings.LogoURL = input.LogoURL
	settings.Currency = input.Currency
	settings.TaxRate = input.TaxRate
	settings.InvoicePrefix = input.InvoicePrefix
	if err := db.UpdateSettings(&settings); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to update settings", nil)
	}
	recordAudit(c, db, userID, "settings.update", "settings", userID.String(), "")
	settings, err = db.GetSettings(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load updated settings", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"settings": settings})
}
