package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/app/models"
)

func endpointResponse(e models.NotificationEndpoint) fiber.Map {
	return fiber.Map{
		"id":         e.ID,
		"target_url": e.TargetURL,
		"events":     e.Events,
		"is_active":  e.IsActive,
		"created_at": e.CreatedAt,
		"updated_at": e.UpdatedAt,
	}
}

func notificationDeliveryResponse(d models.NotificationDelivery) fiber.Map {
	return fiber.Map{
		"id":          d.ID,
		"endpoint_id": d.EndpointID,
		"event_id":    d.EventID,
		"event_type":  d.EventType,
		"target_url":  d.TargetURL,
		"attempt":     d.Attempt,
		"status":      d.Status,
		"resp_code":   d.RespCode,
		"created_at":  d.CreatedAt,
		"updated_at":  d.UpdatedAt,
	}
}
