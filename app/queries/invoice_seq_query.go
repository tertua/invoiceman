package queries

import (
	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"gorm.io/gorm"
)

// ReserveInvoiceNumber increments the org's invoice sequence inside tx and returns the formatted number; both create paths reserve it here, and the atomic increment keeps concurrent calls from producing duplicates.
func (q *InvoiceQueries) ReserveInvoiceNumber(tx *gorm.DB, orgID uuid.UUID) (string, error) {
	settings := models.Settings{OrgID: orgID}
	// Sequence row is keyed by org; orgID rides as userID until register provisions the personal org (6.11).
	if err := tx.Where("org_id = ?", orgID).
		Attrs(models.DefaultSettings(orgID)).
		FirstOrCreate(&settings).Error; err != nil {
		return "", err
	}
	if err := tx.Model(&models.Settings{}).Where("org_id = ?", orgID).
		UpdateColumn("invoice_seq", gorm.Expr("invoice_seq + 1")).Error; err != nil {
		return "", err
	}
	if err := tx.Where("org_id = ?", orgID).First(&settings).Error; err != nil {
		return "", err
	}
	return models.FormatInvoiceSeq(settings.InvoicePrefix, settings.InvoiceSeq), nil
}
