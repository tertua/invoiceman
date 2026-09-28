package routes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The live event stream is session-scoped: browsers connect with cookies,
// so anonymous requests are rejected before any stream state is created.
func TestEventsRequiresSession(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "GET", "/api/events", "", nil)
	defer resp.Body.Close()
	require.Equal(t, 401, resp.StatusCode)
}
