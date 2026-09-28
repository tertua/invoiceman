package routes

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

// MountSPA serves the built Vite entries in dir (SERVE_SPA_DIR, empty = API-only): index.html for the product app and admin.html under /admin — registered after every API/special route so unknown /api/* keeps the JSON 404.
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
	admin := filepath.Join(dir, "admin.html") // pre-MPA dist has no admin entry
	if _, err := os.Stat(admin); err != nil {
		admin = ""
	}
	a.Use("/assets", static.New(filepath.Join(dir, "assets")))
	a.Use(static.New(dir, static.Config{IndexNames: []string{"index.html"}}))
	a.Get("*", func(c fiber.Ctx) error {
		p := c.Path()
		if strings.HasPrefix(p, "/api/") || p == "/api" ||
			strings.HasPrefix(p, "/uploads/") || p == "/uploads" ||
			strings.HasPrefix(p, "/swagger") ||
			p == "/healthz" || p == "/readyz" || p == "/metrics" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": true, "msg": "sorry, endpoint is not found"})
		}
		if admin != "" && (p == "/admin" || strings.HasPrefix(p, "/admin/")) {
			return c.SendFile(admin)
		}
		return c.SendFile(index)
	})
}
