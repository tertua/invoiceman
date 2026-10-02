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
			// Unproven mailbox: linking here would allow takeover via pre-registration.
			return models.User{}, "", errOIDCEmailUnverified
		}
		if err := db.LinkIdentity(user.ID, provider, sub, email); err != nil {
			// Lost the link race: the row owner already claimed this exact subject.
			if isUniqueViolation(err) {
				if winner, ok := oidcIdentityWinner(db, provider, sub); ok {
					return winner, "auth.login.oidc", nil
				}
			}
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
	return provisionOIDCUser(c, db, provider, sub, email, emailVerified, name)
}

// provisionOIDCUser creates a password-less account plus its identity and
// personal org (same tenant path as register). PasswordHash stays empty:
// ComparePasswords("", x) is always false, so an OIDC-only account can never
// log in with a password.
func provisionOIDCUser(c fiber.Ctx, db *database.Queries, provider, sub, email string, emailVerified bool, name string) (models.User, string, error) {
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
	if err := db.ProvisionOIDCUser(user, provider, sub, email); err != nil {
		if isUniqueViolation(err) {
			// Same subject linked first: the row's owner is this same login.
			if winner, ok := oidcIdentityWinner(db, provider, sub); ok {
				return winner, "auth.login.oidc", nil
			}
			// Email inserted first: only a verified mailbox may claim that account.
			if !emailVerified {
				return models.User{}, "", errOIDCEmailUnverified
			}
			if winner, ok := oidcEmailWinner(db, email); ok {
				return winner, "auth.login.oidc", nil
			}
		}
		return models.User{}, "", errOIDCBusy
	}
	return *user, "auth.register.oidc", nil
}

// oidcIdentityWinner resolves the user that owns a (provider, sub) identity,
// used when our insert lost the unique race on that key.
func oidcIdentityWinner(db *database.Queries, provider, sub string) (models.User, bool) {
	identity, err := db.GetByIdentity(provider, sub)
	if err != nil {
		return models.User{}, false
	}
	user, err := db.GetUserByID(identity.UserID)
	if err != nil {
		return models.User{}, false
	}
	return user, true
}

// oidcEmailWinner resolves the account owning an email that won a provision
// race (oldest account wins, matching GetUserByEmail's deterministic ordering).
func oidcEmailWinner(db *database.Queries, email string) (models.User, bool) {
	user, err := db.GetUserByEmail(email)
	if err != nil {
		return models.User{}, false
	}
	return user, true
}
