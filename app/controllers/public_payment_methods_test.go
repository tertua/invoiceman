package controllers

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/tertua/tupay/platform/gateway"
)

type checkableGateway struct {
	fakeGateway
	min       decimal.Decimal
	minErr    error
	seenAsset *string
}

func (g checkableGateway) Methods() []string { return []string{gateway.MethodCrypto} }
func (g checkableGateway) MinAmount(_ context.Context, _, payCurrency string) (decimal.Decimal, error) {
	if g.seenAsset != nil {
		*g.seenAsset = payCurrency
	}
	return g.min, g.minErr
}

func TestBelowLiveMinimum(t *testing.T) {
	ctx := context.Background()
	spec := chargeSpec{AmountDecimal: "3.08", Currency: "USD"}
	// The minimum asked for belongs to the asset the payer picked, never to a
	// hard-coded default.
	var asked string
	below := checkableGateway{fakeGateway: fakeGateway{name: "nowpayments"}, min: decimal.RequireFromString("18.81"), seenAsset: &asked}
	if !belowLiveMinimum(ctx, below, gateway.MethodCrypto, spec, "usdttrc20") {
		t.Fatal("3.08 under an 18.81 minimum must be filtered")
	}
	if asked != "usdttrc20" {
		t.Fatalf("minimum asked for asset %q, want usdttrc20", asked)
	}
	above := checkableGateway{fakeGateway: fakeGateway{name: "nowpayments"}, min: decimal.RequireFromString("1.00")}
	if belowLiveMinimum(ctx, above, gateway.MethodCrypto, spec, "ltc") {
		t.Fatal("3.08 over a 1.00 minimum must stay visible")
	}
	if belowLiveMinimum(ctx, below, gateway.MethodBankTransfer, spec, "usdttrc20") {
		t.Fatal("fiat methods have no provider minimum and must stay visible")
	}
	// No asset picked yet: no asset's minimum applies, so crypto is offered
	// and gated only once the payer chooses one.
	asked = ""
	unpicked := checkableGateway{fakeGateway: fakeGateway{name: "nowpayments"}, min: decimal.RequireFromString("18.81"), seenAsset: &asked}
	if belowLiveMinimum(ctx, unpicked, gateway.MethodCrypto, spec, "") {
		t.Fatal("an unpicked asset must not filter the method list")
	}
	if asked != "" {
		t.Fatalf("no asset chosen, but a minimum was fetched for %q", asked)
	}
	broken := checkableGateway{fakeGateway: fakeGateway{name: "nowpayments"}, minErr: errors.New("boom")}
	if belowLiveMinimum(ctx, broken, gateway.MethodCrypto, spec, "usdttrc20") {
		t.Fatal("a minimum-fetch failure must fail open (keep the method)")
	}
	if belowLiveMinimum(ctx, fakeGateway{name: "midtrans"}, gateway.MethodCrypto, spec, "usdttrc20") {
		t.Fatal("providers without a checker must never filter")
	}
}
