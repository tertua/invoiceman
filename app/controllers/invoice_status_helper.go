package controllers

import (
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/platform/database"
)

// effectiveInvoiceStatus resolves an invoice's display status, letting a live
// gateway transaction (pending) win over a stale stored status such as a
// flipped-back draft. Public/payability gates must use this, not the stored
// status column, or money already in flight reads as an unpayable draft.
func effectiveInvoiceStatus(db database.Queries, invoice models.Invoice) string {
	paid, _ := db.PaidAmount(invoice.ID)
	return models.ResolveEffectiveStatus(invoice.Status, invoice.DueDate, invoice.Total, paid,
		db.PendingInvoiceIDs(invoice.UserID)[invoice.ID])
}
