package routes

import (
	"os"
	"testing"

	"github.com/tertua/tupay/platform/database"
	"github.com/tertua/tupay/platform/gateway"
	"github.com/tertua/tupay/platform/midtrans"
	"github.com/tertua/tupay/platform/nowpayments"
)

// TestMain runs route tests against in-memory SQLite with an in-memory
// session store, so no Postgres or Redis services are required.
func TestMain(m *testing.M) {
	seedRouteTestEnv()

	if err := database.Migrate(); err != nil {
		panic("test migrate failed: " + err.Error())
	}

	// Register gateways for tests (main.go does this in production).
	gateway.Register(midtrans.Gateway{})
	gateway.Register(nowpayments.Gateway{})

	os.Exit(m.Run())
}
