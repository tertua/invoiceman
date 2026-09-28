package configs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWebUIDirEnvFallback: legacy deployments only set SERVE_SPA_DIR, while SERVE_WEBUI wins when both are present.
func TestWebUIDirEnvFallback(t *testing.T) {
	t.Setenv("SERVE_WEBUI", "")
	t.Setenv("SERVE_SPA_DIR", "/legacy/spa")
	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "/legacy/spa", cfg.WebUI.Dir)

	t.Setenv("SERVE_WEBUI", "/new/webui")
	cfg, err = Load()
	require.NoError(t, err)
	assert.Equal(t, "/new/webui", cfg.WebUI.Dir)
}
