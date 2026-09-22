package controllers

import (
	"database/sql"
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/repository"
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

// UpdateUserRole assigns a non-admin role to another account.
// @Description Update a user's role.
// @Summary update user role
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body models.RoleInput true "Role payload"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /admin/users/{id}/role [patch]
func UpdateUserRole(c fiber.Ctx) error {
	adminID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid user id", nil)
	}
	if adminID == userID {
		return utils.Fail(c, fiber.StatusBadRequest, "you cannot change your own role", nil)
	}
	input := &models.RoleInput{}
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
	user, err := db.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "user not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load user", nil)
	}
	if user.UserRole == repository.AdminRoleName {
		return utils.Fail(c, fiber.StatusBadRequest, "admin role cannot be changed", nil)
	}
	if err := db.UpdateUserRole(userID, input.Role); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to update user role", nil)
	}
	recordAudit(c, db, adminID, "user.role.update", "user", userID.String(), `{"role":"`+input.Role+`"}`)
	user.UserRole = input.Role
	return utils.OK(c, fiber.StatusOK, fiber.Map{"user": adminUserResponse(user)})
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
