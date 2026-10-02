package controllers

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/platform/database"
	"github.com/tertua/tupay/platform/events"
)

// notifPayload is the stable v1 envelope forwarded to user webhook endpoints (n8n, GOWA bridges); event_id doubles as the downstream idempotency key and type selects the template on the consumer side.
type notifPayload struct {
	EventID    string `json:"event_id"`
	Type       string `json:"type"`
	Version    string `json:"version"`
	OccurredAt string `json:"occurred_at"`
	UserID     string `json:"user_id"`
	Data       any    `json:"data"`
}

// enqueueNotification fans one event out to every subscribed, active endpoint of a user and nudges the live SSE stream; controllers call it after their write succeeds and return fast, then the background worker forwards and retries.
func enqueueNotification(db *database.Queries, userID uuid.UUID, eventType, eventID string, data any) {
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

// enqueueOrgNotification fans one event out to every owner and staff member of an org (D11): inbox and SSE stay per-user, so the org fan-out repeats the existing per-user path once per member.
func enqueueOrgNotification(db *database.Queries, orgID uuid.UUID, eventType, eventID string, data any) {
	members, err := db.ListByOrg(orgID)
	if err != nil || len(members) == 0 {
		return
	}
	if eventID == "" {
		eventID = "evt_" + uuid.NewString()[:8]
	}
	for _, m := range members {
		enqueueNotification(db, m.UserID, eventType, eventID, data)
	}
}

// enqueueNotificationDelivery queues one event for one endpoint; the caller resolves eventID first so every endpoint in a fan-out shares the same payload event_id and the per-endpoint delivery key adds the endpoint suffix to satisfy the unique index.
func enqueueNotificationDelivery(db *database.Queries, userID uuid.UUID, endpoint models.NotificationEndpoint, eventType, eventID string, data any) {
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
