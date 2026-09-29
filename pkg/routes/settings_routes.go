package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/app/controllers"
	"github.com/tertua/tupay/pkg/middleware"
)

// registerSettingsRoutes wires the per-user settings group: GET for any session user, PATCH owner-only (D10), the Midtrans method picker list, and the logo upload.
func registerSettingsRoutes(route fiber.Router) {
	route.Get("/settings", controllers.GetSettings)
	route.Patch("/settings", middleware.RequireOrgRole("owner"), controllers.UpdateSettings)
	route.Get("/settings/methods", controllers.ListMidtransMethods)
	route.Post("/settings/logo", controllers.UploadLogo)
}
