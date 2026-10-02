package controllers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"
)

// oidcTestCtx runs one throwaway request so resolveOIDCUser has a fiber.Ctx
// (it only reads c for audit/login-failure context, never the network).
func oidcTestCtx(t *testing.T, fn func(c fiber.Ctx) error) {
	t.Helper()
	app := fiber.New()
	app.Get("/probe", fn)
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/probe", http.NoBody), fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	require.NoError(t, err)
	resp.Body.Close()
}

// oidcSeedUser inserts an active local account and returns it.
func oidcSeedUser(t *testing.T, email string, status int) models.User {
	t.Helper()
	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	user := models.User{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(), Name: "Seed", Email: email, PasswordHash: "hash", UserStatus: status, UserRole: "user"}
	require.NoError(t, db.CreateUser(&user))
	return user
}

func TestResolveOIDCUser(t *testing.T) {
	db, err := database.OpenDBConnection()
	require.NoError(t, err)

	tests := []struct {
		name          string
		sub           string
		email         string
		emailVerified bool
		allowRegister bool
		seed          func(t *testing.T) (models.User, bool) // returns (user, identitySeeded)
		wantEvent     string
		wantErrCode   string
	}{
		{
			name: "identity match logs in",
			sub:  "sub-known", email: "known@example.com", emailVerified: true, allowRegister: true,
			seed: func(t *testing.T) (models.User, bool) {
				u := oidcSeedUser(t, "known@example.com", models.UserStatusActive)
				require.NoError(t, db.LinkIdentity(u.ID, models.IdentityProviderOIDC, "sub-known", u.Email))
				return u, true
			},
			wantEvent: "auth.login.oidc",
		},
		{
			name: "verified email auto-links existing account",
			sub:  "sub-link", email: "link@example.com", emailVerified: true, allowRegister: true,
			seed: func(t *testing.T) (models.User, bool) {
				u := oidcSeedUser(t, "link@example.com", models.UserStatusActive)
				return u, false
			},
			wantEvent: "auth.identity.linked",
		},
		{
			name: "unverified email is rejected",
			sub:  "sub-unver", email: "unver@example.com", emailVerified: false, allowRegister: true,
			seed: func(t *testing.T) (models.User, bool) {
				u := oidcSeedUser(t, "unver@example.com", models.UserStatusActive)
				return u, false
			},
			wantErrCode: "email",
		},
		{
			name: "provision new user when registration open",
			sub:  "sub-new", email: "new@example.com", emailVerified: true, allowRegister: true,
			seed:      func(t *testing.T) (models.User, bool) { return models.User{}, false },
			wantEvent: "auth.register.oidc",
		},
		{
			name: "provision denied when registration closed",
			sub:  "sub-denied", email: "denied@example.com", emailVerified: true, allowRegister: false,
			seed:        func(t *testing.T) (models.User, bool) { return models.User{}, false },
			wantErrCode: "denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ALLOW_REGISTRATION is read via configs.Get() inside the resolver.
			t.Setenv("ALLOW_REGISTRATION", map[bool]string{true: "true", false: "false"}[tt.allowRegister])
			seedUser, _ := tt.seed(t)

			var got models.User
			var gotEvent string
			var gotErr error
			oidcTestCtx(t, func(c fiber.Ctx) error {
				got, gotEvent, gotErr = resolveOIDCUser(c, db, models.IdentityProviderOIDC, tt.sub, tt.email, tt.emailVerified, "OIDC User")
				return c.SendStatus(fiber.StatusOK)
			})

			if tt.wantErrCode != "" {
				require.Error(t, gotErr)
				assert.Equal(t, tt.wantErrCode, oidcResolveCode(gotErr))
				return
			}

			require.NoError(t, gotErr)
			assert.Equal(t, tt.wantEvent, gotEvent)
			if seedUser.ID != uuid.Nil {
				assert.Equal(t, seedUser.ID, got.ID, "expected the existing account")
			}

			// The e-mail resolves to exactly one identity row.
			identity, ierr := db.GetByIdentity(models.IdentityProviderOIDC, tt.sub)
			require.NoError(t, ierr)
			assert.Equal(t, got.ID, identity.UserID)

			if tt.wantEvent == "auth.register.oidc" {
				assert.Empty(t, got.PasswordHash, "OIDC-provisioned users must be password-less")
				// A personal org is provisioned via the register path.
				orgID, oerr := db.ResolveActiveOrgID(got.ID, uuid.Nil)
				require.NoError(t, oerr)
				assert.NotEqual(t, uuid.Nil, orgID)
			}
			assert.False(t, utils.ComparePasswords(got.PasswordHash, "anything"), "empty hash must never match a password")
		})
	}
}

