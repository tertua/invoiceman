package nowpayments

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/tertua/tupay/platform/gateway"
)

// Money must never round-trip through float64: the classic 0.1+0.2 case has
// to stay exact when amounts are decoded from JSON and priced to the provider.
func TestToDecimalKeepsExactPrecision(t *testing.T) {
	first, err := toDecimal(json.Number("0.1"))
	if err != nil {
		t.Fatalf("toDecimal(0.1): %v", err)
	}
	second, err := toDecimal(json.Number("0.2"))
	if err != nil {
		t.Fatalf("toDecimal(0.2): %v", err)
	}
	if got := first.Add(second); !got.Equal(decimal.RequireFromString("0.3")) {
		t.Fatalf("0.1+0.2 = %s, want 0.3", got)
	}
}

func TestPriceFromRequestPreservesDecimalString(t *testing.T) {
	amount, currency, err := priceFromRequest(&gateway.CreateTxRequest{
		OrderID:       "INV-1",
		AmountDecimal: "0.30000000000000004",
		Currency:      "usd",
	})
	if err != nil {
		t.Fatalf("priceFromRequest: %v", err)
	}
	if currency != "USD" {
		t.Fatalf("currency = %q, want USD", currency)
	}
	body, err := json.Marshal(json.Number(amount.String()))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(body) != "0.30000000000000004" {
		t.Fatalf("price_amount = %s, want 0.30000000000000004", body)
	}
}

func TestPriceMinorRoundsWithoutFloat(t *testing.T) {
	for _, tc := range []struct {
		amount   string
		currency string
		want     int64
	}{
		{"150000.4", "IDR", 150000},
		{"150000.5", "IDR", 150001},
		{"25.50", "USD", 0},
	} {
		got := priceMinor(decimal.RequireFromString(tc.amount), tc.currency)
		if got != tc.want {
			t.Errorf("priceMinor(%s, %s) = %d, want %d", tc.amount, tc.currency, got, tc.want)
		}
	}
}
