package routes

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// webuiFixture builds a minimal webui/dist stand-in (index + one asset).
func webuiFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>spa</html>"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "assets"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log(1)"), 0o644))
	return dir
}

// TestMountWebUIEmptyDirIsNoOp keeps the default (API-only) image untouched:
// no dir configured means no extra routes are registered.
func TestMountWebUIEmptyDirIsNoOp(t *testing.T) {
	app := fiber.New()
	NotFoundRoute(app)
	MountWebUI(app, "")

	resp, err := app.Test(httptest.NewRequest("GET", "/dashboard", http.NoBody), fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type")[:16])
	resp.Body.Close()
}

// TestMountWebUI covers the embedded-frontend contract: SPA routes fall back
// to index.html, assets are served, and unknown /api/* keeps the JSON 404.
func TestMountWebUI(t *testing.T) {
	app := fiber.New()
	// Mirror main.go ordering: SPA fallback before the final JSON 404.
	MountWebUI(app, webuiFixture(t))
	NotFoundRoute(app)

	// Client-side route falls back to the SPA shell.
	resp, err := app.Test(httptest.NewRequest("GET", "/dashboard", http.NoBody), fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")
	resp.Body.Close()

	// Built asset is served directly.
	resp, err = app.Test(httptest.NewRequest("GET", "/assets/app.js", http.NoBody), fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()

	// Unknown API endpoints keep the JSON 404 (never the SPA shell).
	resp, err = app.Test(httptest.NewRequest("GET", "/api/v1/nope", http.NoBody), fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type")[:16])
	body := decodeBody(t, resp)
	assert.Equal(t, true, body["error"])
}
