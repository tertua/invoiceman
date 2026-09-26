package controllers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/configs"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"
	"github.com/tertua/invoiceman/platform/mail"
	"gorm.io/gorm"
)

func newPaymentToken() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func paymentResponse(row models.PaymentListRow) fiber.Map {
	return fiber.Map{
		"id":               row.PaymentID,
		"invoice_id":       row.InvoiceID,
		"invoice_number":   row.InvoiceNumber,
		"client_name":      row.ClientName,
		"invoice_currency": row.InvoiceCurrency,
		"amount":           row.Amount,
		"method":           row.Method,
		"paid_on":          utils.FormatDate(row.PaidOn),
		"txn_id":           row.TxnID,
		"notes":            row.Notes,
		"can_void":         row.CanVoid(),
	}
}

// ListPayments returns one page of payments plus global totals.
// @Description Get payments of current user.
// @Summary get payments
// @Tags Payments
// @Param page query int false "Page number (default 1)"
// @Param per_page query int false "Items per page (default 20, max 100)"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /payments [get]
func ListPayments(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	paging := utils.ParsePagination(c)
	rows, err := db.ListPayments(userID, paging.Limit(), paging.Offset())
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load payments", nil)
	}
	total, err := db.CountPayments(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to count payments", nil)
	}
	totals, err := db.GetPaymentTotals(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load payment totals", nil)
	}
	payments := make([]fiber.Map, 0, len(rows))
	for _, row := range rows {
		payments = append(payments, paymentResponse(row))
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{
		"payments": payments,
		"totals": fiber.Map{
			"total":     totals.Total,
			"thisMonth": totals.ThisMonth,
		},
		"meta": paging.Meta(total),
	})
}

// CreatePayment records a payment against an invoice owned by the current user.
// @Description Record a payment.
// @Summary create payment
// @Tags Payments
// @Accept json
// @Produce json
// @Param request body models.PaymentInput true "Payment payload"
// @Success 201 {object} map[string]interface{}
// @Security SessionCookie
// @Param Idempotency-Key header string false "Replay protection key (uuid per payment intent)"
// @Router /payments [post]
func CreatePayment(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	input := &models.PaymentInput{}
	if err := c.Bind().Body(input); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
	}
	if err := utils.NewValidator().Struct(input); err != nil {
		return utils.ValidationFailed(c, err)
	}
	if !input.Amount.GreaterThan(decimal.Zero) {
		return utils.Fail(c, fiber.StatusBadRequest, "amount must be greater than zero", nil)
	}
	invoiceID, err := uuid.Parse(input.InvoiceID)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid invoice id", nil)
	}
	paidOn, err := utils.ParseDate(input.PaidOn)
	if err != nil || paidOn == nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid paid_on, expected YYYY-MM-DD", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	invoice, err := db.GetInvoice(userID, invoiceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice", nil)
	}
	paid, err := db.PaidAmount(invoiceID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice payments", nil)
	}
	// Money in flight locks manual payments: a concurrent manual record
	// would settle against a different balance than the Snap intent.
	if db.PendingInvoiceIDs(userID)[invoiceID] {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, "invoice has a pending payment", nil)
	}
	if paid.Add(input.Amount).GreaterThan(invoice.Total) {
		return utils.Fail(c, fiber.StatusBadRequest, "payment exceeds invoice balance", nil)
	}
	now := time.Now()
	payment := &models.Payment{
		ID:        uuid.New(),
		CreatedAt: now,
		UserID:    userID,
		InvoiceID: invoiceID,
		Amount:    input.Amount,
		Method:    input.Method,
		PaidOn:    paidOn,
		TxnID:     input.TxnID,
		Notes:     input.Notes,
	}
	if err := db.CreatePayment(payment); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to create payment", nil)
	}
	// Auto-mark the invoice paid when payments now cover the total, so the
	// stored status column stays in sync (display also uses effective_status).
	if invoice.Status != models.InvoiceStatusPaid && paid.Add(input.Amount).GreaterThanOrEqual(invoice.Total) && invoice.Total.GreaterThan(decimal.Zero) {
		_ = db.UpdateInvoiceStatus(userID, invoiceID, models.InvoiceStatusPaid)
	}
	recordAudit(c, db, userID, "payment.create", "payment", payment.ID.String(),
		`{"invoice_id":"`+invoiceID.String()+`","amount":"`+input.Amount.String()+`"}`)
	invalidateAggregates(c, userID)
	enqueueNotification(db, userID, models.NotifEventPaymentCreated, "", paymentNotifData(db, *payment))
	return utils.OK(c, fiber.StatusCreated, fiber.Map{"payment": fiber.Map{
		"id":         payment.ID,
		"invoice_id": payment.InvoiceID,
		"amount":     payment.Amount,
		"method":     payment.Method,
		"paid_on":    utils.FormatDate(payment.PaidOn),
		"txn_id":     payment.TxnID,
		"notes":      payment.Notes,
	}})
}

