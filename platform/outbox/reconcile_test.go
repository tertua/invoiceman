package outbox

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/platform/database"
	"github.com/tertua/tupay/platform/gateway"
	"github.com/tertua/tupay/platform/midtrans"
)

// seedReconcileInvoice creates a user with one sent invoice for reconcile tests, returning the owning org first.
func seedReconcileInvoice(t *testing.T, db *database.Queries, total string) (orgID, uid uuid.UUID, inv models.Invoice) {
	t.Helper()
	orgID = uuid.New()
	uid = uuid.New()
	require.NoError(t, db.CreateUser(&models.User{
		ID: uid, Name: "Recon", Email: "recon-" + uid.String() + "@example.com",
		PasswordHash: "x", UserStatus: 1, UserRole: "user",
	}))
	inv = models.Invoice{
		ID: uuid.New(), UserID: uid, OrgID: orgID, Status: models.InvoiceStatusSent,
		Currency: "IDR", Total: decimal.RequireFromString(total),
	}
	require.NoError(t, db.CreateInvoice(orgID, &inv, nil))
	return orgID, uid, inv
}

func seedPendingTxn(t *testing.T, db *database.Queries, uid uuid.UUID, inv *models.Invoice) string {
	t.Helper()
	orderID := "PAY-TEST-" + uuid.NewString()[:8]
	old := time.Now().Add(-time.Hour)
	txn := &models.GatewayTransaction{
		OrderID: orderID, ProjectSlug: "local", Gateway: "midtrans",
		AmountIDR: 100000, Currency: "IDR", Status: models.GatewayStatusPending,
		CreatedAt: old, UpdatedAt: old,
	}
	if inv != nil {
		txn.InvoiceID = &inv.ID
		txn.UserID = &uid
	}
	require.NoError(t, db.CreateTransaction(txn))
	return orderID
}

func stubFetchTxStatus(status, gross string) func() {
	old := FetchTxStatus
	FetchTxStatus = func(ctx context.Context, txn models.GatewayTransaction) (*txStatus, error) {
		return &txStatus{
			Status: status, TransactionID: "txn-1", PaymentType: "qris",
			GrossAmount: decimal.RequireFromString(gross), Currency: gateway.FiatIDR,
		}, nil
	}
	return func() { FetchTxStatus = old }
}

// TestReconcileExpired mirrors an expire answer into the stored transaction.
func TestReconcileExpired(t *testing.T) {
	t.Setenv("MIDTRANS_SERVER_KEY", "test")
	db := testDB(t)
	_, uid, inv := seedReconcileInvoice(t, db, "100000")
	orderID := seedPendingTxn(t, db, uid, &inv)
	defer stubFetchTxStatus(models.GatewayStatusExpired, "0")()

	(&Worker{batch: 20}).reconcileGateway(context.Background())

	txn, err := db.GetTransaction(orderID)
	require.NoError(t, err)
	assert.Equal(t, models.GatewayStatusExpired, txn.Status)
	paid, err := db.PaidAmount(inv.ID)
	require.NoError(t, err)
	assert.True(t, paid.IsZero(), "expired must not record a payment")
}

// TestReconcileSuccessSettles settles a local invoice like the webhook path.
func TestReconcileSuccessSettles(t *testing.T) {
	t.Setenv("MIDTRANS_SERVER_KEY", "test")
	db := testDB(t)
	orgID, uid, inv := seedReconcileInvoice(t, db, "100000")
	orderID := seedPendingTxn(t, db, uid, &inv)
	defer stubFetchTxStatus(models.GatewayStatusSuccess, "100000")()

	(&Worker{batch: 20}).reconcileGateway(context.Background())

	txn, err := db.GetTransaction(orderID)
	require.NoError(t, err)
	assert.Equal(t, models.GatewayStatusSuccess, txn.Status)
	paid, err := db.PaidAmount(inv.ID)
	require.NoError(t, err)
	assert.True(t, paid.Equal(decimal.RequireFromString("100000")), "got %s", paid.String())
	updated, err := db.GetInvoice(orgID, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusPaid, updated.Status)
}

// TestReconcilePendingUnchanged only touches a still-pending transaction.
func TestReconcilePendingUnchanged(t *testing.T) {
	t.Setenv("MIDTRANS_SERVER_KEY", "test")
	db := testDB(t)
	_, uid, inv := seedReconcileInvoice(t, db, "100000")
	orderID := seedPendingTxn(t, db, uid, &inv)
	defer stubFetchTxStatus(models.GatewayStatusPending, "0")()

	(&Worker{batch: 20}).reconcileGateway(context.Background())

	txn, err := db.GetTransaction(orderID)
	require.NoError(t, err)
	assert.Equal(t, models.GatewayStatusPending, txn.Status)
	assert.WithinDuration(t, time.Now(), txn.UpdatedAt, 2*time.Minute, "poll must space the next attempt")
}

// TestReconcileSkippedWithoutGateway leaves rows alone when unconfigured.
func TestReconcileSkippedWithoutGateway(t *testing.T) {
	t.Setenv("MIDTRANS_SERVER_KEY", "")
	db := testDB(t)
	_, uid, inv := seedReconcileInvoice(t, db, "100000")
	orderID := seedPendingTxn(t, db, uid, &inv)

	(&Worker{batch: 20}).reconcileGateway(context.Background())

	txn, err := db.GetTransaction(orderID)
	require.NoError(t, err)
	assert.Equal(t, models.GatewayStatusPending, txn.Status)
}

