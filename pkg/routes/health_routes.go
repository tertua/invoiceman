package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/app/controllers"
)

// HealthRoutes registers liveness/readiness probes at the root.
// It must be registered BEFORE auth/gateway groups: probes are public and
// carry no rate-limit so orchestrators are never throttled.
func HealthRoutes(a *fiber.App) {
	a.Get("/healthz", controllers.Health)
	a.Get("/readyz", controllers.Ready)
}
