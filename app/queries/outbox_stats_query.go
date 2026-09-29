package queries

import (
	"time"

	"github.com/tertua/tupay/app/models"
	"gorm.io/gorm"
)

// OutboxStat is one aggregated row of outbox queue depth (status + count + oldest updated_at).
type OutboxStat struct {
	Queue  string
	Status string
	Count  int64
	Oldest *time.Time
}

// oldestTime normalizes MIN(updated_at) scan results (SQLite aggregate yields a raw string, PostgreSQL a time.Time rendered as string).
func oldestTime(s *string) *time.Time {
	if s == nil {
		return nil
	}
	return parseOldest(*s)
}

func parseOldest(s string) *time.Time {
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	} {
		if ts, err := time.Parse(layout, s); err == nil {
			return &ts
		}
	}
	return nil
}

// OutboxStats returns per-status counts and oldest updated_at for every outbox queue.
func (q *MailOutboxQueries) OutboxStats() ([]OutboxStat, error) {
	type row struct {
		Status string
		Count  int64
		Oldest *string
	}
	var out []OutboxStat
	collect := func(queue string, model any, filter func(*gorm.DB) *gorm.DB) error {
		var rows []row
		tx := q.Model(model).Select("status, count(*) as count, min(updated_at) as oldest").Group("status")
		if filter != nil {
			tx = filter(tx)
		}
		if err := tx.Find(&rows).Error; err != nil {
			return err
		}
		for _, r := range rows {
			out = append(out, OutboxStat{Queue: queue, Status: r.Status, Count: r.Count, Oldest: oldestTime(r.Oldest)})
		}
		return nil
	}
	if err := collect("mail", &models.MailOutbox{}, nil); err != nil {
		return nil, err
	}
	if err := collect("notification", &models.NotificationDelivery{}, nil); err != nil {
		return nil, err
	}
	if err := collect("webhook", &models.WebhookDelivery{}, nil); err != nil {
		return nil, err
	}
	// gateway_pending is the reconcile pool: only unsettled intents, not the full business table.
	if err := collect("gateway_pending", &models.GatewayTransaction{}, func(tx *gorm.DB) *gorm.DB {
		return tx.Where("status = ?", models.GatewayStatusPending)
	}); err != nil {
		return nil, err
	}
	return out, nil
}
