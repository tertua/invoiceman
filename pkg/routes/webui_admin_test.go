package routes

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// adminFixture builds a dist stand-in holding both MPA entries (product + console).
func adminFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>product</html>"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "admin.html"), []byte("<html>console</html>"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "assets"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log(1)"), 0o644))
	return dir
}

// htmlBody GETs path and returns the body, asserting a 200.
func htmlBody(t *testing.T, app *fiber.App, path string) string {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest("GET", path, http.NoBody), fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, path)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return string(raw)
}

// TestMountWebUISplitEntries pins the MPA contract: /admin and /admin/* get admin.html, other app routes keep index.html.
func TestMountWebUISplitEntries(t *testing.T) {
	app := fiber.New()
	MountWebUI(app, adminFixture(t))

	for _, path := range []string{"/admin", "/admin/users", "/admin/gateway"} {
		assert.Contains(t, htmlBody(t, app, path), "console", path)
	}
	assert.Contains(t, htmlBody(t, app, "/dashboard"), "product")
}

// TestMountWebUIAdminPrefixIsExact guards against over-matching: /adminfoo is not the console.
func TestMountWebUIAdminPrefixIsExact(t *testing.T) {
	app := fiber.New()
	MountWebUI(app, adminFixture(t))

	assert.Contains(t, htmlBody(t, app, "/adminfoo"), "product")
}

// TestMountWebUIWithoutAdminEntry: an index-only dist (pre-MPA build) still answers /admin with the product shell instead of erroring.
func TestMountWebUIWithoutAdminEntry(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>product</html>"), 0o644))
	app := fiber.New()
	MountWebUI(app, dir)

	assert.Contains(t, htmlBody(t, app, "/admin/users"), "product")
}
