package gateway

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
)

func TestConvertSameCurrencyIsNoop(t *testing.T) {
	amount := decimal.RequireFromString("25.50")
	got, err := Convert(amount, "idr", "IDR", decimal.Zero)
	if err != nil || !got.Equal(amount) {
		t.Fatalf("Convert() = %s, %v; want %s", got, err, amount)
	}
}

func TestConvertUSDToIDR(t *testing.T) {
	got, err := Convert(decimal.RequireFromString("25.50"), "USD", "IDR", decimal.RequireFromString("18000"))
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if !got.Equal(decimal.RequireFromString("459000")) {
		t.Fatalf("got %s, want 459000", got)
	}
}

func TestConvertIDRToUSD(t *testing.T) {
	got, err := Convert(decimal.RequireFromString("459000"), "IDR", "USD", decimal.RequireFromString("18000"))
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if !got.Equal(decimal.RequireFromString("25.5")) {
		t.Fatalf("got %s, want 25.5", got)
	}
}

func TestConvertRequiresRate(t *testing.T) {
	if _, err := Convert(decimal.RequireFromString("10"), "USD", "IDR", decimal.Zero); !errors.Is(err, ErrUnsupportedCurrency) {
		t.Fatalf("expected ErrUnsupportedCurrency, got %v", err)
	}
}

func TestConvertRejectsUnknownPair(t *testing.T) {
	if _, err := Convert(decimal.RequireFromString("10"), "EUR", "IDR", decimal.RequireFromString("18000")); !errors.Is(err, ErrUnsupportedCurrency) {
		t.Fatalf("expected ErrUnsupportedCurrency, got %v", err)
	}
}

func TestSettleAmountFallsBackWithoutRate(t *testing.T) {
	raw := decimal.RequireFromString("459000")
	got := SettleAmount(raw, "IDR", "USD", decimal.Zero)
	if !got.Equal(raw) {
		t.Fatalf("SettleAmount without rate = %s, want raw %s", got, raw)
	}
}
