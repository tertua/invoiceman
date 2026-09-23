package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
)

// newTestApp builds the full API app for flow tests.
func newTestApp() *fiber.App {
	app := fiber.New()
	PublicRoutes(app)
	GatewayRoutes(app)
	PrivateRoutes(app)
	return app
}

// doRequest performs a request against the test app.
func doRequest(t *testing.T, app *fiber.App, method, route, body string, cookies []*http.Cookie) *http.Response {
	t.Helper()

	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, route, reader)
	req.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
		// Mirror the SPA: echo the CSRF cookie as its header.
		if cookie.Name == "csrf_token" && cookie.Value != "" {
			req.Header.Set("X-CSRF-Token", cookie.Value)
		}
	}

	resp, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	require.NoError(t, err)
	return resp
}

// mergeCookies overlays new cookies onto the jar by name (a refresh
// response only carries session cookies; the CSRF cookie persists).
func mergeCookies(old, incoming []*http.Cookie) []*http.Cookie {
	merged := append([]*http.Cookie{}, old...)
	index := map[string]int{}
	for i, c := range merged {
		index[c.Name] = i
	}
	for _, c := range incoming {
		if i, ok := index[c.Name]; ok {
			merged[i] = c
		} else {
			index[c.Name] = len(merged)
			merged = append(merged, c)
		}
	}
	return merged
}

// decodeBody decodes a JSON response body.
func decodeBody(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	defer resp.Body.Close()

	data := map[string]interface{}{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&data))
	return data
}
