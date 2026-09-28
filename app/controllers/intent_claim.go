package controllers

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/platform/database"
	"gorm.io/gorm"
)

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
// within seconds, so an older claim can never complete and must not block
// a new click for the full wait window.
const publicClaimStale = 60 * time.Second

// waitForClaim polls a claimed order id until its charge resolves, returning the payable winner or nil when the claim failed or outlived the deadline.
// missing resolves a vanished claim row (the relay flow deletes its CLM- slot while swapping in the real intent); adopt-in-place claims pass nil and keep polling.
func waitForClaim(ctx context.Context, db database.Queries, orderID string, balance decimal.Decimal, missing func() *models.GatewayTransaction) *models.GatewayTransaction {
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
			if missing != nil && errors.Is(err, sql.ErrNoRows) {
				return missing()
			}
			continue
		}
		if txn.Status == publicIntentProcessing {
			if time.Since(txn.CreatedAt) > publicClaimStale {
				return nil
			}
			continue
		}
		if reusableIntent(txn, balance) {
			return &txn
		}
		return nil
	}
	return nil
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
