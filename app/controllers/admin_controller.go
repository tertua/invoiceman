package controllers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"
)

func adminUserResponse(user models.User) fiber.Map {
	return fiber.Map{
		"id":         user.ID,
		"name":       user.Name,
		"email":      user.Email,
		"role":       user.UserRole,
		"status":     user.UserStatus,
		"created_at": user.CreatedAt,
	}
}

// ListUsers returns one page of accounts for an administrator.
// @Description List registered users.
// @Summary list users
// @Tags Admin
// @Param page query int false "Page number (default 1)"
// @Param per_page query int false "Items per page (default 20, max 100)"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /admin/users [get]
func ListUsers(c fiber.Ctx) error {
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	paging := utils.ParsePagination(c)
	users, err := db.ListUsers(paging.Limit(), paging.Offset())
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load users", nil)
	}
	total, err := db.CountUsers()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to count users", nil)
	}
	result := make([]fiber.Map, 0, len(users))
	for _, user := range users {
		result = append(result, adminUserResponse(user))
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"users": result, "meta": paging.Meta(total)})
}

// ListAuditLogs returns one page of the audit trail for an administrator.
// @Description List audit trail entries.
// @Summary list audit trail
// @Tags Admin
// @Produce json
// @Param page query int false "Page number (default 1)"
// @Param per_page query int false "Items per page (default 20, max 100)"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /admin/audit-logs [get]
func ListAuditLogs(c fiber.Ctx) error {
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	paging := utils.ParsePagination(c)
	rows, err := db.ListAuditLogs(paging.Limit(), paging.Offset())
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load audit logs", nil)
	}
	total, err := db.CountAuditLogs()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to count audit logs", nil)
	}
	out := make([]fiber.Map, 0, len(rows))
	for _, r := range rows {
		out = append(out, fiber.Map{
			"id": r.ID, "user_id": r.UserID, "action": r.Action,
			"entity": r.Entity, "entity_id": r.EntityID,
			"ip": r.IP, "created_at": r.CreatedAt,
		})
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"audit_logs": out, "meta": paging.Meta(total)})
}

// MigrateDown rolls the schema back to a previous version (admin only).
// Emergency use: run this, then deploy the older binary. Dropped columns
// lose their values; forward upgrades re-apply automatically on the next
// startup with a newer binary.
// @Description Roll the database schema back to a previous version.
// @Summary rollback schema version
// @Tags Admin
// @Accept json
// @Produce json
// @Param request body models.MigrateDownInput true "Target version plus explicit confirmation"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /admin/migrate/down [post]
func MigrateDown(c fiber.Ctx) error {
	adminID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	input := &models.MigrateDownInput{}
	if err := c.Bind().Body(input); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
	}
	if err := utils.NewValidator().Struct(input); err != nil {
		return utils.ValidationFailed(c, err)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	version, err := database.MigrateDownTo(input.TargetVersion)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, err.Error(), nil)
	}
	recordAudit(c, db, adminID, "schema.rollback", "schema", "v1", `{"version":`+strconv.Itoa(version)+`}`)
	return utils.OK(c, fiber.StatusOK, fiber.Map{"version": version})
}
