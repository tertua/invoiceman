package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"
)

// RequireRoles permits only authenticated users with one of the given roles.
// AuthRequired must run before this middleware.
func RequireRoles(allowed ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, err := utils.CurrentUserID(c)
		if err != nil {
			return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
		}
		db, err := database.OpenDBConnection()
		if err != nil {
			return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
		}
		user, err := db.GetUserByID(userID)
		if err != nil {
			return utils.Fail(c, fiber.StatusUnauthorized, "user not found", nil)
		}
		if !utils.HasRole(user.UserRole, allowed...) {
			return utils.Fail(c, fiber.StatusForbidden, "insufficient permissions", nil)
		}
		return c.Next()
	}
}
