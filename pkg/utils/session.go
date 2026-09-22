package utils

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/invoiceman/pkg/configs"
)

// Session cookie names.
const (
	AccessCookieName  = "access_token"
	RefreshCookieName = "refresh_token"
)

// SessionUserIDKey is the context locals key holding the authenticated user ID.
const SessionUserIDKey = "userID"

// CurrentUserID returns the authenticated user ID stored by the auth middleware.
func CurrentUserID(c fiber.Ctx) (uuid.UUID, error) {
	value, ok := c.Locals(SessionUserIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "unauthorized, missing session")
	}
	return value, nil
}

// IssueSession generates a new token pair and stores it in HttpOnly cookies.
func IssueSession(c fiber.Ctx, userID uuid.UUID) (*Tokens, error) {
	tokens, err := GenerateNewTokens(userID.String(), nil)
	if err != nil {
		return nil, err
	}

	accessMinutes := configs.Get().JWT.AccessMinutes
	refreshHours := configs.Get().JWT.RefreshHours

	secure := !configs.Get().IsDev()

	c.Cookie(&fiber.Cookie{
		Name:     AccessCookieName,
		Value:    tokens.Access,
		Path:     "/",
		HTTPOnly: true,
		Secure:   secure,
		SameSite: "Lax",
		Expires:  time.Now().Add(time.Minute * time.Duration(accessMinutes)),
	})
	c.Cookie(&fiber.Cookie{
		Name:     RefreshCookieName,
		Value:    tokens.Refresh,
		Path:     "/",
		HTTPOnly: true,
		Secure:   secure,
		SameSite: "Lax",
		Expires:  time.Now().Add(time.Hour * time.Duration(refreshHours)),
	})

	return tokens, nil
}

// ClearSession removes session cookies.
func ClearSession(c fiber.Ctx) {
	expired := time.Now().Add(-time.Hour)
	for _, name := range []string{AccessCookieName, RefreshCookieName} {
		c.Cookie(&fiber.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			HTTPOnly: true,
			SameSite: "Lax",
			Expires:  expired,
		})
	}
}
