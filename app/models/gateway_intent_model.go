package models

// IntentInput is the service-to-service payload for creating a payment.
// Project identity comes from the API key header, not from this body.
// PaymentMethod is the preferred, provider-neutral selector (see
// gateway.IDs); Gateway is an optional explicit provider override kept for
// backward compatibility and defaults to the project's default_gateway.
// Fiat flows use amount_idr; fractional/crypto flows use amount_decimal
// with currency (e.g. "0.0005", "BTC").
type IntentInput struct {
	ExternalOrderID string `json:"external_order_id" validate:"required,lte=128"`
	Gateway         string `json:"gateway" validate:"omitempty,lte=32"`
	// Keep the oneof list in sync with gateway.IDs(); the enum test guards it.
	PaymentMethod string `json:"payment_method" validate:"omitempty,lte=32,oneof=bank_transfer qris gopay credit_card crypto"`
	AmountIDR     int64  `json:"amount_idr" validate:"gte=0"`
	AmountDecimal string `json:"amount_decimal" validate:"omitempty,lte=64"`
	Currency      string `json:"currency" validate:"omitempty,lte=8"`
	CustomerEmail string `json:"customer_email" validate:"omitempty,email,lte=255"`
	CustomerPhone string `json:"customer_phone" validate:"omitempty,lte=64"`
}
