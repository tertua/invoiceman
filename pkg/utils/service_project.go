package utils

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/app/models"
)

// GatewayProjectKey is the context locals key holding the authenticated
// downstream project. It lives in pkg/utils (not pkg/middleware) so that
// app/controllers can read service identity without importing middleware
// wiring, keeping the app -> pkg/middleware direction free.
const GatewayProjectKey = "gatewayProject"

// CurrentServiceProject returns the project stored by GatewayAuth.
func CurrentServiceProject(c fiber.Ctx) (models.GatewayProject, error) {
	project, ok := c.Locals(GatewayProjectKey).(models.GatewayProject)
	if !ok || project.Slug == "" {
		return models.GatewayProject{}, errors.New("missing project")
	}
	return project, nil
}
