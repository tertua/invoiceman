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

// TestOIDCIssuerValidation rejects issuers that are not a usable http(s) URL.
func TestOIDCIssuerValidation(t *testing.T) {
	valid := []string{"http://localhost:9000", "https://idp.example.com"}
	invalid := []string{"::not a url", "ftp://x", "idp.example.com", "mailto:a@b.com"}

	for _, issuer := range valid {
		t.Run("valid "+issuer, func(t *testing.T) {
			t.Setenv("OIDC_ENABLED", "true")
			t.Setenv("OIDC_ISSUER", issuer)
			t.Setenv("OIDC_CLIENT_ID", "client-1")
			t.Setenv("OIDC_CLIENT_SECRET", "secret-1")
			cfg, err := Load()
			require.NoError(t, err)
			assert.True(t, cfg.OIDC.Active())
		})
	}
	for _, issuer := range invalid {
		t.Run("invalid "+issuer, func(t *testing.T) {
			t.Setenv("OIDC_ENABLED", "true")
			t.Setenv("OIDC_ISSUER", issuer)
			t.Setenv("OIDC_CLIENT_ID", "client-1")
			t.Setenv("OIDC_CLIENT_SECRET", "secret-1")
			_, err := Load()
			require.ErrorContains(t, err, "OIDC_ISSUER")
		})
	}
	// An empty issuer still fails the required-field check, not the URL check.
	t.Run("empty", func(t *testing.T) {
		t.Setenv("OIDC_ENABLED", "true")
		t.Setenv("OIDC_ISSUER", "")
		t.Setenv("OIDC_CLIENT_ID", "client-1")
		t.Setenv("OIDC_CLIENT_SECRET", "secret-1")
		_, err := Load()
		require.ErrorContains(t, err, "OIDC_ISSUER")
	})
}

// TestOIDCIssuerNormalized trims whitespace and a trailing slash from the issuer.
func TestOIDCIssuerNormalized(t *testing.T) {
	t.Setenv("OIDC_ENABLED", "true")
	t.Setenv("OIDC_ISSUER", " https://idp.example.com/ ")
	t.Setenv("OIDC_CLIENT_ID", "client-1")
	t.Setenv("OIDC_CLIENT_SECRET", "secret-1")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "https://idp.example.com", cfg.OIDC.Issuer)
	assert.True(t, cfg.OIDC.Active())
}

// TestOIDCRoleClaimDefaults verifies role sync is off out of the box.
func TestOIDCRoleClaimDefaults(t *testing.T) {
	cfg, err := Load()
	require.NoError(t, err)
	assert.False(t, cfg.OIDC.RoleSyncEnabled())
	assert.Empty(t, cfg.OIDC.RoleClaim)
	assert.Equal(t, "admin", cfg.OIDC.AdminRole)
}

// TestOIDCRoleClaimEnabled verifies a configured path activates role sync and
// keeps the default admin value.
func TestOIDCRoleClaimEnabled(t *testing.T) {
	t.Setenv("OIDC_ENABLED", "true")
	t.Setenv("OIDC_ISSUER", "https://idp.example.com")
	t.Setenv("OIDC_CLIENT_ID", "client-1")
	t.Setenv("OIDC_CLIENT_SECRET", "secret-1")
	t.Setenv("OIDC_ROLE_CLAIM", "realm_access.roles")

	cfg, err := Load()
	require.NoError(t, err)
	assert.True(t, cfg.OIDC.RoleSyncEnabled())
	assert.Equal(t, "realm_access.roles", cfg.OIDC.RoleClaim)
	assert.Equal(t, "admin", cfg.OIDC.AdminRole)
}

// TestOIDCRoleClaimCustomAdmin verifies OIDC_ADMIN_ROLE overrides the default.
func TestOIDCRoleClaimCustomAdmin(t *testing.T) {
	t.Setenv("OIDC_ENABLED", "true")
	t.Setenv("OIDC_ISSUER", "https://idp.example.com")
	t.Setenv("OIDC_CLIENT_ID", "client-1")
	t.Setenv("OIDC_CLIENT_SECRET", "secret-1")
	t.Setenv("OIDC_ROLE_CLAIM", "roles")
	t.Setenv("OIDC_ADMIN_ROLE", "superuser")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "superuser", cfg.OIDC.AdminRole)
}

// TestOIDCRoleClaimEmptySegment rejects a malformed dot-path when enabled.
func TestOIDCRoleClaimEmptySegment(t *testing.T) {
	t.Setenv("OIDC_ENABLED", "true")
	t.Setenv("OIDC_ISSUER", "https://idp.example.com")
	t.Setenv("OIDC_CLIENT_ID", "client-1")
	t.Setenv("OIDC_CLIENT_SECRET", "secret-1")
	t.Setenv("OIDC_ROLE_CLAIM", "a..b")

	_, err := Load()
	require.ErrorContains(t, err, "OIDC_ROLE_CLAIM")
}

// TestOIDCRoleClaimBlankAdminFallsBack verifies a blank OIDC_ADMIN_ROLE keeps
// the "admin" default (envOr fallback), so role sync never fails startup over
// it; the empty-admin guard in Validate is a programmatic-only safety net.
func TestOIDCRoleClaimBlankAdminFallsBack(t *testing.T) {
	t.Setenv("OIDC_ENABLED", "true")
	t.Setenv("OIDC_ISSUER", "https://idp.example.com")
	t.Setenv("OIDC_CLIENT_ID", "client-1")
	t.Setenv("OIDC_CLIENT_SECRET", "secret-1")
	t.Setenv("OIDC_ROLE_CLAIM", "roles")
	t.Setenv("OIDC_ADMIN_ROLE", "  ")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "admin", cfg.OIDC.AdminRole)
}

// TestOIDCRoleClaimDisabledInert verifies a path with SSO off stays inert and
// never trips validation (no role mapping runs while SSO is disabled).
func TestOIDCRoleClaimDisabledInert(t *testing.T) {
	t.Setenv("OIDC_ENABLED", "false")
	t.Setenv("OIDC_ROLE_CLAIM", "a..b")

	cfg, err := Load()
	require.NoError(t, err)
	assert.True(t, cfg.OIDC.RoleSyncEnabled())
}
