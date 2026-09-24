package midtrans

import (
	"context"
	"errors"

	"github.com/tertua/invoiceman/platform/gateway"
)

func (Gateway) CreateTransaction(ctx context.Context, req *gateway.CreateTxRequest) (*gateway.CreateTxResponse, error) {
	methods := req.EnabledMethods
	if method := snapPaymentType(req.PaymentMethod); method != "" {
		methods = []string{method}
	}
	snap, err := CreateSnapTransaction(ctx, FromEnv(), req.OrderID, req.AmountMinor, req.Email, req.Phone, methods)
	if err != nil {
		if errors.Is(err, ErrNotConfigured) {
			return nil, gateway.ErrNotConfigured
		}
		return nil, err
	}
	return &gateway.CreateTxResponse{Token: snap.Token, RedirectURL: snap.RedirectURL, PaymentMethod: req.PaymentMethod}, nil
}
