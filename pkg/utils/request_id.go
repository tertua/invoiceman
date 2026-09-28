package utils

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/tertua/tupay/pkg/logger"
)

// RequestID returns the request's X-Request-ID (set by the requestid
// middleware) for log correlation, or "" when absent (tests, workers).
// Background code without a request keeps using logger.L() and adds its
// own job ids instead.
func RequestID(c fiber.Ctx) string {
	if c == nil {
		return ""
	}
	return requestid.FromContext(c)
}

// RequestLogger returns a logger carrying the request id; with no id it
// is exactly the shared logger.
func RequestLogger(c fiber.Ctx) *slog.Logger {
	return logger.WithRequestID(RequestID(c))
}
