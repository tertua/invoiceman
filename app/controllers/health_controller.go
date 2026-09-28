package controllers

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/pkg/configs"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/cache"
	"github.com/tertua/tupay/platform/database"
)

// appVersion is read once from the VERSION file; falls back to "dev".
func appVersion() string {
	raw, err := os.ReadFile("VERSION")
	if err != nil {
		return "dev"
	}
	if v := strings.TrimSpace(string(raw)); v != "" {
		return v
	}
	return "dev"
}

// Health is a liveness probe for orchestrators. Always 200, no auth,
// no DB/Redis access.
// @Description Liveness probe.
// @Summary liveness probe
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /healthz [get]
func Health(c fiber.Ctx) error {
	return utils.OK(c, fiber.StatusOK, fiber.Map{"status": "ok", "version": appVersion()})
}

// Ready is a readiness probe: pings the database and, when REDIS_HOST is
// set, the session store. Failing dependencies return 503 without leaking
// connection details.
// @Description Readiness probe.
// @Summary readiness probe
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 503 {object} map[string]interface{}
// @Router /readyz [get]
func Ready(c fiber.Ctx) error {
	if err := database.Ping(); err != nil {
		return utils.Fail(c, fiber.StatusServiceUnavailable, "not ready", nil)
	}
	if configs.Get().Redis.Enabled() {
		ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
		defer cancel()
		client, err := cache.RedisConnection()
		if err != nil {
			return utils.Fail(c, fiber.StatusServiceUnavailable, "not ready", nil)
		}
		defer func() { _ = client.Close() }()
		if err := client.Ping(ctx).Err(); err != nil {
			return utils.Fail(c, fiber.StatusServiceUnavailable, "not ready", nil)
		}
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"status": "ready", "version": appVersion()})
}
