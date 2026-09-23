package captcha

// Package captcha verifies Cloudflare Turnstile tokens (stdlib only).
//
// Verification is required on abuse-prone public endpoints (register,
// login, forgot-password, reset-password) whenever TURNSTILE_SECRET is
// configured. Empty secret disables verification (dev/test default) so
// local flows and the test suite keep working without network access.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tertua/invoiceman/pkg/configs"
)

// ErrNotConfigured is returned when no secret is set (verification off).
var ErrNotConfigured = errors.New("captcha is not configured")

// ErrFailed is returned when the provider rejects the token.
var ErrFailed = errors.New("captcha verification failed")

// TokenHeader carries the client token (set by the SPA widget).
const TokenHeader = "X-Captcha-Token" // #nosec G101 -- header name, not a credential

// verifyURL is a variable so tests can point at a fake provider.
var verifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// Required reports whether verification is active.
func Required() bool {
	return strings.TrimSpace(configs.Get().Captcha.TurnstileSecret) != ""
}

type verifyResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

// Verify checks token with the provider (3s timeout). A nil error means
// the client solved the challenge; ErrNotConfigured means verification is
// disabled; anything else (including transport errors) fails closed.
func Verify(ctx context.Context, token, remoteIP string) error {
	secret := strings.TrimSpace(configs.Get().Captcha.TurnstileSecret)
	if secret == "" {
		return ErrNotConfigured
	}
	if strings.TrimSpace(token) == "" {
		return ErrFailed
	}

	form := url.Values{
		"secret":   {secret},
		"response": {token},
	}
	if strings.TrimSpace(remoteIP) != "" {
		form.Set("remoteip", remoteIP)
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, verifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	out := verifyResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	if !out.Success {
		return ErrFailed
	}
	return nil
}