// TestReconcileUnknownOrderMarkedFailed terminally fails a row Midtrans never
// heard of: no payment can arrive, the invoice stays untouched, and the row
// leaves the pending poll set so the worker stops warning about it.
func TestReconcileUnknownOrderMarkedFailed(t *testing.T) {
	t.Setenv("MIDTRANS_SERVER_KEY", "test")
	db := testDB(t)
	orgID, uid, inv := seedReconcileInvoice(t, db, "100000")
	orderID := seedPendingTxn(t, db, uid, &inv)
	old := FetchTxStatus
	FetchTxStatus = func(ctx context.Context, txn models.GatewayTransaction) (*txStatus, error) {
		return nil, midtrans.ErrOrderNotFound
	}
	defer func() { FetchTxStatus = old }()

	(&Worker{batch: 20}).reconcileGateway(context.Background())

	txn, err := db.GetTransaction(orderID)
	require.NoError(t, err)
	assert.Equal(t, models.GatewayStatusFailed, txn.Status)
	paid, err := db.PaidAmount(inv.ID)
	require.NoError(t, err)
	assert.True(t, paid.IsZero(), "unknown order must not record a payment")
	updated, err := db.GetInvoice(orgID, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusSent, updated.Status)
}

// seedNowPaymentsTxn stores a stale pending direct-payment row the way the
// public pay widget creates one: payment id in SnapToken plus a pay address.
func seedNowPaymentsTxn(t *testing.T, db *database.Queries, uid uuid.UUID, inv *models.Invoice, paymentID, address string) string {
	t.Helper()
	orderID := "PAY-NP-" + uuid.NewString()[:8]
	old := time.Now().Add(-time.Hour)
	txn := &models.GatewayTransaction{
		OrderID: orderID, ProjectSlug: "local", Gateway: "nowpayments",
		SnapToken: paymentID, Address: address, PayAmount: "5.55", PayCurrency: "usdttrc20",
		AmountIDR: 100000, Currency: "USD", InvoiceCurrency: "IDR",
		InvoiceAmount: decimal.RequireFromString("100000"), Status: models.GatewayStatusPending,
		CreatedAt: old, UpdatedAt: old, InvoiceID: &inv.ID, UserID: &uid,
	}
	require.NoError(t, db.CreateTransaction(txn))
	return orderID
}

// TestReconcileNowPaymentsSettles polls NOWPayments for a direct payment and
// settles the invoice through the IPN's own rules (a USD price carries no
// minor amount, so a finished payment credits the full balance).
func TestReconcileNowPaymentsSettles(t *testing.T) {
	np := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/payment/pay_np_1" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"payment_id":"pay_np_1","order_id":"PAY-NP","payment_status":"finished","price_amount":5.55,"price_currency":"USD","pay_amount":"5.55","pay_currency":"usdttrc20"}`))
	}))
	defer np.Close()
	t.Setenv("NOWPAYMENTS_API_KEY", "test")
	t.Setenv("NOWPAYMENTS_BASE_URL", np.URL+"/v1")
	t.Setenv("MIDTRANS_SERVER_KEY", "test")

	db := testDB(t)
	orgID, uid, inv := seedReconcileInvoice(t, db, "100000")
	orderID := seedNowPaymentsTxn(t, db, uid, &inv, "pay_np_1", "TAddrNP")

	(&Worker{batch: 20}).reconcileGateway(context.Background())

	txn, err := db.GetTransaction(orderID)
	require.NoError(t, err)
	assert.Equal(t, models.GatewayStatusSuccess, txn.Status)
	assert.Equal(t, "pay_np_1", txn.MidtransTxnID)
	assert.Equal(t, "usdttrc20", txn.PaymentType)
	paid, err := db.PaidAmount(inv.ID)
	require.NoError(t, err)
	assert.True(t, paid.Equal(decimal.RequireFromString("100000")), "finished must credit the balance, got %s", paid.String())
	updated, err := db.GetInvoice(orgID, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusPaid, updated.Status)
}

// TestReconcileNowPaymentsHostedInvoiceSkipped keeps a hosted invoice row
// untouched: without a payment id there is nothing to ask the provider.
func TestReconcileNowPaymentsHostedInvoiceSkipped(t *testing.T) {
	calls := 0
	np := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer np.Close()
	t.Setenv("NOWPAYMENTS_API_KEY", "test")
	t.Setenv("NOWPAYMENTS_BASE_URL", np.URL+"/v1")
	t.Setenv("MIDTRANS_SERVER_KEY", "test")

	db := testDB(t)
	_, uid, inv := seedReconcileInvoice(t, db, "100000")
	orderID := seedNowPaymentsTxn(t, db, uid, &inv, "inv_hosted", "")

	(&Worker{batch: 20}).reconcileGateway(context.Background())

	txn, err := db.GetTransaction(orderID)
	require.NoError(t, err)
	assert.Equal(t, models.GatewayStatusPending, txn.Status)
	assert.Zero(t, calls, "hosted invoice rows must not hit the provider")
	assert.WithinDuration(t, time.Now(), txn.UpdatedAt, 2*time.Minute, "the touch still spaces the next attempt")
}
