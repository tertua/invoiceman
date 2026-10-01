package configs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRequireEmailVerificationDefault pins the default-on flag.
func TestRequireEmailVerificationDefault(t *testing.T) {
	cfg, err := Load()
	require.NoError(t, err)
	assert.True(t, cfg.Auth.RequireEmailVerification)
}

// TestRequireEmailVerificationOverride pins the env override.
func TestRequireEmailVerificationOverride(t *testing.T) {
	t.Setenv("REQUIRE_EMAIL_VERIFICATION", "false")
	cfg, err := Load()
	require.NoError(t, err)
	assert.False(t, cfg.Auth.RequireEmailVerification)
}
