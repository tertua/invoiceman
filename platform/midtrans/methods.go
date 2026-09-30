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

// ChargeCurrency reports that Midtrans only charges in IDR; callers convert
// invoice amounts to IDR before creating a charge.
func (Gateway) ChargeCurrency() string { return gateway.FiatIDR }

// DefaultMethods narrows a Midtrans charge to QRIS only: the owner's account
// offers QRIS exclusively, so the checkout must never present another method.
// Declared here (not enforced by the controller) so the narrowing travels with
// the provider.
func (Gateway) DefaultMethods() []string { return []string{gateway.MethodQRIS} }

// RequiresDecimalAmount reports that Midtrans bills whole rupiah, so no
// decimal amount is needed in the request.
func (Gateway) RequiresDecimalAmount() bool { return false }

// BrowserSDK reports that the stored token is a Snap token, meant for the
// Midtrans Snap browser SDK.
func (Gateway) BrowserSDK() bool { return true }

// PayerConfig returns the browser-safe Midtrans settings the pay page needs:
// the public client key, the environment flag, and the checkout mode the
// browser must open. The server key is a secret and is never exposed here.
//
// "checkout": "snap" is a provider-declared value: the pay page dispatches on
// it (webui/src/lib/payerCheckout.js) instead of on a provider name, so a
// provider without a browser SDK simply omits the key and the page falls back
// to its hosted redirect_url/payment_url. client_key stays for older cached
// bundles that still gate on it.
func (Gateway) PayerConfig() map[string]any {
	cfg := FromEnv()
	return map[string]any{
		"checkout":      "snap",
		"client_key":    cfg.ClientKey,
		"is_production": cfg.IsProd,
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
