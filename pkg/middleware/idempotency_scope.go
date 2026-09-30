package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/pkg/utils"
)

// IdempotencyScope extracts the dedup scope for a request
// (e.g. "user:<id>:org:<id>", "project:<slug>", "pay:<token>").
type IdempotencyScope func(c fiber.Ctx) (string, error)

// SessionIdempotencyScope scopes keys to the authenticated session user and
// the active organization. AuthRequired and OrgContext must run before this
// middleware. The org is part of the scope on purpose: a key is a promise
// about one intent in one tenant, so switching orgs must execute rather than
// replay the other tenant's stored response.
func SessionIdempotencyScope(c fiber.Ctx) (string, error) {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return "", err
	}
	orgID, err := utils.CurrentOrgID(c)
	if err != nil {
		return "", err
	}
	return "user:" + userID.String() + ":org:" + orgID.String(), nil
}

// GatewayIdempotencyScope scopes keys to the calling service project.
// GatewayAuth must run before this middleware.
func GatewayIdempotencyScope(c fiber.Ctx) (string, error) {
	project, err := utils.CurrentServiceProject(c)
	if err != nil {
		return "", err
	}
	return "project:" + project.Slug, nil
}

// PublicPayIdempotencyScope scopes keys to the public payment token.
func PublicPayIdempotencyScope(c fiber.Ctx) (string, error) {
	token := strings.TrimSpace(c.Params("token"))
	if token == "" {
		return "", fiber.NewError(fiber.StatusBadRequest, "payment token is required")
	}
	return "pay:" + token, nil
}
