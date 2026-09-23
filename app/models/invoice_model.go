package models

import (
	"time"

	"github.com/google/uuid"
)

// Invoice statuses used by the frontend.
const (
	InvoiceStatusDraft = "draft"
	InvoiceStatusSent  = "sent"
	InvoiceStatusPaid  = "paid"
)

// EffectiveInvoiceStatus values displayed by the frontend.
const (
	InvoiceEffectiveOverdue = "overdue"
)

// Invoice struct to describe an invoice object.
type Invoice struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey" db:"id" json:"id" validate:"required,uuid"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt     *time.Time `db:"updated_at" json:"updated_at"`
	UserID        uuid.UUID  `gorm:"type:uuid" db:"user_id" json:"user_id" validate:"required,uuid"`
	ClientID      *uuid.UUID `gorm:"type:uuid" db:"client_id" json:"client_id"`
	InvoiceNumber string     `db:"invoice_number" json:"invoice_number" validate:"required,lte=50"`
	Status        string     `db:"status" json:"status" validate:"required,oneof=draft sent paid"`
	IssueDate     *time.Time `db:"issue_date" json:"issue_date"`
	DueDate       *time.Time `db:"due_date" json:"due_date"`
	Currency      string     `db:"currency" json:"currency" validate:"required,lte=3"`
	TaxRate       float64    `db:"tax_rate" json:"tax_rate" validate:"gte=0"`
	Discount      float64    `db:"discount" json:"discount" validate:"gte=0"`
	Notes         string     `db:"notes" json:"notes"`
	Terms         string     `db:"terms" json:"terms"`
	Subtotal      float64    `db:"subtotal" json:"subtotal"`
	TaxAmount     float64    `db:"tax_amount" json:"tax_amount"`
	Total         float64    `db:"total" json:"total"`
}

// InvoiceItem struct to describe a single invoice line.
type InvoiceItem struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" db:"id" json:"id" validate:"required,uuid"`
	InvoiceID   uuid.UUID `gorm:"type:uuid" db:"invoice_id" json:"invoice_id" validate:"required,uuid"`
	Description string    `db:"description" json:"description" validate:"required"`
	Quantity    float64   `db:"quantity" json:"quantity" validate:"gte=0"`
	Rate        float64   `db:"rate" json:"rate" validate:"gte=0"`
	Amount      float64   `db:"amount" json:"amount"`
	Position    int       `db:"position" json:"position"`
}

// Payment struct to describe a payment recorded against an invoice.
// Voided payments are never hard-deleted: VoidedAt/VoidReason mark the
// row excluded from every balance, list and aggregate query.
type Payment struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey" db:"id" json:"id" validate:"required,uuid"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UserID         uuid.UUID  `gorm:"type:uuid" db:"user_id" json:"user_id" validate:"required,uuid"`
	InvoiceID      uuid.UUID  `gorm:"type:uuid" db:"invoice_id" json:"invoice_id" validate:"required,uuid"`
	Amount         float64    `db:"amount" json:"amount" validate:"gt=0"`
	Method         string     `db:"method" json:"method" validate:"lte=50"`
	PaidOn         *time.Time `db:"paid_on" json:"paid_on"`
	TxnID          string     `db:"txn_id" json:"txn_id" validate:"lte=255"`
	GatewayOrderID *string    `gorm:"uniqueIndex" db:"gateway_order_id" json:"-"`
	VoidedAt       *time.Time `db:"voided_at" json:"voided_at,omitempty"`
	VoidReason     string     `gorm:"size:500" db:"void_reason" json:"void_reason,omitempty" validate:"lte=500"`
	Notes          string     `db:"notes" json:"notes"`
}

// InvoiceItemInput struct to describe a single invoice line in create/update payloads.
type InvoiceItemInput struct {
	Description string  `json:"description" validate:"required"`
	Quantity    float64 `json:"quantity" validate:"gte=0"`
	Rate        float64 `json:"rate" validate:"gte=0"`
}

// InvoiceInput struct to describe create/update invoice payload.
type InvoiceInput struct {
	ClientID  *string            `json:"client_id"`
	Status    string             `json:"status" validate:"required,oneof=draft sent paid"`
	IssueDate string             `json:"issue_date"`
	DueDate   string             `json:"due_date"`
	Currency  string             `json:"currency" validate:"required,lte=3"`
	TaxRate   float64            `json:"tax_rate" validate:"gte=0"`
	Discount  float64            `json:"discount" validate:"gte=0"`
	Notes     string             `json:"notes"`
	Terms     string             `json:"terms"`
	Items     []InvoiceItemInput `json:"items" validate:"required,min=1,dive"`
}

// InvoiceStatusInput struct to describe update invoice status payload.
type InvoiceStatusInput struct {
	Status string `json:"status" validate:"required,oneof=draft sent paid"`
}

// InvoiceListRow struct to describe an invoice row for list responses.
type InvoiceListRow struct {
	ID            uuid.UUID  `db:"id" json:"id"`
	InvoiceNumber string     `db:"invoice_number" json:"invoice_number"`
	ClientName    string     `db:"client_name" json:"client_name"`
	ClientCompany string     `db:"client_company" json:"client_company"`
	IssueDate     *time.Time `db:"issue_date" json:"-"`
	DueDate       *time.Time `db:"due_date" json:"-"`
	Total         float64    `db:"total" json:"total"`
	Currency      string     `db:"currency" json:"currency"`
	Status        string     `db:"status" json:"-"`
	PaidAmount    float64    `db:"paid_amount" json:"-"`
}

// EffectiveStatus resolves the display status for an invoice row.
func (r InvoiceListRow) EffectiveStatus() string {
	return ResolveEffectiveStatus(r.Status, r.DueDate, r.Total, r.PaidAmount)
}

// ResolveEffectiveStatus computes the display status from stored status,
// due date, total and paid amount.
func ResolveEffectiveStatus(status string, dueDate *time.Time, total, paid float64) string {
	if status == InvoiceStatusPaid {
		return InvoiceStatusPaid
	}
	if total > 0 && paid >= total {
		return InvoiceStatusPaid
	}
	if status == InvoiceStatusSent && dueDate != nil && time.Now().After(*dueDate) {
		return InvoiceEffectiveOverdue
	}
	return status
}
