package queries

import (
	"github.com/google/uuid"
	"github.com/tertua/invoiceman/app/models"
	"gorm.io/gorm"
)

// ReserveInvoiceNumber increments the owner's invoice sequence inside tx and returns the formatted number; both create paths reserve it here, and the atomic increment keeps concurrent calls from producing duplicates.
func (q *InvoiceQueries) ReserveInvoiceNumber(tx *gorm.DB, userID uuid.UUID) (string, error) {
	settings := models.Settings{UserID: userID}
	if err := tx.Where("user_id = ?", userID).FirstOrCreate(
		&settings, models.Settings{UserID: userID},
	).Error; err != nil {
		return "", err
	}
	if err := tx.Model(&models.Settings{}).Where("user_id = ?", userID).
		UpdateColumn("invoice_seq", gorm.Expr("invoice_seq + 1")).Error; err != nil {
		return "", err
	}
	if err := tx.Where("user_id = ?", userID).First(&settings).Error; err != nil {
		return "", err
	}
	return models.FormatInvoiceSeq(settings.InvoicePrefix, settings.InvoiceSeq), nil
}
