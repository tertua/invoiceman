package controllers

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"
)

// CreateOrg creates a brand-new organization and assigns a user as its owner in one transaction (cross-tenant bootstrap). The calling admin is NOT added as a member; the route only carries RequireRoles("admin").
// @Description Create an organization and assign its owner.
// @Summary create organization
// @Tags Admin
// @Accept json
// @Produce json
// @Param request body models.CreateOrgInput true "Organization payload"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Security SessionCookie
// @Router /admin/orgs [post]
func CreateOrg(c fiber.Ctx) error {
	adminID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	input := &models.CreateOrgInput{}
	if err := c.Bind().Body(input); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
	}
	if err := utils.NewValidator().Struct(input); err != nil {
		return utils.ValidationFailed(c, err)
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return utils.Fail(c, fiber.StatusBadRequest, "name is required", nil)
	}
	ownerID, err := uuid.Parse(input.OwnerUserID)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid owner user id", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	owner, err := db.GetUserByID(ownerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "user not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load user", nil)
	}
	// 0 == blocked (see the auth login guard): a blocked account cannot own a tenant.
	if owner.UserStatus != 1 {
		return utils.Fail(c, fiber.StatusBadRequest, "owner user is blocked", nil)
	}
	org, err := db.CreateOrgWithOwner(name, ownerID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to create org", nil)
	}
	recordAudit(c, db, adminID, "org.create", "org", org.ID.String(),
		`{"owner_user_id":"`+ownerID.String()+`"}`)
	return utils.OK(c, fiber.StatusCreated, fiber.Map{
		"org":           fiber.Map{"id": org.ID.String(), "name": org.Name},
		"role":          models.RoleOwner,
		"owner_user_id": ownerID.String(),
	})
}
