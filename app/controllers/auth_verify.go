package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/configs"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"
	"github.com/tertua/tupay/platform/mail"
	"github.com/tertua/tupay/platform/relay"

	"github.com/gofiber/fiber/v3"
)

// verifyEmailTTL is how long a verification link stays valid.
const verifyEmailTTL = 24 * time.Hour

// enqueueVerification mints a fresh single-use verification token for user,
// stores only its hash, and queues the email. Any previous token is deleted
// first (one active link per account). Best-effort on delivery, like
// ForgotPassword: the caller decides whether a failure is fatal.
func enqueueVerification(c fiber.Ctx, db *database.Queries, user models.User) error {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	token := hex.EncodeToString(raw)
	if err := db.DeleteEmailVerificationsByUser(user.ID); err != nil {
		return err
	}
	// Store only the hash: a DB leak must not hand out live verification links (same rule as password resets and service API keys).
	if err := db.CreateEmailVerification(user.ID, relay.HashKey(token), time.Now().Add(verifyEmailTTL)); err != nil {
		return err
	}
	verifyURL := strings.TrimRight(configs.Get().Mail.AppPublicURL, "/") + "/verify-email?token=" + token
	body := fmt.Sprintf("Hello %s,\n\nConfirm your email using this link:\n%s\n\nThis link expires in 24 hours.", user.Name, verifyURL)
	htmlBody, terr := mail.Render("verify_email", mail.TemplateData{
		AppName: configs.Get().AppName, Name: user.Name, URL: verifyURL,
	})
	if terr != nil {
		utils.RequestLogger(c).Warn("verification email template failed", "err", terr)
	}
	if err := db.EnqueueMail(&models.MailOutbox{
		To:       user.Email,
		Subject:  "Verify your " + configs.Get().AppName + " email",
		Body:     body,
		HtmlBody: htmlBody,
	}); err != nil {
		utils.RequestLogger(c).Warn("verification email queue failed", "email", user.Email, "err", err)
	}
	return nil
}

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

	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
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

	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}

	// Always answer generically (enumeration-safe): only a pending account gets a new link, and the response never reveals whether one was sent.
	if user, err := db.GetUserByEmail(payload.Email); err == nil && user.UserStatus == models.UserStatusPending {
		_ = enqueueVerification(c, db, user)
	}

	return utils.OK(c, fiber.StatusAccepted, fiber.Map{"message": "if the account needs verification, a new link was sent"})
}

// verifyEmailGate returns the account status a new registration should get:
// pending when verification is required and this is not the first-install
// bootstrap (count == 0), active otherwise.
func verifyEmailGate(count int64) int {
	if configs.Get().Auth.RequireEmailVerification && count > 0 {
		return models.UserStatusPending
	}
	return models.UserStatusActive
}

// gateAccountStatus rejects login for anything but an active account, writing
// the 403 itself and returning true when the request is done. It runs after
// password verify so a wrong password never leaks the account state. A false
// return means active: the caller may start the session.
func gateAccountStatus(c fiber.Ctx, db *database.Queries, user models.User) bool {
	switch user.UserStatus {
	case models.UserStatusPending:
		recordLoginFailure(c, db, user.Email, "pending", user.ID)
		_ = utils.Fail(c, fiber.StatusForbidden, "account is pending verification", nil)
		return true
	case models.UserStatusActive:
		return false
	default:
		recordLoginFailure(c, db, user.Email, "blocked", user.ID)
		_ = utils.Fail(c, fiber.StatusForbidden, "account is blocked", nil)
		return true
	}
}
