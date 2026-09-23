package routes

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tinyPNG is a 1x1 transparent PNG.
var tinyPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41,
	0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00,
	0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
	0x42, 0x60, 0x82,
}

// multipartFile builds a file upload request, mirroring the SPA (which
// echoes the CSRF cookie as its header).
func multipartFile(t *testing.T, app *fiber.App, method, route, field, filename, contentType string, data []byte, cookies []*http.Cookie) *http.Response {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	h := map[string][]string{
		"Content-Disposition": {fmt.Sprintf(`form-data; name=%q; filename=%q`, field, filename)},
		"Content-Type":        {contentType},
	}
	part, err := w.CreatePart(h)
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req := httptest.NewRequest(method, route, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	for _, c := range cookies {
		req.AddCookie(c)
		if c.Name == "csrf_token" && c.Value != "" {
			req.Header.Set("X-CSRF-Token", c.Value)
		}
	}
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	require.NoError(t, err)
	return resp
}

// TestLogoUpload stores the logo as a file URL instead of a data-URL.
func TestLogoUpload(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Logo User","email":"logo@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	cookies := resp.Cookies()
	resp.Body.Close()

	resp = multipartFile(t, app, "POST", "/api/settings/logo", "logo", "logo.png", "image/png", tinyPNG, cookies)
	require.Equal(t, 200, resp.StatusCode)
	logoURL := decodeBody(t, resp)["settings"].(map[string]interface{})["logo_url"].(string)
	assert.Contains(t, logoURL, "/uploads/logos/")
	assert.NotContains(t, logoURL, "data:")

	// Non-images are rejected.
	resp = multipartFile(t, app, "POST", "/api/settings/logo", "logo", "evil.txt", "text/plain", []byte("nope"), cookies)
	assert.Equal(t, 400, resp.StatusCode)
	resp.Body.Close()

	// SVG is rejected (stored XSS: logos render on public pages).
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
	resp = multipartFile(t, app, "POST", "/api/settings/logo", "logo", "logo.svg", "image/svg+xml", svg, cookies)
	assert.Equal(t, 400, resp.StatusCode)
	resp.Body.Close()
}

// TestReceiptNotExposedViaStaticUploads guards the PRIVATE-receipt
// contract: even with a valid storage key, /uploads must never serve
// receipts — only the ownership-checked proxy may.
func TestReceiptNotExposedViaStaticUploads(t *testing.T) {
	dir := t.TempDir()
	app := fiber.New()
	require.NoError(t, MountUploads(app, dir))

	// Seed a receipt file directly on disk (as UploadReceipt would).
	receiptDir := dir + "/receipts/u1"
	require.NoError(t, os.MkdirAll(receiptDir, 0o755))
	require.NoError(t, os.WriteFile(receiptDir+"/e1.png", tinyPNG, 0o644))

	for _, path := range []string{
		"/uploads/receipts/u1/e1.png",
		"/uploads/RECEIPTS/u1/e1.png",
		"/uploads/logos/../../receipts/u1/e1.png",
		"/uploads/receipts",
		"/uploads",
	} {
		resp, err := app.Test(httptest.NewRequest("GET", path, http.NoBody), fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode, path)
		resp.Body.Close()
	}

	// Logos still serve.
	require.NoError(t, os.WriteFile(dir+"/logos/u1.png", tinyPNG, 0o644))
	resp, err := app.Test(httptest.NewRequest("GET", "/uploads/logos/u1.png", http.NoBody), fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	got, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, tinyPNG, got)
}

// TestReceiptRoundTrip covers upload, proxy download and deletion.
func TestReceiptRoundTrip(t *testing.T) {
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Receipt User","email":"receipt@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	cookies := resp.Cookies()
	resp.Body.Close()

	resp = doRequest(t, app, "POST", "/api/expenses",
		`{"category":"Office","expense_date":"2026-09-01","amount":25,"currency":"USD"}`, cookies)
	require.Equal(t, 201, resp.StatusCode)
	expenseID := decodeBody(t, resp)["expense"].(map[string]interface{})["id"].(string)

	resp = multipartFile(t, app, "POST", "/api/expenses/"+expenseID+"/receipt", "file", "r.png", "image/png", tinyPNG, cookies)
	require.Equal(t, 200, resp.StatusCode)
	receiptURL := decodeBody(t, resp)["expense"].(map[string]interface{})["receipt_url"].(string)
	assert.Contains(t, receiptURL, "/api/v1/expenses/"+expenseID+"/receipt")

	// SVG receipts are rejected even though they sniff as image/*.
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
	resp = multipartFile(t, app, "POST", "/api/expenses/"+expenseID+"/receipt", "file", "r.svg", "image/svg+xml", svg, cookies)
	assert.Equal(t, 400, resp.StatusCode)
	resp.Body.Close()

	// Authenticated proxy streams the bytes back.
	resp = doRequest(t, app, "GET", "/api/expenses/"+expenseID+"/receipt", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	got, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, tinyPNG, got)

	// Another user's receipt is invisible (ownership check).
	resp = doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Other","email":"other-receipt@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	otherCookies := resp.Cookies()
	resp.Body.Close()
	resp = doRequest(t, app, "GET", "/api/expenses/"+expenseID+"/receipt", "", otherCookies)
	assert.Equal(t, 404, resp.StatusCode)
	resp.Body.Close()

	// Deletion clears the attachment.
	resp = doRequest(t, app, "DELETE", "/api/expenses/"+expenseID+"/receipt", "", cookies)
	assert.Equal(t, 204, resp.StatusCode)
	resp = doRequest(t, app, "GET", "/api/expenses/"+expenseID+"/receipt", "", cookies)
	assert.Equal(t, 404, resp.StatusCode)
	resp.Body.Close()
}
