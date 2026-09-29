package controllers

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/middleware"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/cache"
	"github.com/tertua/tupay/platform/database"
	"github.com/tertua/tupay/platform/relay"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// publicUser returns the user fields exposed to the frontend.
func publicUser(u models.User) fiber.Map {
	return fiber.Map{
		"id":    u.ID,
		"name":  u.Name,
		"email": u.Email,
		"role":  u.UserRole,
	}
}

// saveRefreshToken stores the session (sid + refresh token + bound CSRF token + active org) in the session store; overwriting kills any previous session (strict single-session).
func saveRefreshToken(ctx context.Context, userID uuid.UUID, sid, refresh, csrf, activeOrg string) error {
	store, err := cache.Sessions()
	if err != nil {
		return err
	}
	return store.Set(ctx, userID.String(), cache.EncodeSessionValue(sid, refresh, csrf, activeOrg), cache.RefreshTTL())
}

// saveSessionCSRF rebinds the CSRF token of the live session without touching its sid or refresh token (rotation on privilege moments).
func saveSessionCSRF(ctx context.Context, userID uuid.UUID, csrf string) error {
	store, err := cache.Sessions()
	if err != nil {
		return err
	}
	stored, err := store.Get(ctx, userID.String())
	if err != nil {
		return err
	}
	sid, refresh, _, activeOrg, ok := cache.DecodeSessionValue(stored)
	if !ok {
		return cache.ErrSessionNotFound
	}
	return store.Set(ctx, userID.String(), cache.EncodeSessionValue(sid, refresh, csrf, activeOrg), cache.RefreshTTL())
}

// deleteRefreshToken removes the refresh token from the session store.
func deleteRefreshToken(ctx context.Context, userID uuid.UUID) error {
	store, err := cache.Sessions()
	if err != nil {
		return err
	}
	return store.Delete(ctx, userID.String())
}

// recordLoginFailure audits a failed login without leaking the email: the entity id is a one-way hash, the reason stays generic.
func recordLoginFailure(c fiber.Ctx, db *database.Queries, email, reason string, userID uuid.UUID) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	recordAudit(c, db, userID, "auth.login.failed", "auth", relay.HashKey(normalized), `{"reason":"`+reason+`"}`)
}

// issueCSRF mints the double-submit token for a session and writes the readable cookie; the returned value must be bound to the session store (RequireCSRF cross-checks it, so a rotated token invalidates the old one).
func issueCSRF(c fiber.Ctx) (string, error) {
	token, err := middleware.NewCSRFToken()
	if err != nil {
		return "", err
	}
	middleware.SetCSRFCookie(c, token)
	return token, nil
}

// rotateCSRF mints a new CSRF token for a privilege moment; the binding is stored before the cookie is written, so a store failure never leaves browser and server disagreeing.
func rotateCSRF(c fiber.Ctx, userID uuid.UUID) error {
	token, err := middleware.NewCSRFToken()
	if err != nil {
		return err
	}
	if err := saveSessionCSRF(c.Context(), userID, token); err != nil {
		return err
	}
	middleware.SetCSRFCookie(c, token)
	return nil
}

// Login authenticates a user and starts a session.
// @Description Auth user and start session.
// @Summary auth user and start session
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body models.Login true "Login payload"
// @Param X-Captcha-Token header string false "Turnstile token (required when CAPTCHA is enabled)"
// @Success 200 {object} map[string]interface{}
// @Router /auth/login [post]
func Login(c fiber.Ctx) error {
	if !checkCaptcha(c) {
		return nil
	}
	payload := &models.Login{}

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

	user, err := db.GetUserByEmail(payload.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			recordLoginFailure(c, db, payload.Email, "not_found", uuid.Nil)
			return utils.Fail(c, fiber.StatusUnauthorized, "wrong email address or password", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to query user account", nil) // live DB error, not a bad credential
	}
	if !utils.ComparePasswords(user.PasswordHash, payload.Password) {
		recordLoginFailure(c, db, payload.Email, "bad_password", user.ID)
		return utils.Fail(c, fiber.StatusUnauthorized, "wrong email address or password", nil)
	}
	if user.UserStatus != 1 {
		recordLoginFailure(c, db, payload.Email, "blocked", user.ID)
		return utils.Fail(c, fiber.StatusForbidden, "account is blocked", nil)
	}

	tokens, err := utils.IssueSession(c, user.ID, "")
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to create session", nil)
	}
	csrf, err := issueCSRF(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to create session", nil)
	}
	if err := saveRefreshToken(c.Context(), user.ID, tokens.SID, tokens.Refresh, csrf, loginOrgHint(db, user.ID)); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to persist session", nil)
	}
	recordAudit(c, db, user.ID, "auth.login.success", "user", user.ID.String(), "")

	user.PasswordHash = ""
	return utils.OK(c, fiber.StatusOK, fiber.Map{"user": publicUser(user)})
}

// Logout ends the session.
// @Description De-authorize user and delete session.
// @Summary de-authorize user and delete session
// @Tags Auth
// @Accept json
// @Produce json
// @Success 204 {string} status "ok"
// @Security SessionCookie
// @Router /auth/logout [post]
func Logout(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}

	// Audit is best-effort (never fails logout): no handle, no trail.
	if db, err := database.OpenDBConnection(); err == nil {
		recordAudit(c, db, userID, "auth.logout", "user", userID.String(), "")
	}
	if err := deleteRefreshToken(c.Context(), userID); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to delete session", nil)
	}
	utils.ClearSession(c)
	middleware.ClearCSRFCookie(c)

	return c.SendStatus(fiber.StatusNoContent)
}

// Me returns the current session user.
// @Description Get current session user.
// @Summary get current session user
// @Tags Auth
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /auth/me [get]
func Me(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}

	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}

	user, err := db.GetUserByID(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "user not found", nil)
	}

	user.PasswordHash = ""
	return utils.OK(c, fiber.StatusOK, fiber.Map{"user": publicUser(user), "org": orgPayload(db, userID)})
}

// UpdateProfile updates the current user display name.
// @Description Update current user profile.
// @Summary update current user profile
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body models.UpdateProfile true "Update profile payload"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /auth/profile [patch]
func UpdateProfile(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}

	payload := &models.UpdateProfile{}
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

	if err := db.UpdateUserProfile(userID, payload.Name); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to update profile", nil)
	}

	user, err := db.GetUserByID(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "user not found", nil)
	}

	user.PasswordHash = ""
	return utils.OK(c, fiber.StatusOK, fiber.Map{"user": publicUser(user)})
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
	// Privilege moment: a leaked CSRF token must not survive a password change — rotate and rebind, so the old token is rejected server-side.
	if err := rotateCSRF(c, userID); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to rotate csrf token", nil)
	}

	return utils.OK(c, fiber.StatusOK, fiber.Map{"message": "password updated"})
}
