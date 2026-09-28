package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/pkg/configs"
	"github.com/tertua/tupay/pkg/utils"

	jwtMiddleware "github.com/gofiber/contrib/v3/jwt"
)

// JWTProtected func for specify routes group with JWT authentication.
// See: https://github.com/gofiber/contrib/jwt
func JWTProtected() func(fiber.Ctx) error {
	// Create config for JWT authentication middleware.
	config := jwtMiddleware.Config{
		SigningKey:   jwtMiddleware.SigningKey{Key: []byte(configs.Get().JWT.Secret)},
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
