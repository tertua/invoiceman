package controllers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/tertua/tupay/platform/cache"
)

// oidcStateTTL bounds how long an in-flight SSO attempt stays valid.
const oidcStateTTL = 10 * time.Minute

// oidcState is what a pending login remembers between /login and /callback:
// the nonce to bind the ID token to this attempt and the PKCE verifier.
type oidcState struct {
	Nonce    string `json:"nonce"`
	Verifier string `json:"verifier"`
}

// randomHex returns n random bytes as lowercase hex.
func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// oidcStateKey namespaces OIDC state under the session store; the prefix can
// never collide with a session key (a bare user UUID).
func oidcStateKey(state string) string { return "oidc:" + state }

// saveOIDCState stores the nonce+verifier for one state, single-use by TTL.
func saveOIDCState(ctx context.Context, state string, s oidcState) error {
	store, err := cache.Sessions()
	if err != nil {
		return err
	}
	raw, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return store.Set(ctx, oidcStateKey(state), string(raw), oidcStateTTL)
}

// takeOIDCState reads and immediately deletes a state entry (delete-before-use
// so a replayed callback can never consume it twice). A miss/expired entry
// returns ok=false.
func takeOIDCState(ctx context.Context, state string) (oidcState, bool) {
	store, err := cache.Sessions()
	if err != nil {
		return oidcState{}, false
	}
	raw, err := store.Get(ctx, oidcStateKey(state))
	if err != nil {
		return oidcState{}, false
	}
	// Delete first: even if decoding fails below, the attempt is spent.
	_ = store.Delete(ctx, oidcStateKey(state))
	var s oidcState
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return oidcState{}, false
	}
	return s, true
}
