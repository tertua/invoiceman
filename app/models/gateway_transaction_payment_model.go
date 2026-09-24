package models

// GatewayTransactionPayment is the provider-neutral payment method chosen for
// a gateway transaction, embedded to keep the transaction model under budget.
type GatewayTransactionPayment struct {
	PaymentMethod string `gorm:"size:32;index" db:"payment_method" json:"payment_method"`
}
