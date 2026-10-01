package configs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOIDCDefaults verifies SSO stays off out of the box.
func TestOIDCDefaults(t *testing.T) {
	cfg, err := Load()
	require.NoError(t, err)
	assert.False(t, cfg.OIDC.Enabled)
	assert.False(t, cfg.OIDC.Active())
	assert.Empty(t, cfg.OIDC.Issuer)
	assert.Equal(t, "openid email profile", cfg.OIDC.Scopes)
}

// TestOIDCEnabledRequiresFields verifies fail-fast when enabled but incomplete.
func TestOIDCEnabledRequiresFields(t *testing.T) {
	t.Setenv("OIDC_ENABLED", "true")

	_, err := Load()
	require.ErrorContains(t, err, "OIDC_ISSUER")
	require.ErrorContains(t, err, "OIDC_CLIENT_ID")
	require.ErrorContains(t, err, "OIDC_CLIENT_SECRET")

	t.Setenv("OIDC_ISSUER", "https://idp.example.com")
	t.Setenv("OIDC_CLIENT_ID", "client-1")
	_, err = Load()
	require.ErrorContains(t, err, "OIDC_CLIENT_SECRET")
}

// TestOIDCEnabledValid verifies a complete block activates SSO.
func TestOIDCEnabledValid(t *testing.T) {
	t.Setenv("OIDC_ENABLED", "true")
	t.Setenv("OIDC_ISSUER", "https://idp.example.com")
	t.Setenv("OIDC_CLIENT_ID", "client-1")
	t.Setenv("OIDC_CLIENT_SECRET", "secret-1")
	t.Setenv("OIDC_SCOPES", "openid email")

	cfg, err := Load()
	require.NoError(t, err)
	assert.True(t, cfg.OIDC.Enabled)
	assert.True(t, cfg.OIDC.Active())
	assert.Equal(t, []string{"openid", "email"}, cfg.OIDC.ScopeList())
}

// TestOIDCDisabledUnchanged verifies a partial block with the flag off is inert.
func TestOIDCDisabledUnchanged(t *testing.T) {
	t.Setenv("OIDC_ENABLED", "false")
	t.Setenv("OIDC_ISSUER", "")
	t.Setenv("OIDC_CLIENT_ID", "")
	t.Setenv("OIDC_CLIENT_SECRET", "")

	cfg, err := Load()
	require.NoError(t, err)
	assert.False(t, cfg.OIDC.Active())
}
