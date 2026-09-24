package outbox

// Gateway reconciliation polls Midtrans for stale pending transactions and
// mirrors the answer into gateway_transactions. Per-method expiry (QRIS is
// ~15 minutes after the QR is issued, VA ~24 hours) lives at Midtrans, so
// the worker never guesses expiry locally: the provider is the source of
// truth and stored status only changes on its answer. Success settles local
// invoices through the same idempotent path as webhooks, so missed
// notifications heal in both directions (paid and expired).

import (
	"context"
	"strings"
	"time"

	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/configs"
	"github.com/tertua/invoiceman/pkg/logger"
	"github.com/tertua/invoiceman/platform/cache"
	"github.com/tertua/invoiceman/platform/database"
	"github.com/tertua/invoiceman/platform/gateway"
	"github.com/tertua/invoiceman/platform/midtrans"
)

// reconcileAge spaces provider polls per transaction: the UpdatedAt touch on
// every attempt (success or failure) is the backoff, so no extra state is
// needed and several replicas naturally converge.
const reconcileAge = 15 * time.Minute

// FetchTxStatus polls one order. It is a variable so tests can stub the
// provider without network access.
var FetchTxStatus = func(ctx context.Context, orderID string) (*midtrans.TxStatus, error) {
	return midtrans.FetchStatus(ctx, midtrans.FromEnv(), orderID)
}

// reconcileGateway polls stale pending transactions once per tick.
func (w *Worker) reconcileGateway(ctx context.Context) {
	if strings.TrimSpace(configs.Get().Midtrans.ServerKey) == "" {
		return // dev without a gateway: nothing to reconcile
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		logger.L().Warn("outbox reconcile tick skipped, database unavailable", "err", err)
		return
	}
	rows, err := db.StalePendingTransactions(time.Now().Add(-reconcileAge), w.batch)
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
		w.reconcileOne(db, txn)
	}
}

// reconcileOne mirrors one provider answer into the stored transaction.
// The UpdatedAt touch always persists, spacing the next poll even when the
// provider is unreachable or still reports pending.
func (w *Worker) reconcileOne(db *database.Queries, txn models.GatewayTransaction) {
	now := time.Now()
	txn.UpdatedAt = now
	st, err := FetchTxStatus(context.Background(), txn.OrderID)
	if err != nil {
		_ = db.SaveTransaction(&txn)
		logger.L().Warn("outbox reconcile poll failed", "order_id", txn.OrderID, "err", err)
		return
	}
	if st.Status == txn.Status {
		_ = db.SaveTransaction(&txn)
		return
	}
	txn.Status = st.Status
	txn.MidtransTxnID = st.TransactionID
	txn.PaymentType = st.PaymentType
	if st.Status == models.GatewayStatusSuccess {
		txn.PaidAt = &now
	}
	if st.Status == models.GatewayStatusSuccess && txn.ProjectSlug == "local" && txn.InvoiceID != nil && txn.UserID != nil {
		gross := gateway.SettleAmount(st.GrossAmount, gateway.FiatIDR, txn.InvoiceCurrency, txn.UsdToIdr)
		if err := db.SaveTransactionAndSettleInvoice(&txn, gross, reconcileMethod(txn.Gateway)); err != nil {
			logger.L().Warn("outbox reconcile settle failed", "order_id", txn.OrderID, "err", err)
			return
		}
		_ = cache.InvalidateUser(context.Background(), txn.UserID.String())
		logger.L().Info("outbox reconcile settled invoice", "order_id", txn.OrderID)
		return
	}
	_ = db.SaveTransaction(&txn)
	logger.L().Info("outbox reconcile updated transaction", "order_id", txn.OrderID, "status", txn.Status)
}

// reconcileMethod mirrors the webhook payment label for locally settled rows.
func reconcileMethod(provider string) string {
	if provider == "" || provider == "midtrans" {
		return "Midtrans"
	}
	return provider
}
