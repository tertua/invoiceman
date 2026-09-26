package controllers

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/shopspring/decimal"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/logger"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"
	"github.com/tertua/invoiceman/platform/gateway"
	"gorm.io/gorm"
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
func createPublicGatewayIntent(c fiber.Ctx, db database.Queries, link models.PaymentLink, invoice models.Invoice, balance decimal.Decimal, method, payCurrency string) error {
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
	base := localOrderID(invoice.InvoiceNumber, publicIntentSuffix(link.Token, method))
	if method == "" {
		// Legacy default method: keep the pre-rename INV- intent reused
		// forever instead of opening a duplicate at the gateway. Dead
		// claims (failed before reaching the provider) and in-flight
		// claims are not returned as-is; they fall through to the claim
		// loop below, which adopts or joins them.
		if existing, err := db.GetTransaction(base); err == nil && !isDeadClaim(existing) {
			if existing.Status == publicIntentProcessing {
				if winner := waitForPublicIntent(c.Context(), db, base, balance); winner != nil {
					return utils.OK(c, fiber.StatusOK, publicIntentResponse(*winner))
				}
			} else {
				return utils.OK(c, fiber.StatusOK, publicIntentResponse(existing))
			}
		}
		if existing, err := db.GetTransaction(legacyLocalOrderID(invoice.InvoiceNumber, link.Token[:8])); err == nil {
			return utils.OK(c, fiber.StatusOK, publicIntentResponse(existing))
		}
	}
	// Money path: the order id is claimed in the database BEFORE calling the
	// provider, so two concurrent clicks (StrictMode double-effect,
	// double-tap, retry) serialize on the primary key instead of opening two
	// charges at the gateway. The loser waits for the winner's charge and
	// reuses its QR; a failed claim is marked failed so the next attempt
	// advances to a fresh -rN suffix. A crashed claim stays "processing"
	// (never reusable, never polled by the reconciler) and is skipped the
	// same way.
	for attempt := 0; attempt < 3; attempt++ {
		orderID, reuse, rerr := publicRetryOrderID(c.Context(), db, base, invoice, method, balance)
		if rerr != nil {
			return utils.Fail(c, fiber.StatusInternalServerError, "failed to load gateway transaction", nil)
		}
		if reuse != nil {
			return utils.OK(c, fiber.StatusOK, publicIntentResponse(*reuse))
		}
		now := time.Now()
		claim := &models.GatewayTransaction{
			OrderID: orderID, ProjectSlug: "local", Gateway: gw.Name(), PaymentMethod: method,
			ExternalOrderID: invoice.ID.String(),
			InvoiceID:       &invoice.ID, UserID: &link.UserID, AmountIDR: spec.AmountMinor, AmountDecimal: spec.AmountDecimal,
			Currency: spec.Currency, InvoiceCurrency: strings.ToUpper(strings.TrimSpace(invoice.Currency)),
			InvoiceAmount: spec.InvoiceAmount, UsdToIdr: spec.UsdToIdr,
			Status: publicIntentProcessing, CreatedAt: now, UpdatedAt: now,
		}
		// Claim the order id, adopting a dead claim left by a previous
		// attempt that never reached the provider. A live row owned by a
		// sibling click is joined via wait; anything else advances the
		// loop to a fresh suffix.
		claimed, err := claimPublicOrderID(db, claim)
		if err != nil {
			return utils.Fail(c, fiber.StatusInternalServerError, "failed to store transaction", nil)
		}
		if !claimed {
			if winner := waitForPublicIntent(c.Context(), db, orderID, balance); winner != nil {
				return utils.OK(c, fiber.StatusOK, publicIntentResponse(*winner))
			}
			continue
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
			claim.Status = models.GatewayStatusFailed
			claim.UpdatedAt = time.Now()
			if serr := db.SaveTransaction(claim); serr != nil {
				logger.L().Warn("public gateway intent claim store failed", "order_id", claim.OrderID, "err", serr)
			}
			if errors.Is(err, gateway.ErrNotConfigured) {
				return utils.Fail(c, fiber.StatusNotImplemented, "payment gateway is not configured", nil)
			}
			if errors.Is(err, gateway.ErrAmountBelowMinimum) {
				return utils.Fail(c, fiber.StatusBadRequest, "payment amount is below the gateway minimum", fiber.Map{"code": "amount_below_minimum"})
			}
			if errors.Is(err, gateway.ErrRateLimited) {
				return utils.Fail(c, fiber.StatusTooManyRequests, "payment gateway is rate limited, please retry", fiber.Map{"code": "rate_limited"})
			}
			if isDuplicateOrderError(err) {
				if winner, werr := db.LatestIntent("local", invoice.ID.String(), method); werr == nil && reusableIntent(winner, balance) {
					return utils.OK(c, fiber.StatusOK, publicIntentResponse(winner))
				}
				continue
			}
			logger.L().Warn("public gateway intent failed", "gateway", gw.Name(), "method", method, "err", err)
			return utils.Fail(c, fiber.StatusBadGateway, "failed to create gateway transaction", nil)
		}
		claim.Gateway = gw.Name()
		claim.Status = models.GatewayStatusPending
		claim.SnapToken = created.Token
		claim.RedirectURL = created.RedirectURL
		claim.PaymentURL = created.PaymentURL
		claim.Address = created.Address
		claim.PayAmount = created.RawPayload
		claim.PayCurrency = created.PayCurrency
		claim.ExpiresAt = created.ExpiresAt
		claim.UpdatedAt = time.Now()
		if err := db.SaveTransaction(claim); err != nil {
			logger.L().Warn("public gateway intent store failed", "order_id", orderID, "err", err)
			return utils.Fail(c, fiber.StatusInternalServerError, "failed to store transaction", nil)
		}
		return utils.OK(c, fiber.StatusOK, publicIntentResponse(*claim))
	}
	logger.L().Warn("public gateway intent race exhausted", "method", method, "invoice", invoice.ID.String())
	return utils.Fail(c, fiber.StatusBadGateway, "failed to create gateway transaction", nil)
}

