package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/invoiceman/pkg/configs"
	"github.com/tertua/invoiceman/pkg/metrics"
)

// MetricsRoutes exposes the Prometheus-compatible scrape endpoint.
// It is registered alongside HealthRoutes (public, before auth) so
// orchestrators and scrapers never need a session. Scrape via a private
// network or LB in production; disable with METRICS_ENABLED=false.
func MetricsRoutes(a *fiber.App) {
	if !configs.Get().Metrics.Enabled {
		return
	}
	a.Get("/metrics", func(c fiber.Ctx) error {
		c.Set("Content-Type", "text/plain; version=0.0.4")
		return c.SendString(metrics.Shared().Render())
	})
}
