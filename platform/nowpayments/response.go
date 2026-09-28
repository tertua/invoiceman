package nowpayments

import (
	"github.com/tertua/tupay/platform/gateway"
)

func nowPaymentsResponse(inv *Invoice) *gateway.CreateTxResponse {
	return &gateway.CreateTxResponse{Token: inv.ID, PaymentURL: inv.InvoiceURL, PaymentMethod: gateway.MethodCrypto}
}
