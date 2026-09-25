package controllers

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/shopspring/decimal"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/logger"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"
	"github.com/tertua/invoiceman/platform/gateway"
)

// publicIntentRequest is the optional body for creating a public pay intent.
type publicIntentRequest struct {
	PaymentMethod string `json:"payment_method" validate:"omitempty,lte=32"`
	PayCurrency   string `json:"pay_currency" validate:"omitempty,lte=32"`
}

// createPublicGatewayIntent opens (or reuses) the gateway intent for an
// invoice's outstanding balance. method is the provider-neutral choice from
// the public page; an empty method keeps the legacy Midtrans default. The
// charge is converted to the provider's currency with the owner's manual rate
// when they differ.
func createPublicGatewayIntent(c fiber.Ctx, db database.Queries, link models.PaymentLink, invoice models.Invoice, balance decimal.Decimal, method string, payCurrency string) error {
	method = normalizedPaymentMethod(method)
	settings, err := db.GetSettings(link.UserID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load settings", nil)
	}
	gw, err := routePublicGateway(method, midtransMethodAllowlist(settings))
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "unsupported payment method", nil)
	}
	spec, err := buildCharge(gw, invoice.Currency, balance, settings.UsdToIdr)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "currency conversion is not configured", nil)
	}
	applyEnabledMethods(&spec, gw.Name(), settings)
	orderID := localOrderID(invoice.InvoiceNumber, publicIntentSuffix(link.Token, method))
	if existing, err := db.GetTransaction(orderID); err == nil {
		return utils.OK(c, fiber.StatusOK, publicIntentResponse(existing))
	}
	// Backward compat: reuse the pre-rename INV- intent for the same
	// invoice+link instead of opening a duplicate at the gateway.
	if method == "" {
		if existing, err := db.GetTransaction(legacyLocalOrderID(invoice.InvoiceNumber, link.Token[:8])); err == nil {
			return utils.OK(c, fiber.StatusOK, publicIntentResponse(existing))
		}
	}
	req := spec.request(orderID, "", "", method)
	if method == gateway.MethodQRIS {
		req.DirectQRIS = true
	}
	if gw.Name() == "nowpayments" && payCurrency != "" {
		req.PayCurrency = strings.ToUpper(strings.TrimSpace(payCurrency))
	}
	created, err := gw.CreateTransaction(c.Context(), req)
	if err != nil {
		if errors.Is(err, gateway.ErrNotConfigured) {
			return utils.Fail(c, fiber.StatusNotImplemented, "payment gateway is not configured", nil)
		}
		if errors.Is(err, gateway.ErrAmountBelowMinimum) {
			return utils.Fail(c, fiber.StatusBadRequest, "payment amount is below the gateway minimum", fiber.Map{"code": "amount_below_minimum"})
		}
		if errors.Is(err, gateway.ErrRateLimited) {
			return utils.Fail(c, fiber.StatusTooManyRequests, "payment gateway is rate limited, please retry", fiber.Map{"code": "rate_limited"})
		}
		logger.L().Warn("public gateway intent failed", "gateway", gw.Name(), "method", method, "err", err)
		return utils.Fail(c, fiber.StatusBadGateway, "failed to create gateway transaction", nil)
	}
	now := time.Now()
	txn := &models.GatewayTransaction{
		OrderID: orderID, ProjectSlug: "local", Gateway: gw.Name(), PaymentMethod: method,
		ExternalOrderID: invoice.ID.String(),
		InvoiceID:       &invoice.ID, UserID: &link.UserID, AmountIDR: spec.AmountMinor, AmountDecimal: spec.AmountDecimal,
		Currency: spec.Currency, InvoiceCurrency: strings.ToUpper(strings.TrimSpace(invoice.Currency)),
		InvoiceAmount: spec.InvoiceAmount, UsdToIdr: spec.UsdToIdr,
		Status: models.GatewayStatusPending, SnapToken: created.Token, RedirectURL: created.RedirectURL,
		PaymentURL: created.PaymentURL, Address: created.Address, PayAmount: created.RawPayload, PayCurrency: created.PayCurrency,
		ExpiresAt: created.ExpiresAt, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.CreateTransaction(txn); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to store transaction", nil)
	}
	return utils.OK(c, fiber.StatusOK, publicIntentResponse(*txn))
}

// routePublicGateway resolves the provider for a public charge: an explicit
// method routes across configured providers, an empty method keeps Midtrans
// for backward compatibility. The owner's Midtrans allowlist vetoes Midtrans
// for disallowed methods while leaving other providers untouched.
func routePublicGateway(method string, allow map[string]bool) (gateway.Gateway, error) {
	if method == "" {
		return gateway.Get("midtrans")
	}
	return gateway.RouteWhere("", method, func(provider, m string) bool {
		return provider != "midtrans" || methodAllowed(allow, m)
	})
}

// publicIntentSuffix keeps distinct methods from colliding on one order id,
// while an empty method preserves the legacy token-only suffix.
func publicIntentSuffix(token, method string) string {
	if method == "" {
		return token[:8]
	}
	return token[:8] + "-" + sanitizeExternal(method)
}

func publicIntentResponse(t models.GatewayTransaction) fiber.Map {
	out := fiber.Map{
		"order_id":     t.OrderID,
		"gateway":      t.Gateway,
		"redirect_url": t.RedirectURL,
		"payment_url":  t.PaymentURL,
		"address":      t.Address,
		"pay_amount":   t.PayAmount,
		"pay_currency": t.PayCurrency,
		"expires_at":   t.ExpiresAt,
	}
	if t.Gateway == "midtrans" {
		out["snap_token"] = t.SnapToken
	}
	return out
}
