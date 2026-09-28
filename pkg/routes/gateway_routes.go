package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/app/controllers"
	"github.com/tertua/tupay/pkg/middleware"
)

func GatewayRoutes(a *fiber.App) {
	GatewayRoutesAt(a, APILegacyPrefix)
}

// GatewayRoutesAt registers relay endpoints under prefix (see versioning.go); it must run BEFORE PrivateRoutes, whose session AuthRequired group would otherwise capture this prefix, and service identity here comes only from the API key header (see GatewayAuth).
func GatewayRoutesAt(a *fiber.App, prefix string) {
	gateway := a.Group(prefix+"/gateway", middleware.GatewayLimiter(), middleware.GatewayAuth())
	gateway.Post("/intents", middleware.Idempotency(middleware.GatewayIdempotencyScope), controllers.CreateIntent)
	gateway.Get("/methods", controllers.ListGatewayMethods)
	gateway.Post("/invoices", middleware.Idempotency(middleware.GatewayIdempotencyScope), controllers.CreateGatewayInvoice)
	gateway.Get("/intents/:order_id", controllers.GetIntent)
	gateway.Get("/transactions", controllers.ListMyTransactions)
	gateway.Get("/deliveries", controllers.ListMyDeliveries)
	admin := a.Group(prefix+"/admin/gateway", middleware.AuthRequired(), middleware.RequireCSRF(), middleware.RequireRoles("admin"))
	admin.Post("/projects", controllers.CreateProject)
	admin.Get("/projects", controllers.ListProjects)
	admin.Patch("/projects/:slug", controllers.UpdateProject)
	admin.Post("/projects/:slug/rotate-key", controllers.RotateProjectKey)
	admin.Post("/projects/:slug/rotate-secret", controllers.RotateProjectSecret)
	admin.Get("/transactions", controllers.ListAllTransactions)
	admin.Get("/settlement", controllers.GetSettlement)
	admin.Get("/deliveries", controllers.ListDeliveries)
	admin.Post("/deliveries/:id/retry", controllers.RetryDelivery)
}
