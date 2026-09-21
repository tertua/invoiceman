package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
)

func TestPrivateRoutes(t *testing.T) {
	// Define a structure for specifying input and output data of a single test case.
	// These cases have no session cookie, so no database is required.
	tests := []struct {
		description  string
		method       string
		route        string // input route
		expectedCode int
	}{
		{
			description:  "get session user without cookie",
			method:       "GET",
			route:        "/api/auth/me",
			expectedCode: 401,
		},
		{
			description:  "list clients without cookie",
			method:       "GET",
			route:        "/api/clients",
			expectedCode: 401,
		},
		{
			description:  "create client without cookie",
			method:       "POST",
			route:        "/api/clients",
			expectedCode: 401,
		},
		{
			description:  "list invoices without cookie",
			method:       "GET",
			route:        "/api/invoices",
			expectedCode: 401,
		},
		{
			description:  "create invoice without cookie",
			method:       "POST",
			route:        "/api/invoices",
			expectedCode: 401,
		},
		{
			description:  "get dashboard without cookie",
			method:       "GET",
			route:        "/api/dashboard",
			expectedCode: 401,
		},
		{
			description:  "logout without cookie",
			method:       "POST",
			route:        "/api/auth/logout",
			expectedCode: 401,
		},
		{
			description:  "create invoice intent without cookie",
			method:       "POST",
			route:        "/api/gateway/invoice-intents",
			expectedCode: 401,
		},
		{
			description:  "list gateway projects without cookie",
			method:       "GET",
			route:        "/api/admin/gateway/projects",
			expectedCode: 401,
		},
	}

	// Define a new Fiber app.
	app := fiber.New()

	// Define routes.
	PrivateRoutes(app)

	// Iterate through test single test cases
	for _, test := range tests {
		// Create a new http request with the route from the test case.
		req := httptest.NewRequest(test.method, test.route, http.NoBody)
		req.Header.Set("Content-Type", "application/json")

		// Perform the request plain with the app.
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})

		// Verify, that no error occurred.
		assert.Equalf(t, false, err != nil, test.description)

		// Verify, if the status code is as expected.
		assert.Equalf(t, test.expectedCode, resp.StatusCode, test.description)
	}
}
