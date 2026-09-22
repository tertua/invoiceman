package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/tertua/invoiceman/pkg/configs"
)

// Tokens struct to describe tokens object.
type Tokens struct {
	Access string
	// SID identifies the session generation: a new login mints a new SID
	// and instantly kills the previous session (strict single-session).
	SID     string
	Refresh string
}

// GenerateNewTokens func for generate a new Access & Refresh tokens.
// An empty sid mints a fresh session; a non-empty one reuses the session
// (transparent refresh must not kick sibling tabs sharing the session).
func GenerateNewTokens(id, sid string) (*Tokens, error) {
	if sid == "" {
		sid = uuid.NewString()
	}

	// Generate JWT Access token.
	accessToken, err := generateNewAccessToken(id, sid)
	if err != nil {
		// Return token generation error.
		return nil, err
	}

	// Generate JWT Refresh token.
	refreshToken, err := generateNewRefreshToken()
	if err != nil {
		// Return token generation error.
		return nil, err
	}

	return &Tokens{
		Access:  accessToken,
		SID:     sid,
		Refresh: refreshToken,
	}, nil
}

func generateNewAccessToken(id, sid string) (string, error) {
	// Set secret key and expiry from the central config.
	cfg := configs.Get().JWT
	secret := cfg.Secret
	minutesCount := cfg.AccessMinutes

	// Create a new claims.
	claims := jwt.MapClaims{}

	// Set public claims:
	claims["id"] = id
	claims["sid"] = sid
	claims["exp"] = time.Now().Add(time.Minute * time.Duration(minutesCount)).Unix()

	// Create a new JWT access token with claims.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate token.
	t, err := token.SignedString([]byte(secret))
	if err != nil {
		// Return error, it JWT token generation failed.
		return "", err
	}

	return t, nil
}

func generateNewRefreshToken() (string, error) {
	// Create a new SHA256 hash.
	hash := sha256.New()

	// Create a new now date and time string with salt.
	refresh := configs.Get().JWT.RefreshKey + time.Now().String()

	// See: https://pkg.go.dev/io#Writer.Write
	_, err := hash.Write([]byte(refresh))
	if err != nil {
		// Return error, it refresh token generation failed.
		return "", err
	}

	// Set expires hours count for refresh key from the central config.
	hoursCount := configs.Get().JWT.RefreshHours

	// Set expiration time.
	expireTime := fmt.Sprint(time.Now().Add(time.Hour * time.Duration(hoursCount)).Unix())

	// Create a new refresh token (sha256 string with salt + expire time).
	t := hex.EncodeToString(hash.Sum(nil)) + "." + expireTime

	return t, nil
}

// ParseRefreshToken func for parse second argument from refresh token.
func ParseRefreshToken(refreshToken string) (int64, error) {
	parts := strings.Split(refreshToken, ".")
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return 0, fmt.Errorf("invalid refresh token format")
	}
	return strconv.ParseInt(parts[1], 0, 64)
}
