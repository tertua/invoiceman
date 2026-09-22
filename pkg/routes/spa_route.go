package routes

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

// MountSPA serves the built Vite SPA (web/dist) from the same origin as
// the API, so one container can host the whole app. It must be registered
// AFTER every API/special route: unknown /api/* keeps the JSON 404, only
// non-API misses fall back to index.html (client-side routing).
//
// Empty dir = no-op, so the default Dockerfile stays API-only. Set
// SERVE_SPA_DIR to the build output (Dockerfile.dev uses /spa).
func MountSPA(a *fiber.App, dir string) {
	if strings.TrimSpace(dir) == "" {
		return
	}
	index, err := filepath.Abs(filepath.Join(dir, "index.html"))
	if err != nil {
		return
	}
	if _, err := os.Stat(index); err != nil {
		return
	}
	a.Use("/assets", static.New(filepath.Join(dir, "assets")))
	a.Use(static.New(dir, static.Config{IndexNames: []string{"index.html"}}))
	a.Get("*", func(c fiber.Ctx) error {
		p := c.Path()
		if strings.HasPrefix(p, "/api/") || p == "/api" ||
			strings.HasPrefix(p, "/uploads/") || p == "/uploads" ||
			strings.HasPrefix(p, "/swagger") ||
			p == "/healthz" || p == "/readyz" || p == "/metrics" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": true,
				"msg":   "sorry, endpoint is not found",
			})
		}
		return c.SendFile(index)
	})
}
