package configs

import (
	"time"

	"github.com/gofiber/fiber/v3"
)

// FiberConfig func for configuration Fiber app.
// See: https://docs.gofiber.io/api/fiber#config
func FiberConfig() fiber.Config {
	// Define server settings from the central config.
	cfg := Get()

	// Return Fiber configuration.
	return fiber.Config{ReadTimeout: time.Second * time.Duration(cfg.Server.ReadTimeoutSec)}
}
