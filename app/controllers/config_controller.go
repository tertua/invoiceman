package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/pkg/configs"
	"github.com/tertua/tupay/pkg/utils"
)

// AppName returns the configured application brand name.
func AppName() string {
	return configs.Get().AppName
}

// AppConfig returns public branding configuration for the MPA.
// @Description Get public application configuration.
// @Summary get app config
// @Tags Config
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /config [get]
func AppConfig(c fiber.Ctx) error {
	cfg := configs.Get()
	return utils.OK(c, fiber.StatusOK, fiber.Map{"appName": cfg.AppName, "allowRegistration": cfg.Auth.AllowRegistration, "oidcEnabled": cfg.OIDC.Active()})
}
