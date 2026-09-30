package queries

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tertua/tupay/app/models"
)

// SettlementStatusRow aggregates one transaction status for the admin settlement summary.
type SettlementStatusRow struct {
	Status    string `gorm:"column:status" json:"status"`
	Count     int64  `gorm:"column:count" json:"count"`
	AmountIDR int64  `gorm:"column:amount_idr" json:"amount_idr"`
}

// SettlementSummary groups every gateway transaction by status with a count and the summed IDR amount (the physical column is amount_id_r — default GORM naming, which ignores the struct's db tag).
func (q *GatewayQueries) SettlementSummary() ([]SettlementStatusRow, error) {
	out := []SettlementStatusRow{}
	err := q.Model(&models.GatewayTransaction{}).
		Select("status, COUNT(*) AS count, COALESCE(SUM(amount_id_r), 0) AS amount_idr").
		Group("status").
		Order("status").
		Scan(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SaveTransactionAndSettleInvoice atomically stores a successful local
// transaction and its invoice payment. GatewayOrderID makes webhook replays
// idempotent; the invoice row lock serializes distinct payments against its
// remaining balance on PostgreSQL (SQLite serializes writers itself).
// method is the human payment label recorded on the payment row.
func (q *GatewayQueries) SaveTransactionAndSettleInvoice(t *models.GatewayTransaction, gross decimal.Decimal, method string) error {
	return q.Transaction(func(tx *gorm.DB) error {
		if t.InvoiceID == nil || t.UserID == nil {
			return tx.Save(t).Error
		}

		var invoice models.Invoice
		loadErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", *t.InvoiceID, *t.UserID).
			First(&invoice).Error
		if loadErr != nil && !errors.Is(loadErr, gorm.ErrRecordNotFound) {
			return loadErr
		}

		// Record the provider's success even when the invoice was deleted
		// after the intent was created (nothing left to settle).
		if err := tx.Save(t).Error; err != nil {
			return err
		}
		if loadErr != nil {
			return nil
		}

		var existing models.Payment
		err := tx.Where("gateway_order_id = ?", t.OrderID).First(&existing).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var paid decimal.Decimal
		if err := tx.Model(&models.Payment{}).Where("invoice_id = ? AND voided_at IS NULL", invoice.ID).
			Select("COALESCE(SUM(amount), 0)").Scan(&paid).Error; err != nil {
			return err
		}
		balance := invoice.Total.Sub(paid)
		if !balance.GreaterThan(decimal.Zero) {
			return nil
		}
		amount := balance
		if gross.GreaterThan(decimal.Zero) && gross.LessThan(balance) {
			amount = gross
		}
		now := time.Now()
		orderID := t.OrderID
		if err := tx.Create(&models.Payment{
			ID: uuid.New(), CreatedAt: now, UserID: *t.UserID,
			OrgID: invoice.OrgID, InvoiceID: invoice.ID, Amount: amount,
			Method: method, PaidOn: &now,
			TxnID: t.ProviderTxnID, GatewayOrderID: &orderID,
			Notes: method + " " + t.OrderID,
		}).Error; err != nil {
			return err
		}
		// Auto-mark paid when the settlement covers the invoice total.
		if invoice.Status != models.InvoiceStatusPaid && invoice.Total.GreaterThan(decimal.Zero) && paid.Add(amount).GreaterThanOrEqual(invoice.Total) {
			if err := tx.Model(&models.Invoice{}).Where("id = ?", invoice.ID).
				Updates(map[string]any{"status": models.InvoiceStatusPaid, "updated_at": now}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// OrgIDForTransaction resolves the owning org: its invoice when present, else the active org of the payer.
func (q *GatewayQueries) OrgIDForTransaction(txn *models.GatewayTransaction) (uuid.UUID, error) {
	if txn.InvoiceID != nil {
		var invoice models.Invoice
		if err := q.Where("id = ?", *txn.InvoiceID).First(&invoice).Error; err == nil {
			return invoice.OrgID, nil
		}
	}
	if txn.UserID != nil {
		return (&OrgQueries{DB: q.DB}).ResolveActiveOrgID(*txn.UserID, uuid.Nil)
	}
	return uuid.Nil, errors.New("transaction has neither invoice nor user to resolve an org")
}
