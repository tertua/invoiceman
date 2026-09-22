package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/invoiceman/app/controllers"
	"github.com/tertua/invoiceman/pkg/middleware"
)

// PublicRoutes func for describe group of public routes.
//
// NOTE: limiters are attached per-route (not via Group("", ...)): in Fiber
// a subgroup with an empty prefix mounts its middleware on the parent
// prefix, which would throttle every /api route including private ones.
func PublicRoutes(a *fiber.App) {
	// Create routes group matching the frontend apiClient baseURL ("/api").
	route := a.Group("/api")

	// Brute-forceable auth endpoints get the strict limiter.
	auth := middleware.AuthLimiter()
	route.Post("/auth/register", auth, controllers.Register)              // register a new user
	route.Post("/auth/login", auth, controllers.Login)                    // auth, start session
	route.Post("/auth/forgot-password", auth, controllers.ForgotPassword) // request password reset
	route.Post("/auth/reset-password", auth, controllers.ResetPassword)   // reset password with token

	// Public payment pages (shared links, higher abuse potential).
	publicPay := middleware.PublicPayLimiter()
	route.Get("/public/pay/:token", publicPay, controllers.GetPublicPayment)
	route.Post("/public/pay/:token/transaction", publicPay, middleware.Idempotency(middleware.PublicPayIdempotencyScope), controllers.CreatePublicTransaction)
	route.Get("/public/pay/:token/status", publicPay, controllers.GetPublicPaymentStatus)
	route.Get("/public/gateway/config", publicPay, controllers.GatewayConfig)

	route.Get("/config", controllers.AppConfig) // public branding for the SPA

	// Provider webhooks (HMAC-verified, but still rate-limited per IP).
	webhooks := middleware.WebhookLimiter()
	// Single Midtrans notification URL for the whole account (central relay).
	route.Post("/webhooks/midtrans", webhooks, controllers.HandleMidtransWebhook)
	// Generic provider webhooks for current and future gateways (e.g. crypto).
	route.Post("/webhooks/:gateway", webhooks, controllers.HandleGatewayWebhook)
}
