package controllers

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/platform/database"
	"github.com/tertua/tupay/platform/events"
)

// notifPayload is the stable v1 envelope forwarded to user webhook
// endpoints (n8n, GOWA bridges). event_id doubles as the downstream
// idempotency key; type selects the template on the consumer side.
type notifPayload struct {
	EventID    string    `json:"event_id"`
	Type       string    `json:"type"`
	Version    string    `json:"version"`
	OccurredAt string    `json:"occurred_at"`
	UserID     string    `json:"user_id"`
	Data       fiber.Map `json:"data"`
}

// enqueueNotification fans one event out to every subscribed, active
// endpoint of a user. Controllers call it after their write succeeds and
// return fast; the background worker forwards and retries. It also nudges
// the user's live SSE stream (every caller here changes aggregates), even
// when no webhook endpoint exists — the two channels are independent.
func enqueueNotification(db *database.Queries, userID uuid.UUID, eventType, eventID string, data fiber.Map) {
	events.Default.Publish(userID.String(), events.Event{Type: eventType, Data: "{}"})
	endpoints, err := db.ListActiveEndpoints(userID, eventType)
	if err != nil || len(endpoints) == 0 {
		return
	}
	if eventID == "" {
		eventID = "evt_" + uuid.NewString()[:8]
	}
	for _, e := range endpoints {
		enqueueNotificationDelivery(db, userID, e, eventType, eventID, data)
	}
}

// enqueueNotificationDelivery queues one event for one endpoint. The caller
// resolves eventID first so every endpoint in a fan-out shares the same
// payload event_id; the per-endpoint delivery key adds the endpoint suffix
// to satisfy the unique index.
func enqueueNotificationDelivery(db *database.Queries, userID uuid.UUID, endpoint models.NotificationEndpoint, eventType, eventID string, data fiber.Map) {
	raw, _ := json.Marshal(notifPayload{
		EventID:    eventID,
		Type:       eventType,
		Version:    "v1",
		OccurredAt: time.Now().UTC().Format(time.RFC3339),
		UserID:     userID.String(),
		Data:       data,
	})
	_ = db.EnqueueDelivery(&models.NotificationDelivery{
		UserID:     userID,
		EndpointID: endpoint.ID,
		EventID:    eventID + ":" + endpoint.ID.String(),
		EventType:  eventType,
		TargetURL:  endpoint.TargetURL,
		Payload:    string(raw),
	})
}
