package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/cache"
	"github.com/tertua/tupay/platform/database"
	"gorm.io/gorm"
)

// TestMain runs the middleware tests against in-memory SQLite with an in-memory session store, mirroring the route test setup.
func TestMain(m *testing.M) {
	os.Setenv("STAGE_STATUS", "dev")
	os.Setenv("JWT_SECRET_KEY", "test-secret")
	os.Setenv("JWT_SECRET_KEY_EXPIRE_MINUTES_COUNT", "15")
	os.Setenv("JWT_REFRESH_KEY", "test-refresh")
	os.Setenv("JWT_REFRESH_KEY_EXPIRE_HOURS_COUNT", "720")
	os.Setenv("SQL_DSN", "")
	os.Setenv("SQLITE_PATH", "file::memory:?cache=shared")
	os.Setenv("REDIS_HOST", "")
	if err := database.Migrate(); err != nil {
		panic("test migrate failed: " + err.Error())
	}
	ensureOrgTables()
	os.Exit(m.Run())
}

// ensureOrgTables creates the org tables on the shared in-memory database; idempotent, and a no-op once the v16 migrate includes them.
func ensureOrgTables() {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("org table setup failed: " + err.Error())
	}
	if err := db.AutoMigrate(&models.Organization{}, &models.Membership{}); err != nil {
		panic("org table setup failed: " + err.Error())
	}
}

// seedUser inserts a plain user row and returns its id.
func seedUser(t *testing.T, email string) uuid.UUID {
	t.Helper()
	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	user := models.User{ID: uuid.New(), Name: "Org Tester", Email: email, PasswordHash: "hash", UserStatus: 1, UserRole: "user"}
	require.NoError(t, db.CreateUser(&user))
	return user.ID
}

// seedOrg creates an org owned by the user and returns its id.
func seedOrg(t *testing.T, userID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	org, err := db.CreateOrg(name)
	require.NoError(t, err)
	require.NoError(t, db.CreateOwner(org.ID, userID))
	return org.ID
}

// orgTestApp wires the middleware under test behind a stub that mimics AuthRequired locals.
func orgTestApp(t *testing.T, userID uuid.UUID) *fiber.App {
	t.Helper()
	withUser := func(c fiber.Ctx) error {
		c.Locals(utils.SessionUserIDKey, userID)
		return c.Next()
	}
	withRole := func(role string) fiber.Handler {
		return func(c fiber.Ctx) error {
			c.Locals(utils.SessionOrgRoleKey, role)
			return c.Next()
		}
	}
	echo := func(c fiber.Ctx) error {
		orgID, _ := utils.CurrentOrgID(c)
		role, _ := utils.CurrentOrgRole(c)
		return utils.OK(c, fiber.StatusOK, fiber.Map{"orgID": orgID.String(), "role": role})
	}
	ok := func(c fiber.Ctx) error {
		return utils.OK(c, fiber.StatusOK, fiber.Map{"ok": true})
	}
	app := fiber.New()
	app.Get("/ctx", withUser, OrgContext(), echo)
	app.Get("/owner", withUser, OrgContext(), RequireOrgRole("owner"), ok)
	app.Get("/staff-guard", withRole(models.RoleStaff), RequireOrgRole("owner"), ok)
	app.Get("/owner-guard", withRole(models.RoleOwner), RequireOrgRole("owner"), ok)
	app.Get("/owner-or-staff", withRole(models.RoleStaff), RequireOrgRole(models.RoleOwner, models.RoleStaff), ok)
	return app
}

// orgGet performs a GET and returns the status plus decoded JSON body.
func orgGet(t *testing.T, app *fiber.App, route string) (status int, body map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, route, http.NoBody)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	require.NoError(t, err)
	defer resp.Body.Close()
	body = map[string]any{}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return resp.StatusCode, body
}

// errorMessage pulls error.message out of a utils.Fail envelope.
func errorMessage(body map[string]any) string {
	errObj, _ := body["error"].(map[string]any)
	message, _ := errObj["message"].(string)
	return message
}

