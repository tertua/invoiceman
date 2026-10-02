package routes

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/tupay/platform/cache"
)

// TestOIDCTokenClaimGaps covers ID-token claim defects: each must fail closed
// with the stable provider/email code and never start a session.
func TestOIDCTokenClaimGaps(t *testing.T) {
	idp := newFakeIdP(t)
	seedOIDCEnv(t, idp)
	app := newTestApp()

	// Each case starts a fresh flow, mutates only the claims under test, and
	// asserts the redirect code.
	cases := []struct {
		name    string
		code    string
		prepare func(q map[string]string)
	}{
		{
			name: "missing id_token", code: "provider",
			prepare: func(q map[string]string) { idp.omitIDToken = true },
		},
		{
			name: "bad signature", code: "provider",
			prepare: func(q map[string]string) { idp.badSigner = newBadSigner(t) },
		},
		{
			name: "wrong audience", code: "provider",
			prepare: func(q map[string]string) {
				idp.tokenOverride = mutatingTokenOverride(func(c map[string]any) { c["aud"] = "other-client" })
			},
		},
		{
			name: "expired token", code: "provider",
			prepare: func(q map[string]string) {
				idp.tokenOverride = mutatingTokenOverride(func(c map[string]any) { c["exp"] = time.Now().Add(-time.Hour).Unix() })
			},
		},
		{
			name: "missing sub", code: "email",
			prepare: func(q map[string]string) {
				idp.tokenOverride = mutatingTokenOverride(func(c map[string]any) { delete(c, "sub") })
			},
		},
		{
			name: "missing email", code: "email",
			prepare: func(q map[string]string) {
				idp.tokenOverride = mutatingTokenOverride(func(c map[string]any) { delete(c, "email") })
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset harness mutators, then apply this case's defect.
			idp.omitIDToken, idp.badSigner, idp.tokenOverride = false, nil, nil
			_, q := oidcStart(t, app)
			idp.nextClaims = idp.claimSet("sub-claim-"+tc.name, "claim-"+tc.name+"@example.com", true, q.Get("nonce"))
			tc.prepare(map[string]string{"state": q.Get("state")})

			resp := doRequest(t, app, "GET", oidcCallback(q.Get("state"), "test-code"), "", nil)
			defer resp.Body.Close()
			require.Equal(t, http.StatusFound, resp.StatusCode)
			assert.Equal(t, "/login?oidc_error="+tc.code, fetchLocation(resp))
		})
	}
}

// TestOIDCStateTTLExpiry proves an expired state entry is rejected: the entry
// is written with a short TTL and the callback runs after it lapses.
func TestOIDCStateTTLExpiry(t *testing.T) {
	idp := newFakeIdP(t)
	seedOIDCEnv(t, idp)
	app := newTestApp()

	store, err := cache.Sessions()
	require.NoError(t, err)
	state := "expired-state"
	raw, err := json.Marshal(map[string]string{"nonce": "n", "verifier": "v"})
	require.NoError(t, err)
	require.NoError(t, store.Set(t.Context(), "oidc:"+state, string(raw), 10*time.Millisecond))
	time.Sleep(30 * time.Millisecond)

	resp := doRequest(t, app, "GET", oidcCallback(state, "test-code"), "", nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusFound, resp.StatusCode)
	assert.Equal(t, "/login?oidc_error=state", fetchLocation(resp))
}
