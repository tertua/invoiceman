package queries

import (
	"time"

	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"gorm.io/gorm"
)

// ItemQueries provides catalog item persistence operations.
type ItemQueries struct {
	*gorm.DB
}

// ListItems returns one page of catalog items owned by an org.
func (q *ItemQueries) ListItems(orgID uuid.UUID, limit, offset int) ([]models.Item, error) {
	items := []models.Item{}
	if err := q.Where("org_id = ?", orgID).Order("name ASC").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		return items, err
	}
	return items, nil
}

// CountItems returns the total catalog items of an org.
func (q *ItemQueries) CountItems(orgID uuid.UUID) (int64, error) {
	var total int64
	if err := q.Model(&models.Item{}).Where("org_id = ?", orgID).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// GetItem returns one catalog item owned by an org.
func (q *ItemQueries) GetItem(orgID, id uuid.UUID) (models.Item, error) {
	item := models.Item{}
	if err := q.Where("id = ? AND org_id = ?", id, orgID).First(&item).Error; err != nil {
		return item, notFound(err)
	}
	return item, nil
}

// CreateItem persists a catalog item; an unset OrgID rides as the creator UserID (rides-as), UserID stays audit.
func (q *ItemQueries) CreateItem(item *models.Item) error {
	return rideOrg(q.DB, item, &item.OrgID, item.UserID)
}

// UpdateItem updates a catalog item owned by an org.
func (q *ItemQueries) UpdateItem(item *models.Item) error {
	return q.Model(&models.Item{}).Where("id = ? AND org_id = ?", item.ID, item.OrgID).
		Updates(map[string]any{
			"updated_at":  time.Now(),
			"name":        item.Name,
			"description": item.Description,
			"rate":        item.Rate,
			"unit":        item.Unit,
		}).Error
}

// DeleteItem deletes a catalog item owned by an org.
func (q *ItemQueries) DeleteItem(orgID, id uuid.UUID) error {
	return q.Where("id = ? AND org_id = ?", id, orgID).Delete(&models.Item{}).Error
}
