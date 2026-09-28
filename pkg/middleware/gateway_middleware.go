package middleware

import (
	"database/sql"
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"
	"github.com/tertua/invoiceman/platform/relay"
)

// GatewayAuth authenticates downstream projects by service API key.
// Identity is derived server-side from the key hash; client-supplied
// project fields are never trusted.
func GatewayAuth() fiber.Handler {
	return func(c fiber.Ctx) error {
		key := relay.ExtractKey(c.Get("Authorization"), c.Get("X-Api-Key"))
		if key == "" {
			return utils.Fail(c, fiber.StatusUnauthorized, "missing api key", nil)
		}
		db, err := database.OpenDBConnection()
		if err != nil {
			return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
		}
		project, err := db.GetProjectByKeyHash(relay.HashKey(key))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return utils.Fail(c, fiber.StatusUnauthorized, "invalid api key", nil)
			}
			return utils.Fail(c, fiber.StatusInternalServerError, "failed to load project", nil)
		}
		if !project.IsActive {
			return utils.Fail(c, fiber.StatusForbidden, "project is disabled", nil)
		}
		c.Locals(utils.GatewayProjectKey, project)
		return c.Next()
	}
}
