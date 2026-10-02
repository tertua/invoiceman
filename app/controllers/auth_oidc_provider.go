package controllers

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/pkg/configs"

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
