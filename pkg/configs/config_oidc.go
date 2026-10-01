package configs

import (
	"fmt"
	"os"
	"strings"
)

// OIDCConfig holds the single external OIDC provider used for "Login with SSO".
// It is split out of config.go (file-size ratchet), following the
// config_gateway.go pattern: the struct, accessors, parsing and validation
// stay here while Config only carries the OIDC field.
type OIDCConfig struct {
	Enabled      bool   // OIDC_ENABLED (default false)
	Issuer       string // OIDC_ISSUER: discovery base URL
	ClientID     string // OIDC_CLIENT_ID
	ClientSecret string // OIDC_CLIENT_SECRET
	Scopes       string // OIDC_SCOPES (default "openid email profile")
}

// Active reports whether SSO login is usable: enabled with the mandatory
// issuer and client id present. Scopes always has the openid scope by default.
func (o OIDCConfig) Active() bool {
	return o.Enabled && o.Issuer != "" && o.ClientID != ""
}

// ScopeList splits Scopes into the OAuth scope slice (blanks dropped).
func (o OIDCConfig) ScopeList() []string {
	var out []string
	for _, s := range strings.Fields(o.Scopes) {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// Validate fail-fasts when SSO is switched on without the fields it needs;
// a disabled block (the default) validates unconditionally.
func (o OIDCConfig) Validate() error {
	if !o.Enabled {
		return nil
	}
	var missing []string
	if o.Issuer == "" {
		missing = append(missing, "OIDC_ISSUER")
	}
	if o.ClientID == "" {
		missing = append(missing, "OIDC_CLIENT_ID")
	}
	if o.ClientSecret == "" {
		missing = append(missing, "OIDC_CLIENT_SECRET")
	}
	if len(missing) > 0 {
		return fmt.Errorf("OIDC_ENABLED=true requires %s", strings.Join(missing, " and "))
	}
	return nil
}

// loadOIDC parses the OIDC env block (defaults keep SSO off).
func loadOIDC() OIDCConfig {
	return OIDCConfig{
		Enabled:      envBool("OIDC_ENABLED", false),
		Issuer:       strings.TrimSpace(os.Getenv("OIDC_ISSUER")),
		ClientID:     strings.TrimSpace(os.Getenv("OIDC_CLIENT_ID")),
		ClientSecret: strings.TrimSpace(os.Getenv("OIDC_CLIENT_SECRET")),
		Scopes:       envOr("OIDC_SCOPES", "openid email profile"),
	}
}
