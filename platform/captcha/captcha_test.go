package captcha

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVerifyDisabled passes through without a secret.
func TestVerifyDisabled(t *testing.T) {
	t.Setenv("TURNSTILE_SECRET", "")
	assert.False(t, Required())
	assert.ErrorIs(t, Verify(context.Background(), "anything", ""), ErrNotConfigured)
}

// TestVerifySuccessAndFailure exercises the provider contract.
func TestVerifySuccessAndFailure(t *testing.T) {
	t.Setenv("TURNSTILE_SECRET", "test-secret")
	assert.True(t, Required())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		if r.Form.Get("secret") == "test-secret" && r.Form.Get("response") == "good-token" {
			_, _ = w.Write([]byte(`{"success":true}`))
			return
		}
		_, _ = w.Write([]byte(`{"success":false,"error-codes":["invalid-input-response"]}`))
	}))
	defer server.Close()

	old := verifyURL
	verifyURL = server.URL
	defer func() { verifyURL = old }()

	assert.NoError(t, Verify(context.Background(), "good-token", "127.0.0.1"))
	assert.ErrorIs(t, Verify(context.Background(), "bad-token", ""), ErrFailed)
	assert.ErrorIs(t, Verify(context.Background(), "", ""), ErrFailed)
}
