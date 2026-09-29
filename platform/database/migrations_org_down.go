package database

import (
	"time"

	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"gorm.io/gorm"
)

// settingsDown mirrors models.Settings as of v15: user_id primary key and every settings column except org_id.
type settingsDown struct {
	UserID      uuid.UUID `gorm:"type:uuid;primaryKey" db:"user_id" json:"user_id" validate:"required,uuid"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
	CompanyName string    `db:"company_name" json:"company_name" validate:"lte=255"`
	Email       string    `db:"email" json:"email" validate:"omitempty,email,lte=255"`
	Phone       string    `db:"phone" json:"phone" validate:"lte=100"`
	Address     string    `db:"address" json:"address"`
	LogoURL     string    `db:"logo_url" json:"logo_url"`
	Currency    string    `db:"currency" json:"currency" validate:"required,lte=3"`
	TaxRate     float64   `db:"tax_rate" json:"tax_rate" validate:"gte=0"`
	models.SettingsGatewayConversion
	models.SettingsGatewayMethods
	InvoicePrefix string `db:"invoice_prefix" json:"invoice_prefix" validate:"required,lte=20"`
	InvoiceSeq    int    `db:"invoice_seq" json:"-"`
	Language      string `db:"language" json:"language" validate:"omitempty,oneof=en id"`
}

// TableName parks the rebuild on a scratch table until it replaces settings.
func (settingsDown) TableName() string { return "settings_down_tmp" }

// dropSettingsOrgID rebuilds settings without org_id: the SQLite migrator drops the column but keeps PRIMARY KEY (org_id), so a plain DropColumn fails.
func dropSettingsOrgID(db *gorm.DB) error {
	if !db.Migrator().HasColumn(&models.Settings{}, "org_id") {
		if db.Migrator().HasTable(&settingsDown{}) { // finish a rollback interrupted between drop and rename
			return db.Migrator().RenameTable(&settingsDown{}, &models.Settings{})
		}
		return nil
	}
	if db.Migrator().HasTable(&settingsDown{}) { // stale scratch table from an interrupted rollback
		if err := db.Migrator().DropTable(&settingsDown{}); err != nil {
			return err
		}
	}
	if err := db.AutoMigrate(&settingsDown{}); err != nil {
		return err
	}
	var rows []settingsDown
	if err := db.Table("settings").Find(&rows).Error; err != nil {
		return err
	}
	if err := db.CreateInBatches(&rows, 100).Error; err != nil { // row copy drops org_id and keeps every settings value under the user_id key
		return err
	}
	if err := db.Migrator().DropTable(&models.Settings{}); err != nil {
		return err
	}
	return db.Migrator().RenameTable(&settingsDown{}, &models.Settings{})
}
