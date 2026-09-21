package relay

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
)

// GenerateAPIKey creates a service API key shown once to the project owner.
func GenerateAPIKey() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return "inv_live_" + hex.EncodeToString(raw), nil
}

// GenerateSecret creates a webhook signing secret shown once.
func GenerateSecret() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return "whsec_" + hex.EncodeToString(raw), nil
}

// HashKey hashes an API key for storage (the plaintext is never stored).
func HashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// SignPayload returns hex(HMAC-SHA256(body, secret)).
func SignPayload(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifySignature compares a presented signature with the expected HMAC.
func VerifySignature(body []byte, secret, presented string) bool {
	presented = strings.TrimSpace(presented)
	if presented == "" || secret == "" {
		return false
	}
	expected := SignPayload(body, secret)
	return subtle.ConstantTimeCompare([]byte(presented), []byte(expected)) == 1
}

// ExtractKey reads a service key from "Authorization: Bearer <key>"
// with fallback to the X-Api-Key header (some proxies strip Authorization).
func ExtractKey(authorization, apiKeyHeader string) string {
	const prefix = "Bearer "
	if len(authorization) > len(prefix) && authorization[:len(prefix)] == prefix {
		if key := strings.TrimSpace(authorization[len(prefix):]); key != "" {
			return key
		}
	}
	return strings.TrimSpace(apiKeyHeader)
}
