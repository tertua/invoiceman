package utils

import (
	"fmt"

	"github.com/tertua/invoiceman/pkg/configs"
)

// ConnectionURLBuilder func for building URL connection.
//
// Supported names follow the one-api pattern used by this service:
//   - "redis" -> host:port from the Redis settings
//   - "fiber" -> host:port for the HTTP listener
//
// The database is selected via SQL_DSN (see platform/database), so the
// legacy postgres/mysql branches were removed.
func ConnectionURLBuilder(n string) (string, error) {
	cfg := configs.Get()

	// Switch given names.
	switch n {
	case "redis":
		// URL for Redis connection.
		return cfg.Redis.Addr(), nil
	case "fiber":
		// URL for Fiber connection.
		return cfg.ListenAddr(), nil
	default:
		// Return error message.
		return "", fmt.Errorf("connection name '%v' is not supported", n)
	}
}
