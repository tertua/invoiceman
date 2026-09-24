package models

// SettingsGatewayMethods holds the per-account Midtrans payment method shown
// on payment links. It stores a single gateway method id; legacy empty or
// multi-method values normalize to gopay.
type SettingsGatewayMethods struct {
	// MidtransMethods is a single gateway method id (e.g. gopay).
	MidtransMethods string `gorm:"size:255" db:"midtrans_methods" json:"midtrans_methods"`
}
