package controllers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/shopspring/decimal"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/app/queries"
	"github.com/tertua/tupay/pkg/logger"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"
	"github.com/tertua/tupay/platform/gateway"
	"gorm.io/gorm"
)

// createRelayIntent creates one provider charge for a service project. The
// caller (CreateIntent) already enforced the Idempotency-Key header, so an
// exact retry replays the stored response in middleware and never reaches
// here. What remains is the concurrent-submit gap: two requests with
// different keys for the same external id. An order id slot is claimed in
// the database before charging, so the loser joins the winner instead of
// opening a second payable charge at the provider.
func createRelayIntent(c fiber.Ctx) error {
	project, err := utils.CurrentServiceProject(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized", nil)
	}
	input := &models.IntentInput{}
	if err := c.Bind().Body(input); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
	}
	if err := utils.NewValidator().Struct(input); err != nil {
		return utils.ValidationFailed(c, err)
	}
	if input.AmountIDR <= 0 && strings.TrimSpace(input.AmountDecimal) == "" {
		return utils.Fail(c, fiber.StatusBadRequest, "amount_idr or amount_decimal is required", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}

	// The canonical amount is expressed in the request currency; amount_idr
	// implies IDR. Conversion to the provider's currency happens after routing.
	invoiceCurrency := strings.ToUpper(strings.TrimSpace(input.Currency))
	if invoiceCurrency == "" {
		invoiceCurrency = "IDR"
	}
	balance := models.MoneyFromMinor(input.AmountIDR)
	if raw := strings.TrimSpace(input.AmountDecimal); raw != "" {
		parsed, err := decimal.NewFromString(raw)
		if err != nil {
			return utils.Fail(c, fiber.StatusBadRequest, "invalid amount_decimal", nil)
		}
		balance = parsed
	} else {
		invoiceCurrency = "IDR"
	}

	// Reuse a pending intent for the same external id + amount (safe retry).
	// The method only has to match when the retry asks for one explicitly, so
	// provider-resolved defaults (e.g. NOWPayments "crypto") still reuse.
	// Claim rows are invisible here by construction (LatestRelayIntent skips
	// the CLM- namespace); a fresh one is joined below instead.
	if existing, err := db.LatestRelayIntent(project.Slug, input.ExternalOrderID); err == nil {
		requested := normalizedPaymentMethod(input.PaymentMethod)
		methodMatches := requested == "" || existing.PaymentMethod == requested
		sameAmount := existing.InvoiceCurrency == invoiceCurrency && existing.InvoiceAmount.Equal(balance)
		if existing.InvoiceCurrency == "" { // pre-conversion rows
			sameAmount = existing.AmountIDR == input.AmountIDR && existing.AmountDecimal == strings.TrimSpace(input.AmountDecimal)
		}
		if existing.Status == models.GatewayStatusPending && sameAmount && methodMatches && existing.ProviderToken != "" {
			return utils.OK(c, fiber.StatusOK, intentResponse(existing))
		}
	}

	gw, err := routeIntentGateway(input, project.DefaultGateway)
	if err != nil {
		if errors.Is(err, gateway.ErrUnsupportedPaymentMethod) {
			return utils.Fail(c, fiber.StatusBadRequest, "unsupported payment method", nil)
		}
		return utils.Fail(c, fiber.StatusBadRequest, "unknown payment gateway", nil)
	}
	if provider, ok := gw.(gateway.DecimalAmountProvider); ok && provider.RequiresDecimalAmount() && strings.TrimSpace(input.AmountDecimal) == "" {
		return utils.Fail(c, fiber.StatusBadRequest, "amount_decimal is required for this payment method", nil)
	}
	usdToIdr := decimal.Zero
	if project.OwnerUserID != nil {
		if s, err := db.GetSettings(project.OrgID); err == nil {
			usdToIdr = s.UsdToIdr
		}
	}
	spec, err := buildCharge(gw, invoiceCurrency, balance, usdToIdr)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "currency conversion is not configured", nil)
	}
	applyEnabledMethods(&spec, gw)
	method := normalizedPaymentMethod(input.PaymentMethod)
	claimID := relayClaimID(project.Slug, input.ExternalOrderID, method, invoiceCurrency, balance)
	now := time.Now()
	claim := &models.GatewayTransaction{
		OrderID: claimID, ProjectSlug: project.Slug, Gateway: gw.Name(), PaymentMethod: method,
		ExternalOrderID: input.ExternalOrderID,
		AmountIDR:       spec.AmountMinor, AmountDecimal: spec.AmountDecimal,
		Currency: spec.Currency, InvoiceCurrency: invoiceCurrency,
		InvoiceAmount: spec.InvoiceAmount, UsdToIdr: spec.UsdToIdr,
		CustomerEmail: input.CustomerEmail, CustomerPhone: input.CustomerPhone,
		Status: publicIntentProcessing, CreatedAt: now, UpdatedAt: now,
	}
	slot, cerr := claimPublicOrderID(*db, claim)
	if cerr != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to store transaction", nil)
	}
	swap := slot
	if !slot {
		// A sibling request owns this slot: join its charge instead of
		// opening a second one. Only a stuck claim falls through, and the
		// provider call below is then genuinely the only live attempt.
		if winner := waitForClaim(c.Context(), *db, claimID, balance, relayClaimLookup(*db, project.Slug, input.ExternalOrderID, balance)); winner != nil {
			return utils.OK(c, fiber.StatusOK, intentResponse(*winner))
		}
		logger.L().Warn("relay intent claim stuck, charging without slot", "project", project.Slug, "external", input.ExternalOrderID)
	}
	orderID, err := relayOrderID(project.Slug, input.ExternalOrderID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to create order", nil)
	}
	created, err := gw.CreateTransaction(c.Context(), spec.request(orderID, input.CustomerEmail, input.CustomerPhone, method))
	if err != nil {
		if swap {
			claim.Status = models.GatewayStatusFailed
			claim.UpdatedAt = time.Now()
			if serr := db.SaveTransaction(claim); serr != nil {
				logger.L().Warn("relay intent claim store failed", "order_id", claim.OrderID, "err", serr)
			}
		}
		if errors.Is(err, gateway.ErrNotConfigured) {
			return utils.Fail(c, fiber.StatusNotImplemented, "payment gateway is not configured", nil)
		}
		return utils.Fail(c, fiber.StatusBadGateway, "failed to create gateway transaction", nil)
	}
	raw, merr := json.Marshal(input)
	if merr != nil {
		logger.L().Warn("relay intent audit payload marshal failed", "project", project.Slug, "external", input.ExternalOrderID, "err", merr)
		raw = []byte("{}")
	}
	finished := time.Now()
	txn := &models.GatewayTransaction{
		OrderID:         orderID,
		ProjectSlug:     project.Slug,
		Gateway:         gw.Name(),
		PaymentMethod:   created.PaymentMethod,
		ExternalOrderID: input.ExternalOrderID,
		AmountIDR:       spec.AmountMinor,
		AmountDecimal:   spec.AmountDecimal,
		Currency:        spec.Currency,
		InvoiceCurrency: invoiceCurrency,
		InvoiceAmount:   spec.InvoiceAmount,
		UsdToIdr:        spec.UsdToIdr,
		CustomerEmail:   input.CustomerEmail,
		CustomerPhone:   input.CustomerPhone,
		Status:          models.GatewayStatusPending,
		ProviderToken:   created.Token,
		RedirectURL:     created.RedirectURL,
		PaymentURL:      created.PaymentURL,
		Address:         created.Address,
		RawIntent:       string(raw),
		CreatedAt:       finished,
		UpdatedAt:       finished,
	}
	if swap {
		if err := swapRelayClaim(*db, claimID, txn); err != nil {
			logger.L().Warn("relay intent swap failed", "order_id", orderID, "err", err)
			return utils.Fail(c, fiber.StatusInternalServerError, "failed to store transaction", nil)
		}
	} else if err := db.CreateTransaction(txn); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to store transaction", nil)
	}
	return utils.OK(c, fiber.StatusCreated, intentResponse(*txn))
}

