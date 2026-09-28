package controllers

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"
)

// recordAudit stores one audit trail entry. It never fails the user action:
// write errors are only logged.
func recordAudit(c fiber.Ctx, db *database.Queries, userID uuid.UUID, action, entity, entityID, meta string) {
	entry := &models.AuditLog{
		UserID: userID, Action: action, Entity: entity,
		EntityID: entityID, Meta: meta, IP: c.IP(), CreatedAt: time.Now(),
	}
	if err := db.RecordAudit(entry); err != nil {
		utils.RequestLogger(c).Warn("audit write failed", "action", action, "err", err)
	}
}
