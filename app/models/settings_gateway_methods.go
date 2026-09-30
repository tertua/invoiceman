package models

// SettingsGatewayMethods keeps the legacy single-provider method column for
// backward compatibility: midtrans_methods is read as the allowlist for
// provider "midtrans". The default provider's account only offers QRIS, so the
// resolved allowlist is QRIS-only (declared by platform/midtrans.DefaultMethods,
// never by this stored value). A future provider gets its own column when it
// needs per-owner narrowing; until then it is unlisted (means "all methods").
type SettingsGatewayMethods struct {
	// MidtransMethods is the legacy single-provider method id column. For the
	// default provider it is written as "qris" and its stored value is ignored
	// when resolving the allowlist.
	MidtransMethods string `gorm:"size:255" db:"midtrans_methods" json:"midtrans_methods"`
}
