package routes

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOIDCProviderDiscoveryCached proves end-to-end that discovery runs once
// per issuer: the login start and its callback share one cached provider, so
// the provider's well-known endpoint is hit exactly once for the flow.
func TestOIDCProviderDiscoveryCached(t *testing.T) {
	idp := newFakeIdP(t)
	seedOIDCEnv(t, idp)
	app := newTestApp()

	_, q := oidcStart(t, app)
	idp.nextClaims = idp.claimSet("sub-cache", "cache@example.com", true, q.Get("nonce"))

	resp := doRequest(t, app, "GET", oidcCallback(q.Get("state"), "test-code"), "", nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusFound, resp.StatusCode)
	assert.Equal(t, "/dashboard", fetchLocation(resp))

	// login + callback resolved the same issuer -> exactly one discovery fetch.
	assert.EqualValues(t, 1, idp.discoveryHits.Load(), "discovery must be cached per issuer")
}