// TestOrgContextAutoProvisionsPersonalOrg proves the D3 fallback: a user with no membership resolves to a freshly provisioned personal org as owner.
func TestOrgContextAutoProvisionsPersonalOrg(t *testing.T) {
	userID := seedUser(t, "org-ctx-new@example.com")
	status, body := orgGet(t, orgTestApp(t, userID), "/ctx")
	require.Equal(t, fiber.StatusOK, status, body)

	orgID, err := uuid.Parse(body["orgID"].(string))
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, orgID)
	assert.Equal(t, models.RoleOwner, body["role"])

	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	role, err := db.GetRole(orgID, userID)
	require.NoError(t, err)
	assert.Equal(t, models.RoleOwner, role)
}

// TestOrgContextHonorsSessionActiveOrgHint proves the D2 path: a session pointing at a second membership wins over the oldest one.
func TestOrgContextHonorsSessionActiveOrgHint(t *testing.T) {
	userID := seedUser(t, "org-ctx-hint@example.com")
	oldest := seedOrg(t, userID, "Hint Org A")
	hinted := seedOrg(t, userID, "Hint Org B")
	require.NoError(t, cache.WriteSessionValue(context.Background(), userID.String(), cache.SessionValue{SID: "sid", Refresh: "refresh", CSRF: "csrf", ActiveOrgID: hinted.String()}))

	status, body := orgGet(t, orgTestApp(t, userID), "/ctx")
	require.Equal(t, fiber.StatusOK, status, body)
	assert.Equal(t, hinted.String(), body["orgID"])
	assert.NotEqual(t, oldest.String(), body["orgID"])
}

// TestOrgContextWritesResolvedOrgBackToSession proves the best-effort write-back keeps sid/refresh/csrf intact.
func TestOrgContextWritesResolvedOrgBackToSession(t *testing.T) {
	userID := seedUser(t, "org-ctx-writeback@example.com")
	orgID := seedOrg(t, userID, "Writeback Org")
	require.NoError(t, cache.WriteSessionValue(context.Background(), userID.String(), cache.SessionValue{SID: "sid", Refresh: "refresh", CSRF: "csrf"}))

	status, _ := orgGet(t, orgTestApp(t, userID), "/ctx")
	require.Equal(t, fiber.StatusOK, status)

	sv, ok, err := cache.ReadSessionValue(context.Background(), userID.String())
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, orgID.String(), sv.ActiveOrgID)
	assert.Equal(t, "sid", sv.SID)
	assert.Equal(t, "refresh", sv.Refresh)
	assert.Equal(t, "csrf", sv.CSRF)
}

// TestOrgContextRejectsUnknownUser proves the defensive 403 when resolution yields no membership.
func TestOrgContextRejectsUnknownUser(t *testing.T) {
	status, body := orgGet(t, orgTestApp(t, uuid.New()), "/ctx")
	assert.Equal(t, fiber.StatusForbidden, status)
	assert.Equal(t, "org.notMember", errorMessage(body))
}

// TestRequireOrgRoleForbidsStaff proves a staff role cannot pass an owner guard (403 org.ownerRequired).
func TestRequireOrgRoleForbidsStaff(t *testing.T) {
	status, body := orgGet(t, orgTestApp(t, uuid.New()), "/staff-guard")
	assert.Equal(t, fiber.StatusForbidden, status)
	assert.Equal(t, "org.ownerRequired", errorMessage(body))
}

// TestRequireOrgRoleAllowsOwner proves the owner role passes the same guard, both standalone and chained after OrgContext.
func TestRequireOrgRoleAllowsOwner(t *testing.T) {
	userID := seedUser(t, "org-ctx-owner@example.com")
	seedOrg(t, userID, "Owner Org")
	app := orgTestApp(t, userID)

	status, _ := orgGet(t, app, "/owner-guard")
	assert.Equal(t, fiber.StatusOK, status)

	status, _ = orgGet(t, app, "/owner")
	assert.Equal(t, fiber.StatusOK, status)
}

// TestRequireOrgRoleMultiRoleAllowsStaff proves the variadic guard admits any listed role: staff passes "owner","staff" while the single-role owner guard still refuses it.
func TestRequireOrgRoleMultiRoleAllowsStaff(t *testing.T) {
	app := orgTestApp(t, uuid.New())

	status, _ := orgGet(t, app, "/owner-or-staff")
	assert.Equal(t, fiber.StatusOK, status)

	status, body := orgGet(t, app, "/staff-guard")
	assert.Equal(t, fiber.StatusForbidden, status)
	assert.Equal(t, "org.ownerRequired", errorMessage(body))
}
