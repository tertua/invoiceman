package routes

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/platform/database"
	"github.com/tertua/tupay/platform/relay"

	"github.com/gofiber/fiber/v3"
)

// uniqueEmail returns a fresh address so a rerun never trips the dup-email 409.
func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%d@example.com", prefix, time.Now().UnixNano())
}

// mustRead drains a response body as a string (for message assertions) and closes it.
func mustRead(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return raw
}

// verifyFlagOn flips verification on for one test (TestMain seeds it off) and
// guarantees an existing user so the first-install bootstrap (count == 0) does
// not exempt the account under test when this file runs alone.
func verifyFlagOn(t *testing.T, app *fiber.App) {
	t.Helper()
	t.Setenv("RATE_LIMIT_AUTH", "100")
	seedBootstrapUser(t, app)
	t.Setenv("REQUIRE_EMAIL_VERIFICATION", "true")
}

// seedBootstrapUser registers one active account with verification off so a
// later register in the same isolated run is not treated as the first install.
func seedBootstrapUser(t *testing.T, app *fiber.App) {
	t.Helper()
	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	count, err := db.CountUsers()
	require.NoError(t, err)
	if count > 0 {
		return
	}
	t.Setenv("REQUIRE_EMAIL_VERIFICATION", "false")
	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Bootstrap","email":"`+uniqueEmail("bootstrap")+`","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode, "bootstrap register")
	resp.Body.Close()
}

// TestEmailVerificationPendingRegistration covers the core pending flow:
// register with the flag ON makes a pending account, mails a link, starts no
// session, and blocks login until the token is redeemed.
func TestEmailVerificationPendingRegistration(t *testing.T) {
	app := newTestApp()
	verifyFlagOn(t, app)
	email := uniqueEmail("pending")
	const password = "secret123"

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Pending User","email":"`+email+`","password":"`+password+`"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	body := decodeBody(t, resp)
	assert.Equal(t, "verification_required", body["status"])
	_, hasUser := body["user"]
	assert.False(t, hasUser, "pending register must not echo a user")
	assert.Empty(t, resp.Cookies(), "pending register must not start a session")

	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	user, err := db.GetUserByEmail(email)
	require.NoError(t, err)
	uid := user.ID
	assert.Equal(t, models.UserStatusPending, user.UserStatus)

	var mails int64
	require.NoError(t, db.MailOutboxQueries.DB.Table("mail_outbox").Where("`to` = ?", email).Count(&mails).Error)
	assert.GreaterOrEqual(t, mails, int64(1), "a verification email must be queued")

	// Login is blocked while pending (after correct password).
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"`+email+`","password":"`+password+`"}`, nil)
	assert.Equal(t, 403, resp.StatusCode)
	assert.Contains(t, string(mustRead(t, resp)), "account is pending verification")

	// Mint the token row ourselves (only the hash is stored) and redeem it.
	raw := strings.Repeat("v", 64)
	require.NoError(t, db.CreateEmailVerification(uid, relay.HashKey(raw), time.Now().Add(time.Hour)))
	resp = doRequest(t, app, "POST", "/api/auth/verify-email", `{"token":"`+raw+`"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	verified := decodeBody(t, resp)
	assert.NotNil(t, verified["user"])
	assert.NotEmpty(t, resp.Cookies(), "verify must start a session")

	active, err := db.GetUserByID(uid)
	require.NoError(t, err)
	assert.Equal(t, models.UserStatusActive, active.UserStatus)

	// The token is single-use: the row is gone and a second click 400s.
	_, err = db.GetEmailVerification(relay.HashKey(raw))
	assert.Error(t, err)
	resp = doRequest(t, app, "POST", "/api/auth/verify-email", `{"token":"`+raw+`"}`, nil)
	assert.Equal(t, 400, resp.StatusCode)
	resp.Body.Close()
}

// TestEmailVerificationExpiredToken covers the expiry branch and its cleanup.
func TestEmailVerificationExpiredToken(t *testing.T) {
	app := newTestApp()
	verifyFlagOn(t, app)
	email := uniqueEmail("expired")
	const password = "secret123"

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Expired User","email":"`+email+`","password":"`+password+`"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)

	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	user, err := db.GetUserByEmail(email)
	require.NoError(t, err)

	raw := strings.Repeat("e", 64)
	require.NoError(t, db.CreateEmailVerification(user.ID, relay.HashKey(raw), time.Now().Add(-time.Hour)))
	resp = doRequest(t, app, "POST", "/api/auth/verify-email", `{"token":"`+raw+`"}`, nil)
	assert.Equal(t, 400, resp.StatusCode)
	assert.Contains(t, string(mustRead(t, resp)), "invalid or expired token")

	// The expired row is cleaned up.
	_, err = db.GetEmailVerification(relay.HashKey(raw))
	assert.Error(t, err)
}

