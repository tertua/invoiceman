package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/cache"
)

// invalidateAggregates drops cached dashboard/report aggregates after a
// write. Best-effort: the write already committed, so a cache failure only
// costs TTL staleness — it is logged, never fatal.
func invalidateAggregates(c fiber.Ctx, userID uuid.UUID) {
	if err := cache.InvalidateUser(c.Context(), userID.String()); err != nil {
		utils.RequestLogger(c).Warn("aggregate cache invalidation failed", "err", err)
	}
}
