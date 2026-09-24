package controllers

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/shopspring/decimal"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/configs"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"
	"github.com/tertua/invoiceman/platform/gateway"
)

func publicPaymentData(db database.Queries, link models.PaymentLink) (fiber.Map, error) {
	detail, err := invoiceDetail(db, link.UserID, link.InvoiceID)
	if err != nil {
		return nil, err
	}
	settings, err := db.GetSettings(link.UserID)
	if err != nil {
		return nil, err
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
	if invoice.Status == models.InvoiceStatusDraft {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, "invoice is still a draft", nil)
	}
	paid, err := db.PaidAmount(invoice.ID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice payments", nil)
	}
	if !invoice.Total.GreaterThan(paid) {
		return utils.Fail(c, fiber.StatusBadRequest, "invoice is already paid", nil)
	}
	return createPublicGatewayIntent(c, *db, link, invoice, invoice.Total.Sub(paid))
}

func createPublicGatewayIntent(c fiber.Ctx, db database.Queries, link models.PaymentLink, invoice models.Invoice, balance decimal.Decimal) error {
	if !strings.EqualFold(strings.TrimSpace(invoice.Currency), "IDR") {
		return utils.Fail(c, fiber.StatusBadRequest, "online payment is currently available for IDR invoices only", nil)
	}
	gw, err := gateway.Get("midtrans")
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "payment gateway is not registered", nil)
	}
	orderID := localOrderID(invoice.InvoiceNumber, link.Token[:8])
	if existing, err := db.GetTransaction(orderID); err == nil {
		return utils.OK(c, fiber.StatusOK, fiber.Map{"snap_token": existing.SnapToken, "redirect_url": existing.RedirectURL, "order_id": existing.OrderID})
	}
	// Backward compat: reuse the pre-rename INV- intent for the same
	// invoice+link instead of opening a duplicate at the gateway.
	if existing, err := db.GetTransaction(legacyLocalOrderID(invoice.InvoiceNumber, link.Token[:8])); err == nil {
		return utils.OK(c, fiber.StatusOK, fiber.Map{"snap_token": existing.SnapToken, "redirect_url": existing.RedirectURL, "order_id": existing.OrderID})
	}
	amountIDR := balance.IntPart()
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
