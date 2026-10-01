package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/app/controllers"
	"github.com/tertua/tupay/pkg/middleware"
)

// registerPublicAuthRoutes wires the brute-forceable auth endpoints (all under
// the strict AuthLimiter) onto the public group.
func registerPublicAuthRoutes(route fiber.Router) {
	auth := middleware.AuthLimiter()
	route.Post("/auth/register", auth, controllers.Register)                      // register a new user
	route.Post("/auth/login", auth, controllers.Login)                            // auth, start session
	route.Post("/auth/forgot-password", auth, controllers.ForgotPassword)         // request password reset
	route.Post("/auth/reset-password", auth, controllers.ResetPassword)           // reset password with token
	route.Post("/auth/verify-email", auth, controllers.VerifyEmail)               // verify email + start session
	route.Post("/auth/verify-email/resend", auth, controllers.ResendVerification) // resend verification link
	registerOIDCRoutes(route)                                                     // OIDC SSO login + callback
}
