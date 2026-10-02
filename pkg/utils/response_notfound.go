package utils

import (
	"database/sql"
	"errors"

	"github.com/gofiber/fiber/v3"
)

// NotFoundOrFailed maps an "no rows" lookup error to 404 and any other error
// to 500, with messages shaped as "<resource> not found" / "failed to load <resource>".
// It writes the response and returns the same error from utils.Fail, so callers can
// simply `return utils.NotFoundOrFailed(c, err, "user")`.
func NotFoundOrFailed(c fiber.Ctx, err error, resource string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return Fail(c, fiber.StatusNotFound, resource+" not found", nil)
	}
	return Fail(c, fiber.StatusInternalServerError, "failed to load "+resource, nil)
}
