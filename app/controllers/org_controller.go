package controllers

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/cache"
	"github.com/tertua/tupay/platform/database"
)

// orgRenameInput is the PATCH /orgs/{id} payload.
type orgRenameInput struct {
	Name string `json:"name"`
}

// GetMyOrg returns the organization the session is working in plus the caller's role.
// @Description Get the active organization of the current session with the caller's role.
// @Summary get active organization
// @Tags Org
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /orgs/me [get]
func GetMyOrg(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	role, err := utils.CurrentOrgRole(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusForbidden, "org.notMember", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	org, err := db.GetOrg(orgID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "org not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load org", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"org": fiber.Map{"id": org.ID.String(), "name": org.Name}, "role": role})
}

// ActivateOrg makes :id the session's active org for members; repeating it is a no-op.
// @Description Activate an organization for the current session; the caller must be a member.
// @Summary activate organization
// @Tags Org
// @Produce json
// @Param id path string true "Organization ID"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /orgs/{id}/activate [post]
func ActivateOrg(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	orgID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid org id", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	role, err := db.GetRole(orgID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusForbidden, "org.notMember", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load membership", nil)
	}
	org, err := db.GetOrg(orgID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "org not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load org", nil)
	}
	if err := persistActiveOrg(c, userID, orgID); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to persist session", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"org": fiber.Map{"id": org.ID.String(), "name": org.Name}, "role": role})
}

// RenameOrg renames the active organization; RequireOrgRole guards the owner role at the route.
// @Description Rename the active organization of the current session.
// @Summary rename organization
// @Tags Org
// @Accept json
// @Produce json
// @Param id path string true "Organization ID"
// @Param request body object true "Rename payload with name"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /orgs/{id} [patch]
func RenameOrg(c fiber.Ctx) error {
	orgID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid org id", nil)
	}
	current, err := utils.CurrentOrgID(c)
	if err != nil || current != orgID {
		return utils.Fail(c, fiber.StatusForbidden, "org.notMember", nil)
	}
	input := &orgRenameInput{}
	if err := c.Bind().Body(input); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return utils.Fail(c, fiber.StatusBadRequest, "name is required", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	if err := db.RenameOrg(orgID, name); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to rename org", nil)
	}
	recordAudit(c, db, utils.CurrentActorID(c), "org.rename", "org", orgID.String(), "")
	return utils.OK(c, fiber.StatusOK, fiber.Map{"org": fiber.Map{"id": orgID.String(), "name": name}})
}

// persistActiveOrg stores the active-org hint on the session value keeping sid/refresh/csrf; an absent entry has no hint to update.
func persistActiveOrg(c fiber.Ctx, userID, orgID uuid.UUID) error {
	sv, ok, err := cache.ReadSessionValue(c.Context(), userID.String())
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	sv.ActiveOrgID = orgID.String()
	return cache.WriteSessionValue(c.Context(), userID.String(), sv)
}
