package middleware

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/tertua/invoiceman/pkg/utils"
)

// FiberMiddleware provide Fiber's built-in middlewares.
// Order matters (outer -> inner):
// recover -> requestid -> helmet -> cors -> logger -> (route limiters).
// See: https://docs.gofiber.io/api/middleware
func FiberMiddleware(a *fiber.App) {
	a.Use(
		// Recover panics into a JSON 500 envelope instead of crashing.
		recover.New(recover.Config{
			PanicHandler: func(c fiber.Ctx, _ any) error {
				return utils.Fail(c, fiber.StatusInternalServerError, "internal server error", nil)
			},
		}),
		// Propagate X-Request-ID for log correlation.
		requestid.New(requestid.Config{
			Header: "X-Request-ID",
		}),
		// Security headers. CSP is intentionally empty: this service is a
		// pure JSON API, the SPA is served separately via Vite.
		helmet.New(helmet.Config{
			ContentSecurityPolicy: "",
		}),
		// CORS: permissive default for local dev (vite proxies /api
		// same-origin, so credentials never cross origins). In production
		// set CORS_ORIGINS so cookie sessions work cross-origin.
		cors.New(corsConfig()),
		// Request log with request id; skip health probes to avoid noise.
		logger.New(logger.Config{
			Format: "[${time}] ${ip} ${status} - ${latency} ${method} ${path} rid=${reqHeader:X-Request-ID} ${error}\n",
			Skip: func(c fiber.Ctx) bool {
				return c.Path() == "/healthz" || c.Path() == "/readyz"
			},
		}),
	)
}

// corsConfig allows credentialed requests only from the configured origins
// (CORS_ORIGINS, comma-separated). Empty CORS_ORIGINS keeps the permissive
// dev default.
func corsConfig() cors.Config {
	origins := []string{}
	for _, o := range strings.Split(strings.TrimSpace(os.Getenv("CORS_ORIGINS")), ",") {
		if o = strings.TrimSpace(o); o != "" {
			origins = append(origins, o)
		}
	}
	if len(origins) == 0 {
		return cors.Config{}
	}
	return cors.Config{
		AllowOrigins:     origins,
		AllowCredentials: true,
	}
}
