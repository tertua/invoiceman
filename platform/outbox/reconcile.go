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
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/configs"
	"github.com/tertua/invoiceman/pkg/logger"
	"github.com/tertua/invoiceman/platform/cache"
	"github.com/tertua/invoiceman/platform/database"
	"github.com/tertua/invoiceman/platform/gateway"
	"github.com/tertua/invoiceman/platform/midtrans"
	"github.com/tertua/invoiceman/platform/nowpayments"
)

// reconcileAge spaces provider polls per transaction: the UpdatedAt touch on
// every attempt (success or failure) is the backoff, so no extra state is
// needed and several replicas naturally converge.
const reconcileAge = 15 * time.Minute

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
	if strings.TrimSpace(txn.SnapToken) == "" || strings.TrimSpace(txn.Address) == "" {
		return nil, errNoPoll
	}
	notif, err := nowpayments.StatusNotification(ctx, nowpayments.FromEnv(), txn.SnapToken)
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
	if configs.Get().Midtrans.ServerKey == "" && nowpayments.FromEnv().APIKey == "" {
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
	st, err := FetchTxStatus(context.Background(), txn)
	if errors.Is(err, errNoPoll) {
		_ = db.SaveTransaction(&txn)
		return
	}
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
		gross := gateway.SettleAmount(st.GrossAmount, st.Currency, txn.InvoiceCurrency, txn.UsdToIdr)
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
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", "midtrans":
		return "Midtrans"
	case "nowpayments":
		return "NOWPayments"
	default:
		return provider
	}
}
