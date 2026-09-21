package middleware

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
)

// FiberMiddleware provide Fiber's built-in middlewares.
// See: https://docs.gofiber.io/api/middleware
func FiberMiddleware(a *fiber.App) {
	a.Use(
		// CORS: permissive default for local dev (vite proxies /api
		// same-origin, so credentials never cross origins). In production
		// set CORS_ORIGINS so cookie sessions work cross-origin.
		cors.New(corsConfig()),
		// Add simple logger.
		logger.New(),
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
