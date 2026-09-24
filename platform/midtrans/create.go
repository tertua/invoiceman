package midtrans

import (
	"context"
	"errors"

	"github.com/tertua/invoiceman/platform/gateway"
)

func (Gateway) CreateTransaction(ctx context.Context, req *gateway.CreateTxRequest) (*gateway.CreateTxResponse, error) {
	methods := SnapMethods(req.EnabledMethods)
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

// SnapMethods translates provider-neutral method ids into the Snap
// enabled_payments codes, dropping any id Midtrans cannot express. An empty
// input yields nil so Snap keeps its full default list.
func SnapMethods(methods []string) []string {
	if len(methods) == 0 {
		return nil
	}
	out := make([]string, 0, len(methods))
	for _, method := range methods {
		if code := snapPaymentType(method); code != "" {
			out = append(out, code)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
