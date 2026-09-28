// Package nowpayments implements gateway.Gateway for NOWPayments crypto
// invoices (https://api.nowpayments.io/v1) plus IPN verification.
//
// Design notes:
//   - Invoice creation is hosted-checkout shaped: the customer pays on
//     invoice_url, so the result maps to CreateTxResponse.PaymentURL.
//   - Direct USDT (TRC20) payments skip the hosted checkout and return an
//     on-page deposit address instead of a redirect.
//   - IPN authenticity is verified by fetching the payment status with the
//     API key ("API fetch" in the Gateway contract) instead of the
//     x-nowpayments-sig header, which never reaches ParseAndVerify. The
//     fetched payment_id and order_id must match the callback body.
//   - NOWPayments never sends callbacks after expiry; expiry is observed
//     when the next status fetch (or a fresh callback) reports it.
package nowpayments

import (
	"context"

	"github.com/tertua/tupay/platform/gateway"
)

// GatewayName is the registry name for NOWPayments.
const GatewayName = "nowpayments"

// Gateway implements gateway.Gateway for NOWPayments invoices + IPN.
type Gateway struct{}

// Name returns the registry name.
func (Gateway) Name() string { return GatewayName }

// ParseAndVerify decodes and verifies a NOWPayments IPN body.
func (Gateway) ParseAndVerify(raw []byte) (*gateway.NotificationResult, error) {
	cfg := FromEnv()
	if cfg.APIKey == "" {
		return nil, gateway.ErrNotConfigured
	}
	return VerifyNotification(context.Background(), cfg, raw)
}
