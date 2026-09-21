package controllers

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/invoiceman/pkg/utils"
)

// AppName returns the configured application brand name.
func AppName() string {
	name := strings.TrimSpace(os.Getenv("APP_NAME"))
	if name == "" {
		return "Invoiceman"
	}
	return name
}

// AppConfig returns public branding configuration for the SPA.
// @Description Get public application configuration.
// @Summary get app config
// @Tags Config
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /config [get]
func AppConfig(c fiber.Ctx) error {
	return utils.OK(c, fiber.StatusOK, fiber.Map{"appName": AppName()})
}
