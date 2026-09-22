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
// An empty sid mints a fresh session (login/register: kills any previous
// session for this user); a non-empty one keeps the session id so a
// transparent refresh never kicks sibling tabs sharing the session.
// The access cookie deliberately outlives the JWT inside it (crypto expiry
// stays short via JWT.AccessMinutes; the cookie lives as long as the
// refresh cookie). Browsers drop expired cookies, so without this the
// expired token hint would vanish after 15 minutes of idle and the
// transparent refresh in AuthRequired could never fire — forcing a
// re-login despite the long refresh TTL.
func IssueSession(c fiber.Ctx, userID uuid.UUID, sid string) (*Tokens, error) {
	tokens, err := GenerateNewTokens(userID.String(), sid)
	if err != nil {
		return nil, err
	}

	refreshHours := configs.Get().JWT.RefreshHours

	secure := !configs.Get().IsDev()

	c.Cookie(&fiber.Cookie{
		Name:     AccessCookieName,
		Value:    tokens.Access,
		Path:     "/",
		HTTPOnly: true,
		Secure:   secure,
		SameSite: "Lax",
		Expires:  time.Now().Add(time.Hour * time.Duration(refreshHours)),
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

// ClearSession removes session cookies. Flags mirror IssueSession:
// browsers treat a Secure cookie and its non-Secure twin as distinct, so
// omitting Secure here would leave the prod cookie behind after logout.
func ClearSession(c fiber.Ctx) {
	expired := time.Now().Add(-time.Hour)
	secure := !configs.Get().IsDev()
	for _, name := range []string{AccessCookieName, RefreshCookieName} {
		c.Cookie(&fiber.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			HTTPOnly: true,
			Secure:   secure,
			SameSite: "Lax",
			Expires:  expired,
		})
	}
}
