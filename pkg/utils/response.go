package utils

import (
	"github.com/gofiber/fiber/v3"
)

// OK sends a success JSON response with the given data fields.
func OK(c fiber.Ctx, status int, data fiber.Map) error {
	return c.Status(status).JSON(data)
}

// Fail sends an error JSON response shaped for the frontend apiClient,
// which reads message from error.message and details from error.details.
func Fail(c fiber.Ctx, status int, message string, details interface{}) error {
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
