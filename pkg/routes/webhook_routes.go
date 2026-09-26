package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/invoiceman/app/controllers"
	"github.com/tertua/invoiceman/pkg/middleware"
)

// registerWebhooks mounts the provider webhook endpoint on a route group.
// Split out of public_routes.go so that group stays within its size ratchet.
// One generic handler serves every provider: POST /webhooks/midtrans and
// POST /webhooks/nowpayments both resolve here, so per-provider URLs are
// stable without duplicate routes.
func registerWebhooks(route fiber.Router) {
	webhooks := middleware.WebhookLimiter()
	route.Post("/webhooks/:gateway", webhooks, controllers.HandleGatewayWebhook)
}
