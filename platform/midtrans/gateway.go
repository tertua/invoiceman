package midtrans

import "github.com/tertua/invoiceman/platform/gateway"

// GatewayName is the registry name for Midtrans.
const GatewayName = "midtrans"

// Gateway implements gateway.Gateway for Midtrans Snap + notifications.
type Gateway struct{}

// Name returns the registry name.
func (Gateway) Name() string { return GatewayName }

// Configured reports whether Midtrans credentials are present.
func (Gateway) Configured() bool { return FromEnv().ServerKey != "" }

// Sandbox reports whether Midtrans runs against its sandbox environment.
func (Gateway) Sandbox() bool { return !FromEnv().IsProd }

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
		PaymentMethod: StandardizePaymentType(notif.PaymentType),
		GrossMinor:    notif.GrossAmountValue().IntPart(),
		GrossDecimal:  notif.GrossAmount,
		Currency:      "IDR",
	}, nil
}
