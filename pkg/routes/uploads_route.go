package routes

import (
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

// MountUploads serves public company logos for the local storage backend.
// Receipts are PRIVATE by contract (proxied through the ownership-checked
// GET /expenses/:id/receipt), so only /uploads/logos/* is mounted —
// /uploads/receipts/* matches no route and falls through to the JSON 404.
func MountUploads(a *fiber.App, storageDir string) error {
	logosDir := filepath.Join(storageDir, "logos")
	if err := os.MkdirAll(logosDir, 0o750); err != nil {
		return err
	}
	a.Use("/uploads/logos", static.New(logosDir, static.Config{
		NotFoundHandler: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": true,
				"msg":   "sorry, endpoint is not found",
			})
		},
	}))
	return nil
}
