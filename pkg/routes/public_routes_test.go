package routes

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
)

func TestPublicRoutes(t *testing.T) {
	// Define a structure for specifying input and output data of a single test case.
	// These cases fail on validation, so no database is required.
	tests := []struct {
		description  string
		method       string
		route        string // input route
		body         string // input body
		expectedCode int
	}{
		{
			description:  "register with empty body",
			method:       "POST",
			route:        "/api/auth/register",
			body:         `{}`,
			expectedCode: 400,
		},
		{
			description:  "register with invalid email",
			method:       "POST",
			route:        "/api/auth/register",
			body:         `{"name":"Test","email":"not-an-email","password":"secret123"}`,
			expectedCode: 400,
		},
		{
			description:  "login with empty body",
			method:       "POST",
			route:        "/api/auth/login",
			body:         `{}`,
			expectedCode: 400,
		},
		{
			description:  "forgot password with invalid email",
			method:       "POST",
			route:        "/api/auth/forgot-password",
			body:         `{"email":"not-an-email"}`,
			expectedCode: 400,
		},
		{
			description:  "reset password with empty body",
			method:       "POST",
			route:        "/api/auth/reset-password",
			body:         `{}`,
			expectedCode: 400,
		},
	}

	// Define Fiber app.
	app := fiber.New()

	// Define routes.
	PublicRoutes(app)

	// Iterate through test single test cases
	for _, test := range tests {
		// Create a new http request with the route from the test case.
		req := httptest.NewRequest(test.method, test.route, strings.NewReader(test.body))
		req.Header.Set("Content-Type", "application/json")

		// Perform the request plain with the app.
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})

		// Verify, that no error occurred.
		assert.Equalf(t, false, err != nil, test.description)

		// Verify, if the status code is as expected.
		assert.Equalf(t, test.expectedCode, resp.StatusCode, test.description)
	}
}
