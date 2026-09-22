package controllers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/configs"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"
	"github.com/tertua/invoiceman/platform/gateway"
	"github.com/tertua/invoiceman/platform/mail"
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
	}
}

// ListPayments returns payments and totals for the current user.
// @Description Get payments of current user.
// @Summary get payments
// @Tags Payments
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
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
	rows, err := db.ListPayments(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load payments", nil)
	}
	now := time.Now()
	total := 0.0
	thisMonth := 0.0
	payments := make([]fiber.Map, 0, len(rows))
	for _, row := range rows {
		total += row.Amount
		if row.PaidOn != nil && row.PaidOn.Year() == now.Year() && row.PaidOn.Month() == now.Month() {
			thisMonth += row.Amount
		}
		payments = append(payments, paymentResponse(row))
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{
		"payments": payments,
		"totals": fiber.Map{
			"total":     total,
			"thisMonth": thisMonth,
		},
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
// @Security ApiKeyAuth
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
	if paid+input.Amount > invoice.Total {
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

// DeletePayment deletes a payment owned by the current user.
// @Description Delete a payment.
// @Summary delete payment
// @Tags Payments
// @Produce json
// @Param id path string true "Payment ID"
// @Success 204 {string} status "ok"
// @Security ApiKeyAuth
// @Router /payments/{id} [delete]
func DeletePayment(c fiber.Ctx) error {
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
	if _, err := db.GetPayment(userID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "payment not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load payment", nil)
	}
	if err := db.DeletePayment(userID, id); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to delete payment", nil)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// CreateOnlineLink creates a public payment link without contacting a gateway.
// @Description Create a public payment link.
// @Summary create payment link
// @Tags Payments
// @Accept json
// @Produce json
// @Param request body map[string]string true "Invoice ID"
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
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
	if invoice.Status == models.InvoiceStatusDraft {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, "invoice is still a draft", nil)
	}
	link, err := db.GetPaymentLinkForInvoice(invoiceID, userID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusInternalServerError, "failed to load payment link", nil)
		}
		if invoice.Status != models.InvoiceStatusSent {
			return utils.Fail(c, fiber.StatusBadRequest, "invoice is already paid", nil)
		}
		paid, err := db.PaidAmount(invoiceID)
		if err != nil {
			return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice payments", nil)
		}
		if invoice.Total <= paid {
			return utils.Fail(c, fiber.StatusBadRequest, "invoice is already paid", nil)
		}
		token, tokenErr := newPaymentToken()
		if tokenErr != nil {
			return utils.Fail(c, fiber.StatusInternalServerError, "failed to create payment link", nil)
		}
		link = models.PaymentLink{Token: token, InvoiceID: invoiceID, UserID: userID, CreatedAt: time.Now()}
		if err := db.CreatePaymentLink(&link); err != nil {
			return utils.Fail(c, fiber.StatusInternalServerError, "failed to create payment link", nil)
		}
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
// @Security ApiKeyAuth
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
	if invoice.Status == models.InvoiceStatusDraft {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, "invoice is still a draft", nil)
	}
	link, err := db.GetPaymentLinkForInvoice(invoiceID, userID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusInternalServerError, "failed to load payment link", nil)
		}
		if invoice.Status != models.InvoiceStatusSent {
			return utils.Fail(c, fiber.StatusBadRequest, "invoice is already paid", nil)
		}
		paid, err := db.PaidAmount(invoiceID)
		if err != nil {
			return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice payments", nil)
		}
		if invoice.Total <= paid {
			return utils.Fail(c, fiber.StatusBadRequest, "invoice is already paid", nil)
		}
		token, tokenErr := newPaymentToken()
		if tokenErr != nil {
			return utils.Fail(c, fiber.StatusInternalServerError, "failed to create payment link", nil)
		}
		link = models.PaymentLink{Token: token, InvoiceID: invoiceID, UserID: userID, CreatedAt: time.Now()}
		if err := db.CreatePaymentLink(&link); err != nil {
			return utils.Fail(c, fiber.StatusInternalServerError, "failed to create payment link", nil)
		}
	}
	mailer, err := mail.NewFromEnv()
	if errors.Is(err, mail.ErrNotConfigured) {
		return utils.Fail(c, fiber.StatusNotImplemented, "email provider is not configured", nil)
	}
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "email provider configuration is invalid", nil)
	}
	url := publicURL("/pay/" + link.Token)
	body := fmt.Sprintf("Hello,\n\nPlease use the following link to pay invoice %s:\n%s\n\nThank you.", invoice.InvoiceNumber, url)
	if err := mailer.Send(input.Email, "Payment link for invoice "+invoice.InvoiceNumber, body); err != nil {
		return utils.Fail(c, fiber.StatusBadGateway, "failed to send payment link email", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"message": "payment link sent"})
}

