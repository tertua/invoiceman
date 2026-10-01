package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/app/controllers"
	"github.com/tertua/tupay/pkg/middleware"
)

// registerOIDCRoutes wires the SSO login endpoints. /login is brute-forceable
// (strict AuthLimiter); /callback must always be reachable once with a valid
// state, so it takes the looser public limiter (state single-use + TTL is the
// real protection).
func registerOIDCRoutes(route fiber.Router) {
	route.Get("/auth/oidc/login", middleware.AuthLimiter(), controllers.OIDCLogin)
	route.Get("/auth/oidc/callback", middleware.PublicPayLimiter(), controllers.OIDCCallback)
}
