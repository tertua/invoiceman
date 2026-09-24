package outbox

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/platform/database"
	"github.com/tertua/invoiceman/platform/midtrans"
)

// seedReconcileInvoice creates a user with one sent invoice for reconcile tests.
func seedReconcileInvoice(t *testing.T, db *database.Queries, total string) (uuid.UUID, models.Invoice) {
	t.Helper()
	uid := uuid.New()
	require.NoError(t, db.CreateUser(&models.User{
		ID: uid, Name: "Recon", Email: "recon-" + uid.String() + "@example.com",
		PasswordHash: "x", UserStatus: 1, UserRole: "user",
	}))
	inv := models.Invoice{
		ID: uuid.New(), UserID: uid, Status: models.InvoiceStatusSent,
		Currency: "IDR", Total: decimal.RequireFromString(total),
	}
	require.NoError(t, db.CreateInvoice(uid, &inv, nil))
	return uid, inv
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
	FetchTxStatus = func(ctx context.Context, orderID string) (*midtrans.TxStatus, error) {
		return &midtrans.TxStatus{
			OrderID: orderID, Status: status, TransactionID: "txn-1",
			PaymentType: "qris", GrossAmount: decimal.RequireFromString(gross),
		}, nil
	}
	return func() { FetchTxStatus = old }
}

// TestReconcileExpired mirrors an expire answer into the stored transaction.
func TestReconcileExpired(t *testing.T) {
	t.Setenv("MIDTRANS_SERVER_KEY", "test")
	db := testDB(t)
	uid, inv := seedReconcileInvoice(t, db, "100000")
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
	uid, inv := seedReconcileInvoice(t, db, "100000")
	orderID := seedPendingTxn(t, db, uid, &inv)
	defer stubFetchTxStatus(models.GatewayStatusSuccess, "100000")()

	(&Worker{batch: 20}).reconcileGateway(context.Background())

	txn, err := db.GetTransaction(orderID)
	require.NoError(t, err)
	assert.Equal(t, models.GatewayStatusSuccess, txn.Status)
	paid, err := db.PaidAmount(inv.ID)
	require.NoError(t, err)
	assert.True(t, paid.Equal(decimal.RequireFromString("100000")), "got %s", paid.String())
	updated, err := db.GetInvoice(uid, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusPaid, updated.Status)
}

// TestReconcilePendingUnchanged only touches a still-pending transaction.
func TestReconcilePendingUnchanged(t *testing.T) {
	t.Setenv("MIDTRANS_SERVER_KEY", "test")
	db := testDB(t)
	uid, inv := seedReconcileInvoice(t, db, "100000")
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
	uid, inv := seedReconcileInvoice(t, db, "100000")
	orderID := seedPendingTxn(t, db, uid, &inv)

	(&Worker{batch: 20}).reconcileGateway(context.Background())

	txn, err := db.GetTransaction(orderID)
	require.NoError(t, err)
	assert.Equal(t, models.GatewayStatusPending, txn.Status)
}
