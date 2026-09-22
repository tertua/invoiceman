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

	fiberCfg := fiber.Config{ReadTimeout: time.Second * time.Duration(cfg.Server.ReadTimeoutSec)}

	// Trust X-Forwarded-* headers only from the configured proxies (empty =
	// direct, client headers are ignored so they cannot be spoofed).
	if len(cfg.Server.TrustedProxies) > 0 {
		fiberCfg.TrustProxy = true
		fiberCfg.ProxyHeader = cfg.Server.ProxyHeader
		fiberCfg.TrustProxyConfig = fiber.TrustProxyConfig{Proxies: cfg.Server.TrustedProxies}
	}

	return fiberCfg
}
