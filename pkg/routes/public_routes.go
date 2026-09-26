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
	PublicRoutesAt(a, APILegacyPrefix)
}

// PublicRoutesAt registers public routes under prefix (see versioning.go).
func PublicRoutesAt(a *fiber.App, prefix string) {
	// Create routes group matching the frontend apiClient baseURL.
	route := a.Group(prefix)

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
	route.Get("/public/pay/:token/qr", publicPay, controllers.GetPublicQrImage)
	route.Get("/public/gateway/config", publicPay, controllers.GatewayConfig)
	route.Get("/public/gateway/status", publicPay, controllers.GatewayStatus)

	route.Get("/config", controllers.AppConfig) // public branding for the SPA

	registerWebhooks(route)
}
