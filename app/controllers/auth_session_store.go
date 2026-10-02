package controllers

import (
	"context"

	"github.com/tertua/tupay/pkg/middleware"
	"github.com/tertua/tupay/platform/cache"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// saveRefreshToken stores the session (sid + refresh token + bound CSRF token + active org) in the session store; overwriting kills any previous session (strict single-session).
func saveRefreshToken(ctx context.Context, userID uuid.UUID, sid, refresh, csrf, activeOrg string) error {
	store, err := cache.Sessions()
	if err != nil {
		return err
	}
	return store.Set(ctx, userID.String(), cache.EncodeSessionValue(sid, refresh, csrf, activeOrg), cache.RefreshTTL())
}

// saveSessionCSRF rebinds the CSRF token of the live session without touching its sid or refresh token (rotation on privilege moments).
func saveSessionCSRF(ctx context.Context, userID uuid.UUID, csrf string) error {
	store, err := cache.Sessions()
	if err != nil {
		return err
	}
	stored, err := store.Get(ctx, userID.String())
	if err != nil {
		return err
	}
	sid, refresh, _, activeOrg, ok := cache.DecodeSessionValue(stored)
	if !ok {
		return cache.ErrSessionNotFound
	}
	return store.Set(ctx, userID.String(), cache.EncodeSessionValue(sid, refresh, csrf, activeOrg), cache.RefreshTTL())
}

// deleteRefreshToken removes the refresh token from the session store.
func deleteRefreshToken(ctx context.Context, userID uuid.UUID) error {
	store, err := cache.Sessions()
	if err != nil {
		return err
	}
	return store.Delete(ctx, userID.String())
}

// issueCSRF mints the double-submit token for a session and writes the readable cookie; the returned value must be bound to the session store (RequireCSRF cross-checks it, so a rotated token invalidates the old one).
func issueCSRF(c fiber.Ctx) (string, error) {
	token, err := middleware.NewCSRFToken()
	if err != nil {
		return "", err
	}
	middleware.SetCSRFCookie(c, token)
	return token, nil
}

// rotateCSRF mints a new CSRF token for a privilege moment; the binding is stored before the cookie is written, so a store failure never leaves browser and server disagreeing.
func rotateCSRF(c fiber.Ctx, userID uuid.UUID) error {
	token, err := middleware.NewCSRFToken()
	if err != nil {
		return err
	}
	if err := saveSessionCSRF(c.Context(), userID, token); err != nil {
		return err
	}
	middleware.SetCSRFCookie(c, token)
	return nil
}
