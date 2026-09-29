package controllers

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/configs"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"
	"github.com/tertua/tupay/platform/mail"
)

// CreateOnlineLink creates a public payment link without contacting a gateway.
// @Description Create a public payment link.
// @Summary create payment link
// @Tags Payments
// @Accept json
// @Produce json
// @Param request body map[string]string true "Invoice ID"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Param Idempotency-Key header string false "Replay protection key (uuid per payment intent)"
// @Router /payments/online [post]
func CreateOnlineLink(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	var input struct {
		InvoiceID string `json:"invoiceId"`
	}
	if err := c.Bind().Body(&input); err != nil || input.InvoiceID == "" {
		return utils.Fail(c, fiber.StatusBadRequest, "invoiceId is required", nil)
	}
	invoiceID, err := uuid.Parse(input.InvoiceID)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid invoice id", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	invoice, err := db.GetInvoice(orgID, invoiceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice", nil)
	}
	if effectiveInvoiceStatus(*db, invoice) == models.InvoiceStatusDraft {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, "invoice is still a draft", nil)
	}
	// A second link would open a second Snap intent for the same invoice; the existing link (returned below when present) stays usable.
	if db.PendingInvoiceIDs(orgID)[invoiceID] {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, "invoice has a pending payment", nil)
	}
	link, err := ensurePaymentLink(*db, orgID, invoice)
	if err != nil {
		return linkFail(c, err)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"url": "/pay/" + link.Token, "token": link.Token})
}

func publicURL(path string) string {
	base := strings.TrimRight(configs.Get().Mail.AppPublicURL, "/")
	return base + path
}

// SendOnlineLink emails a public payment link to the requested recipient.
// @Description Send a public payment link by email.
// @Summary send payment link email
// @Tags Payments
// @Accept json
// @Produce json
// @Param request body models.OnlineLinkEmailInput true "Payment link email payload"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /payments/online/send [post]
func SendOnlineLink(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	input := &models.OnlineLinkEmailInput{}
	if err := c.Bind().Body(input); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
	}
	if err := utils.NewValidator().Struct(input); err != nil {
		return utils.ValidationFailed(c, err)
	}
	invoiceID, err := uuid.Parse(input.InvoiceID)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid invoice id", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	invoice, err := db.GetInvoice(orgID, invoiceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice", nil)
	}
	if effectiveInvoiceStatus(*db, invoice) == models.InvoiceStatusDraft {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, "invoice is still a draft", nil)
	}
	link, err := ensurePaymentLink(*db, orgID, invoice)
	if err != nil {
		return linkFail(c, err)
	}
	// Fail fast when mail is not configured (501 contract), then queue for async delivery by the worker.
	if _, err := mail.NewFromEnv(); errors.Is(err, mail.ErrNotConfigured) {
		return utils.Fail(c, fiber.StatusNotImplemented, "email provider is not configured", nil)
	} else if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "email provider configuration is invalid", nil)
	}
	url := publicURL("/pay/" + link.Token)
	body := fmt.Sprintf("Hello,\n\nPlease use the following link to pay invoice %s:\n%s\n\nThank you.", invoice.InvoiceNumber, url)
	htmlBody, terr := mail.Render("payment_link", mail.TemplateData{
		AppName: configs.Get().AppName, URL: url, InvoiceNumber: invoice.InvoiceNumber,
	})
	if terr != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to render payment link email", nil)
	}
	if err := db.EnqueueMail(&models.MailOutbox{
		To:       input.Email,
		Subject:  "Payment link for invoice " + invoice.InvoiceNumber,
		Body:     body,
		HtmlBody: htmlBody,
	}); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to queue payment link email", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"message": "payment link queued", "queued": true})
}
