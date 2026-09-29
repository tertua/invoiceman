package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/app/controllers"
)

// registerNotificationRoutes wires the notification webhook routes (user-owned targets, e.g. n8n).
func registerNotificationRoutes(route fiber.Router) {
	route.Get("/notifications/endpoints", controllers.ListEndpoints)
	route.Post("/notifications/endpoints", controllers.CreateEndpoint)
	route.Patch("/notifications/endpoints/:id", controllers.UpdateEndpoint)
	route.Delete("/notifications/endpoints/:id", controllers.DeleteEndpoint)
	route.Post("/notifications/endpoints/:id/rotate-secret", controllers.RotateEndpointSecret)
	route.Post("/notifications/endpoints/:id/test", controllers.TestEndpoint)
	route.Get("/notifications/deliveries", controllers.ListNotificationDeliveries)
	route.Post("/notifications/deliveries/:id/retry", controllers.RetryNotificationDelivery)
}