// publicIntentProcessing marks an order id claimed in the database while its
// provider charge is still in flight. It is deliberately not one of the
// GatewayStatus values: reusableIntent only reuses pending rows and the
// reconciler only polls pending rows, so a claim is invisible to both until
// it completes. Status is a free-form string column, so no migration is
// needed for this value.
const publicIntentProcessing = "processing"

// publicClaimWait bounds how long a losing click waits for the winner's
// charge before falling through to a retry suffix. The winner's Core API
// call typically resolves in ~2s against a 15s timeout, so this covers it
// without holding the payer's request.
const publicClaimWait = 12 * time.Second

// publicClaimStep is the poll interval while waiting for a claimed intent.
const publicClaimStep = 100 * time.Millisecond

// publicClaimStale is the age past which a "processing" claim is treated as
// crashed instead of in flight: the provider call it guards always resolves
// within seconds, so an older claim can never complete and must not block a
// new click for the full wait window.
const publicClaimStale = 60 * time.Second

// waitForPublicIntent polls a claimed order id until its charge completes.
// It returns the winner while it is still payable, or nil when the claim
// failed or is still in flight past the deadline (the caller then retries
// under a fresh suffix).
func waitForPublicIntent(ctx context.Context, db database.Queries, orderID string, balance decimal.Decimal) *models.GatewayTransaction {
	deadline := time.Now().Add(publicClaimWait)
	step := publicClaimStep
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(step):
		}
		step *= 2
		if step > time.Second {
			step = time.Second
		}
		txn, err := db.GetTransaction(orderID)
		if err != nil {
			continue
		}
		if txn.Status == publicIntentProcessing {
			continue
		}
		if reusableIntent(txn, balance) {
			return &txn
		}
		return nil
	}
	return nil
}

