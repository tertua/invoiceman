package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Settings struct to describe per-user business settings.
type Settings struct {
	UserID      uuid.UUID `gorm:"type:uuid;primaryKey" db:"user_id" json:"user_id" validate:"required,uuid"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
	CompanyName string    `db:"company_name" json:"company_name" validate:"lte=255"`
	Email       string    `db:"email" json:"email" validate:"omitempty,email,lte=255"`
	Phone       string    `db:"phone" json:"phone" validate:"lte=100"`
	Address     string    `db:"address" json:"address"`
	LogoURL     string    `db:"logo_url" json:"logo_url"`
	Currency    string    `db:"currency" json:"currency" validate:"required,lte=3"`
	TaxRate     float64   `db:"tax_rate" json:"tax_rate" validate:"gte=0"`
	SettingsGatewayConversion
	SettingsGatewayMethods
	InvoicePrefix string `db:"invoice_prefix" json:"invoice_prefix" validate:"required,lte=20"`
	InvoiceSeq    int    `db:"invoice_seq" json:"-"`
	Language      string `db:"language" json:"language" validate:"omitempty,oneof=en id"`
}

// DefaultSettings returns settings defaults for a new user: a usable first-run
// tax rate and manual USD→IDR rate instead of empty zeros.
func DefaultSettings(userID uuid.UUID) *Settings {
	s := &Settings{UserID: userID, UpdatedAt: time.Now(), Currency: "IDR", TaxRate: 11, InvoicePrefix: "INV-"}
	s.UsdToIdr = decimal.NewFromInt(18000)
	return s
}
