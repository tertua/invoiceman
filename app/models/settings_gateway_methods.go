package models

// SettingsGatewayMethods holds the per-account allowlist of provider-neutral
// payment methods exposed through Midtrans. Empty means "all Midtrans
// methods": an owner that never touched the setting keeps the previous
// behaviour.
type SettingsGatewayMethods struct {
	// MidtransMethods is a comma-separated list of gateway method ids. Only
	// ids Midtrans actually supports can be stored; the UI uses an empty
	// value to mean "all".
	MidtransMethods string `gorm:"size:255" db:"midtrans_methods" json:"midtrans_methods"`
}
