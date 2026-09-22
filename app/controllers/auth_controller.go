package controllers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/configs"
	"github.com/tertua/invoiceman/pkg/logger"
	"github.com/tertua/invoiceman/pkg/middleware"
	"github.com/tertua/invoiceman/pkg/repository"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/cache"
	"github.com/tertua/invoiceman/platform/captcha"
	"github.com/tertua/invoiceman/platform/database"

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

// saveRefreshToken stores the refresh token in the session store.
func saveRefreshToken(userID uuid.UUID, refresh string) error {
	store, err := cache.Sessions()
	if err != nil {
		return err
	}
	return store.Set(context.Background(), userID.String(), refresh, cache.RefreshTTL())
}

// deleteRefreshToken removes the refresh token from the session store.
func deleteRefreshToken(userID uuid.UUID) error {
	store, err := cache.Sessions()
	if err != nil {
		return err
	}
	return store.Delete(context.Background(), userID.String())
}

// issueCSRF mints the double-submit token for a fresh session.
func issueCSRF(c fiber.Ctx) error {
	token, err := middleware.NewCSRFToken()
	if err != nil {
		return err
	}
	middleware.SetCSRFCookie(c, token)
	return nil
}

// checkCaptcha enforces Turnstile verification when configured.
// Disabled (no secret) in dev/test; fails closed otherwise.
// It writes the 403 response itself: callers must return nil when it
// reports false (utils.Fail returns nil after writing).
func checkCaptcha(c fiber.Ctx) bool {
	if !captcha.Required() {
		return true
	}
	if err := captcha.Verify(c.Context(), c.Get(captcha.TokenHeader), c.IP()); err != nil {
		_ = utils.Fail(c, fiber.StatusForbidden, "captcha verification failed", nil)
		return false
	}
	return true
}

// Register creates a new user and starts a session.
// @Description Register a new user.
// @Summary register a new user
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body models.Register true "Register payload"
// @Param X-Captcha-Token header string false "Turnstile token (required when CAPTCHA is enabled)"
// @Success 201 {object} map[string]interface{}
// @Router /auth/register [post]
func Register(c fiber.Ctx) error {
	if !checkCaptcha(c) {
		return nil
	}
	payload := &models.Register{}

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

	if _, err := db.GetUserByEmail(payload.Email); err == nil {
		return utils.Fail(c, fiber.StatusConflict, "email is already registered", nil)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return utils.Fail(c, fiber.StatusInternalServerError, "database query error", nil)
	}

	role := repository.UserRoleName
	count, err := db.CountUsers()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to determine account role", nil)
	}
	if count == 0 {
		role = repository.AdminRoleName
	}

	user := &models.User{
		ID:           uuid.New(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Name:         payload.Name,
		Email:        payload.Email,
		PasswordHash: utils.GeneratePassword(payload.Password),
		UserStatus:   1, // 0 == blocked, 1 == active
		UserRole:     role,
	}
	if err := utils.NewValidator().Struct(user); err != nil {
		return utils.ValidationFailed(c, err)
	}
	if err := db.CreateUser(user); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to create user", nil)
	}

	// Create default settings row (ignored when it already exists).
	settings := models.DefaultSettings(user.ID)
	if err := db.CreateSettings(settings); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to create user settings", nil)
	}

	tokens, err := utils.IssueSession(c, user.ID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to create session", nil)
	}
	if err := saveRefreshToken(user.ID, tokens.Refresh); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to persist session", nil)
	}
	if err := issueCSRF(c); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to create session", nil)
	}

	user.PasswordHash = ""
	return utils.OK(c, fiber.StatusCreated, fiber.Map{"user": publicUser(*user)})
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
		return utils.Fail(c, fiber.StatusUnauthorized, "wrong email address or password", nil)
	}
	if !utils.ComparePasswords(user.PasswordHash, payload.Password) {
		return utils.Fail(c, fiber.StatusUnauthorized, "wrong email address or password", nil)
	}
	if user.UserStatus != 1 {
		return utils.Fail(c, fiber.StatusForbidden, "account is blocked", nil)
	}

	tokens, err := utils.IssueSession(c, user.ID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to create session", nil)
	}
	if err := saveRefreshToken(user.ID, tokens.Refresh); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to persist session", nil)
	}
	if err := issueCSRF(c); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to create session", nil)
	}

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
// @Security ApiKeyAuth
// @Router /auth/logout [post]
func Logout(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}

	if err := deleteRefreshToken(userID); err != nil {
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
// @Security ApiKeyAuth
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
	return utils.OK(c, fiber.StatusOK, fiber.Map{"user": publicUser(user)})
}

// UpdateProfile updates the current user display name.
// @Description Update current user profile.
// @Summary update current user profile
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body models.UpdateProfile true "Update profile payload"
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
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
// @Security ApiKeyAuth
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

	return utils.OK(c, fiber.StatusOK, fiber.Map{"message": "password updated"})
}

// ForgotPassword creates a password reset token for the given email.
// @Description Request password reset token.
// @Summary request password reset token
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body models.ForgotPassword true "Forgot password payload"
// @Param X-Captcha-Token header string false "Turnstile token (required when CAPTCHA is enabled)"
// @Success 200 {object} map[string]interface{}
// @Router /auth/forgot-password [post]
func ForgotPassword(c fiber.Ctx) error {
	if !checkCaptcha(c) {
		return nil
	}
	payload := &models.ForgotPassword{}

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

	// Always respond generically to avoid email enumeration.
	// Delivery is async: the worker sends the email, retries included.
	if user, err := db.GetUserByEmail(payload.Email); err == nil {
		raw := make([]byte, 32)
		if _, err := rand.Read(raw); err == nil {
			token := hex.EncodeToString(raw)
			_ = db.DeletePasswordResetsByUser(user.ID)
			if err := db.CreatePasswordReset(user.ID, token, time.Now().Add(time.Hour)); err != nil {
				return utils.Fail(c, fiber.StatusInternalServerError, "failed to create password reset request", nil)
			}
			resetURL := strings.TrimRight(configs.Get().Mail.AppPublicURL, "/") + "/reset-password?token=" + token
			body := fmt.Sprintf("Hello %s,\n\nReset your password using this link:\n%s\n\nThis link expires in one hour.", user.Name, resetURL)
			if err := db.EnqueueMail(&models.MailOutbox{
				To:      user.Email,
				Subject: "Reset your Invoiceman password",
				Body:    body,
			}); err != nil {
				logger.L().Warn("password reset email queue failed", "email", user.Email, "err", err)
			}
		}
	}

	return utils.OK(c, fiber.StatusOK, fiber.Map{"message": "if the email exists, a reset link was sent"})
}

// ResetPassword resets the password using a reset token.
// @Description Reset password with token.
// @Summary reset password with token
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body models.ResetPassword true "Reset password payload"
// @Success 200 {object} map[string]interface{}
// @Router /auth/reset-password [post]
func ResetPassword(c fiber.Ctx) error {
	payload := &models.ResetPassword{}

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

	reset, err := db.GetPasswordReset(payload.Token)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid or expired token", nil)
	}
	if time.Now().After(reset.ExpiresAt) {
		_ = db.DeletePasswordResetsByUser(reset.UserID)
		return utils.Fail(c, fiber.StatusBadRequest, "invalid or expired token", nil)
	}

	if err := db.UpdateUserPassword(reset.UserID, utils.GeneratePassword(payload.NewPassword)); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to update password", nil)
	}
	_ = db.DeletePasswordResetsByUser(reset.UserID)

	return utils.OK(c, fiber.StatusOK, fiber.Map{"message": "password updated"})
}
