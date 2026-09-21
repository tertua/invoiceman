package midtrans

import (
	"context"
	"errors"

	"github.com/tertua/invoiceman/platform/gateway"
)

// GatewayName is the registry name for Midtrans.
const GatewayName = "midtrans"

// Gateway implements gateway.Gateway for Midtrans Snap + notifications.
type Gateway struct{}

// Name returns the registry name.
func (Gateway) Name() string { return GatewayName }

// CreateTransaction creates a Snap transaction via the gateway-agnostic request.
func (Gateway) CreateTransaction(ctx context.Context, req *gateway.CreateTxRequest) (*gateway.CreateTxResponse, error) {
	cfg := FromEnv()
	snap, err := CreateSnapTransaction(ctx, cfg, req.OrderID, req.AmountMinor, req.Email, req.Phone)
	if err != nil {
		if errors.Is(err, ErrNotConfigured) {
			return nil, gateway.ErrNotConfigured
		}
		return nil, err
	}
	return &gateway.CreateTxResponse{Token: snap.Token, RedirectURL: snap.RedirectURL}, nil
}

// ParseAndVerify decodes and verifies a Midtrans notification body.
func (Gateway) ParseAndVerify(raw []byte) (*gateway.NotificationResult, error) {
	notif, err := ParseNotification(raw)
	if err != nil {
		return nil, err
	}
	cfg := FromEnv()
	if cfg.ServerKey == "" {
		return nil, gateway.ErrNotConfigured
	}
	if !VerifySignature(notif, cfg.ServerKey) {
		return nil, gateway.ErrInvalidSignature
	}
	return &gateway.NotificationResult{
		OrderID:       notif.OrderID,
		TransactionID: notif.TransactionID,
		Status:        MapStatus(notif.TransactionStatus),
		PaymentType:   notif.PaymentType,
		GrossMinor:    int64(notif.GrossAmountValue() + 0.5),
		GrossDecimal:  notif.GrossAmount,
		Currency:      "IDR",
	}, nil
}
