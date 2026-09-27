package controllers

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/shopspring/decimal"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/configs"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"
)

func publicPaymentData(ctx context.Context, db database.Queries, link models.PaymentLink, payCurrency string) (fiber.Map, error) {
	detail, err := invoiceDetail(db, link.UserID, link.InvoiceID)
	if err != nil {
		return nil, err
	}
	settings, err := db.GetSettings(link.UserID)
	if err != nil {
		return nil, err
	}
	invoice, err := db.GetInvoice(link.UserID, link.InvoiceID)
	if err != nil {
		return nil, err
	}
	paid, err := db.PaidAmount(invoice.ID)
	if err != nil {
		return nil, err
	}
	balance := invoice.Total.Sub(paid)
	if balance.IsNegative() {
		balance = decimal.Zero
	}
	delete(detail, "client_id")
	delete(detail, "client_email")
	delete(detail, "payment_link")
	if raw, ok := detail["payments"].([]fiber.Map); ok {
		clean := make([]fiber.Map, 0, len(raw))
		for _, p := range raw {
			clean = append(clean, fiber.Map{
				"id":      p["id"],
				"amount":  p["amount"],
				"paid_on": p["paid_on"],
				"method":  p["method"],
			})
		}
		detail["payments"] = clean
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
		"methods": availableChargeMethods(ctx, invoice.Currency, balance, settings.UsdToIdr, midtransMethodAllowlist(), payCurrency),
		"can_pay": detail["effective_status"] != models.InvoiceStatusPaid,
	}, nil
}

// GetPublicPayment returns invoice data for a public payment token.
// @Description Get a public payment invoice.
// @Summary get public payment
// @Tags Public Payments
// @Produce json
// @Param token path string true "Payment token"
// @Param pay_currency query string false "Crypto asset (e.g. usdtbsc) whose live minimum filters the method list"
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
	if effectiveInvoiceStatus(*db, invoice) == models.InvoiceStatusDraft {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, "invoice is still a draft", nil)
	}
	// Optional crypto asset the payer picked: its own live minimum filters the
	// list. No asset yet means no minimum gate — crypto stays offered.
	payCurrency := strings.TrimSpace(c.Query("pay_currency"))
	if len(payCurrency) > 32 {
		payCurrency = ""
	}
	data, err := publicPaymentData(c.Context(), *db, link, payCurrency)
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
// @Param Idempotency-Key header string false "Replay protection key (uuid per payment intent)"
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
	if effectiveInvoiceStatus(*db, invoice) == models.InvoiceStatusDraft {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, "invoice is still a draft", nil)
	}
	paid, err := db.PaidAmount(invoice.ID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice payments", nil)
	}
	if !invoice.Total.GreaterThan(paid) {
		return utils.Fail(c, fiber.StatusBadRequest, "invoice is already paid", nil)
	}
	input := &publicIntentRequest{}
	_ = c.Bind().Body(input) // body is optional; an empty body keeps the default
	return createPublicGatewayIntent(c, *db, link, invoice, invoice.Total.Sub(paid), input.PaymentMethod, input.PayCurrency)
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
