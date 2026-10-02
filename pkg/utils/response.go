package utils

import (
	"database/sql"
	"errors"

	"github.com/gofiber/fiber/v3"
)

// OK sends a success JSON response with the given data fields.
func OK(c fiber.Ctx, status int, data fiber.Map) error {
	return c.Status(status).JSON(data)
}

// Fail sends an error JSON response shaped for the frontend apiClient,
// which reads message from error.message and details from error.details.
func Fail(c fiber.Ctx, status int, message string, details any) error {
	errObject := fiber.Map{"message": message}
	if details != nil {
		errObject["details"] = details
	}
	return c.Status(status).JSON(fiber.Map{"error": errObject})
}

// ValidationFailed sends a 400 response for validator errors.
func ValidationFailed(c fiber.Ctx, err error) error {
	return Fail(c, fiber.StatusBadRequest, "validation failed", ValidatorErrors(err))
}

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
