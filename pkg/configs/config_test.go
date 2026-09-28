package configs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadDefaults verifies zero-config dev works out of the box.
func TestLoadDefaults(t *testing.T) {
	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "dev", cfg.Stage)
	assert.Equal(t, "Invoiceman", cfg.AppName)
	assert.Equal(t, "0.0.0.0:5000", cfg.ListenAddr())
	assert.Equal(t, 100, cfg.RateLimit.General)
	assert.Equal(t, 10, cfg.RateLimit.Auth)
	assert.Equal(t, 30, cfg.AI.TimeoutSec)
	assert.False(t, cfg.Redis.Enabled())
	assert.True(t, cfg.Metrics.Enabled)
	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, 24, cfg.Idempotency.TTLHours)
	assert.Equal(t, 10, cfg.Outbox.PollSeconds)
	assert.Equal(t, 20, cfg.Outbox.Batch)
	assert.Equal(t, 60, cfg.Cache.AggTTLSeconds)
	assert.Equal(t, "local", cfg.Storage.Backend)
	assert.True(t, cfg.Auth.AllowRegistration)
	assert.Empty(t, cfg.WebUI.Dir)
	assert.Empty(t, cfg.Server.TrustedProxies)
	assert.Equal(t, "X-Forwarded-For", cfg.Server.ProxyHeader)
}

// TestLoadOverrides verifies env values win over defaults.
func TestLoadOverrides(t *testing.T) {
	t.Setenv("APP_NAME", "Acme")
	t.Setenv("RATE_LIMIT_AUTH", "5")
	t.Setenv("CORS_ORIGINS", "https://a.example.com, https://b.example.com")
	t.Setenv("REDIS_HOST", "redis")
	t.Setenv("LOG_LEVEL", "DEBUG")
	t.Setenv("ALLOW_REGISTRATION", "false")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "Acme", cfg.AppName)
	assert.Equal(t, 5, cfg.RateLimit.Auth)
	assert.Equal(t, []string{"https://a.example.com", "https://b.example.com"}, cfg.CORS.Origins)
	assert.True(t, cfg.Redis.Enabled())
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.False(t, cfg.Auth.AllowRegistration)
}

// TestValidateProxyTrust covers the TRUSTED_PROXIES allowlist parsing.
func TestValidateProxyTrust(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "10.0.0.1, 192.168.0.0/16, 2001:db8::1")
	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, []string{"10.0.0.1", "192.168.0.0/16", "2001:db8::1"}, cfg.Server.TrustedProxies)

	t.Setenv("TRUSTED_PROXIES", "not-an-ip")
	_, err = Load()
	require.ErrorContains(t, err, "TRUSTED_PROXIES")

	t.Setenv("TRUSTED_PROXIES", "10.0.0.0/33")
	_, err = Load()
	require.ErrorContains(t, err, "TRUSTED_PROXIES")

	// Blank header falls back to the default (envOr), never an error.
	t.Setenv("TRUSTED_PROXIES", "")
	t.Setenv("PROXY_HEADER", "   ")
	cfg, err = Load()
	require.NoError(t, err)
	assert.Equal(t, "X-Forwarded-For", cfg.Server.ProxyHeader)

	t.Setenv("PROXY_HEADER", "X-Real-IP")
	cfg, err = Load()
	require.NoError(t, err)
	assert.Equal(t, "X-Real-IP", cfg.Server.ProxyHeader)
}
func TestValidateProdSecrets(t *testing.T) {
	t.Setenv("STAGE_STATUS", "prod")
	t.Setenv("JWT_SECRET_KEY", "secret")
	t.Setenv("JWT_REFRESH_KEY", "refresh")

	_, err := Load()
	require.ErrorContains(t, err, "JWT_SECRET_KEY")
}

// TestValidateBadValues covers DSN, ports and unknown stages.
func TestValidateBadValues(t *testing.T) {
	t.Setenv("SQL_DSN", "mysql://user:pass@localhost/db")
	_, err := Load()
	require.ErrorContains(t, err, "SQL_DSN")

	t.Setenv("SQL_DSN", "")
	t.Setenv("STAGE_STATUS", "staging")
	_, err = Load()
	require.ErrorContains(t, err, "STAGE_STATUS")

	t.Setenv("STAGE_STATUS", "dev")
	t.Setenv("SERVER_PORT", "99999")
	_, err = Load()
	require.ErrorContains(t, err, "SERVER_PORT")

	t.Setenv("SERVER_PORT", "5000")
	t.Setenv("LOG_LEVEL", "verbose")
	_, err = Load()
	require.ErrorContains(t, err, "LOG_LEVEL")
}

// TestValidatePostgresDSN passes with a production-like DSN and secrets.
func TestValidatePostgresDSN(t *testing.T) {
	t.Setenv("STAGE_STATUS", "prod")
	t.Setenv("JWT_SECRET_KEY", "test-secret-value")
	t.Setenv("JWT_REFRESH_KEY", "test-refresh-value")
	t.Setenv("SQL_DSN", "postgres://postgres:password@localhost/postgres?sslmode=disable")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "postgres://postgres:password@localhost/postgres?sslmode=disable", cfg.DSN())
}
