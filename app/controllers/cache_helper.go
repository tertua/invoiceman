package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/cache"
)

// invalidateAggregates drops cached dashboard/report aggregates after a write; best-effort, a failure only costs TTL staleness and is logged, never fatal.
func invalidateAggregates(c fiber.Ctx, orgID uuid.UUID) {
	if err := cache.InvalidateOrg(c.Context(), orgID.String()); err != nil {
		utils.RequestLogger(c).Warn("aggregate cache invalidation failed", "err", err)
	}
}
