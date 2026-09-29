package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/tertua/tupay/pkg/logger"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/cache"
	"github.com/tertua/tupay/platform/database"
)

// OrgContext resolves the active organization after AuthRequired (D2): it reads the session itself, validates membership and exposes orgID/orgRole locals for controllers.
func OrgContext() fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, ok := orgContextUser(c)
		if !ok {
			return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
		}
		db, err := database.OpenDBConnection()
		if err != nil {
			return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
		}
		orgID, role := orgContextMembership(c, db, userID)
		if orgID == uuid.Nil {
			return utils.Fail(c, fiber.StatusForbidden, "org.notMember", nil)
		}
		c.Locals(utils.SessionOrgIDKey, orgID)
		c.Locals(utils.SessionOrgRoleKey, role)
		return c.Next()
	}
}

// RequireOrgRole permits only the given active-org role; OrgContext must run first (D4 keeps role guards at the route layer).
func RequireOrgRole(role string) fiber.Handler {
	return func(c fiber.Ctx) error {
		current, err := utils.CurrentOrgRole(c)
		if err != nil || current != role {
			return utils.Fail(c, fiber.StatusForbidden, "org.ownerRequired", nil)
		}
		return c.Next()
	}
}

// orgContextUser takes the user AuthRequired stored in locals, falling back to the session cookie identity when the chain omitted it (D2: OrgContext reads its own cookie).
func orgContextUser(c fiber.Ctx) (uuid.UUID, bool) {
	if userID, ok := c.Locals(utils.SessionUserIDKey).(uuid.UUID); ok && userID != uuid.Nil {
		return userID, true
	}
	userID, _, ok := userIDFromAccess(c.Cookies(utils.AccessCookieName))
	return userID, ok
}

// orgContextMembership validates the session active-org hint, else falls back to D3 resolution (0 memberships → personal org, >1 → oldest) plus a best-effort session write-back.
func orgContextMembership(c fiber.Ctx, db *database.Queries, userID uuid.UUID) (orgID uuid.UUID, role string) {
	if hint := sessionActiveOrg(c, userID); hint != uuid.Nil {
		if role, err := db.GetRole(hint, userID); err == nil {
			return hint, role
		}
	}
	orgID, err := db.ResolveActiveOrgID(userID, uuid.Nil)
	if err != nil {
		logger.L().Debug("org context: resolve failed", "userID", userID, "err", err)
		return uuid.Nil, ""
	}
	role, err = db.GetRole(orgID, userID)
	if err != nil {
		logger.L().Debug("org context: membership missing after resolve", "userID", userID, "err", err)
		return uuid.Nil, ""
	}
	rememberActiveOrg(c, userID, orgID)
	return orgID, role
}

// sessionActiveOrg decodes the stored session value for the active-org hint; an absent, legacy or malformed entry yields no hint.
func sessionActiveOrg(c fiber.Ctx, userID uuid.UUID) uuid.UUID {
	sv, ok, err := cache.ReadSessionValue(c.Context(), userID.String())
	if err != nil || !ok || sv.ActiveOrgID == "" {
		return uuid.Nil
	}
	hint, err := uuid.Parse(sv.ActiveOrgID)
	if err != nil {
		return uuid.Nil
	}
	return hint
}

// rememberActiveOrg persists the resolved org into the stored session value keeping sid/refresh/csrf; a failed write is logged at debug and ignored (D3).
func rememberActiveOrg(c fiber.Ctx, userID, orgID uuid.UUID) {
	sv, ok, err := cache.ReadSessionValue(c.Context(), userID.String())
	if err != nil {
		logger.L().Debug("org context: session read failed, skipping write-back", "err", err)
		return
	}
	if !ok {
		return
	}
	sv.ActiveOrgID = orgID.String()
	if err := cache.WriteSessionValue(c.Context(), userID.String(), sv); err != nil {
		logger.L().Debug("org context: active org write-back failed", "err", err)
	}
}
