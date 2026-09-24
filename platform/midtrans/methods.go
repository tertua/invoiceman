package midtrans

import (
	"strings"

	"github.com/tertua/invoiceman/platform/gateway"
)

var paymentTypes = map[string]string{
	"bank_transfer": gateway.MethodBankTransfer,
	"bca":           gateway.MethodBankTransfer, "bni": gateway.MethodBankTransfer,
	"bri": gateway.MethodBankTransfer, "echannel": gateway.MethodBankTransfer,
	"permata": gateway.MethodBankTransfer,
	"qris":    gateway.MethodQRIS,
	"gopay":   gateway.MethodGopay,
	"cstore":  gateway.MethodConvenience, "indomaret": gateway.MethodConvenience,
	"alfamart":       gateway.MethodConvenience,
	"credit_card":    gateway.MethodCreditCard,
	"akulaku":        gateway.MethodAkulaku,
	"klik_bca":       gateway.MethodKlikBca,
	"bca_klikpay":    gateway.MethodBcaKlikpay,
	"cimb_clicks":    gateway.MethodCimbClicks,
	"danamon_online": gateway.MethodDanamonOnline,
}

func (Gateway) Methods() []string {
	return []string{
		gateway.MethodBankTransfer, gateway.MethodQRIS, gateway.MethodGopay,
		gateway.MethodConvenience, gateway.MethodCreditCard, gateway.MethodAkulaku,
		gateway.MethodKlikBca, gateway.MethodBcaKlikpay, gateway.MethodCimbClicks,
		gateway.MethodDanamonOnline,
	}
}

func StandardizePaymentType(raw string) string {
	if method, ok := paymentTypes[strings.ToLower(strings.TrimSpace(raw))]; ok {
		return method
	}
	return gateway.MethodOther
}

func snapPaymentType(method string) string {
	switch method {
	case gateway.MethodBankTransfer:
		return "bank_transfer"
	case gateway.MethodQRIS:
		return "qris"
	case gateway.MethodGopay:
		return "gopay"
	case gateway.MethodConvenience:
		return "cstore"
	case gateway.MethodCreditCard:
		return "credit_card"
	case gateway.MethodAkulaku:
		return "akulaku"
	case gateway.MethodKlikBca:
		return "klik_bca"
	case gateway.MethodBcaKlikpay:
		return "bca_klikpay"
	case gateway.MethodCimbClicks:
		return "cimb_clicks"
	case gateway.MethodDanamonOnline:
		return "danamon_online"
	default:
		return ""
	}
}