// publicRetryOrderID resolves the order id for one attempt: a still-payable
// intent is returned for reuse, otherwise the next retry suffix (-rN) is
// appended because the provider rejects duplicate order ids. The legacy
// empty method keeps its suffix-less id forever. A fresh "processing" claim
// (a sibling click whose charge is still in flight) is waited on instead of
// being suffixed past — suffixing it would open a second charge at the
// gateway for the same user click.
func publicRetryOrderID(ctx context.Context, db database.Queries, base string, invoice models.Invoice, method string, balance decimal.Decimal) (string, *models.GatewayTransaction, error) {
	if method == "" {
		return base, nil, nil
	}
	latest, err := db.LatestIntent("local", invoice.ID.String(), method)
	switch {
	case err == nil:
		if latest.Status == publicIntentProcessing && time.Since(latest.CreatedAt) < publicClaimStale {
			if winner := waitForPublicIntent(ctx, db, latest.OrderID, balance); winner != nil {
				return "", winner, nil
			}
		}
		if isDeadClaim(latest) {
			return latest.OrderID, nil, nil
		}
		if reusableIntent(latest, balance) {
			return "", &latest, nil
		}
		n, cerr := db.CountIntents("local", invoice.ID.String(), method)
		if cerr != nil {
			return "", nil, cerr
		}
		return base + "-r" + strconv.FormatInt(n, 10), nil, nil
	case errors.Is(err, sql.ErrNoRows):
		return base, nil, nil
	default:
		return "", nil, err
	}
}

// isDeadClaim reports a claim that provably never reached the provider: it
// failed while holding no provider payload (no Snap token, redirect, QR, or
// deposit address). Its order id is safe to adopt because the gateway never
// saw it, so adopting can never double-charge.
func isDeadClaim(t models.GatewayTransaction) bool {
	if t.Status != models.GatewayStatusFailed {
		return false
	}
	return t.SnapToken == "" && t.RedirectURL == "" && t.PaymentURL == "" && t.Address == ""
}

// claimPublicOrderID atomically claims an order id for the caller's charge.
// A fresh id is inserted; a collision on a dead claim (failed before reaching
// the provider) or a stale one (processing past publicClaimStale, whose
// guarded provider call can never still be running) adopts that row (reset
// to processing); a collision on anything else reports false so the caller
// joins the live claim via wait. A non-duplicate storage error is returned.
func claimPublicOrderID(db database.Queries, claim *models.GatewayTransaction) (bool, error) {
	if err := db.CreateTransaction(claim); err == nil {
		return true, nil
	} else if !isDuplicateKeyError(err) {
		return false, err
	}
	existing, gerr := db.GetTransaction(claim.OrderID)
	if gerr != nil {
		return false, nil
	}
	if !isDeadClaim(existing) && !isStaleClaim(existing) {
		return false, nil
	}
	existing.Status = publicIntentProcessing
	existing.UpdatedAt = time.Now()
	if serr := db.SaveTransaction(&existing); serr != nil {
		return false, serr
	}
	*claim = existing
	return true, nil
}

// isStaleClaim reports a processing claim older than publicClaimStale. The
// provider call it guarded always resolves within seconds, so an older claim
// belongs to a crashed attempt and its slot may be adopted.
func isStaleClaim(t models.GatewayTransaction) bool {
	return t.Status == publicIntentProcessing && time.Since(t.CreatedAt) > publicClaimStale
}
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	// Typed check first: both dialectors translate unique violations to
	// gorm.ErrDuplicatedKey (TranslateError is enabled in platform/database).
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	// Fallback for untranslated driver errors.
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique constraint") || strings.Contains(msg, "unique_index") || strings.Contains(msg, "primary key")
}

// isDuplicateOrderError reports a provider-side duplicate order id rejection
// (Midtrans refuses to recharge an order id it already holds).
func isDuplicateOrderError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "already used") || strings.Contains(msg, "order_id")
}

// reusableIntent reports whether a stored public intent can still be paid:
// pending, not past the expiry the provider itself reported (QRIS ~15 minutes,
// crypto deposit windows too; Snap/gopay rows carry none), and still covering
// the current balance. Failed/settled/foreign-amount rows are never reused.
func reusableIntent(t models.GatewayTransaction, balance decimal.Decimal) bool {
	if t.Status != models.GatewayStatusPending {
		return false
	}
	if t.ExpiresAt != "" {
		if at, err := time.Parse(time.RFC3339, t.ExpiresAt); err == nil && !at.After(time.Now()) {
			return false
		}
	}
	// Rows written before conversion existed carry no currency to compare.
	return t.InvoiceCurrency == "" || t.InvoiceAmount.Equal(balance)
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
