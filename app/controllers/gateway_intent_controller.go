package controllers

import (
	"database/sql"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"
	"github.com/tertua/tupay/platform/gateway"
	"github.com/tertua/tupay/platform/midtrans"
)

// CreateIntent creates a Midtrans Snap transaction.
// Project identity is taken from the API key (GatewayAuth), never from the body.
// @Description Create a relayed payment intent.
// @Summary create payment intent
// @Tags Gateway
// @Accept json
// @Produce json
// @Param request body models.IntentInput true "Intent payload"
// @Success 201 {object} map[string]interface{}
// @Param Idempotency-Key header string true "Replay protection key (uuid per payment intent)"
// @Router /gateway/intents [post]
func CreateIntent(c fiber.Ctx) error {
	if strings.TrimSpace(c.Get("Idempotency-Key")) == "" {
		return utils.Fail(c, fiber.StatusBadRequest, "Idempotency-Key header is required", nil)
	}
	return createRelayIntent(c)
}

// GetIntent returns one intent owned by the calling project.
// @Description Get a payment intent status.
// @Summary get payment intent
// @Tags Gateway
// @Produce json
// @Param order_id path string true "Global order ID"
// @Success 200 {object} map[string]interface{}
// @Router /gateway/intents/{order_id} [get]
func GetIntent(c fiber.Ctx) error {
	project, err := utils.CurrentServiceProject(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	txn, err := db.GetTransaction(c.Params("order_id"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "transaction not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load transaction", nil)
	}
	if txn.ProjectSlug != project.Slug {
		return utils.Fail(c, fiber.StatusNotFound, "transaction not found", nil)
	}
	return utils.OK(c, fiber.StatusOK, intentResponse(txn))
}

// ListMyTransactions returns one page of recent intents for the calling project.
// @Description List own payment intents.
// @Summary list own intents
// @Tags Gateway
// @Produce json
// @Param page query int false "Page number (default 1)"
// @Param per_page query int false "Items per page (default 20, max 100)"
// @Success 200 {object} map[string]interface{}
// @Router /gateway/transactions [get]
func ListMyTransactions(c fiber.Ctx) error {
	project, err := utils.CurrentServiceProject(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	paging := utils.ParsePagination(c)
	rows, err := db.ListTransactionsByProject(project.Slug, paging.Limit(), paging.Offset())
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load transactions", nil)
	}
	total, err := db.CountTransactionsByProject(project.Slug)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to count transactions", nil)
	}
	out := make([]fiber.Map, 0, len(rows))
	for _, t := range rows {
		out = append(out, intentResponse(t))
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"transactions": out, "meta": paging.Meta(total)})
}

// GatewayConfig returns public browser configuration for the active gateway.
// Server credentials are never exposed here; the client key is intentionally
// public and is required by the provider's browser SDK.
// @Description Get public payment gateway browser configuration.
// @Summary get gateway browser config
// @Tags Gateway
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /gateway/config [get]
func GatewayConfig(c fiber.Ctx) error {
	cfg := midtrans.FromEnv()
	return utils.OK(c, fiber.StatusOK, fiber.Map{
		"gateway":       "midtrans",
		"client_key":    cfg.ClientKey,
		"is_production": cfg.IsProd,
		"configured":    cfg.ServerKey != "",
	})
}

// GatewayStatus returns public per-gateway availability for admins and
// downstream services. Only booleans are exposed here, never keys or
// secrets, so this endpoint is safe to call without authentication.
// @Description Get public payment gateway availability.
// @Summary get gateway status
// @Tags Gateway
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /public/gateway/status [get]
func GatewayStatus(c fiber.Ctx) error {
	names := gateway.Names()
	sort.Strings(names)
	out := make([]fiber.Map, 0, len(names))
	for _, name := range names {
		gw, err := gateway.Get(name)
		if err != nil {
			continue
		}
		status := fiber.Map{"name": name, "configured": gateway.ProviderReady(gw), "sandbox": false}
		if provider, ok := gw.(gateway.SandboxProvider); ok {
			status["sandbox"] = provider.Sandbox()
		}
		out = append(out, status)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"gateways": out})
}

// CreateInvoiceIntent creates a Snap transaction for a local invoice.
// Uses the session user (dashboard), not a service API key. It lives under the
// invoices namespace so it is never swept by the API-key gateway middleware.
// @Description Create a Snap transaction for a local invoice.
// @Summary create invoice intent
// @Tags Invoices
// @Produce json
// @Param id path string true "Invoice ID"
// @Success 201 {object} map[string]interface{}
// @Param Idempotency-Key header string false "Replay protection key (uuid per payment intent)"
// @Router /invoices/{id}/intents [post]
func CreateInvoiceIntent(c fiber.Ctx) error {
	orgID, userID, err := currentUserOrg(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	invoiceID, err := uuid.Parse(c.Params("id"))
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
	if invoice.Status == models.InvoiceStatusDraft {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, "invoice is still a draft", nil)
	}
	paid, err := db.PaidAmount(invoiceID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice payments", nil)
	}
	balance := invoice.Total.Sub(paid)
	if !balance.GreaterThan(decimal.Zero) {
		return utils.Fail(c, fiber.StatusBadRequest, "invoice is already paid", nil)
	}
	gw, err := gateway.Get(gateway.DefaultProvider())
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "payment gateway is not registered", nil)
	}
	settings, err := db.GetSettings(orgID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load settings", nil)
	}
	spec, err := buildCharge(gw, invoice.Currency, balance, settings.UsdToIdr)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "currency conversion is not configured", nil)
	}
	applyEnabledMethods(&spec, gw.Name())
	suffix, err := randHex(4)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to create order", nil)
	}
	orderID := localOrderID(invoice.InvoiceNumber, suffix)
	created, err := gw.CreateTransaction(c.Context(), spec.request(orderID, "", "", ""))
	if err != nil {
		if errors.Is(err, gateway.ErrNotConfigured) {
			return utils.Fail(c, fiber.StatusNotImplemented, "payment gateway is not configured", nil)
		}
		return utils.Fail(c, fiber.StatusBadGateway, "failed to create gateway transaction", nil)
	}
	now := time.Now()
	txn := &models.GatewayTransaction{
		OrderID:         orderID,
		ProjectSlug:     "local",
		Gateway:         gw.Name(),
		ExternalOrderID: invoice.ID.String(),
		InvoiceID:       &invoice.ID,
		UserID:          &userID,
		AmountIDR:       spec.AmountMinor,
		AmountDecimal:   spec.AmountDecimal,
		Currency:        spec.Currency,
		InvoiceCurrency: strings.ToUpper(strings.TrimSpace(invoice.Currency)),
		InvoiceAmount:   spec.InvoiceAmount,
		UsdToIdr:        spec.UsdToIdr,
		Status:          models.GatewayStatusPending,
		SnapToken:       created.Token,
		RedirectURL:     created.RedirectURL,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.CreateTransaction(txn); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to store transaction", nil)
	}
	return utils.OK(c, fiber.StatusCreated, intentResponse(*txn))
}
