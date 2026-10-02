package routes

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRouteURLNoPluralDatabase guards a standing route-URL rule: the database
// segment is always `database`, never `databases`. It walks every route Fiber
// actually registered (public + gateway + private, v1 and legacy), so a bad
// path fails here instead of in production.
func TestRouteURLNoPluralDatabase(t *testing.T) {
	app := newVersionedApp()

	routes := app.GetRoutes()
	assert.NotEmpty(t, routes, "expected the app to register routes")

	var offenders []string
	for _, r := range routes {
		if strings.Contains(r.Path, "databases") {
			offenders = append(offenders, r.Path)
		}
	}
	assert.Empty(t, offenders,
		"route URLs must say `database`, never `databases`: %v", offenders)
}
