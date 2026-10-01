package controllers

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/configs"
	"github.com/tertua/tupay/pkg/repository"
	"github.com/tertua/tupay/platform/database"
)

// OIDC resolve outcomes that map to stable redirect codes (never rendered).
var (
	errOIDCEmailUnverified = errors.New("oidc: email not verified for auto-link")
	errOIDCDenied          = errors.New("oidc: registration is disabled")
	errOIDCBusy            = errors.New("oidc: concurrent account creation")
)

// resolveOIDCUser decides which local user an authenticated OIDC identity maps
// to, deterministically: existing identity -> auto-link by verified email ->
// auto-provision (gated by ALLOW_REGISTRATION). It returns the user plus the
// audit event to record on the successful path. err is one of the sentinel
// redirect codes above on rejection.
func resolveOIDCUser(c fiber.Ctx, db *database.Queries, provider, sub, email string, emailVerified bool, name string) (models.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	// 1. Known identity: straight login.
	if identity, err := db.GetByIdentity(provider, sub); err == nil {
		user, err := db.GetUserByID(identity.UserID)
		if err != nil {
			return models.User{}, "", errOIDCBusy
		}
		return user, "auth.login.oidc", nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return models.User{}, "", errOIDCBusy
	}

	// 2. Existing account with this email.
	if user, err := db.GetUserByEmail(email); err == nil {
		if !emailVerified {
			// Email matches but the IdP has not proven mailbox ownership:
			// linking here would allow account takeover via pre-registration.
			return models.User{}, "", errOIDCEmailUnverified
		}
		if err := db.LinkIdentity(user.ID, provider, sub, email); err != nil {
			return models.User{}, "", errOIDCBusy
		}
		return user, "auth.identity.linked", nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return models.User{}, "", errOIDCBusy
	}

	// 3. New account: provision only when registration is open.
	if !configs.Get().Auth.AllowRegistration {
		return models.User{}, "", errOIDCDenied
	}
	return provisionOIDCUser(c, db, provider, sub, email, name)
}

// provisionOIDCUser creates a password-less account plus its identity and
// personal org (same tenant path as register). PasswordHash stays empty:
// ComparePasswords("", x) is always false, so an OIDC-only account can never
// log in with a password.
func provisionOIDCUser(c fiber.Ctx, db *database.Queries, provider, sub, email, name string) (models.User, string, error) {
	if strings.TrimSpace(name) == "" {
		name = email
	}
	user := &models.User{
		ID:           uuid.New(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Name:         name,
		Email:        email,
		PasswordHash: "",
		UserStatus:   models.UserStatusActive,
		UserRole:     repository.UserRoleName,
	}
	if err := db.CreateUser(user); err != nil {
		return models.User{}, "", errOIDCBusy
	}
	if err := db.CreateIdentity(&models.UserIdentity{
		ID:        uuid.New(),
		UserID:    user.ID,
		Provider:  provider,
		Sub:       sub,
		Email:     email,
		CreatedAt: time.Now(),
	}); err != nil {
		return models.User{}, "", errOIDCBusy
	}
	orgID, err := db.EnsurePersonalOrg(user.ID)
	if err != nil {
		return models.User{}, "", errOIDCBusy
	}
	if err := db.CreateSettings(models.DefaultSettings(orgID)); err != nil {
		return models.User{}, "", errOIDCBusy
	}
	return *user, "auth.register.oidc", nil
}
