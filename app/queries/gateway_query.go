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

// ListProjects returns one page of registered projects ordered by slug.
func (q *GatewayQueries) ListProjects(limit, offset int) ([]models.GatewayProject, error) {
	out := []models.GatewayProject{}
	if err := q.Order("slug ASC").Limit(limit).Offset(offset).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// CountProjects returns the total registered projects.
func (q *GatewayQueries) CountProjects() (int64, error) {
	var total int64
	if err := q.Model(&models.GatewayProject{}).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
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

// ListTransactionsByProject returns one page of transactions for one
// project, newest first.
func (q *GatewayQueries) ListTransactionsByProject(projectSlug string, limit, offset int) ([]models.GatewayTransaction, error) {
	out := []models.GatewayTransaction{}
	if err := q.Where("project_slug = ?", projectSlug).Order("created_at DESC").Limit(limit).Offset(offset).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// CountTransactionsByProject returns the total transactions of a project.
func (q *GatewayQueries) CountTransactionsByProject(projectSlug string) (int64, error) {
	var total int64
	if err := q.Model(&models.GatewayTransaction{}).Where("project_slug = ?", projectSlug).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// ListAllTransactions returns one page of transactions across projects.
func (q *GatewayQueries) ListAllTransactions(limit, offset int) ([]models.GatewayTransaction, error) {
	out := []models.GatewayTransaction{}
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// CountAllTransactions returns the total transactions across projects.
func (q *GatewayQueries) CountAllTransactions() (int64, error) {
	var total int64
	if err := q.Model(&models.GatewayTransaction{}).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
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

// ListRecentDeliveries returns one page of recent deliveries for admins.
func (q *GatewayQueries) ListRecentDeliveries(limit, offset int) ([]models.WebhookDelivery, error) {
	out := []models.WebhookDelivery{}
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// CountDeliveries returns the total webhook deliveries.
func (q *GatewayQueries) CountDeliveries() (int64, error) {
	var total int64
	if err := q.Model(&models.WebhookDelivery{}).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// SaveDelivery persists delivery updates.
func (q *GatewayQueries) SaveDelivery(d *models.WebhookDelivery) error {
	return q.Save(d).Error
}

// FailDelivery records a failed forward attempt without touching payload
// or signature. Used by the background worker.
func (q *GatewayQueries) FailDelivery(id uuid.UUID, status string, attempt int, retry *time.Time, respBody string, now time.Time) error {
	return q.Model(&models.WebhookDelivery{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status": status, "attempt": attempt, "next_retry_at": retry,
			"resp_body": respBody, "updated_at": now,
		}).Error
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

// ClaimDelivery atomically marks one due delivery as claimed for sending.
// Only one worker wins the claim; losers get claimed=false and skip it.
func (q *GatewayQueries) ClaimDelivery(id uuid.UUID, now time.Time) (bool, error) {
	res := q.Model(&models.WebhookDelivery{}).
		Where("id = ? AND status IN ? AND (next_retry_at IS NULL OR next_retry_at <= ?)",
			id, []string{"pending", "failed"}, now).
		Updates(map[string]interface{}{"updated_at": now})
	return res.RowsAffected > 0, res.Error
}
