package middleware

import (
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/invoiceman/pkg/utils"

	jwtMiddleware "github.com/gofiber/contrib/v3/jwt"
)

// JWTProtected func for specify routes group with JWT authentication.
// See: https://github.com/gofiber/contrib/jwt
func JWTProtected() func(fiber.Ctx) error {
	// Create config for JWT authentication middleware.
	config := jwtMiddleware.Config{
		SigningKey:   jwtMiddleware.SigningKey{Key: []byte(os.Getenv("JWT_SECRET_KEY"))},
		ErrorHandler: jwtError,
	}

	return jwtMiddleware.New(config)
}

func jwtError(c fiber.Ctx, err error) error {
	// Keep the shared {"error":{"message","details"}} envelope so the
	// frontend apiClient interceptor reads the server message correctly.
	if err.Error() == "Missing or malformed JWT" {
		return utils.Fail(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	// Return status 401 and failed authentication error.
	return utils.Fail(c, fiber.StatusUnauthorized, err.Error(), nil)
}
