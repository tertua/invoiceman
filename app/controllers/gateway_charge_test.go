package controllers

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/tertua/tupay/platform/gateway"
)

type fakeGateway struct{ name string }

func (f fakeGateway) Name() string { return f.name }
func (f fakeGateway) CreateTransaction(context.Context, *gateway.CreateTxRequest) (*gateway.CreateTxResponse, error) {
	return &gateway.CreateTxResponse{}, nil
}
func (f fakeGateway) ParseAndVerify([]byte) (*gateway.NotificationResult, error) { return nil, nil }

// declaringGateway is a fake with the ChargeCurrencyProvider capability.
type declaringGateway struct {
	fakeGateway
	currency string
}

func (g declaringGateway) ChargeCurrency() string { return g.currency }

// buildCharge converts a USD invoice into IDR for the IDR-only provider.
func TestBuildChargeConvertsForMidtrans(t *testing.T) {
	spec, err := buildCharge(declaringGateway{fakeGateway{name: "midtrans"}, gateway.FiatIDR}, "USD", decimal.RequireFromString("25.50"), decimal.RequireFromString("18000"))
	if err != nil {
		t.Fatalf("buildCharge: %v", err)
	}
	if spec.Currency != gateway.FiatIDR {
		t.Fatalf("currency = %q, want IDR", spec.Currency)
	}
	if spec.AmountMinor != 459000 {
		t.Fatalf("AmountMinor = %d, want 459000", spec.AmountMinor)
	}
	if !spec.UsdToIdr.Equal(decimal.RequireFromString("18000")) {
		t.Fatalf("UsdToIdr = %s, want 18000", spec.UsdToIdr)
	}
	if !spec.InvoiceAmount.Equal(decimal.RequireFromString("25.50")) {
		t.Fatalf("InvoiceAmount = %s, want 25.50", spec.InvoiceAmount)
	}
}

// A USD invoice keeps USD for NOWPayments and needs no rate.
func TestBuildChargeKeepsInvoiceCurrencyForCrypto(t *testing.T) {
	spec, err := buildCharge(declaringGateway{fakeGateway{name: "nowpayments"}, gateway.FiatUSD}, "USD", decimal.RequireFromString("25.50"), decimal.Zero)
	if err != nil {
		t.Fatalf("buildCharge: %v", err)
	}
	if spec.Currency != "USD" || spec.AmountDecimal != "25.5" {
		t.Fatalf("unexpected spec: %+v", spec)
	}
	if !spec.UsdToIdr.IsZero() {
		t.Fatalf("UsdToIdr = %s, want 0", spec.UsdToIdr)
	}
}

// An IDR invoice is converted to USD for NOWPayments with the manual rate:
// the hosted checkout rejects IDR, so it must never be sent as-is.
func TestBuildChargeConvertsIDRToUSDForCrypto(t *testing.T) {
	spec, err := buildCharge(declaringGateway{fakeGateway{name: "nowpayments"}, gateway.FiatUSD}, "IDR", decimal.RequireFromString("222000"), decimal.RequireFromString("18000"))
	if err != nil {
		t.Fatalf("buildCharge: %v", err)
	}
	if spec.Currency != gateway.FiatUSD {
		t.Fatalf("currency = %q, want USD", spec.Currency)
	}
	want := decimal.RequireFromString("222000").Div(decimal.RequireFromString("18000"))
	if got, _ := decimal.NewFromString(spec.AmountDecimal); !got.Equal(want) {
		t.Fatalf("AmountDecimal = %s, want %s", spec.AmountDecimal, want)
	}
	if !spec.UsdToIdr.Equal(decimal.RequireFromString("18000")) {
		t.Fatalf("UsdToIdr = %s, want 18000", spec.UsdToIdr)
	}
	if !spec.InvoiceAmount.Equal(decimal.RequireFromString("222000")) {
		t.Fatalf("InvoiceAmount = %s, want 222000", spec.InvoiceAmount)
	}
}

// A provider without the capability charges in the invoice currency.
func TestBuildChargeUsesInvoiceCurrencyWithoutCapability(t *testing.T) {
	spec, err := buildCharge(fakeGateway{name: "unknownpay"}, "EUR", decimal.RequireFromString("10.00"), decimal.Zero)
	if err != nil {
		t.Fatalf("buildCharge: %v", err)
	}
	if spec.Currency != "EUR" {
		t.Fatalf("currency = %q, want EUR", spec.Currency)
	}
	if !spec.UsdToIdr.IsZero() {
		t.Fatalf("UsdToIdr = %s, want 0", spec.UsdToIdr)
	}
}

// Missing rate for a cross-currency charge is a hard error, never a silent guess.
func TestBuildChargeRequiresRateForCrossCurrency(t *testing.T) {
	if _, err := buildCharge(declaringGateway{fakeGateway{name: "midtrans"}, gateway.FiatIDR}, "USD", decimal.RequireFromString("25"), decimal.Zero); err == nil {
		t.Fatal("expected error, got nil")
	}
	if _, err := buildCharge(declaringGateway{fakeGateway{name: "nowpayments"}, gateway.FiatUSD}, "IDR", decimal.RequireFromString("222000"), decimal.Zero); err == nil {
		t.Fatal("expected error for IDR crypto without rate, got nil")
	}
}
