package controllers

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/pkg/configs"
	"github.com/tertua/tupay/pkg/utils"
)

// outboxQueues are the fixed queue names with their zero-fill status enums.
var outboxQueues = []struct {
	queue    string
	statuses []string
}{
	{"mail", []string{"pending", "processing", "sent", "failed", "dead"}},
	{"notification", []string{"pending", "processing", "delivered", "failed", "dead"}},
	{"webhook", []string{"pending", "failed", "delivered", "dead"}},
	{"gateway_pending", []string{"pending"}},
}

// ageSeconds reports now-t in whole seconds, or nil when t is absent.
func ageSeconds(now time.Time, t *time.Time) *int64 {
	if t == nil {
		return nil
	}
	sec := int64(now.Sub(*t).Seconds())
	return &sec
}

// OutboxStatus returns queue depth per status plus oldest-row age signals.
// @Description Queue depth per status for every outbox queue, with oldest pending/processing age and worker poll settings.
// @Summary get outbox queue status
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /admin/outbox/status [get]
func OutboxStatus(c fiber.Ctx) error {
	db, ok := openDB(c)
	if !ok {
		return nil
	}
	stats, err := db.OutboxStats()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load outbox status", nil)
	}
	now := time.Now()
	queues := make([]fiber.Map, 0, len(outboxQueues))
	for _, q := range outboxQueues {
		statuses := make(map[string]int64, len(q.statuses))
		for _, st := range q.statuses {
			statuses[st] = 0
		}
		var oldestPending, oldestProcessing *time.Time
		for _, s := range stats {
			if s.Queue != q.queue {
				continue
			}
			statuses[s.Status] = s.Count
			if s.Status == "pending" {
				oldestPending = s.Oldest
			}
			if s.Status == "processing" {
				oldestProcessing = s.Oldest
			}
		}
		queues = append(queues, fiber.Map{
			"queue":                     q.queue,
			"statuses":                  statuses,
			"oldest_pending_seconds":    ageSeconds(now, oldestPending),
			"oldest_processing_seconds": ageSeconds(now, oldestProcessing),
		})
	}
	out := configs.Get().Outbox
	return utils.OK(c, fiber.StatusOK, fiber.Map{
		"queues":       queues,
		"poll_seconds": out.PollSeconds,
		"batch_size":   out.Batch,
	})
}
