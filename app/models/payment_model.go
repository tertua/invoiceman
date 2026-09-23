package models

import (
	"time"

	"github.com/google/uuid"
)

// PaymentInput describes a recorded payment.
type PaymentInput struct {
	InvoiceID string          `json:"invoiceId" validate:"required,uuid4"`
	Amount    Money `json:"amount"`
	Method    string          `json:"method" validate:"required,lte=50"`
	PaidOn    string          `json:"paid_on" validate:"required"`
	TxnID     string          `json:"txn_id" validate:"lte=255"`
	Notes     string          `json:"notes"`
}

// PaymentVoidInput describes a void request for a recorded payment.
type PaymentVoidInput struct {
	Reason string `json:"reason" validate:"required,lte=500"`
}

// OnlineLinkEmailInput describes a request to email a public payment link.
type OnlineLinkEmailInput struct {
	InvoiceID string `json:"invoiceId" validate:"required,uuid4"`
	Email     string `json:"email" validate:"required,email,lte=255"`
}

// PaymentListRow contains payment data formatted for the payments page.
type PaymentListRow struct {
	PaymentID       uuid.UUID       `db:"payment_id"`
	InvoiceID       uuid.UUID       `db:"invoice_id"`
	InvoiceNumber   string          `db:"invoice_number"`
	ClientName      string          `db:"client_name"`
	InvoiceCurrency string          `db:"invoice_currency"`
	Amount          Money `gorm:"type:decimal(19,4)" db:"amount"`
	Method          string          `db:"method"`
	PaidOn          *time.Time      `db:"paid_on"`
	TxnID           string          `db:"txn_id"`
	Notes           string          `db:"notes"`
	GatewayOrderID  *string         `db:"gateway_order_id"`
}

// CanVoid reports whether a payment may be voided from the UI. Gateway
// settlements are provider ledger entries and are rejected by VoidPayment.
func (r PaymentListRow) CanVoid() bool {
	return !gatewaySettled(r.GatewayOrderID)
}
