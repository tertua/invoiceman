package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newDiscoveryServer serves a minimal OIDC discovery document and counts hits.
func newDiscoveryServer(t *testing.T, hits *atomic.Int64) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                "http://" + r.Host,
			"authorization_endpoint":                "http://" + r.Host + "/authorize",
			"token_endpoint":                        "http://" + r.Host + "/token",
			"jwks_uri":                              "http://" + r.Host + "/jwks",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// TestOIDCProviderCache pins the per-issuer discovery cache: same issuer hits
// discovery once, a different issuer fetches again, and reset re-fetches.
func TestOIDCProviderCache(t *testing.T) {
	resetOIDCProviderCache()
	t.Cleanup(resetOIDCProviderCache)

	var hits atomic.Int64
	idp := newDiscoveryServer(t, &hits)
	ctx := context.Background()

	first, err := oidcProviderFor(ctx, idp.URL)
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.EqualValues(t, 1, hits.Load(), "first call fetches discovery")

	second, err := oidcProviderFor(ctx, idp.URL)
	require.NoError(t, err)
	assert.Same(t, first, second, "second call returns the cached provider")
	assert.EqualValues(t, 1, hits.Load(), "cache hit must not re-fetch discovery")

	// A different issuer string is a different cache key.
	other := newDiscoveryServer(t, &hits)
	_, err = oidcProviderFor(ctx, other.URL)
	require.NoError(t, err)
	assert.EqualValues(t, 2, hits.Load(), "a new issuer fetches discovery")

	// Reset drops the cache so the next call re-fetches.
	resetOIDCProviderCache()
	_, err = oidcProviderFor(ctx, idp.URL)
	require.NoError(t, err)
	assert.EqualValues(t, 3, hits.Load(), "after reset the issuer re-fetches")
}

// TestOIDCProviderCacheDoesNotCacheErrors proves a discovery failure is not
// cached: the next call retries instead of returning a stale nil.
func TestOIDCProviderCacheDoesNotCacheErrors(t *testing.T) {
	resetOIDCProviderCache()
	t.Cleanup(resetOIDCProviderCache)

	_, err := oidcProviderFor(context.Background(), "http://127.0.0.1:1/nope")
	require.Error(t, err)

	oidcProviderMu.RLock()
	_, cached := oidcProviderCache["http://127.0.0.1:1/nope"]
	oidcProviderMu.RUnlock()
	assert.False(t, cached, "a failed discovery must not be cached")
}
