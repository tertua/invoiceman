package controllers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/configs"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"
)

// syncOIDCRole maps the verified token's role claim to the local platform role
// and persists it when it changed. It is a no-op when role sync is disabled
// (OIDC_ROLE_CLAIM empty), so a console-promoted admin keeps its role.
//
// claims is the FULL verified ID-token payload as JSON; oidcRoleFromClaims walks
// the configured dot-path inside it.
//
// It never returns an error that should fail the login: a write failure is
// logged and the previous role is kept (documented fail-open on DB error, plan
// D10 — a DB outage already fails user resolution earlier in the flow).
func syncOIDCRole(c fiber.Ctx, db *database.Queries, user *models.User, claims json.RawMessage) {
	cfg := configs.Get().OIDC
	if !cfg.RoleSyncEnabled() {
		return
	}
	target, _ := oidcRoleFromClaims(claims, cfg.RoleClaim, cfg.AdminRole)
	if target == "" || target == user.UserRole {
		return // no change -> no write, no audit
	}
	if err := db.UpdateUserRole(user.ID, target); err != nil {
		utils.RequestLogger(c).Warn("oidc role sync failed", "user", user.ID, "err", err)
		return
	}
	user.UserRole = target // keep the in-memory copy consistent for the caller
	recordAudit(c, db, user.ID, "auth.role.sync", "user", user.ID.String(), `{"role":"`+target+`"}`)
}
