package controllers

import "github.com/tertua/tupay/pkg/configs"

// appConfigResponse is the public branding configuration payload. Keys keep
// their historical camelCase spelling.
type appConfigResponse struct {
	AppName           string `json:"appName"`
	AllowRegistration bool   `json:"allowRegistration"`
	OIDCEnabled       bool   `json:"oidcEnabled"`
}

// newAppConfigResponse lifts the public branding config into its wire shape.
func newAppConfigResponse(cfg configs.Config) appConfigResponse {
	return appConfigResponse{
		AppName:           cfg.AppName,
		AllowRegistration: cfg.Auth.AllowRegistration,
		OIDCEnabled:       cfg.OIDC.Active(),
	}
}
