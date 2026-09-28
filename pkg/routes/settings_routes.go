package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/app/controllers"
	"github.com/tertua/tupay/pkg/middleware"
)

// registerSettingsRoutes wires the per-user settings group: GET for any
// session user, PATCH restricted to admin/user, the Midtrans method picker
// list, and the logo upload.
func registerSettingsRoutes(route fiber.Router) {
	route.Get("/settings", controllers.GetSettings)
	route.Patch("/settings", middleware.RequireRoles("admin", "user"), controllers.UpdateSettings)
	route.Get("/settings/methods", controllers.ListMidtransMethods)
	route.Post("/settings/logo", controllers.UploadLogo)
}
