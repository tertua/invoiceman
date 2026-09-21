package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/invoiceman/app/controllers"
)

// PublicRoutes func for describe group of public routes.
func PublicRoutes(a *fiber.App) {
	// Create routes group matching the frontend apiClient baseURL ("/api").
	route := a.Group("/api")

	// Routes for POST method:
	route.Post("/auth/register", controllers.Register)              // register a new user
	route.Post("/auth/login", controllers.Login)                    // auth, start session
	route.Post("/auth/forgot-password", controllers.ForgotPassword) // request password reset
	route.Post("/auth/reset-password", controllers.ResetPassword)   // reset password with token
	route.Get("/public/pay/:token", controllers.GetPublicPayment)
	route.Post("/public/pay/:token/transaction", controllers.CreatePublicTransaction)
	route.Get("/public/pay/:token/status", controllers.GetPublicPaymentStatus)
	// Single Midtrans notification URL for the whole account (central relay).
	route.Post("/webhooks/midtrans", controllers.HandleMidtransWebhook)
	// Generic provider webhooks for current and future gateways (e.g. crypto).
	route.Post("/webhooks/:gateway", controllers.HandleGatewayWebhook)
	route.Get("/public/gateway/config", controllers.GatewayConfig)
}
