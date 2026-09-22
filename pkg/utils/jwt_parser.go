package utils

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/tertua/invoiceman/pkg/configs"
)

// TokenMetadata struct to describe metadata in JWT.
type TokenMetadata struct {
	UserID      uuid.UUID
	Credentials map[string]bool
	Expires     int64
}

// ExtractTokenMetadata func to extract metadata from JWT.
func ExtractTokenMetadata(c fiber.Ctx) (*TokenMetadata, error) {
	token, err := verifyToken(c)
	if err != nil {
		return nil, err
	}

	// Setting and checking token and credentials.
	claims, ok := token.Claims.(jwt.MapClaims)
	if ok && token.Valid {
		// User ID.
		rawID, ok := claims["id"].(string)
		if !ok {
			return nil, err
		}
		userID, err := uuid.Parse(rawID)
		if err != nil {
			return nil, err
		}

		// Expires time.
		expFloat, ok := claims["exp"].(float64)
		if !ok {
			return nil, err
		}
		expires := int64(expFloat)

		// User credentials.
		bookCreate, _ := claims["book:create"].(bool)
		bookUpdate, _ := claims["book:update"].(bool)
		bookDelete, _ := claims["book:delete"].(bool)
		credentials := map[string]bool{
			"book:create": bookCreate,
			"book:update": bookUpdate,
			"book:delete": bookDelete,
		}

		return &TokenMetadata{
			UserID:      userID,
			Credentials: credentials,
			Expires:     expires,
		}, nil
	}

	return nil, err
}

func extractToken(c fiber.Ctx) string {
	// Cookie session first (set by IssueSession).
	if token := c.Cookies(AccessCookieName); token != "" {
		return token
	}

	bearToken := c.Get("Authorization")

	// Normally Authorization HTTP header.
	onlyToken := strings.Split(bearToken, " ")
	if len(onlyToken) == 2 {
		return onlyToken[1]
	}

	return ""
}

func verifyToken(c fiber.Ctx) (*jwt.Token, error) {
	tokenString := extractToken(c)

	token, err := jwt.Parse(tokenString, jwtKeyFunc)
	if err != nil {
		return nil, err
	}

	return token, nil
}

func jwtKeyFunc(token *jwt.Token) (interface{}, error) {
	if token.Method != jwt.SigningMethodHS256 {
		return nil, errors.New("unexpected signing method")
	}
	return []byte(configs.Get().JWT.Secret), nil
}
