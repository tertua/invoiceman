package controllers

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/tertua/invoiceman/platform/gateway"
)

type checkableGateway struct {
	fakeGateway
	min    decimal.Decimal
	minErr error
}

func (g checkableGateway) Methods() []string { return []string{gateway.MethodCrypto} }
func (g checkableGateway) MinAmount(context.Context, string) (decimal.Decimal, error) {
	return g.min, g.minErr
}

func TestBelowLiveMinimum(t *testing.T) {
	ctx := context.Background()
	spec := chargeSpec{AmountDecimal: "3.08", Currency: "USD"}
	below := checkableGateway{fakeGateway: fakeGateway{name: "nowpayments"}, min: decimal.RequireFromString("18.81")}
	if !belowLiveMinimum(ctx, below, gateway.MethodCrypto, spec) {
		t.Fatal("3.08 under an 18.81 minimum must be filtered")
	}
	above := checkableGateway{fakeGateway: fakeGateway{name: "nowpayments"}, min: decimal.RequireFromString("1.00")}
	if belowLiveMinimum(ctx, above, gateway.MethodCrypto, spec) {
		t.Fatal("3.08 over a 1.00 minimum must stay visible")
	}
	if belowLiveMinimum(ctx, below, gateway.MethodBankTransfer, spec) {
		t.Fatal("fiat methods have no provider minimum and must stay visible")
	}
	broken := checkableGateway{fakeGateway: fakeGateway{name: "nowpayments"}, minErr: errors.New("boom")}
	if belowLiveMinimum(ctx, broken, gateway.MethodCrypto, spec) {
		t.Fatal("a minimum-fetch failure must fail open (keep the method)")
	}
	if belowLiveMinimum(ctx, fakeGateway{name: "midtrans"}, gateway.MethodCrypto, spec) {
		t.Fatal("providers without a checker must never filter")
	}
}
