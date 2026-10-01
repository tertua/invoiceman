package controllers

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/tupay/pkg/middleware"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"
	"github.com/tertua/tupay/platform/relay"
)

// normalizeEmail lowercases and trims an email at an input boundary so lookups
// and writes agree on one canonical form (matching the startup normalization).
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// clearStaleSession drops the session and CSRF cookies for a request that failed
// authentication: the browser is left holding tokens the server already rejects,
// so an explicit 401 should also clean them up.
func clearStaleSession(c fiber.Ctx) {
	utils.ClearSession(c)
	middleware.ClearCSRFCookie(c)
}

// recordLoginFailure audits a failed login without leaking the email: the entity
// id is a one-way hash, the reason stays generic.
func recordLoginFailure(c fiber.Ctx, db *database.Queries, email, reason string, userID uuid.UUID) {
	recordAudit(c, db, userID, "auth.login.failed", "auth", relay.HashKey(normalizeEmail(email)), `{"reason":"`+reason+`"}`)
}
