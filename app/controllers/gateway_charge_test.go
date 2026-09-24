package controllers

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/tertua/invoiceman/platform/gateway"
)

type fakeGateway struct{ name string }

func (f fakeGateway) Name() string { return f.name }
func (f fakeGateway) CreateTransaction(context.Context, *gateway.CreateTxRequest) (*gateway.CreateTxResponse, error) {
	return &gateway.CreateTxResponse{}, nil
}
func (f fakeGateway) ParseAndVerify([]byte) (*gateway.NotificationResult, error) { return nil, nil }

// buildCharge converts a USD invoice into IDR for the IDR-only provider.
func TestBuildChargeConvertsForMidtrans(t *testing.T) {
	spec, err := buildCharge(fakeGateway{name: "midtrans"}, "USD", decimal.RequireFromString("25.50"), decimal.RequireFromString("18000"))
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

// A non-IDR provider keeps the invoice currency and needs no rate.
func TestBuildChargeKeepsInvoiceCurrencyForCrypto(t *testing.T) {
	spec, err := buildCharge(fakeGateway{name: "nowpayments"}, "USD", decimal.RequireFromString("25.50"), decimal.Zero)
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

// Missing rate for a cross-currency charge is a hard error, never a silent guess.
func TestBuildChargeRequiresRateForCrossCurrency(t *testing.T) {
	if _, err := buildCharge(fakeGateway{name: "midtrans"}, "USD", decimal.RequireFromString("25"), decimal.Zero); err == nil {
		t.Fatal("expected error, got nil")
	}
}
