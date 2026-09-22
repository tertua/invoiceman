package middleware

import (
	"context"
	"errors"
	"time"

	"github.com/tertua/invoiceman/pkg/configs"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/cache"
	"github.com/tertua/invoiceman/platform/database"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AuthRequired protects routes with a cookie (or Bearer) JWT session.
// Expired access tokens are refreshed transparently via the refresh cookie,
// so the frontend never needs token handling logic.
func AuthRequired() fiber.Handler {
	return func(c fiber.Ctx) error {
		if userID, ok := validAccessToken(accessTokenString(c)); ok {
			c.Locals(utils.SessionUserIDKey, userID)
			return c.Next()
		}

		userID, ok := refreshSession(c)
		if !ok {
			return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
		}

		c.Locals(utils.SessionUserIDKey, userID)
		return c.Next()
	}
}

// accessTokenString reads the access token from cookie or Authorization header.
func accessTokenString(c fiber.Ctx) string {
	if token := c.Cookies(utils.AccessCookieName); token != "" {
		return token
	}
	if parts := splitBearer(c.Get("Authorization")); parts != "" {
		return parts
	}
	return ""
}

// validAccessToken parses and validates an access token.
func validAccessToken(tokenString string) (uuid.UUID, bool) {
	if tokenString == "" {
		return uuid.Nil, false
	}

	token, err := jwt.Parse(tokenString, jwtKeyFunc)
	if err != nil || !token.Valid {
		return uuid.Nil, false
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, false
	}

	rawID, ok := claims["id"].(string)
	if !ok {
		return uuid.Nil, false
	}
	userID, err := uuid.Parse(rawID)
	if err != nil {
		return uuid.Nil, false
	}

	return userID, true
}

// refreshSession issues a new token pair using the refresh cookie.
// The access cookie outlives the JWT inside it (see IssueSession), so the
// browser keeps sending the expired token as an identity hint long after
// its 15-minute cryptographic expiry — the refresh token then renews the
// session transparently without forcing a re-login.
func refreshSession(c fiber.Ctx) (uuid.UUID, bool) {
	accessString := accessTokenString(c)
	refreshString := c.Cookies(utils.RefreshCookieName)
	if refreshString == "" {
		return uuid.Nil, false
	}

	userID, ok := userIDFromAccess(accessString)
	if !ok {
		return uuid.Nil, false
	}

	// Check refresh token expiry.
	expiresRefresh, err := utils.ParseRefreshToken(refreshString)
	if err != nil || time.Now().Unix() >= expiresRefresh {
		return uuid.Nil, false
	}

	// Check refresh token against the session store.
	store, err := cache.Sessions()
	if err != nil {
		return uuid.Nil, false
	}
	stored, err := store.Get(context.Background(), userID.String())
	if err != nil || stored != refreshString {
		return uuid.Nil, false
	}

	// Check user still exists.
	db, err := database.OpenDBConnection()
	if err != nil {
		return uuid.Nil, false
	}
	if _, err := db.GetUserByID(userID); err != nil {
		return uuid.Nil, false
	}

	// Issue new tokens and persist the refresh token.
	tokens, err := utils.IssueSession(c, userID)
	if err != nil {
		return uuid.Nil, false
	}
	if err := store.Set(context.Background(), userID.String(), tokens.Refresh, cache.RefreshTTL()); err != nil {
		return uuid.Nil, false
	}

	return userID, true
}

// userIDFromAccess recovers the user ID from an access token without
// expiry validation. Only expiry is recoverable: malformed or badly-signed
// tokens always fail, and an empty input (no cookie/header at all) fails —
// the refresh token carries no identity, so there is nothing to look up.
func userIDFromAccess(accessString string) (uuid.UUID, bool) {
	if accessString == "" {
		return uuid.Nil, false
	}
	if _, err := jwt.Parse(accessString, jwtKeyFunc); err != nil {
		if !errors.Is(err, jwt.ErrTokenExpired) {
			return uuid.Nil, false
		}
	}
	parser := jwt.NewParser()
	unverified, _, err := parser.ParseUnverified(accessString, jwt.MapClaims{})
	if err != nil {
		return uuid.Nil, false
	}
	claims, ok := unverified.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, false
	}
	id, ok := claims["id"].(string)
	if !ok {
		return uuid.Nil, false
	}
	userID, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, false
	}
	return userID, true
}

// splitBearer extracts the token from a "Bearer <token>" header value.
func splitBearer(header string) string {
	const prefix = "Bearer "
	if len(header) > len(prefix) && header[:len(prefix)] == prefix {
		return header[len(prefix):]
	}
	return ""
}

// jwtKeyFunc returns the JWT signing key, accepting only HS256.
// Without the alg pin a token signed with another algorithm (e.g. "none"
// on a permissive parser) could slip through key confusion.
func jwtKeyFunc(token *jwt.Token) (interface{}, error) {
	if token.Method != jwt.SigningMethodHS256 {
		return nil, errors.New("unexpected signing method")
	}
	return []byte(configs.Get().JWT.Secret), nil
}
