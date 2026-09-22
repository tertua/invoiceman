package middleware

import (
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/timeout"
	"github.com/tertua/invoiceman/pkg/utils"
)

// aiTimeout returns the per-request timeout for AI + gateway intent routes
// (AI_TIMEOUT_SECONDS, default 30s). Webhooks and CRUD keep no extra timeout
// so provider retries and large payloads are not cut off.
func aiTimeout() time.Duration {
	if v, err := strconv.Atoi(os.Getenv("AI_TIMEOUT_SECONDS")); err == nil && v > 0 {
		return time.Duration(v) * time.Second
	}
	return 30 * time.Second
}

// WithAITimeout wraps a handler with a request timeout. On timeout the
// client gets a 504 envelope instead of a dropped connection.
func WithAITimeout(h fiber.Handler) fiber.Handler {
	return timeout.New(h, timeout.Config{
		Timeout: aiTimeout(),
		OnTimeout: func(c fiber.Ctx) error {
			return utils.Fail(c, fiber.StatusGatewayTimeout, "request timed out, try again later", nil)
		},
	})
}
