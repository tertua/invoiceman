package outbox

// Gateway reconciliation polls the owning provider for stale pending
// transactions and mirrors the answer into gateway_transactions. Midtrans
// rows ask GET /v2/{order_id}/status; NOWPayments direct payments ask
// GET /v1/payment/{payment_id} and settle through the IPN's own rules, so a
// missed webhook heals the same way a delivered one does. Per-method expiry
// (QRIS is ~15 minutes after the QR is issued, VA ~24 hours) lives at the
// provider, so the worker never guesses expiry locally: the provider is the
// source of truth and stored status only changes on its answer. Success
// settles local invoices through the same idempotent path as webhooks, so
// missed notifications heal in both directions (paid and expired).

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/configs"
	"github.com/tertua/tupay/pkg/logger"
	"github.com/tertua/tupay/platform/cache"
	"github.com/tertua/tupay/platform/database"
	"github.com/tertua/tupay/platform/gateway"
	"github.com/tertua/tupay/platform/midtrans"
	"github.com/tertua/tupay/platform/nowpayments"
)

// Poll spacing (GATEWAY_RECONCILE_MINUTES, default 15m): the UpdatedAt touch
// on every attempt (success or failure) is the backoff, so no extra state is
// needed and several replicas naturally converge.

// txStatus is one provider's answer in relay terms. GrossAmount is expressed
// in Currency; a zero gross (NOWPayments prices outside IDR carry no minor
// amount) makes settlement credit the full remaining balance, which is
// exactly what the IPN path does on 'finished'.
type txStatus struct {
	Status        string
	TransactionID string
	PaymentType   string
	GrossAmount   decimal.Decimal
	Currency      string
}

// errNoPoll marks a row that cannot be asked anything yet: a hosted
// NOWPayments invoice has no payment id until its IPN arrives. The
// UpdatedAt touch still runs so the row is re-examined later.
var errNoPoll = errors.New("no provider status to poll yet")

// FetchTxStatus polls one pending row from its provider. It is a variable so
// tests can stub the answer without network access.
var FetchTxStatus = func(ctx context.Context, txn models.GatewayTransaction) (*txStatus, error) {
	if strings.EqualFold(strings.TrimSpace(txn.Gateway), "nowpayments") {
		return nowpaymentsTxStatus(ctx, txn)
	}
	st, err := midtrans.FetchStatus(ctx, midtrans.FromEnv(), txn.OrderID)
	if err != nil {
		return nil, err
	}
	return &txStatus{
		Status: st.Status, TransactionID: st.TransactionID, PaymentType: st.PaymentType,
		GrossAmount: st.GrossAmount, Currency: gateway.FiatIDR,
	}, nil
}

// nowpaymentsTxStatus polls a direct payment by its stored payment id and
// reuses the IPN rendering so status mapping and gross rules never drift
// between the webhook and the worker.
func nowpaymentsTxStatus(ctx context.Context, txn models.GatewayTransaction) (*txStatus, error) {
	if strings.TrimSpace(txn.ProviderToken) == "" || strings.TrimSpace(txn.Address) == "" {
		return nil, errNoPoll
	}
	notif, err := nowpayments.StatusNotification(ctx, nowpayments.FromEnv(), txn.ProviderToken)
	if err != nil {
		return nil, err
	}
	return &txStatus{
		Status: notif.Status, TransactionID: notif.TransactionID, PaymentType: notif.PaymentType,
		GrossAmount: models.MoneyFromMinor(notif.GrossMinor), Currency: notif.Currency,
	}, nil
}

// reconcileGateway polls stale pending transactions once per tick.
func (w *Worker) reconcileGateway(ctx context.Context) {
	if configs.Get().ProviderString(midtrans.GatewayName, "server_key", "") == "" && nowpayments.FromEnv().APIKey == "" {
		return // dev without a gateway: nothing to reconcile
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		logger.L().Warn("outbox reconcile tick skipped, database unavailable", "err", err)
		return
	}
	rows, err := db.StalePendingTransactions(time.Now().Add(-configs.Get().Gateway.ReconcileAge()), w.batch)
	if err != nil {
		logger.L().Warn("outbox reconcile tick failed", "err", err)
		return
	}
	for _, txn := range rows {
		select {
		case <-ctx.Done():
			return
		default:
		}
		w.reconcileOne(ctx, db, txn)
	}
}

// reconcileOne mirrors one provider answer into the stored transaction.
// The UpdatedAt touch always persists, spacing the next poll even when the
// provider is unreachable or still reports pending.
func (w *Worker) reconcileOne(ctx context.Context, db *database.Queries, txn models.GatewayTransaction) {
	now := time.Now()
	txn.UpdatedAt = now
	st, err := FetchTxStatus(ctx, txn)
	if errors.Is(err, errNoPoll) {
		recordErr("save reconcile touch", db.SaveTransaction(&txn), "order_id", txn.OrderID)
		return
	}
	if errors.Is(err, midtrans.ErrOrderNotFound) {
		// The provider never heard of this order, so no payment can arrive:
		// fail the row so it stops polling. The invoice itself is untouched
		// and the next pay click recharges under a fresh order id.
		txn.Status = models.GatewayStatusFailed
		recordErr("save reconcile terminal", db.SaveTransaction(&txn), "order_id", txn.OrderID)
		logger.L().Info("outbox reconcile order unknown at provider, marked failed", "order_id", txn.OrderID)
		return
	}
	if err != nil {
		recordErr("save reconcile touch", db.SaveTransaction(&txn), "order_id", txn.OrderID)
		logger.L().Warn("outbox reconcile poll failed", "order_id", txn.OrderID, "err", err)
		return
	}
	if st.Status == txn.Status {
		recordErr("save reconcile touch", db.SaveTransaction(&txn), "order_id", txn.OrderID)
		return
	}
	txn.Status = st.Status
	txn.ProviderTxnID = st.TransactionID
	txn.PaymentType = st.PaymentType
	if st.Status == models.GatewayStatusSuccess {
		txn.PaidAt = &now
	}
	if st.Status == models.GatewayStatusSuccess && txn.ProjectSlug == "local" && txn.InvoiceID != nil && txn.UserID != nil {
		gross := gateway.SettleAmount(st.GrossAmount, st.Currency, txn.InvoiceCurrency, txn.UsdToIdr)
		if err := db.SaveTransactionAndSettleInvoice(&txn, gross, gateway.DisplayName(txn.Gateway)); err != nil {
			logger.L().Warn("outbox reconcile settle failed", "order_id", txn.OrderID, "err", err)
			return
		}
		orgID, oerr := db.OrgIDForTransaction(&txn)
		recordErr("invalidate org cache", errors.Join(oerr, cache.InvalidateOrg(ctx, orgID.String())), "order_id", txn.OrderID)
		logger.L().Info("outbox reconcile settled invoice", "order_id", txn.OrderID)
		return
	}
	recordErr("save reconciled transaction", db.SaveTransaction(&txn), "order_id", txn.OrderID)
	logger.L().Info("outbox reconcile updated transaction", "order_id", txn.OrderID, "status", txn.Status)
}