// VoidPayment voids a payment owned by the current user instead of deleting
// it. The row stays for audit but is excluded from every balance, list and
// aggregate query, so it disappears from the frontend.
// @Description Void a payment.
// @Summary void payment
// @Tags Payments
// @Accept json
// @Produce json
// @Param id path string true "Payment ID"
// @Param reason query string false "Void reason (required, also accepted as JSON body {reason})"
// @Param request body models.PaymentVoidInput false "Void payload (alternative to query)"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Param Idempotency-Key header string false "Replay protection key (uuid per void intent)"
// @Router /payments/{id} [delete]
func VoidPayment(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid payment id", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	payment, err := db.GetPayment(userID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "payment not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load payment", nil)
	}
	if payment.VoidedAt != nil {
		return utils.Fail(c, fiber.StatusConflict, "payment is already voided", nil)
	}
	// Gateway-settled payments are provider ledger entries: voiding them
	// here would silently diverge from the gateway's record.
	if !payment.CanVoid() {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, "gateway payment cannot be voided", nil)
	}
	// A pending overlay locks money movement: voiding would change the
	// balance the live Snap intent settles against.
	if _, err := db.GetInvoice(userID, payment.InvoiceID); err == nil {
		if db.PendingInvoiceIDs(userID)[payment.InvoiceID] {
			return utils.Fail(c, fiber.StatusUnprocessableEntity, "invoice has a pending payment", nil)
		}
	}
	reason := strings.TrimSpace(c.Query("reason"))
	if reason == "" && len(c.Body()) > 0 {
		input := &models.PaymentVoidInput{}
		if err := c.Bind().Body(input); err != nil {
			return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
		}
		reason = strings.TrimSpace(input.Reason)
	}
	if reason == "" {
		return utils.Fail(c, fiber.StatusBadRequest, "void reason is required", nil)
	}
	if err := utils.NewValidator().Var(reason, "lte=500"); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "void reason is too long", nil)
	}
	if err := db.VoidPayment(userID, id, reason); err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.Fail(c, fiber.StatusConflict, "payment is already voided", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to void payment", nil)
	}
	// Reopen the invoice when a void drops payments below the total again.
	if invoice, err := db.GetInvoice(userID, payment.InvoiceID); err == nil {
		if invoice.Status == models.InvoiceStatusPaid {
			if remaining, err := db.PaidAmount(payment.InvoiceID); err == nil && remaining.LessThan(invoice.Total) {
				_ = db.UpdateInvoiceStatus(userID, payment.InvoiceID, models.InvoiceStatusSent)
			}
		}
	}
	meta, _ := json.Marshal(fiber.Map{
		"invoice_id": payment.InvoiceID.String(),
		"amount":     payment.Amount,
		"reason":     reason,
	})
	recordAudit(c, db, userID, "payment.void", "payment", id.String(), string(meta))
	voidData := paymentNotifData(db, payment)
	voidData["void_reason"] = reason
	enqueueNotification(db, userID, models.NotifEventPaymentVoided, "", voidData)
	invalidateAggregates(c, userID)
	return utils.OK(c, fiber.StatusOK, fiber.Map{"payment": fiber.Map{
		"id":         payment.ID,
		"invoice_id": payment.InvoiceID,
		"voided":     true,
	}})
}

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
	userID, err := utils.CurrentUserID(c)
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
	invoice, err := db.GetInvoice(userID, invoiceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice", nil)
	}
	if effectiveInvoiceStatus(*db, invoice) == models.InvoiceStatusDraft {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, "invoice is still a draft", nil)
	}
	// A second link would open a second Snap intent for the same invoice;
	// the existing link (returned below when present) stays usable.
	if db.PendingInvoiceIDs(userID)[invoiceID] {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, "invoice has a pending payment", nil)
	}
	link, err := ensurePaymentLink(*db, userID, invoice)
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
	userID, err := utils.CurrentUserID(c)
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
	invoice, err := db.GetInvoice(userID, invoiceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice", nil)
	}
	if effectiveInvoiceStatus(*db, invoice) == models.InvoiceStatusDraft {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, "invoice is still a draft", nil)
	}
	link, err := ensurePaymentLink(*db, userID, invoice)
	if err != nil {
		return linkFail(c, err)
	}
	// Fail fast when mail is not configured (501 contract), then queue
	// for async delivery by the worker.
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
