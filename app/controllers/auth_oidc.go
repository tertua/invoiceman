package controllers

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/configs"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"
	"golang.org/x/oauth2"
)

// oidcCallbackPath is the registered callback route (versioned API prefix) the
// provider must be told to redirect to; register it as the redirect URI.
const oidcCallbackPath = "/api/v1/auth/oidc/callback"

// oidcRedirectBase is the MPA login URL error codes land on.
const oidcRedirectBase = "/login"

// oidcErrorRedirect sends the browser back to the login page with a stable
// code query param (the only error surfaced to the user).
func oidcErrorRedirect(c fiber.Ctx, code string) error {
	return c.Redirect().Status(fiber.StatusFound).To(oidcRedirectBase + "?oidc_error=" + code)
}

// Provider discovery is cached per issuer: configs.Get() re-reads env on every
// call and tests swap OIDC_ISSUER (@t.Setenv), so the key must be the issuer.
var (
	oidcProviderMu    sync.RWMutex
	oidcProviderCache = map[string]*oidc.Provider{}
)

// oidcProviderFor returns the cached discovery provider for issuer, fetching
// and caching it on a miss. Errors are never cached, so the next attempt
// re-fetches.
func oidcProviderFor(ctx context.Context, issuer string) (*oidc.Provider, error) {
	oidcProviderMu.RLock()
	p := oidcProviderCache[issuer]
	oidcProviderMu.RUnlock()
	if p != nil {
		return p, nil
	}
	p, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}
	oidcProviderMu.Lock()
	oidcProviderCache[issuer] = p
	oidcProviderMu.Unlock()
	return p, nil
}

// resetOIDCProviderCache clears the discovery cache (test isolation).
func resetOIDCProviderCache() {
	oidcProviderMu.Lock()
	oidcProviderCache = map[string]*oidc.Provider{}
	oidcProviderMu.Unlock()
}

