package models

import (
	"github.com/google/uuid"
	"strings"
	"time"
)

type InvoiceItem struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" db:"id" json:"id" validate:"required,uuid"`
	InvoiceID   uuid.UUID `gorm:"type:uuid" db:"invoice_id" json:"invoice_id" validate:"required,uuid"`
	Description string    `db:"description" json:"description" validate:"required"`
	Quantity    float64   `db:"quantity" json:"quantity" validate:"gte=0"`
	Rate        Money     `gorm:"type:decimal(19,4)" db:"rate" json:"rate"`
	Amount      Money     `gorm:"type:decimal(19,4)" db:"amount" json:"amount"`
	Position    int       `db:"position" json:"position"`
}

type Payment struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey" db:"id" json:"id" validate:"required,uuid"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UserID         uuid.UUID  `gorm:"type:uuid" db:"user_id" json:"user_id" validate:"required,uuid"`
	OrgID          uuid.UUID  `gorm:"type:uuid;index" db:"org_id" json:"org_id"`
	InvoiceID      uuid.UUID  `gorm:"type:uuid" db:"invoice_id" json:"invoice_id" validate:"required,uuid"`
	Amount         Money      `gorm:"type:decimal(19,4)" db:"amount" json:"amount"`
	Method         string     `db:"method" json:"method" validate:"lte=50"`
	PaidOn         *time.Time `db:"paid_on" json:"paid_on"`
	TxnID          string     `db:"txn_id" json:"txn_id" validate:"lte=255"`
	GatewayOrderID *string    `gorm:"uniqueIndex" db:"gateway_order_id" json:"-"`
	VoidedAt       *time.Time `db:"voided_at" json:"voided_at,omitempty"`
	VoidReason     string     `gorm:"size:500" db:"void_reason" json:"void_reason,omitempty" validate:"lte=500"`
	Notes          string     `db:"notes" json:"notes"`
}

func (p Payment) CanVoid() bool { return !gatewaySettled(p.GatewayOrderID) }
func gatewaySettled(orderID *string) bool {
	return orderID != nil && strings.TrimSpace(*orderID) != ""
}
