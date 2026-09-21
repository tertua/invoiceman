package queries

import (
	"time"

	"github.com/google/uuid"
	"github.com/tertua/invoiceman/app/models"
	"gorm.io/gorm"
)

// GatewayQueries provides persistence for the central payment relay.
type GatewayQueries struct {
	*gorm.DB
}

// CreateProject registers a downstream project.
func (q *GatewayQueries) CreateProject(p *models.GatewayProject) error {
	return q.Create(p).Error
}

// GetProjectBySlug returns a project by slug.
func (q *GatewayQueries) GetProjectBySlug(slug string) (models.GatewayProject, error) {
	p := models.GatewayProject{}
	if err := q.Where("slug = ?", slug).First(&p).Error; err != nil {
		return p, notFound(err)
	}
	return p, nil
}

// GetProjectByKeyHash resolves project identity from an API key hash.
func (q *GatewayQueries) GetProjectByKeyHash(hash string) (models.GatewayProject, error) {
	p := models.GatewayProject{}
	if err := q.Where("api_key_hash = ?", hash).First(&p).Error; err != nil {
		return p, notFound(err)
	}
	return p, nil
}

// ListProjects returns all registered projects ordered by slug.
func (q *GatewayQueries) ListProjects() ([]models.GatewayProject, error) {
	out := []models.GatewayProject{}
	if err := q.Order("slug ASC").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// SaveProject persists project edits.
func (q *GatewayQueries) SaveProject(p *models.GatewayProject) error {
	return q.Save(p).Error
}

// CreateTransaction stores a new gateway transaction.
func (q *GatewayQueries) CreateTransaction(t *models.GatewayTransaction) error {
	return q.Create(t).Error
}

// GetTransaction returns a transaction by global order id.
func (q *GatewayQueries) GetTransaction(orderID string) (models.GatewayTransaction, error) {
	t := models.GatewayTransaction{}
	if err := q.Where("order_id = ?", orderID).First(&t).Error; err != nil {
		return t, notFound(err)
	}
	return t, nil
}

// GetTransactionByExternal returns the latest transaction for a project + external id.
func (q *GatewayQueries) GetTransactionByExternal(projectSlug, external string) (models.GatewayTransaction, error) {
	t := models.GatewayTransaction{}
	if err := q.Where("project_slug = ? AND external_order_id = ?", projectSlug, external).
		Order("created_at DESC").First(&t).Error; err != nil {
		return t, notFound(err)
	}
	return t, nil
}

// ListTransactionsByProject returns transactions for one project, newest first.
func (q *GatewayQueries) ListTransactionsByProject(projectSlug string, limit int) ([]models.GatewayTransaction, error) {
	out := []models.GatewayTransaction{}
	if err := q.Where("project_slug = ?", projectSlug).Order("created_at DESC").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// ListAllTransactions returns recent transactions across projects for admins.
func (q *GatewayQueries) ListAllTransactions(limit int) ([]models.GatewayTransaction, error) {
	out := []models.GatewayTransaction{}
	if err := q.Order("created_at DESC").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// SaveTransaction persists transaction updates.
func (q *GatewayQueries) SaveTransaction(t *models.GatewayTransaction) error {
	return q.Save(t).Error
}

// CreateEvent stores a raw gateway event for audit.
func (q *GatewayQueries) CreateEvent(e *models.GatewayEvent) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return q.Create(e).Error
}

// CreateDelivery stores a webhook delivery attempt record.
func (q *GatewayQueries) CreateDelivery(d *models.WebhookDelivery) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return q.Create(d).Error
}

// GetDelivery returns a delivery by id.
func (q *GatewayQueries) GetDelivery(id uuid.UUID) (models.WebhookDelivery, error) {
	d := models.WebhookDelivery{}
	if err := q.Where("id = ?", id).First(&d).Error; err != nil {
		return d, notFound(err)
	}
	return d, nil
}

// ListDeliveriesByOrder returns deliveries for an order, newest first.
func (q *GatewayQueries) ListDeliveriesByOrder(orderID string) ([]models.WebhookDelivery, error) {
	out := []models.WebhookDelivery{}
	if err := q.Where("order_id = ?", orderID).Order("created_at DESC").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// ListRecentDeliveries returns recent deliveries for admins.
func (q *GatewayQueries) ListRecentDeliveries(limit int) ([]models.WebhookDelivery, error) {
	out := []models.WebhookDelivery{}
	if err := q.Order("created_at DESC").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// SaveDelivery persists delivery updates.
func (q *GatewayQueries) SaveDelivery(d *models.WebhookDelivery) error {
	return q.Save(d).Error
}

// PendingDeliveries returns failed/pending deliveries due for retry.
func (q *GatewayQueries) PendingDeliveries(now time.Time, limit int) ([]models.WebhookDelivery, error) {
	out := []models.WebhookDelivery{}
	if err := q.Where("status IN ? AND (next_retry_at IS NULL OR next_retry_at <= ?)",
		[]string{"pending", "failed"}, now).Order("created_at ASC").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}
