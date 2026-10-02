package controllers

import (
	"context"
	"encoding/json"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/configs"
	"github.com/tertua/tupay/pkg/utils"
	"golang.org/x/oauth2"
)

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
	oidcClaims, allClaimsJSON, redirectCode := verifyOIDCIDToken(c, ctx, provider, cfg, token, stored.Nonce)
	if redirectCode != "" {
		return oidcErrorRedirect(c, redirectCode)
	}

	db, ok := openDB(c)
	if !ok {
		return oidcErrorRedirect(c, "busy")
	}
	user, auditEvent, err := resolveOIDCUser(c, db, models.IdentityProviderOIDC, oidcClaims.Sub, oidcClaims.Email, oidcClaims.EmailVerified, oidcClaims.Name)
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
	// Sync after the status gate: a denied account must not change its role.
	// The role is read from the DB per request, so this write is enough for the
	// fresh session to see the mapped role (no token re-issue needed).
	syncOIDCRole(c, db, &user, allClaimsJSON)

	if _, err := startSession(c, db, user.ID, loginOrgHint(db, user.ID)); err != nil {
		return oidcErrorRedirect(c, "busy")
	}
	recordAudit(c, db, user.ID, auditEvent, "user", user.ID.String(), "")

	return c.Redirect().Status(fiber.StatusFound).To("/dashboard")
}

// verifyOIDCIDToken runs the full provider-side verification of the ID token
// (signature, claims, nonce) and returns both the typed claims we use directly
// and the full payload as JSON for the configurable role-claim path. It writes
// the right redirect code on any failure so callers can simply return it.
func verifyOIDCIDToken(c fiber.Ctx, ctx context.Context, provider *oidc.Provider, cfg configs.OIDCConfig, token *oauth2.Token, storedNonce string) (claims struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
}, allClaimsJSON []byte, redirect string) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return claims, nil, "provider"
	}
	idToken, err := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}).Verify(ctx, rawIDToken)
	if err != nil {
		utils.RequestLogger(c).Warn("oidc id token verify failed", "err", err)
		return claims, nil, "provider"
	}
	if idToken.Nonce == "" || idToken.Nonce != storedNonce {
		return claims, nil, "state"
	}
	if err := idToken.Claims(&claims); err != nil || claims.Sub == "" || claims.Email == "" {
		return claims, nil, "email"
	}
	// Decode the full verified payload once more so the configurable role path
	// (OIDC_ROLE_CLAIM) can be walked; same source, same verification guarantees.
	var allClaims map[string]any
	_ = idToken.Claims(&allClaims)
	allClaimsJSON, _ = json.Marshal(allClaims)
	return claims, allClaimsJSON, ""
}
