package models

import (
	"time"

	"github.com/google/uuid"
)

// Gateway transaction statuses, aligned with the Midtrans notification mapping.
const (
	GatewayStatusPending         = "pending"
	GatewayStatusSuccess         = "success"
	GatewayStatusFailed          = "failed"
	GatewayStatusExpired         = "expired"
	GatewayStatusRefunded        = "refunded"
	GatewayStatusPartialRefunded = "partially_refunded"
)

// GatewayTransaction is the central record for one payment intent.
// OrderID is the global Midtrans order id (EXT-<slug>-<external>-<rand>
// for relayed projects, PAY-<invoice>-<rand> for local invoices;
// pre-rename local rows use the legacy INV- prefix).
// ExternalOrderID preserves the downstream id (e.g. one-api topup_*)
// so downstream services can match without migrating their schema.
type GatewayTransaction struct {
	OrderID     string `gorm:"primaryKey;size:128" db:"order_id" json:"order_id"`
	ProjectSlug string `gorm:"size:64;index" db:"project_slug" json:"project_slug"`
	Gateway     string `gorm:"size:32;index;default:midtrans" db:"gateway" json:"gateway"`
	GatewayTransactionPayment
	ExternalOrderID string     `gorm:"size:128;index" db:"external_order_id" json:"external_order_id"`
	InvoiceID       *uuid.UUID `gorm:"type:uuid;index" db:"invoice_id" json:"invoice_id"`
	UserID          *uuid.UUID `gorm:"type:uuid;index" db:"user_id" json:"user_id"`
	AmountIDR       int64      `db:"amount_idr" json:"amount_idr"`
	AmountDecimal   string     `gorm:"size:64" db:"amount_decimal" json:"amount_decimal"`
	Currency        string     `gorm:"size:8;default:IDR" db:"currency" json:"currency"`
	CustomerEmail   string     `gorm:"size:255" db:"customer_email" json:"customer_email"`
	CustomerPhone   string     `gorm:"size:64" db:"customer_phone" json:"customer_phone"`
	Status          string     `gorm:"size:32;index;default:pending" db:"status" json:"status"`
	SnapToken       string     `gorm:"size:255" db:"snap_token" json:"snap_token"`
	RedirectURL     string     `gorm:"size:1024" db:"redirect_url" json:"redirect_url"`
	PaymentURL      string     `gorm:"size:1024" db:"payment_url" json:"payment_url"`
	Address         string     `gorm:"size:255" db:"address" json:"address"`
	MidtransTxnID   string     `gorm:"size:128" db:"midtrans_txn_id" json:"midtrans_txn_id"`
	PaymentType     string     `gorm:"size:64" db:"payment_type" json:"payment_type"`
	RawIntent       string     `db:"raw_intent" json:"-"`
	RawNotification string     `db:"raw_notification" json:"-"`
	PaidAt          *time.Time `db:"paid_at" json:"paid_at"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
}

// GatewayEvent stores the raw Midtrans notification for audit.
type GatewayEvent struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" db:"id" json:"id"`
	OrderID   string    `gorm:"size:128;index" db:"order_id" json:"order_id"`
	Source    string    `gorm:"size:32" db:"source" json:"source"`
	Verified  bool      `db:"verified" json:"verified"`
	Payload   string    `db:"payload" json:"-"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// WebhookDelivery tracks one forward attempt to a downstream project.
type WebhookDelivery struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" db:"id" json:"id"`
	OrderID     string     `gorm:"size:128;index" db:"order_id" json:"order_id"`
	ProjectSlug string     `gorm:"size:64;index" db:"project_slug" json:"project_slug"`
	Gateway     string     `gorm:"size:32;index;default:midtrans" db:"gateway" json:"gateway"`
	TargetURL   string     `gorm:"size:1024" db:"target_url" json:"target_url"`
	Payload     string     `db:"payload" json:"-"`
	Signature   string     `gorm:"size:128" db:"signature" json:"-"`
	Attempt     int        `db:"attempt" json:"attempt"`
	Status      string     `gorm:"size:16;index;default:pending" db:"status" json:"status"`
	RespCode    int        `db:"resp_code" json:"resp_code"`
	RespBody    string     `db:"resp_body" json:"-"`
	NextRetryAt *time.Time `db:"next_retry_at" json:"next_retry_at"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}
