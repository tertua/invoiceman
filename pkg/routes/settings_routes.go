package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/app/controllers"
	"github.com/tertua/tupay/pkg/middleware"
)

// registerSettingsRoutes wires the per-user settings group: GET for any session user, PATCH and the logo upload owner-only (D10), plus the provider method picker list.
func registerSettingsRoutes(route fiber.Router) {
	route.Get("/settings", controllers.GetSettings)
	route.Patch("/settings", middleware.RequireOrgRole("owner"), controllers.UpdateSettings)
	route.Get("/settings/methods", controllers.ListSettingsMethods)
	route.Post("/settings/logo", middleware.RequireOrgRole("owner"), controllers.UploadLogo)
}
