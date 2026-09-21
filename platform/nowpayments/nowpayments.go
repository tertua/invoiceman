// Package nowpayments implements gateway.Gateway for NOWPayments crypto
// invoices (https://api.nowpayments.io/v1) plus IPN verification.
//
// Design notes:
//   - Invoice creation is hosted-checkout shaped: the customer pays on
//     invoice_url, so the result maps to CreateTxResponse.PaymentURL.
//   - IPN authenticity is verified by fetching the payment status with the
//     API key ("API fetch" in the Gateway contract) instead of the
//     x-nowpayments-sig header, which never reaches ParseAndVerify. The
//     fetched payment_id and order_id must match the callback body.
//   - NOWPayments never sends callbacks after expiry; expiry is observed
//     when the next status fetch (or a fresh callback) reports it.
package nowpayments

import (
	"context"
	"errors"

	"github.com/tertua/invoiceman/platform/gateway"
)

// GatewayName is the registry name for NOWPayments.
const GatewayName = "nowpayments"

// ErrNotConfigured is returned when the NOWPayments API key is missing.
var ErrNotConfigured = errors.New("payment gateway is not configured")

// Gateway implements gateway.Gateway for NOWPayments invoices + IPN.
type Gateway struct{}

// Name returns the registry name.
func (Gateway) Name() string { return GatewayName }

// CreateTransaction creates a NOWPayments hosted invoice.
func (Gateway) CreateTransaction(ctx context.Context, req *gateway.CreateTxRequest) (*gateway.CreateTxResponse, error) {
	cfg := FromEnv()
	inv, err := CreateInvoice(ctx, cfg, req)
	if err != nil {
		if errors.Is(err, ErrNotConfigured) {
			return nil, gateway.ErrNotConfigured
		}
		return nil, err
	}
	return &gateway.CreateTxResponse{Token: inv.ID, PaymentURL: inv.InvoiceURL}, nil
}

// ParseAndVerify decodes and verifies a NOWPayments IPN body.
func (Gateway) ParseAndVerify(raw []byte) (*gateway.NotificationResult, error) {
	cfg := FromEnv()
	if cfg.APIKey == "" {
		return nil, gateway.ErrNotConfigured
	}
	return VerifyNotification(context.Background(), cfg, raw)
}
