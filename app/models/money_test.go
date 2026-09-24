package models

import (
	"testing"
	"time"

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
	if got := ResolveEffectiveStatus(InvoiceStatusSent, nil, total, decimal.RequireFromString("99.9999"), false); got != InvoiceStatusSent {
		t.Fatalf("underpaid invoice marked %s", got)
	}
	if got := ResolveEffectiveStatus(InvoiceStatusSent, nil, total, total, false); got != InvoiceStatusPaid {
		t.Fatalf("exactly paid invoice marked %s", got)
	}
}

func TestResolveEffectiveStatusPending(t *testing.T) {
	total := decimal.RequireFromString("100.0000")
	overdue := time.Now().Add(-48 * time.Hour)
	if got := ResolveEffectiveStatus(InvoiceStatusSent, nil, total, decimal.Zero, true); got != InvoiceEffectivePending {
		t.Fatalf("awaiting-payment invoice marked %s", got)
	}
	if got := ResolveEffectiveStatus(InvoiceStatusSent, &overdue, total, decimal.Zero, true); got != InvoiceEffectivePending {
		t.Fatalf("pending beats overdue, got %s", got)
	}
	if got := ResolveEffectiveStatus(InvoiceStatusPaid, nil, total, total, true); got != InvoiceStatusPaid {
		t.Fatalf("paid beats pending, got %s", got)
	}
	if got := ResolveEffectiveStatus(InvoiceStatusDraft, nil, total, decimal.Zero, false); got != InvoiceStatusDraft {
		t.Fatalf("draft without live transaction marked %s", got)
	}
}