// TestEmailVerificationUnknownToken is the generic-token case.
func TestEmailVerificationUnknownToken(t *testing.T) {
	app := newTestApp()
	verifyFlagOn(t, app)

	resp := doRequest(t, app, "POST", "/api/auth/verify-email", `{"token":"garbage"}`, nil)
	assert.Equal(t, 400, resp.StatusCode)
	assert.Contains(t, string(mustRead(t, resp)), "invalid or expired token")
}

// TestResendVerificationEnumerationSafe covers the always-202 resend: a pending
// account gets a fresh link, an active/unknown email gets nothing but the same
// generic response.
func TestResendVerificationEnumerationSafe(t *testing.T) {
	app := newTestApp()
	verifyFlagOn(t, app)
	email := uniqueEmail("resend")
	const password = "secret123"

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Resend User","email":"`+email+`","password":"`+password+`"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)

	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	user, err := db.GetUserByEmail(email)
	require.NoError(t, err)

	// Plant a known token, resend, and confirm the old hash is replaced.
	old := strings.Repeat("o", 64)
	require.NoError(t, db.CreateEmailVerification(user.ID, relay.HashKey(old), time.Now().Add(time.Hour)))

	var before int64
	require.NoError(t, db.MailOutboxQueries.DB.Table("mail_outbox").Where("`to` = ?", email).Count(&before).Error)

	resp = doRequest(t, app, "POST", "/api/auth/verify-email/resend", `{"email":"`+email+`"}`, nil)
	require.Equal(t, 202, resp.StatusCode)
	decodeBody(t, resp)

	_, err = db.GetEmailVerification(relay.HashKey(old))
	assert.Error(t, err, "old token must be invalidated by resend")

	var after int64
	require.NoError(t, db.MailOutboxQueries.DB.Table("mail_outbox").Where("`to` = ?", email).Count(&after).Error)
	assert.Greater(t, after, before, "a new verification email must be queued")

	// Active and unknown emails still get the generic 202 with no new mail.
	other := uniqueEmail("active")
	resp = doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Active User","email":"`+other+`","password":"`+password+`"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)
	otherUser, err := db.GetUserByEmail(other)
	require.NoError(t, err)
	require.NoError(t, db.UpdateUserStatus(otherUser.ID, models.UserStatusActive))

	var activeMailsBefore int64
	require.NoError(t, db.MailOutboxQueries.DB.Table("mail_outbox").Where("`to` = ?", other).Count(&activeMailsBefore).Error)
	resp = doRequest(t, app, "POST", "/api/auth/verify-email/resend", `{"email":"`+other+`"}`, nil)
	assert.Equal(t, 202, resp.StatusCode)
	resp.Body.Close()
	var activeMailsAfter int64
	require.NoError(t, db.MailOutboxQueries.DB.Table("mail_outbox").Where("`to` = ?", other).Count(&activeMailsAfter).Error)
	assert.Equal(t, activeMailsBefore, activeMailsAfter, "an active account gets no new link")

	resp = doRequest(t, app, "POST", "/api/auth/verify-email/resend", `{"email":"nobody@example.com"}`, nil)
	assert.Equal(t, 202, resp.StatusCode)
	resp.Body.Close()
}

// TestLoginBlockedAccountUnchanged keeps the blocked-account 403 message stable.
func TestLoginBlockedAccountUnchanged(t *testing.T) {
	app := newTestApp()
	verifyFlagOn(t, app)
	email := uniqueEmail("blocked")
	const password = "secret123"

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Blocked User","email":"`+email+`","password":"`+password+`"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	decodeBody(t, resp)

	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	user, err := db.GetUserByEmail(email)
	require.NoError(t, err)
	require.NoError(t, db.UpdateUserStatus(user.ID, models.UserStatusBlocked))

	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"`+email+`","password":"`+password+`"}`, nil)
	assert.Equal(t, 403, resp.StatusCode)
	assert.Contains(t, string(mustRead(t, resp)), "account is blocked")
}

// TestRegisterAutoLoginFlagOff is the legacy path: with the flag OFF register
// returns the user and starts a session (the shared suite's default).
func TestRegisterAutoLoginFlagOff(t *testing.T) {
	t.Setenv("REQUIRE_EMAIL_VERIFICATION", "false")
	t.Setenv("RATE_LIMIT_AUTH", "100")
	app := newTestApp()
	email := uniqueEmail("flagoff")

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Flag Off","email":"`+email+`","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	body := decodeBody(t, resp)
	assert.NotNil(t, body["user"])
	assert.NotEmpty(t, resp.Cookies(), "flag off must auto-login")
}
