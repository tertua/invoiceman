package controllers

import (
	"strings"
	"time"

	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/relay"

	"github.com/gofiber/fiber/v3"
)

// VerifyEmail activates a pending account from an emailed token and starts a session.
// @Description Verify an email address and start a session.
// @Summary verify email address
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body models.VerifyEmail true "Verify email payload"
// @Success 200 {object} map[string]interface{}
// @Router /auth/verify-email [post]
func VerifyEmail(c fiber.Ctx) error {
	payload := &models.VerifyEmail{}

	if err := c.Bind().Body(payload); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
	}
	if err := utils.NewValidator().Struct(payload); err != nil {
		return utils.ValidationFailed(c, err)
	}

	db, ok := openDB(c)
	if !ok {
		return nil
	}

	ev, err := db.GetEmailVerification(relay.HashKey(strings.TrimSpace(payload.Token)))
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid or expired token", nil)
	}
	if time.Now().After(ev.ExpiresAt) {
		_ = db.DeleteEmailVerificationsByUser(ev.UserID)
		return utils.Fail(c, fiber.StatusBadRequest, "invalid or expired token", nil)
	}

	user, err := db.GetUserByID(ev.UserID)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid or expired token", nil)
	}
	// A second click on the same link is idempotent: the row may be gone but
	// the account is already active, so land the user in a session anyway.
	if user.UserStatus != models.UserStatusActive {
		if err := db.UpdateUserStatus(user.ID, models.UserStatusActive); err != nil {
			return utils.Fail(c, fiber.StatusInternalServerError, "failed to activate account", nil)
		}
		user.UserStatus = models.UserStatusActive
	}
	_ = db.DeleteEmailVerificationsByUser(user.ID)

	if _, err := startSession(c, db, user.ID, loginOrgHint(db, user.ID)); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to start session", nil)
	}
	recordAudit(c, db, user.ID, "auth.verify_email", "user", user.ID.String(), "")

	user.PasswordHash = ""
	return utils.OK(c, fiber.StatusOK, fiber.Map{"user": publicUser(user)})
}

// ResendVerification re-sends a verification link when the account still needs one.
// @Description Resend the email verification link.
// @Summary resend email verification link
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body models.ResendVerification true "Resend verification payload"
// @Param X-Captcha-Token header string false "Turnstile token (required when CAPTCHA is enabled)"
// @Success 202 {object} map[string]interface{}
// @Router /auth/verify-email/resend [post]
func ResendVerification(c fiber.Ctx) error {
	if !checkCaptcha(c) {
		return nil
	}
	payload := &models.ResendVerification{}

	if err := c.Bind().Body(payload); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
	}
	if err := utils.NewValidator().Struct(payload); err != nil {
		return utils.ValidationFailed(c, err)
	}

	db, ok := openDB(c)
	if !ok {
		return nil
	}

	// Always answer generically (enumeration-safe): only a pending account gets a new link, and the response never reveals whether one was sent.
	if user, err := db.GetUserByEmail(payload.Email); err == nil && user.UserStatus == models.UserStatusPending {
		_ = enqueueVerification(c, db, user)
	}

	return utils.OK(c, fiber.StatusAccepted, fiber.Map{"message": "if the account needs verification, a new link was sent"})
}
