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

// GatewayProjectKey is the context locals key holding the authenticated project.
const GatewayProjectKey = "gatewayProject"

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
func CurrentProject(c fiber.Ctx) (models.GatewayProject, error) {
	project, ok := c.Locals(GatewayProjectKey).(models.GatewayProject)
	if !ok || project.Slug == "" {
		return models.GatewayProject{}, errors.New("missing project")
	}
	return project, nil
}
