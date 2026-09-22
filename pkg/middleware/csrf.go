package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/invoiceman/pkg/configs"
	"github.com/tertua/invoiceman/pkg/utils"
)

// CSRF double-submit protection for cookie-session mutations.
//
// The server sets a csrf_token cookie (readable by JS) on login/register;
// every session-authenticated POST/PATCH/DELETE must echo it back in the
// X-CSRF-Token header. An attacker site cannot read the cookie (same-origin
// policy) and therefore cannot forge the header, while the SPA reads it
// trivially. API-key relay routes and public GETs are exempt (no cookies).
const (
	CSRFCookieName = "csrf_token"
	CSRFHeaderName = "X-CSRF-Token"
)

// NewCSRFToken generates one unguessable token per session.
func NewCSRFToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

// SetCSRFCookie writes the readable CSRF cookie (JS needs document.cookie
// access, so HttpOnly is false; Secure follows the session cookies).
func SetCSRFCookie(c fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		HTTPOnly: false,
		Secure:   !configs.Get().IsDev(),
		SameSite: "Lax",
	})
}

// ClearCSRFCookie removes the CSRF cookie on logout.
func ClearCSRFCookie(c fiber.Ctx) {
	c.Cookie(&fiber.Cookie{Name: CSRFCookieName, Value: "", Path: "/", SameSite: "Lax"})
}

// RequireCSRF rejects session-cookie mutations without a matching header.
// Safe methods pass through; requests without any session cookie (API-key
// relay, public pages) are not its concern and pass to their own auth.
func RequireCSRF() fiber.Handler {
	return func(c fiber.Ctx) error {
		switch c.Method() {
		case fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions:
			return c.Next()
		}
		if c.Cookies(utils.AccessCookieName) == "" && c.Cookies(utils.RefreshCookieName) == "" {
			return c.Next()
		}
		cookie := strings.TrimSpace(c.Cookies(CSRFCookieName))
		header := strings.TrimSpace(c.Get(CSRFHeaderName))
		if cookie == "" || header == "" || cookie != header {
			return utils.Fail(c, fiber.StatusForbidden, "csrf token missing or mismatched", nil)
		}
		return c.Next()
	}
}
