package metrics

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestObserveAndRender covers counters, histogram and exposition keys.
func TestObserveAndRender(t *testing.T) {
	r := New("9.9.9-test")
	r.Observe("GET", "/api/clients/:id", 200, 12*time.Millisecond)
	r.Observe("GET", "/api/clients/:id", 200, 3*time.Millisecond)
	r.Observe("POST", "/api/auth/login", 401, 1500*time.Millisecond)

	out := r.Render()
	assert.Contains(t, out, `tupay_app_info{version="9.9.9-test"} 1`)
	assert.Contains(t, out, `tupay_http_requests_total{method="GET",route="/api/clients/:id",status="200"} 2`)
	assert.Contains(t, out, `tupay_http_requests_total{method="POST",route="/api/auth/login",status="401"} 1`)
	assert.Contains(t, out, "tupay_http_request_duration_seconds_count 3")
	assert.Contains(t, out, `le="+Inf"`)
	assert.Contains(t, out, "tupay_uptime_seconds")
}

// TestUnknownRoute normalizes empty routes instead of exploding cardinality.
func TestUnknownRoute(t *testing.T) {
	r := New("dev")
	r.Observe("GET", "", 404, time.Millisecond)
	assert.Contains(t, r.Render(), `route="unknown"`)
}

// TestDBStatsHook renders pool gauges only when the hook reports ok.
func TestDBStatsHook(t *testing.T) {
	r := New("dev")
	assert.NotContains(t, r.Render(), "tupay_db_open_conns{")

	r.SetDBStats(func() (int, int, int, string, bool) { return 5, 3, 2, "sqlite", true })
	out := r.Render()
	assert.Contains(t, out, `tupay_db_open_conns{backend="sqlite"} 5`)
	assert.Contains(t, out, `tupay_db_idle_conns{backend="sqlite"} 3`)
	assert.Contains(t, out, `tupay_db_in_use_conns{backend="sqlite"} 2`)

	r.SetDBStats(func() (int, int, int, string, bool) { return 0, 0, 0, "", false })
	assert.NotContains(t, r.Render(), "tupay_db_open_conns{")

	require.True(t, strings.HasPrefix(out, "# HELP"))
}
