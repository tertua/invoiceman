package controllers

import (
	"time"

	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
)

func endpointResponse(e models.NotificationEndpoint) endpointResponseRow {
	return endpointResponseRow{
		ID:        e.ID,
		TargetURL: e.TargetURL,
		Events:    e.Events,
		IsActive:  e.IsActive,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

// endpointResponseRow mirrors the historical hand-built map. secret is only
// set on the create/rotate paths (the generated secret is non-empty, so
// omitempty is byte-exact); the list path never sets it and the key stays
// absent as before.
type endpointResponseRow struct {
	ID        uuid.UUID `json:"id"`
	TargetURL string    `json:"target_url"`
	Events    string    `json:"events"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Secret    string    `json:"secret,omitempty"`
}

func notificationDeliveryResponse(d models.NotificationDelivery) notificationDeliveryRow {
	return notificationDeliveryRow{
		ID:         d.ID,
		EndpointID: d.EndpointID,
		EventID:    d.EventID,
		EventType:  d.EventType,
		TargetURL:  d.TargetURL,
		Attempt:    d.Attempt,
		Status:     d.Status,
		RespCode:   d.RespCode,
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
	}
}

type notificationDeliveryRow struct {
	ID         uuid.UUID `json:"id"`
	EndpointID uuid.UUID `json:"endpoint_id"`
	EventID    string    `json:"event_id"`
	EventType  string    `json:"event_type"`
	TargetURL  string    `json:"target_url"`
	Attempt    int       `json:"attempt"`
	Status     string    `json:"status"`
	RespCode   int       `json:"resp_code"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