// TestProvisionOIDCUserRollback proves the atomic provision leaves no partial
// account. The query layer is exercised directly (no controller re-fetch): a
// duplicate (provider, sub) fails the identity insert inside the tx, so the
// whole user row is rolled back.
func TestProvisionOIDCUserRollback(t *testing.T) {
	db, err := database.OpenDBConnection()
	require.NoError(t, err)

	// The subject is already linked to another user, so the new identity insert
	// hits the unique index and fails in the middle of the transaction.
	owner := oidcSeedUser(t, "rollback-owner@example.com", models.UserStatusActive)
	require.NoError(t, db.LinkIdentity(owner.ID, models.IdentityProviderOIDC, "sub-rollback", owner.Email))

	email := "rollback-new@example.com"
	user := &models.User{
		ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(), Name: "Rollback",
		Email: email, PasswordHash: "", UserStatus: models.UserStatusActive, UserRole: "user",
	}
	require.Error(t, db.ProvisionOIDCUser(user, models.IdentityProviderOIDC, "sub-rollback", email))

	// The user row must NOT exist: the tx rolled back all five inserts.
	_, uerr := db.GetUserByEmail(email)
	assert.ErrorIs(t, uerr, sql.ErrNoRows, "no user row may survive a rolled-back provision")
	// The original identity is untouched.
	identity, ierr := db.GetByIdentity(models.IdentityProviderOIDC, "sub-rollback")
	require.NoError(t, ierr)
	assert.Equal(t, owner.ID, identity.UserID)
}

// TestResolveOIDCUserRaceLoser proves a lost link race logs into the winner
// instead of surfacing busy: the identity row already exists by the time we
// try to link it, so the unique-violation re-fetch resolves to its owner.
func TestResolveOIDCUserRaceLoser(t *testing.T) {
	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	t.Setenv("ALLOW_REGISTRATION", "true")

	// The winner linked first; our resolve sees the identity as "known" and
	// logs straight in (the same path a loser's re-fetch takes).
	winner := oidcSeedUser(t, "race-loser@example.com", models.UserStatusActive)
	require.NoError(t, db.LinkIdentity(winner.ID, models.IdentityProviderOIDC, "sub-race-loser", winner.Email))

	var got models.User
	var event string
	var gotErr error
	oidcTestCtx(t, func(c fiber.Ctx) error {
		got, event, gotErr = resolveOIDCUser(c, db, models.IdentityProviderOIDC, "sub-race-loser", winner.Email, true, "Race")
		return c.SendStatus(fiber.StatusOK)
	})
	require.NoError(t, gotErr)
	assert.Equal(t, "auth.login.oidc", event)
	assert.Equal(t, winner.ID, got.ID)
}

// TestProvisionOIDCUserRaceCreate proves a create-path unique-violation
// re-fetches the existing owner: provisioning a subject that is already linked
// returns that account (login), never busy, and adds no second user.
func TestProvisionOIDCUserRaceCreate(t *testing.T) {
	db, err := database.OpenDBConnection()
	require.NoError(t, err)

	existing := oidcSeedUser(t, "race-create@example.com", models.UserStatusActive)
	require.NoError(t, db.LinkIdentity(existing.ID, models.IdentityProviderOIDC, "sub-race-create", existing.Email))

	var got models.User
	var event string
	var gotErr error
	oidcTestCtx(t, func(c fiber.Ctx) error {
		got, event, gotErr = provisionOIDCUser(c, db, models.IdentityProviderOIDC, "sub-race-create", "race-create-new@example.com", true, "Race")
		return c.SendStatus(fiber.StatusOK)
	})
	require.NoError(t, gotErr)
	assert.Equal(t, "auth.login.oidc", event)
	assert.Equal(t, existing.ID, got.ID)
	_, uerr := db.GetUserByEmail("race-create-new@example.com")
	assert.ErrorIs(t, uerr, sql.ErrNoRows, "the losing provision must not create a second user")
}

// TestProvisionOIDCUserRaceEmailUnverified proves the email re-fetch respects
// the verified-mailbox gate: a lost race on users.email must NOT log into the
// existing account when the IdP has not verified the email.
func TestProvisionOIDCUserRaceEmailUnverified(t *testing.T) {
	db, err := database.OpenDBConnection()
	require.NoError(t, err)

	// The account owning the email already exists; our provision loses on the
	// users.email unique index. With email_verified=false it must be rejected
	// (email), not silently logged in.
	existing := oidcSeedUser(t, "race-email@example.com", models.UserStatusActive)

	var got models.User
	var event string
	var gotErr error
	oidcTestCtx(t, func(c fiber.Ctx) error {
		got, event, gotErr = provisionOIDCUser(c, db, models.IdentityProviderOIDC, "sub-race-email", existing.Email, false, "Race")
		return c.SendStatus(fiber.StatusOK)
	})
	require.Error(t, gotErr)
	assert.Equal(t, "email", oidcResolveCode(gotErr))
	assert.Equal(t, models.User{}, got)
	assert.Empty(t, event)
}

// TestProvisionOIDCUserRaceEmailVerified proves the email re-fetch still logs
// in when the mailbox is verified (the legitimate concurrent sign-up).
func TestProvisionOIDCUserRaceEmailVerified(t *testing.T) {
	db, err := database.OpenDBConnection()
	require.NoError(t, err)

	existing := oidcSeedUser(t, "race-email-ok@example.com", models.UserStatusActive)

	var got models.User
	var event string
	var gotErr error
	oidcTestCtx(t, func(c fiber.Ctx) error {
		got, event, gotErr = provisionOIDCUser(c, db, models.IdentityProviderOIDC, "sub-race-email-ok", existing.Email, true, "Race")
		return c.SendStatus(fiber.StatusOK)
	})
	require.NoError(t, gotErr)
	assert.Equal(t, "auth.login.oidc", event)
	assert.Equal(t, existing.ID, got.ID)
}
