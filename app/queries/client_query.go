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
