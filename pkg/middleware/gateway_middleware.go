package middleware

import (
	"database/sql"
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"
	"github.com/tertua/invoiceman/platform/relay"
)

// GatewayProjectKey aliases the canonical key in pkg/utils so existing
// callers keep compiling; new code should use utils.GatewayProjectKey.
const GatewayProjectKey = utils.GatewayProjectKey

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
		c.Locals(GatewayProjectKey, project)
		return c.Next()
	}
}

// CurrentProject returns the project stored by GatewayAuth.
// Deprecated: prefer utils.CurrentServiceProject to keep app/ free of
// middleware imports; kept here for backward compatibility.
func CurrentProject(c fiber.Ctx) (models.GatewayProject, error) {
	return utils.CurrentServiceProject(c)
}
