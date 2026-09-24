package gateway

const (
	MethodBankTransfer  = "bank_transfer"
	MethodQRIS          = "qris"
	MethodGopay         = "gopay"
	MethodConvenience   = "convenience_store"
	MethodCreditCard    = "credit_card"
	MethodCrypto        = "crypto"
	MethodAkulaku       = "akulaku"
	MethodKlikBca       = "klik_bca"
	MethodBcaKlikpay    = "bca_klikpay"
	MethodCimbClicks    = "cimb_clicks"
	MethodDanamonOnline = "danamon_online"
	MethodOther         = "other"
)

// IDs lists every routable provider-neutral method id, oldest first.
// MethodOther is excluded: it only labels unmapped provider notifications and
// is never something a client may request.
func IDs() []string {
	return []string{
		MethodBankTransfer, MethodQRIS, MethodGopay, MethodConvenience,
		MethodCreditCard, MethodCrypto, MethodAkulaku, MethodKlikBca,
		MethodBcaKlikpay, MethodCimbClicks, MethodDanamonOnline,
	}
}

func MethodName(method string) string {
	labels := map[string]string{MethodBankTransfer: "Bank Transfer", MethodQRIS: "QRIS", MethodGopay: "GoPay", MethodConvenience: "Convenience Store", MethodCreditCard: "Credit Card", MethodCrypto: "Cryptocurrency", MethodAkulaku: "Akulaku", MethodKlikBca: "KlikBCA", MethodBcaKlikpay: "BCA KlikPay", MethodCimbClicks: "CIMB Clicks", MethodDanamonOnline: "Danamon Online Banking", MethodOther: "Other"}
	if label, ok := labels[method]; ok {
		return label
	}
	return method
}
