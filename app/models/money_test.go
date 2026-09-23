package models

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestMoneyAvoidsBinaryFloatRounding(t *testing.T) {
	got := decimal.RequireFromString("0.1").Add(decimal.RequireFromString("0.2"))
	if !got.Equal(decimal.RequireFromString("0.3")) {
		t.Fatalf("expected 0.3, got %s", got)
	}
}

func TestResolveEffectiveStatusUsesExactThreshold(t *testing.T) {
	total := decimal.RequireFromString("100.0000")
	if got := ResolveEffectiveStatus(InvoiceStatusSent, nil, total, decimal.RequireFromString("99.9999")); got != InvoiceStatusSent {
		t.Fatalf("underpaid invoice marked %s", got)
	}
	if got := ResolveEffectiveStatus(InvoiceStatusSent, nil, total, total); got != InvoiceStatusPaid {
		t.Fatalf("exactly paid invoice marked %s", got)
	}
}
