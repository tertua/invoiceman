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

// MethodName maps a stable method ID to its display label. NOTE (intentional,
// not a typo): MethodGopay displays as "QRIS" and MethodQRIS displays as
// "Other QRIS" per owner request; the IDs ("gopay"/"qris") stay unchanged for
// Midtrans Snap, allowlists, and the API.
func MethodName(method string) string {
	labels := map[string]string{MethodBankTransfer: "Bank Transfer", MethodQRIS: "Other QRIS", MethodGopay: "QRIS", MethodCreditCard: "Credit Card", MethodCrypto: "Cryptocurrency", MethodOther: "Other"}
	if label, ok := labels[method]; ok {
		return label
	}
	return method
}