func publicPaymentData(db database.Queries, link models.PaymentLink) (fiber.Map, error) {
	detail, err := invoiceDetail(db, link.UserID, link.InvoiceID)
	if err != nil {
		return nil, err
	}
	settings, err := db.GetSettings(link.UserID)
	if err != nil {
		return nil, err
	}
	return fiber.Map{
		"invoice": detail,
		"branding": fiber.Map{
			"company_name": settings.CompanyName,
			"logo_url":     settings.LogoURL,
		},
		"gateway": fiber.Map{
			"name":          "midtrans",
			"client_key":    configs.Get().Midtrans.ClientKey,
			"is_production": configs.Get().Midtrans.IsProd,
		},
		"can_pay": detail["effective_status"] != models.InvoiceStatusPaid,
	}, nil
}

// GetPublicPayment returns invoice data for a public payment token.
// @Description Get a public payment invoice.
// @Summary get public payment
// @Tags Public Payments
// @Produce json
// @Param token path string true "Payment token"
// @Success 200 {object} map[string]interface{}
// @Router /public/pay/{token} [get]
func GetPublicPayment(c fiber.Ctx) error {
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	link, err := db.GetPaymentLink(c.Params("token"))
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "payment link not found", nil)
	}
	invoice, err := db.GetInvoice(link.UserID, link.InvoiceID)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
	}
	if invoice.Status == models.InvoiceStatusDraft {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, "invoice is still a draft", nil)
	}
	data, err := publicPaymentData(*db, link)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
	}
	return utils.OK(c, fiber.StatusOK, data)
}

// CreatePublicTransaction creates a real gateway intent for the invoice balance.
// @Description Create a public payment gateway transaction.
// @Summary create public payment transaction
// @Tags Public Payments
// @Produce json
// @Param token path string true "Payment token"
// @Success 200 {object} map[string]interface{}
// @Router /public/pay/{token}/transaction [post]
func CreatePublicTransaction(c fiber.Ctx) error {
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	link, err := db.GetPaymentLink(c.Params("token"))
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "payment link not found", nil)
	}
	invoice, err := db.GetInvoice(link.UserID, link.InvoiceID)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
	}
	if invoice.Status == models.InvoiceStatusDraft {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, "invoice is still a draft", nil)
	}
	paid, err := db.PaidAmount(invoice.ID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice payments", nil)
	}
	if invoice.Total <= paid {
		return utils.Fail(c, fiber.StatusBadRequest, "invoice is already paid", nil)
	}
	return createPublicGatewayIntent(c, *db, link, invoice, invoice.Total-paid)
}

func createPublicGatewayIntent(c fiber.Ctx, db database.Queries, link models.PaymentLink, invoice models.Invoice, balance float64) error {
	if !strings.EqualFold(strings.TrimSpace(invoice.Currency), "IDR") {
		return utils.Fail(c, fiber.StatusBadRequest, "online payment is currently available for IDR invoices only", nil)
	}
	gw, err := gateway.Get("midtrans")
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "payment gateway is not registered", nil)
	}
	orderID := "INV-" + strings.ReplaceAll(invoice.InvoiceNumber, " ", "") + "-" + link.Token[:8]
	if existing, err := db.GetTransaction(orderID); err == nil {
		return utils.OK(c, fiber.StatusOK, fiber.Map{"snap_token": existing.SnapToken, "redirect_url": existing.RedirectURL, "order_id": existing.OrderID})
	}
	amountIDR := int64(balance + 0.5)
	created, err := gw.CreateTransaction(c.Context(), &gateway.CreateTxRequest{OrderID: orderID, AmountMinor: amountIDR, Currency: invoice.Currency})
	if err != nil {
		if errors.Is(err, gateway.ErrNotConfigured) {
			return utils.Fail(c, fiber.StatusNotImplemented, "payment gateway is not configured", nil)
		}
		return utils.Fail(c, fiber.StatusBadGateway, "failed to create gateway transaction", nil)
	}
	now := time.Now()
	txn := &models.GatewayTransaction{
		OrderID: orderID, ProjectSlug: "local", Gateway: "midtrans", ExternalOrderID: invoice.ID.String(),
		InvoiceID: &invoice.ID, UserID: &link.UserID, AmountIDR: amountIDR, Currency: invoice.Currency,
		Status: models.GatewayStatusPending, SnapToken: created.Token, RedirectURL: created.RedirectURL,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := db.CreateTransaction(txn); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to store transaction", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"snap_token": txn.SnapToken, "redirect_url": txn.RedirectURL, "order_id": txn.OrderID})
}

// GetPublicPaymentStatus returns the current public payment status.
// @Description Get public payment status.
// @Summary get public payment status
// @Tags Public Payments
// @Produce json
// @Param token path string true "Payment token"
// @Success 200 {object} map[string]interface{}
// @Router /public/pay/{token}/status [get]
func GetPublicPaymentStatus(c fiber.Ctx) error {
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	link, err := db.GetPaymentLink(c.Params("token"))
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "payment link not found", nil)
	}
	detail, err := invoiceDetail(*db, link.UserID, link.InvoiceID)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"status": detail["effective_status"], "paid": detail["paid_amount"], "balance": detail["balance"]})
}
