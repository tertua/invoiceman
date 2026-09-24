package relay

// Payload is the normalized v1 webhook body forwarded to downstream projects.
// Gateway identifies the provider; fiat flows use gross_amount_idr (fiat minor
// units) while fractional flows use amount_decimal + currency. PaymentType
// stays provider-specific for compatibility; PaymentMethod carries the
// provider-neutral method (one of gateway.Method*).
type Payload struct {
	EventID         string `json:"event_id"`
	OrderID         string `json:"order_id"`
	ProjectSlug     string `json:"project_slug"`
	Gateway         string `json:"gateway"`
	ExternalOrderID string `json:"external_order_id"`
	Status          string `json:"status"`
	GrossAmountIDR  int64  `json:"gross_amount_idr"`
	AmountDecimal   string `json:"amount_decimal"`
	Currency        string `json:"currency"`
	TransactionID   string `json:"transaction_id"`
	PaymentType     string `json:"payment_type"`
	PaymentMethod   string `json:"payment_method"`
	PaidAt          string `json:"paid_at"`
}
