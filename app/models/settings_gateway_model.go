package models

// SettingsGatewayConversion holds the manual FX rate applied to online gateway
// charges. It is embedded in Settings so the base settings model stays small.
type SettingsGatewayConversion struct {
	// UsdToIdr is IDR per 1 USD. Zero disables conversion; the rate is a fixed
	// owner-set value, never fetched from a realtime feed.
	UsdToIdr Money `gorm:"type:decimal(19,4)" db:"usd_to_idr" json:"usd_to_idr"`
}