// relayClaimID derives the deterministic claim slot for one relayed intent.
// Equal content maps to one slot (concurrent submits serialize on it);
// anything different charges independently, exactly like distinct intents.
func relayClaimID(projectSlug, external, method, currency string, amount decimal.Decimal) string {
	sum := sha256.Sum256([]byte(projectSlug + "\n" + external + "\n" + method + "\n" + currency + "\n" + amount.String()))
	return queries.RelayClaimPrefix + sanitizeExternal(projectSlug) + "-" + hex.EncodeToString(sum[:])[:16]
}

// relayClaimLookup resolves a vanished CLM- claim: the winner swapped it for the real intent, so the newest payable relay row for this external id is returned.
func relayClaimLookup(db database.Queries, projectSlug, external string, balance decimal.Decimal) func() *models.GatewayTransaction {
	return func() *models.GatewayTransaction {
		swapped, err := db.LatestRelayIntent(projectSlug, external)
		if err != nil || !reusableIntent(swapped, balance) {
			return nil
		}
		return &swapped
	}
}

// swapRelayClaim atomically exchanges a held claim slot for the charged
// intent. The provider never saw the claim id, so no notification or poll
// can ever reference it; deletion is safe and immediate.
func swapRelayClaim(db database.Queries, claimID string, created *models.GatewayTransaction) error {
	return db.GatewayQueries.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&models.GatewayTransaction{}, "order_id = ?", claimID).Error; err != nil {
			return err
		}
		return tx.Create(created).Error
	})
}
