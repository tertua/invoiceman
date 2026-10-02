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
