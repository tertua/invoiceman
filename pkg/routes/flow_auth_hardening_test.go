package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/tupay/platform/database"
)

// TestRegisterNormalizesEmail pins the email canonicalization at the input
// boundary: a padded/cased register lands lowercase, the same address logs in,
// a duplicate register is a 409 and the stored address is the canonical form.
func TestRegisterNormalizesEmail(t *testing.T) {
	t.Setenv("RATE_LIMIT_AUTH", "100")
	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Case User","email":"  CaseUser@Example.com ","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	registered := decodeBody(t, resp)["user"].(map[string]interface{})["email"].(string)
	assert.Equal(t, "caseuser@example.com", registered)

	// Login with the canonical form succeeds.
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"caseuser@example.com","password":"secret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	cookies := resp.Cookies()
	resp.Body.Close()

	// Registering the same address again (any casing) is a conflict.
	resp = doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Case User","email":"caseuser@example.com","password":"secret123"}`, nil)
	assert.Equal(t, 409, resp.StatusCode)
	resp.Body.Close()

	// The session reports the canonical address.
	resp = doRequest(t, app, "GET", "/api/auth/me", "", cookies)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "caseuser@example.com", decodeBody(t, resp)["user"].(map[string]interface{})["email"])
}

// TestDuplicateEmailRace fires N concurrent registers for one email and asserts
// exactly one wins: the check-then-insert guard plus the unique index must let a
// single 201 through and turn every loser into a 409 (never a 500), with one row.
func TestDuplicateEmailRace(t *testing.T) {
	t.Setenv("RATE_LIMIT_AUTH", "1000")
	app := newTestApp()

	const workers = 8
	// The routes suite shares one process-wide database across tests, so the
	// raced email must be unique per invocation (a fixed address would already
	// exist on the second run and turn the "winner" into a 409).
	email := "race-" + uuid.NewString() + "@example.com"
	body := `{"name":"Race User","email":"` + email + `","password":"secret123"}`

	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		status []int
	)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
			if err != nil {
				mu.Lock()
				status = append(status, -1)
				mu.Unlock()
				return
			}
			defer resp.Body.Close()
			mu.Lock()
			status = append(status, resp.StatusCode)
			mu.Unlock()
		}()
	}
	wg.Wait()

	created, conflict := 0, 0
	for _, code := range status {
		switch code {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			conflict++
		default:
			t.Fatalf("unexpected register status %d (want 201 or 409)", code)
		}
	}
	assert.Equal(t, 1, created, "exactly one concurrent register should win")
	assert.Equal(t, workers-1, conflict, "every loser should be a 409")

	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	var count int64
	require.NoError(t, db.UserQueries.DB.Table("users").Where("email = ?", email).Count(&count).Error)
	assert.EqualValues(t, 1, count, "exactly one user row should exist for the raced email")
}

// TestChangePasswordKillsOldRefresh pins the change-password session rotation
// (decision D4): the pre-change refresh token stops working, the user is not
// force-logged-out (the response mints a fresh session), and the new password
// logs in.
func TestChangePasswordKillsOldRefresh(t *testing.T) {
	t.Setenv("RATE_LIMIT_AUTH", "100")
	app := newTestApp()

	email := uniqueEmail("rotate-cp")
	cookies := registerUser(t, app, email, "secret123")

	// Capture the pre-change session cookies.
	var oldRefresh, oldAccess string
	for _, c := range cookies {
		switch c.Name {
		case "refresh_token":
			oldRefresh = c.Value
		case "access_token":
			oldAccess = c.Value
		}
	}
	require.NotEmpty(t, oldRefresh)
	require.NotEmpty(t, oldAccess)

	resp := doRequest(t, app, "PATCH", "/api/auth/password",
		`{"currentPassword":"secret123","newPassword":"newsecret123"}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	rotated := resp.Cookies()
	resp.Body.Close()

	// The response carries a fresh refresh token, distinct from the old one.
	var newRefresh string
	for _, c := range rotated {
		if c.Name == "refresh_token" {
			newRefresh = c.Value
		}
	}
	require.NotEmpty(t, newRefresh)
	assert.NotEqual(t, oldRefresh, newRefresh, "change-password should rotate the refresh token")

	// The old session cookies can no longer authenticate: neither the old access
	// token (new sid) nor the old refresh token (overwritten) works.
	resp = doRequest(t, app, "GET", "/api/auth/me", "", []*http.Cookie{
		{Name: "access_token", Value: oldAccess},
		{Name: "refresh_token", Value: oldRefresh},
	})
	assert.Equal(t, 401, resp.StatusCode)
	resp.Body.Close()

	// The freshly minted session still works.
	resp = doRequest(t, app, "GET", "/api/auth/me", "", rotated)
	require.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()

	// Login with the new password succeeds (session rotation, not logout).
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"`+email+`","password":"newsecret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()
}
