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

// ListUsers returns all accounts for an administrator.
// @Description List registered users.
// @Summary list users
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /admin/users [get]
func ListUsers(c fiber.Ctx) error {
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	users, err := db.ListUsers()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load users", nil)
	}
	result := make([]fiber.Map, 0, len(users))
	for _, user := range users {
		result = append(result, adminUserResponse(user))
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"users": result})
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
// @Security ApiKeyAuth
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
	user.UserRole = input.Role
	return utils.OK(c, fiber.StatusOK, fiber.Map{"user": adminUserResponse(user)})
}
