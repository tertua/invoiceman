package midtrans

import (
	"strings"

	"github.com/tertua/tupay/platform/gateway"
)

var paymentTypes = map[string]string{
	"bank_transfer": gateway.MethodBankTransfer,
	"bca":           gateway.MethodBankTransfer, "bni": gateway.MethodBankTransfer,
	"bri": gateway.MethodBankTransfer, "echannel": gateway.MethodBankTransfer,
	"permata":     gateway.MethodBankTransfer,
	"qris":        gateway.MethodQRIS,
	"gopay":       gateway.MethodGopay,
	"credit_card": gateway.MethodCreditCard,
}

func (Gateway) Methods() []string {
	return []string{
		gateway.MethodBankTransfer, gateway.MethodQRIS, gateway.MethodGopay,
		gateway.MethodCreditCard,
	}
}

func StandardizePaymentType(raw string) string {
	if method, ok := paymentTypes[strings.ToLower(strings.TrimSpace(raw))]; ok {
		return method
	}
	return gateway.MethodOther
}

// snapPaymentType maps a neutral method id to the code Midtrans Snap accepts
// for enabled_payments. Snap has no "qris" code: QRIS is surfaced inside the
// GoPay method ("QRIS by GoPay"), so the neutral qris method maps to gopay
// for Snap. The on-page QRIS path uses the Core API directly instead.
func snapPaymentType(method string) string {
	switch method {
	case gateway.MethodBankTransfer:
		return "bank_transfer"
	case gateway.MethodQRIS:
		return "gopay"
	case gateway.MethodGopay:
		return "gopay"
	case gateway.MethodCreditCard:
		return "credit_card"
	default:
		return ""
	}
}
