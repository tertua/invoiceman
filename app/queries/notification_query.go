package queries

import (
	"time"

	"github.com/google/uuid"
	"github.com/tertua/invoiceman/app/models"
	"gorm.io/gorm"
)

// NotificationQueries persists user-owned webhook endpoints and their
// outbound deliveries for the background worker.
type NotificationQueries struct {
	*gorm.DB
}

// CreateEndpoint registers a webhook target for a user.
func (q *NotificationQueries) CreateEndpoint(e *models.NotificationEndpoint) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	now := time.Now()
	e.CreatedAt = now
	e.UpdatedAt = now
	return q.Create(e).Error
}

// GetEndpoint returns one endpoint owned by a user.
func (q *NotificationQueries) GetEndpoint(userID, id uuid.UUID) (models.NotificationEndpoint, error) {
	e := models.NotificationEndpoint{}
	if err := q.Where("id = ? AND user_id = ?", id, userID).First(&e).Error; err != nil {
		return e, notFound(err)
	}
	return e, nil
}

// ListEndpoints returns all endpoints of a user, oldest first.
func (q *NotificationQueries) ListEndpoints(userID uuid.UUID) ([]models.NotificationEndpoint, error) {
	out := []models.NotificationEndpoint{}
	if err := q.Where("user_id = ?", userID).Order("created_at ASC").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// ListActiveEndpoints returns subscribed, active endpoints of a user for
// one event type. Filtering happens in Go (Events is a CSV); endpoint
// counts per user stay small.
func (q *NotificationQueries) ListActiveEndpoints(userID uuid.UUID, eventType string) ([]models.NotificationEndpoint, error) {
	all, err := q.ListEndpoints(userID)
	if err != nil {
		return nil, err
	}
	out := []models.NotificationEndpoint{}
	for _, e := range all {
		if !e.IsActive {
			continue
		}
		if models.EndpointSubscribed(e.Events, eventType) {
			out = append(out, e)
		}
	}
	return out, nil
}

// SaveEndpoint persists endpoint edits.
func (q *NotificationQueries) SaveEndpoint(e *models.NotificationEndpoint) error {
	return q.Save(e).Error
}

// DeleteEndpoint removes an endpoint owned by a user.
func (q *NotificationQueries) DeleteEndpoint(userID, id uuid.UUID) error {
	return q.Where("id = ? AND user_id = ?", id, userID).Delete(&models.NotificationEndpoint{}).Error
}

// EnqueueDelivery stores one event forward for async delivery.
// EventID is unique per forward so replays never duplicate rows.
func (q *NotificationQueries) EnqueueDelivery(d *models.NotificationDelivery) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	now := time.Now()
	d.Status = models.NotifStatusPending
	d.CreatedAt = now
	d.UpdatedAt = now
	return q.Create(d).Error
}

// DueNotifications returns pending/failed deliveries due for retry.
func (q *NotificationQueries) DueNotifications(now time.Time, limit int) ([]models.NotificationDelivery, error) {
	out := []models.NotificationDelivery{}
	if err := q.Where("status IN ? AND (next_retry_at IS NULL OR next_retry_at <= ?)",
		[]string{models.NotifStatusPending, models.NotifStatusFailed}, now).
		Order("created_at ASC").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// ClaimNotification atomically marks one due delivery as claimed.
// Only one worker wins the claim; losers get claimed=false and skip it.
func (q *NotificationQueries) ClaimNotification(id uuid.UUID, now time.Time) (bool, error) {
	return DoRetryValue(func() (bool, error) {
		res := q.Model(&models.NotificationDelivery{}).
			Where("id = ? AND status IN ? AND (next_retry_at IS NULL OR next_retry_at <= ?)",
				id, []string{models.NotifStatusPending, models.NotifStatusFailed}, now).
			Updates(map[string]interface{}{"status": models.NotifStatusProcessing, "updated_at": now})
		return res.RowsAffected > 0, res.Error
	})
}

// MarkNotificationSent records a successful delivery.
func (q *NotificationQueries) MarkNotificationSent(id uuid.UUID, code int, body string, now time.Time) error {
	return DoRetry(func() error {
		return q.Model(&models.NotificationDelivery{}).Where("id = ?", id).
			Updates(map[string]interface{}{
				"status": models.NotifStatusDelivered, "resp_code": code,
				"resp_body": body, "updated_at": now, "next_retry_at": nil,
			}).Error
	})
}

// MarkNotificationFailed records a failed attempt with the next retry time,
// or parks the row as dead when attempts are exhausted.
func (q *NotificationQueries) MarkNotificationFailed(id uuid.UUID, attempt int, retryAt *time.Time, code int, body string, now time.Time) error {
	status := models.NotifStatusFailed
	if retryAt == nil {
		status = models.NotifStatusDead
	}
	return DoRetry(func() error {
		return q.Model(&models.NotificationDelivery{}).Where("id = ?", id).
			Updates(map[string]interface{}{
				"status": status, "attempt": attempt, "next_retry_at": retryAt,
				"resp_code": code, "resp_body": body, "updated_at": now,
			}).Error
	})
}

// ListDeliveriesByUser returns one page of deliveries for a user.
func (q *NotificationQueries) ListDeliveriesByUser(userID uuid.UUID, eventType, status string, limit, offset int) ([]models.NotificationDelivery, error) {
	out := []models.NotificationDelivery{}
	tx := q.Where("user_id = ?", userID)
	if eventType != "" {
		tx = tx.Where("event_type = ?", eventType)
	}
	if status != "" {
		tx = tx.Where("status = ?", status)
	}
	if err := tx.Order("created_at DESC").Limit(limit).Offset(offset).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// CountDeliveriesByUser returns the total deliveries for a user filter.
func (q *NotificationQueries) CountDeliveriesByUser(userID uuid.UUID, eventType, status string) (int64, error) {
	var total int64
	tx := q.Model(&models.NotificationDelivery{}).Where("user_id = ?", userID)
	if eventType != "" {
		tx = tx.Where("event_type = ?", eventType)
	}
	if status != "" {
		tx = tx.Where("status = ?", status)
	}
	if err := tx.Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// GetDeliveryByUser returns one delivery owned by a user.
func (q *NotificationQueries) GetDeliveryByUser(userID, id uuid.UUID) (models.NotificationDelivery, error) {
	d := models.NotificationDelivery{}
	if err := q.Where("id = ? AND user_id = ?", id, userID).First(&d).Error; err != nil {
		return d, notFound(err)
	}
	return d, nil
}

// RequeueNotification resets a failed/dead delivery to pending for the
// next worker tick. Sends stay async; the worker owns the HTTP attempt.
func (q *NotificationQueries) RequeueNotification(id uuid.UUID, now time.Time) error {
	return DoRetry(func() error {
		return q.Model(&models.NotificationDelivery{}).Where("id = ?", id).
			Updates(map[string]interface{}{
				"status": models.NotifStatusPending, "next_retry_at": nil, "updated_at": now,
			}).Error
	})
}
