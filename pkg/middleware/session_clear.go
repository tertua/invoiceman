package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/pkg/utils"
)

// clearAuthCookies removes the session and CSRF cookies and returns the standard
// 401: a dead/absent session leaves stale cookies in the browser, so the 401
// response also cleans them up instead of letting the client keep replaying them.
func clearAuthCookies(c fiber.Ctx) error {
	utils.ClearSession(c)
	ClearCSRFCookie(c)
	return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
}
