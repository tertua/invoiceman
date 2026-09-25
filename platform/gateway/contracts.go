package gateway

type CreateTxRequest struct {
	OrderID        string
	AmountMinor    int64
	AmountDecimal  string
	Currency       string
	Email          string
	Phone          string
	PaymentMethod  string
	EnabledMethods []string
	PayCurrency    string
}

type CreateTxResponse struct {
	Token         string
	RedirectURL   string
	PaymentURL    string
	Address       string
	ExpiresAt     string
	PaymentMethod string
	RawPayload    string
	PayCurrency   string
}

type NotificationResult struct {
	OrderID       string
	TransactionID string
	Status        string
	// PaymentType is the provider's raw payment type (e.g. Midtrans "bni",
	// NOWPayments "btc"); kept for downstream compatibility.
	PaymentType string
	// PaymentMethod is the provider-neutral method (one of the Method*
	// constants) derived from the raw type.
	PaymentMethod string
	GrossMinor    int64
	GrossDecimal  string
	Currency      string
}

type PaymentMethodProvider interface {
	Methods() []string
}

// ConfiguredProvider is an optional capability: providers that can tell
// whether their credentials are present report it so endpoints can hide
// unusable methods.
type ConfiguredProvider interface {
	Configured() bool
}
