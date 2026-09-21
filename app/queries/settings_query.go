package queries

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/tertua/invoiceman/app/models"
	"gorm.io/gorm"
)

// SettingsQueries struct for queries from Settings model.
type SettingsQueries struct {
	*gorm.DB
}

// GetSettings returns user settings, creating defaults on first use.
func (q *SettingsQueries) GetSettings(userID uuid.UUID) (models.Settings, error) {
	settings := models.Settings{UserID: userID}

	if err := q.Where("user_id = ?", userID).First(&settings).Error; err == nil {
		return settings, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return settings, err
	}

	defaults := models.DefaultSettings(userID)
	if err := q.Create(defaults).Error; err != nil {
		return settings, err
	}
	settings = *defaults
	return settings, nil
}

// CreateSettings creates the default settings row for a user.
func (q *SettingsQueries) CreateSettings(s *models.Settings) error {
	if err := q.Create(s).Error; err != nil {
		return err
	}

	return nil
}

// UpdateSettings updates user settings.
func (q *SettingsQueries) UpdateSettings(s *models.Settings) error {
	if err := q.Model(&models.Settings{}).Where("user_id = ?", s.UserID).
		Updates(map[string]interface{}{
			"updated_at":     time.Now(),
			"company_name":   s.CompanyName,
			"email":          s.Email,
			"phone":          s.Phone,
			"address":        s.Address,
			"logo_url":       s.LogoURL,
			"currency":       s.Currency,
			"tax_rate":       s.TaxRate,
			"invoice_prefix": s.InvoicePrefix,
		}).Error; err != nil {
		return err
	}

	return nil
}
