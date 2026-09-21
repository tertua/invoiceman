package queries

import (
	"time"

	"github.com/google/uuid"
	"github.com/tertua/invoiceman/app/models"
	"gorm.io/gorm"
)

// ItemQueries provides catalog item persistence operations.
type ItemQueries struct {
	*gorm.DB
}

// ListItems returns catalog items owned by a user.
func (q *ItemQueries) ListItems(userID uuid.UUID) ([]models.Item, error) {
	items := []models.Item{}
	if err := q.Where("user_id = ?", userID).Order("name ASC").Find(&items).Error; err != nil {
		return items, err
	}
	return items, nil
}

// GetItem returns one catalog item owned by a user.
func (q *ItemQueries) GetItem(userID, id uuid.UUID) (models.Item, error) {
	item := models.Item{}
	if err := q.Where("id = ? AND user_id = ?", id, userID).First(&item).Error; err != nil {
		return item, notFound(err)
	}
	return item, nil
}

// CreateItem persists a catalog item.
func (q *ItemQueries) CreateItem(item *models.Item) error {
	return q.Create(item).Error
}

// UpdateItem updates a catalog item owned by a user.
func (q *ItemQueries) UpdateItem(item *models.Item) error {
	return q.Model(&models.Item{}).Where("id = ? AND user_id = ?", item.ID, item.UserID).
		Updates(map[string]interface{}{
			"updated_at":  time.Now(),
			"name":        item.Name,
			"description": item.Description,
			"rate":        item.Rate,
			"unit":        item.Unit,
		}).Error
}

// DeleteItem deletes a catalog item owned by a user.
func (q *ItemQueries) DeleteItem(userID, id uuid.UUID) error {
	return q.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Item{}).Error
}
