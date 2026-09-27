package database

import (
	"github.com/tertua/invoiceman/app/models"
	"gorm.io/gorm"
)

// migrations lists every rollback known to this binary, oldest first.
// Ups run implicitly through AutoMigrate at startup; only the Down
// rollback is recorded here, keyed by the SchemaVersion that introduced
// the change.
var migrations = []Migration{
	{
		Version:     2,
		Description: "expense receipt_url + mail_outbox html_body",
		Down: func(db *gorm.DB) error {
			if err := db.Migrator().DropColumn(&models.Expense{}, "receipt_url"); err != nil {
				return err
			}
			return db.Migrator().DropColumn(&models.MailOutbox{}, "html_body")
		},
	},
	{
		Version:     3,
		Description: "admin_claims singleton for atomic first-admin bootstrap",
		Down: func(db *gorm.DB) error {
			return db.Migrator().DropTable(&models.AdminClaim{})
		},
	},
	{
		Version:     4,
		Description: "idempotent local gateway payment settlement",
		Down: func(db *gorm.DB) error {
			return db.Migrator().DropColumn(&models.Payment{}, "gateway_order_id")
		},
	},
	{
		Version:     5,
		Description: "notification endpoints + deliveries",
		Down: func(db *gorm.DB) error {
			if err := db.Migrator().DropTable(&models.NotificationDelivery{}); err != nil {
				return err
			}
			return db.Migrator().DropTable(&models.NotificationEndpoint{})
		},
	},
	{
		Version:     6,
		Description: "payment void columns (voided_at + void_reason)",
		Down: func(db *gorm.DB) error {
			if err := db.Migrator().DropColumn(&models.Payment{}, "voided_at"); err != nil {
				return err
			}
			return db.Migrator().DropColumn(&models.Payment{}, "void_reason")
		},
	},
	{
		Version:     7,
		Description: "settings language (per-user UI locale)",
		Down: func(db *gorm.DB) error {
			return db.Migrator().DropColumn(&models.Settings{}, "language")
		},
	},
	{
		Version:     8,
		Description: "fixed-point decimal money columns",
		Down: func(db *gorm.DB) error {
			// Money fields are part of the base tables; fresh dev databases are
			// recreated for this schema change rather than backfilled in place.
			return nil
		},
	},
	{
		Version:     9,
		Description: "gateway project ownership and external invoice identities",
		Down: func(db *gorm.DB) error {
			if err := db.Migrator().DropColumn(&models.GatewayProject{}, "owner_user_id"); err != nil {
				return err
			}
			if err := db.Migrator().DropColumn(&models.Client{}, "gateway_project_slug"); err != nil {
				return err
			}
			if err := db.Migrator().DropColumn(&models.Client{}, "external_id"); err != nil {
				return err
			}
			if err := db.Migrator().DropColumn(&models.Invoice{}, "gateway_project_slug"); err != nil {
				return err
			}
			return db.Migrator().DropColumn(&models.Invoice{}, "external_id")
		},
	},
	{
		Version:     10,
		Description: "gateway transaction payment method",
		Down: func(db *gorm.DB) error {
			return db.Migrator().DropColumn(&models.GatewayTransaction{}, "payment_method")
		},
	},
	{
		Version:     11,
		Description: "manual USD/IDR gateway conversion",
		Down: func(db *gorm.DB) error {
			if err := db.Migrator().DropColumn(&models.Settings{}, "usd_to_idr"); err != nil {
				return err
			}
			if err := db.Migrator().DropColumn(&models.GatewayTransaction{}, "invoice_currency"); err != nil {
				return err
			}
			if err := db.Migrator().DropColumn(&models.GatewayTransaction{}, "invoice_amount"); err != nil {
				return err
			}
			return db.Migrator().DropColumn(&models.GatewayTransaction{}, "usd_to_idr")
		},
	},
	{
		Version:     12,
		Description: "per-account Midtrans payment method allowlist",
		Down: func(db *gorm.DB) error {
			return db.Migrator().DropColumn(&models.Settings{}, "midtrans_methods")
		},
	},
	{
		Version:     13,
		Description: "nowpayments direct payment fields (pay_amount, pay_currency, expires_at)",
		Down: func(db *gorm.DB) error {
			if err := db.Migrator().DropColumn(&models.GatewayTransaction{}, "pay_amount"); err != nil {
				return err
			}
			if err := db.Migrator().DropColumn(&models.GatewayTransaction{}, "pay_currency"); err != nil {
				return err
			}
			return db.Migrator().DropColumn(&models.GatewayTransaction{}, "expires_at")
		},
	},
	{
		Version:     14,
		Description: "invoice informational payment_method column",
		Down: func(db *gorm.DB) error {
			return db.Migrator().DropColumn(&models.Invoice{}, "payment_method")
		},
	},
	{
		Version:     15,
		Description: "gateway transaction qr_string (on-page QRIS payload)",
		Down: func(db *gorm.DB) error {
			return db.Migrator().DropColumn(&models.GatewayTransaction{}, "qr_string")
		},
	},
}
