package routes

import (
	"os"
	"testing"

	"github.com/tertua/invoiceman/platform/database"
)

// TestMain runs route tests against in-memory SQLite with an in-memory
// session store, so no Postgres or Redis services are required.
func TestMain(m *testing.M) {
	os.Setenv("STAGE_STATUS", "dev")
	os.Setenv("JWT_SECRET_KEY", "test-secret")
	os.Setenv("JWT_SECRET_KEY_EXPIRE_MINUTES_COUNT", "15")
	os.Setenv("JWT_REFRESH_KEY", "test-refresh")
	os.Setenv("JWT_REFRESH_KEY_EXPIRE_HOURS_COUNT", "720")
	os.Setenv("SQL_DSN", "")
	os.Setenv("SQLITE_PATH", "file::memory:?cache=shared")
	os.Setenv("REDIS_HOST", "")

	if err := database.Migrate(); err != nil {
		panic("test migrate failed: " + err.Error())
	}

	os.Exit(m.Run())
}
