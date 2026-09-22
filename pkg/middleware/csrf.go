package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/invoiceman/pkg/configs"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/cache"
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

// ClearCSRFCookie removes the CSRF cookie on logout. Flags mirror
// SetCSRFCookie (Secure included) so the prod cookie is actually cleared.
func ClearCSRFCookie(c fiber.Ctx) {
	c.Cookie(&fiber.Cookie{Name: CSRFCookieName, Value: "", Path: "/", Secure: !configs.Get().IsDev(), SameSite: "Lax"})
}

// RequireCSRF rejects session-cookie mutations without a matching header.
// Safe methods pass through; requests without any session cookie (API-key
// relay, public pages) are not its concern and pass to their own auth.
//
// Beyond the cookie/header match, the token is cross-checked against the
// session store binding: rotation on privilege moments (password/role
// change) rebinds the stored token, so a leaked old token is rejected even
// when attacker and victim pairs are self-consistent. Sessions predating
// the binding keep the pure double-submit check.
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
		if userID, _, ok := userIDFromAccess(accessTokenString(c)); ok {
			if !csrfBound(userID, cookie) {
				return utils.Fail(c, fiber.StatusForbidden, "csrf token missing or mismatched", nil)
			}
		}
		return c.Next()
	}
}

// csrfBound reports whether the presented token matches the session-store
// binding. Unbound (legacy) sessions pass — the cookie/header match above
// is their only check; their next transparent refresh binds them. A missing
// or unreadable store entry fails closed: AuthRequired just proved the
// session exists, so a miss means it died mid-request.
func csrfBound(userID uuid.UUID, token string) bool {
	store, err := cache.Sessions()
	if err != nil {
		return false
	}
	stored, err := store.Get(context.Background(), userID.String())
	if err != nil {
		return false
	}
	_, _, bound, ok := cache.DecodeSessionValue(stored)
	if !ok || bound == "" {
		return true
	}
	return bound == token
}