// OIDCLogin starts the SSO flow: it mints state+nonce+PKCE verifier, stores
// them under oidc:<state> and redirects to the provider's authorization URL.
// @Description Start the OIDC SSO login redirect.
// @Summary start OIDC SSO login
// @Tags Auth
// @Success 302 {string} string "redirect to the identity provider"
// @Router /auth/oidc/login [get]
func OIDCLogin(c fiber.Ctx) error {
	cfg := configs.Get().OIDC
	if !cfg.Active() {
		return oidcErrorRedirect(c, "disabled")
	}

	ctx := c.Context()
	provider, err := oidcProviderFor(ctx, cfg.Issuer)
	if err != nil {
		utils.RequestLogger(c).Warn("oidc discovery failed", "err", err)
		return oidcErrorRedirect(c, "provider")
	}

	state, err := randomHex(32)
	if err != nil {
		return oidcErrorRedirect(c, "provider")
	}
	nonce, err := randomHex(32)
	if err != nil {
		return oidcErrorRedirect(c, "provider")
	}
	verifier := oauth2.GenerateVerifier()
	if err := saveOIDCState(c.Context(), state, oidcState{Nonce: nonce, Verifier: verifier}); err != nil {
		return oidcErrorRedirect(c, "provider")
	}

	oauthCfg := oauth2Config(c, provider, cfg)
	url := oauthCfg.AuthCodeURL(state,
		oidc.Nonce(nonce),
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	return c.Redirect().Status(fiber.StatusFound).To(url)
}

// OIDCCallback finishes the SSO flow: it verifies the ID token (signature via
// JWKS, iss/aud/exp, nonce), resolves the local user and starts exactly the
// same session as a password Login.
// @Description Complete the OIDC SSO login and start a session.
// @Summary complete OIDC SSO login
// @Tags Auth
// @Param code query string false "Authorization code"
// @Param state query string false "Opaque state"
// @Param error query string false "Provider error code"
// @Success 302 {string} string "redirect to /dashboard or /login?oidc_error=..."
// @Router /auth/oidc/callback [get]
func OIDCCallback(c fiber.Ctx) error {
	cfg := configs.Get().OIDC
	if !cfg.Active() {
		return oidcErrorRedirect(c, "disabled")
	}
	// The provider may bounce back with an error instead of a code; never echo it.
	if c.Query("error") != "" {
		return oidcErrorRedirect(c, "provider")
	}

	state := c.Query("state")
	if state == "" {
		return oidcErrorRedirect(c, "state")
	}
	stored, ok, err := takeOIDCState(c.Context(), state)
	if err != nil {
		return oidcErrorRedirect(c, "busy")
	}
	if !ok {
		return oidcErrorRedirect(c, "state")
	}

	ctx := c.Context()
	provider, err := oidcProviderFor(ctx, cfg.Issuer)
	if err != nil {
		utils.RequestLogger(c).Warn("oidc discovery failed", "err", err)
		return oidcErrorRedirect(c, "provider")
	}

	oauthCfg := oauth2Config(c, provider, cfg)
	token, err := oauthCfg.Exchange(ctx, c.Query("code"), oauth2.VerifierOption(stored.Verifier))
	if err != nil {
		utils.RequestLogger(c).Warn("oidc token exchange failed", "err", err)
		return oidcErrorRedirect(c, "provider")
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return oidcErrorRedirect(c, "provider")
	}

	idToken, err := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}).Verify(ctx, rawIDToken)
	if err != nil {
		utils.RequestLogger(c).Warn("oidc id token verify failed", "err", err)
		return oidcErrorRedirect(c, "provider")
	}
	if idToken.Nonce == "" || idToken.Nonce != stored.Nonce {
		return oidcErrorRedirect(c, "state")
	}

	var claims struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil || claims.Sub == "" || claims.Email == "" {
		return oidcErrorRedirect(c, "email")
	}

	db, err := database.OpenDBConnection()
	if err != nil {
		return oidcErrorRedirect(c, "busy")
	}
	user, auditEvent, err := resolveOIDCUser(c, db, models.IdentityProviderOIDC, claims.Sub, claims.Email, claims.EmailVerified, claims.Name)
	if err != nil {
		return oidcErrorRedirect(c, oidcResolveCode(err))
	}
	// gateAccountStatus is a JSON-403 helper; the SSO flow is a browser
	// redirect, so suspended/pending accounts map to the stable denied code.
	// The audit reason still comes from the shared helper, so the SSO path
	// reports the same "pending"/"blocked" word as the password path.
	if !statusAllowed(user.UserStatus) {
		recordLoginFailure(c, db, user.Email, accountStatusReason(user.UserStatus), user.ID)
		return oidcErrorRedirect(c, "denied")
	}

	tokens, err := utils.IssueSession(c, user.ID, "")
	if err != nil {
		return oidcErrorRedirect(c, "busy")
	}
	csrf, err := issueCSRF(c)
	if err != nil {
		return oidcErrorRedirect(c, "busy")
	}
	if err := saveRefreshToken(c.Context(), user.ID, tokens.SID, tokens.Refresh, csrf, loginOrgHint(db, user.ID)); err != nil {
		return oidcErrorRedirect(c, "busy")
	}
	recordAudit(c, db, user.ID, auditEvent, "user", user.ID.String(), "")

	return c.Redirect().Status(fiber.StatusFound).To("/dashboard")
}

// oauth2Config builds the oauth2 client config for the provider endpoint.
func oauth2Config(c fiber.Ctx, provider *oidc.Provider, cfg configs.OIDCConfig) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  strings.TrimRight(c.BaseURL(), "/") + oidcCallbackPath,
		Scopes:       cfg.ScopeList(),
	}
}

// oidcResolveCode maps a resolve sentinel error to its redirect code.
func oidcResolveCode(err error) string {
	switch {
	case errors.Is(err, errOIDCEmailUnverified):
		return "email"
	case errors.Is(err, errOIDCDenied):
		return "denied"
	default:
		return "busy"
	}
}
