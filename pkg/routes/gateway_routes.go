package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/invoiceman/app/controllers"
	"github.com/tertua/invoiceman/pkg/middleware"
)

// GatewayRoutes registers the service-to-service relay endpoints.
// It must be registered BEFORE PrivateRoutes: the session AuthRequired
// group matches the /api prefix, so anything registered after it would
// be forced through cookie sessions. Service identity here comes only
// from the API key header (see GatewayAuth).
func GatewayRoutes(a *fiber.App) {
	gateway := a.Group("/api/gateway", middleware.GatewayAuth())
	gateway.Post("/intents", controllers.CreateIntent)
	gateway.Get("/intents/:order_id", controllers.GetIntent)
	gateway.Get("/transactions", controllers.ListMyTransactions)
	gateway.Get("/deliveries", controllers.ListMyDeliveries)
}
