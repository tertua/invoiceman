package queries

import (
	"time"

	"github.com/google/uuid"
	"github.com/tertua/invoiceman/app/models"
	"gorm.io/gorm"
)

// ClientQueries struct for queries from Client model.
type ClientQueries struct {
	*gorm.DB
}

// ListClients returns all clients of a user with billing aggregates.
func (q *ClientQueries) ListClients(userID uuid.UUID) ([]models.ClientListRow, error) {
	clients := []models.ClientListRow{}

	paidSubquery := q.Model(&models.Payment{}).
		Select("invoices.client_id AS client_id, SUM(payments.amount) AS paid").
		Joins("JOIN invoices ON invoices.id = payments.invoice_id").
		Where("invoices.user_id = ?", userID).
		Group("invoices.client_id")

	billedSubquery := q.Model(&models.Invoice{}).
		Select("client_id, SUM(total) AS total_billed").
		Where("user_id = ?", userID).
		Group("client_id")

	if err := q.Table("clients AS c").
		Select(`c.*, COALESCE(inv.total_billed, 0) AS total_billed,
			COALESCE(inv.total_billed, 0) - COALESCE(pay.paid, 0) AS outstanding`).
		Joins("LEFT JOIN (?) AS inv ON inv.client_id = c.id", billedSubquery).
		Joins("LEFT JOIN (?) AS pay ON pay.client_id = c.id", paidSubquery).
		Where("c.user_id = ?", userID).
		Order("c.created_at DESC").
		Scan(&clients).Error; err != nil {
		return clients, err
	}

	return clients, nil
}

// GetClient returns one client of a user by ID.
func (q *ClientQueries) GetClient(userID, id uuid.UUID) (models.Client, error) {
	client := models.Client{}

	if err := q.Where("id = ? AND user_id = ?", id, userID).First(&client).Error; err != nil {
		return client, notFound(err)
	}

	return client, nil
}

// CreateClient creates a new client.
func (q *ClientQueries) CreateClient(c *models.Client) error {
	if err := q.Create(c).Error; err != nil {
		return err
	}

	return nil
}

// UpdateClient updates a client of a user.
func (q *ClientQueries) UpdateClient(c *models.Client) error {
	if err := q.Model(&models.Client{}).Where("id = ? AND user_id = ?", c.ID, c.UserID).
		Updates(map[string]interface{}{
			"updated_at": time.Now(),
			"name":       c.Name,
			"email":      c.Email,
			"company":    c.Company,
			"phone":      c.Phone,
			"address":    c.Address,
			"notes":      c.Notes,
		}).Error; err != nil {
		return err
	}

	return nil
}

// DeleteClient deletes a client of a user.
func (q *ClientQueries) DeleteClient(userID, id uuid.UUID) error {
	if err := q.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Client{}).Error; err != nil {
		return err
	}

	return nil
}
