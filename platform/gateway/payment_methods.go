package gateway

const (
	MethodBankTransfer = "bank_transfer"
	MethodQRIS         = "qris"
	MethodGopay        = "gopay"
	MethodCreditCard   = "credit_card"
	MethodCrypto       = "crypto"
	MethodOther        = "other"
)

// IDs lists every routable provider-neutral method id, oldest first.
// MethodOther is excluded: it only labels unmapped provider notifications and
// is never something a client may request.
func IDs() []string {
	return []string{
		MethodBankTransfer, MethodQRIS, MethodGopay,
		MethodCreditCard, MethodCrypto,
	}
}

// MethodName maps a stable method ID to its display label.
func MethodName(method string) string {
	labels := map[string]string{MethodBankTransfer: "Bank Transfer", MethodQRIS: "QRIS", MethodGopay: "GoPay", MethodCreditCard: "Credit Card", MethodCrypto: "Cryptocurrency", MethodOther: "Other"}
	if label, ok := labels[method]; ok {
		return label
	}
	return method
}
