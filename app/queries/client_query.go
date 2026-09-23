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

// ListClients returns one page of clients of a user with billing aggregates.
func (q *ClientQueries) ListClients(userID uuid.UUID, limit, offset int) ([]models.ClientListRow, error) {
	clients := []models.ClientListRow{}

	// Billed invoices are sent + paid; drafts are not billed yet. Paid must
	// stay in total_billed so auto-marking paid never shrinks client totals.
	paidSubquery := q.Model(&models.Payment{}).
		Select("invoices.client_id AS client_id, SUM(payments.amount) AS paid").
		Joins("JOIN invoices ON invoices.id = payments.invoice_id").
		Where("invoices.user_id = ? AND invoices.status IN ? AND payments.voided_at IS NULL", userID, []string{models.InvoiceStatusSent, models.InvoiceStatusPaid}).
		Group("invoices.client_id")

	billedSubquery := q.Model(&models.Invoice{}).
		Select("client_id, SUM(total) AS total_billed").
		Where("user_id = ? AND status IN ?", userID, []string{models.InvoiceStatusSent, models.InvoiceStatusPaid}).
		Group("client_id")

	if err := q.Table("clients AS c").
		Select(`c.*, COALESCE(inv.total_billed, 0) AS total_billed,
			COALESCE(inv.total_billed, 0) - COALESCE(pay.paid, 0) AS outstanding`).
		Joins("LEFT JOIN (?) AS inv ON inv.client_id = c.id", billedSubquery).
		Joins("LEFT JOIN (?) AS pay ON pay.client_id = c.id", paidSubquery).
		Where("c.user_id = ?", userID).
		Order("c.created_at DESC").
		Limit(limit).Offset(offset).
		Scan(&clients).Error; err != nil {
		return clients, err
	}

	return clients, nil
}

// CountClients returns the total clients of a user for pagination meta.
func (q *ClientQueries) CountClients(userID uuid.UUID) (int64, error) {
	var total int64
	if err := q.Model(&models.Client{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
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
