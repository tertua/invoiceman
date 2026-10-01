package controllers

import (
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// rotateSession mints a fresh session for the active device (new sid, refresh
// token and CSRF binding) and stores it, overwriting the previous entry so the
// old refresh token and CSRF binding stop working. The user stays logged in: the
// new cookies are set on the same response.
func rotateSession(c fiber.Ctx, db *database.Queries, userID uuid.UUID) error {
	tokens, err := utils.IssueSession(c, userID, "")
	if err != nil {
		return err
	}
	csrf, err := issueCSRF(c)
	if err != nil {
		return err
	}
	return saveRefreshToken(c.Context(), userID, tokens.SID, tokens.Refresh, csrf, loginOrgHint(db, userID))
}

// ChangePassword changes the current user password.
// @Description Change current user password.
// @Summary change current user password
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body models.ChangePassword true "Change password payload"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /auth/password [patch]
func ChangePassword(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}

	payload := &models.ChangePassword{}
	if err := c.Bind().Body(payload); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
	}
	if err := utils.NewValidator().Struct(payload); err != nil {
		return utils.ValidationFailed(c, err)
	}

	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}

	user, err := db.GetUserByID(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "user not found", nil)
	}
	if !utils.ComparePasswords(user.PasswordHash, payload.CurrentPassword) {
		return utils.Fail(c, fiber.StatusBadRequest, "current password is wrong", nil)
	}

	if err := db.UpdateUserPassword(userID, utils.GeneratePassword(payload.NewPassword)); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to update password", nil)
	}
	recordAudit(c, db, userID, "auth.password.change", "user", userID.String(), "")
	// Privilege moment: a leaked CSRF token or the pre-change refresh token must
	// not survive. Mint a brand-new session for this device (fresh sid + refresh
	// token + CSRF binding), which overwrites the store entry and kills the old
	// refresh token — without force-logging the user out (plan decision D4).
	if err := rotateSession(c, db, userID); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to rotate session", nil)
	}

	return utils.OK(c, fiber.StatusOK, fiber.Map{"message": "password updated"})
}
