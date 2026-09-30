package models

// SettingsGatewayMethods keeps the single-provider method column for backward
// compatibility: provider_methods (renamed from midtrans_methods in v17) is read
// as the allowlist for the default provider. The default provider's account only
// offers QRIS, so the resolved allowlist is QRIS-only (declared by
// platform/midtrans.DefaultMethods, never by this stored value). A future
// provider gets its own column when it needs per-owner narrowing; until then it
// is unlisted (means "all methods").
type SettingsGatewayMethods struct {
	// ProviderMethods is the single-provider method id column, written as "qris"
	// for the default provider. Its stored value is ignored when resolving the
	// allowlist. The response emits both provider_methods (this key) and the
	// deprecated midtrans_methods alias — see settingsResponse in the controller.
	ProviderMethods string `gorm:"size:255" db:"provider_methods" json:"provider_methods"`
}
