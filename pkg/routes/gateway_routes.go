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
	registerGatewayAdminRoutes(a, prefix)
}
